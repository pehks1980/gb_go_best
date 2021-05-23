package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/html"
)

/*
1. Доработать программу из практической части так, чтобы при отправке ей сигнала SIGUSR1 она увеличивала глубину поиска на 10.
kill -10 pid
cделано но лучше смотреть через дибаггер программа печатает много инфы

go run crawler.go -url http://google.com -depth 4 > output 2>&1
jobs -l
почему то в файле невозможно найти вывод команды что maxDepth был изменен, хотя по дебаггу оно появляется.

2. Добавить общий таймаут на выполнение следующих операций: работа парсера, получений ссылок со страницы, формирование заголовка.
*/
// Объект который имеет все необходимые поля и методы для поиска
type VisitedPages struct {
	sync.Mutex
	wg *sync.WaitGroup
	// карта с url которые мы уже обработали
	visited         map[string]string
	depthFromParent map[string]int
	maxDepth        int
}

var (
	shLink bool = false
)

// конструктор структуры
func NewVisitedPages(maxDepth int) *VisitedPages {
	return &VisitedPages{
		visited:         make(map[string]string),
		depthFromParent: make(map[string]int),
		maxDepth:        maxDepth,
		wg:              new(sync.WaitGroup),
	}
}

// метод структуры - рекурсивно сканируем страницы
func (visitedPages *VisitedPages) analize(ctx context.Context, // упрвление контекстом
	url, // ссылка
	baseurl string, // базовый адрес
	urlDepth int,
	resultChan chan string, // ссылка на канал занесения результатов
	errChan chan error) { // ссылка на канал занесения ошибок
	// При использовании синхронизаций при помощи каналов или
	// вэйтгруп лучше как можно раньше объявлять отложенные вызовы (defer)
	// Это может не раз спасти вас, ведь даже когда где-то внутри неожиданно
	// возникнет паника по - мы всё равно сможем отработать более-менее корректно
	// в конце безусловное завершение wg
	defer visitedPages.wg.Done()

	// проверяем что контекст исполнения актуален
	select {
	case <-ctx.Done():
		// управление - досрочное завершения
		errChan <- fmt.Errorf("cancel analize page %s", url)
		return
	default:
		// проверка глубины
		// если она больше какого то макс значения - выход
		// иначе увеличение глубины на 1
		// похоже что выниесено в отдельную ф. т.к. там лок анлок при работе с структурой этой
		// внимание! здесь селект не блокирует горутину т.к. есть default
		if visitedPages.isMaxDepth(url) == 1 || visitedPages.isMaxDepth(url) == -1 {
			return
		}

		visitedPages.Lock()
		fmt.Printf("Пытаемся открыть ссылку: %s (уровень %d)\n", url, visitedPages.depthFromParent[url])
		visitedPages.Unlock()
		// open link (visit)

		page, err := pingParse(url, 30)

		if err != nil {
			// ошибку отправляем в канал ошибок, а не обрабатываем на месте
			errChan <- fmt.Errorf("error when getting page %s: %s", url, err)
			return
		}
		title, _ := pingPageTitle(page, 5) // тайтл странички

		links, _ := pingPageLinks(nil, page, 5) //мапа линков со странички

		// блокировка требуется, т.к. мы модифицируем карту объекта в рекурсии с новыми гоуртинами
		visitedPages.Lock() // блокировка ставится здесь т.к. в if тоже идет считывание структуры
		// проверяем что другая гошка уже не посешала эту url

		if visitedPages.visited[url] == "" {
			// если нет, то вводим в посещенные эту сцылку
			visitedPages.visited[url] = title
			resultChan <- fmt.Sprintf("visited link %s -> %s\n", url, title)
		}

		visitedPages.Unlock()

		// рекурсивно ищем ссылки
		// cпускаемся на уровень больше(ниже)
		urlDepth++
		baseurlWO := baseurl[7:] // убрать http:// из baseurl
		for link := range links {
			// отфильтровываем только сцылки которые имеют в своем содержимом base название
			if strings.Contains(link, baseurlWO) {
				//прописываем каждой новой ссылке уровень
				visitedPages.Lock() // блокировка ставится здесь т.к. в if тоже идет считывание структуры
				// проверяем что другая гошка уже не посешала эту url
				if _, inMap := visitedPages.visited[link]; !inMap {
					// задаем данные по ссылке - depth, название ""  значит не заходили
					visitedPages.depthFromParent[link] = urlDepth
					visitedPages.visited[link] = ""

					if shLink {
						fmt.Printf("got new link %s (уровень %d)\n", link, visitedPages.depthFromParent[link])
					}

				}
				visitedPages.Unlock()

				visitedPages.checkAndRecurseCall(ctx, link, baseurl, urlDepth, resultChan, errChan)

			}

		}
	}
}

//по сути тот же анализ только он для создания новой гошки обработки каждой сцылки - как
// папки подпапки каждая подпапка будет запускать свою гошку и тд.
func (visitedPages *VisitedPages) checkAndRecurseCall(ctx context.Context,
	newURL, baseurl string, urlDepth int, resultChan chan string, errChan chan error) {
	visitedPages.Lock()
	defer visitedPages.Unlock()
	// если ссылка найдена, то запускаем анализ по новой ссылке
	// если у такой сцылки нет тайтла страницы (получается тайтл страницы как флаг посешения)
	// и ссылка имеет в составе базовый адреес то запускаем анализ этой сцылки типа как рекурсию но в отдельной
	// гошке
	if visitedPages.visited[newURL] == "" { //&& strings.HasPrefix(newURL, baseurl) {
		visitedPages.wg.Add(1)
		go visitedPages.analize(ctx, newURL, baseurl, urlDepth, resultChan, errChan)
	}
}

// проверяем не слишком ли глубоко мы нырнули
func (visitedPages *VisitedPages) isMaxDepth(url string) int {
	visitedPages.Lock()
	defer visitedPages.Unlock()
	// проверяем сцылки имеющие уровень depth
	if _, inMap := visitedPages.depthFromParent[url]; inMap {
		if visitedPages.depthFromParent[url] >= visitedPages.maxDepth {
			//fmt.Printf("reached maxDepth(%d) %d for link %s\n ",visitedPages.maxDepth, visitedPages.depthFromParent[url], url)
			return 1
		}
		//visitedPages.depthFromParent[url]++
		return 0
	}
	return -1
}

// функция увеличивет порог сканирования глубины на i
func (visitedPages *VisitedPages) updateDepthLev(i int) {
	visitedPages.Lock()
	defer visitedPages.Unlock()
	visitedPages.maxDepth += i
	fmt.Printf("\n\n\n!!!!!!!!!!!!!!!! maxDepth level was increased by %d to %d  !!!!!!!!!!!!!!!!\n\n\n", i, visitedPages.maxDepth)
}

// адрес в интернете
var url string

// насколько глубоко нам надо смотреть
var maxDepthProperty int

func init() {
	// задаём и парсим флаги
	flag.StringVar(&url, "url", "", "url address")
	flag.IntVar(&maxDepthProperty, "depth", 3, "max depth for analize")
	flag.Parse()

	// Проверяем обязательное условие
	if url == "" {
		log.Print("no url set by flag")
		flag.PrintDefaults()
		os.Exit(1)
	}
}

func main() {
	started := time.Now()

	// задаём количество ошибок достигнув которое работа приложения будет прервана
	maxErrorsBeforeChancel := 3

	visitedPages := NewVisitedPages(maxDepthProperty)
	// создаём канал для результатов - внимание буфер 100 сообщений
	resultChan := make(chan string, 100)
	// создаём канал для ошибок
	errChan := make(chan error, 100)
	//нулевой уровень это сам базовый url
	visitedPages.depthFromParent[url] = 0

	// создаём вспомогательные каналы для горутины которая будет вычитывать сообщения
	// из канала с результатами и с ошибками
	shutdownChanForReaders := make(chan struct{})
	readersDone := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())

	go catchOs(cancel, visitedPages)
	// запускаем горутину для чтения из каналов
	go startReaders(cancel, resultChan, errChan, shutdownChanForReaders, readersDone, maxErrorsBeforeChancel)

	// синхронизация окончания обхода анализа страниц через вэйтгруппу
	visitedPages.wg.Add(1)

	// запуск основной логики
	// внутри есть рекурсивные запуски анализа в других горутинах
	go visitedPages.analize(ctx, url, url, 0, resultChan, errChan)

	// дожидаемся когда весь анализ окончен
	visitedPages.wg.Wait()

	// после окончания анализа мы могли успеть обработать не всю информацию из каналов
	// поэтому нам следует сообщить что новых данных не будет
	// конструкция с буферизированные каналами будет выглядеть намного проще,
	// но в нашем случае она не подходит
	shutdownChanForReaders <- struct{}{}

	// хороший тон всегда закрывать каналы, которые точно не будут использованы
	close(errChan)
	close(resultChan)

	// ждём завершения работы чтения в своей горутине
	<-readersDone

	log.Println(time.Since(started))
}

// ловим сигналы выключения
func catchOs(cancel context.CancelFunc, visitedPages *VisitedPages) {
	osSignalChan := make(chan os.Signal)

	signal.Notify(osSignalChan, syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGUSR1)
	for {
		// поточек ждет сигнал в канале
		select {
		case sig := <-osSignalChan:
			log.Printf("got %s signal", sig.String())
			switch sig.String() {
			case "interrupt":
				cancel()
				return
			case "quit":
				cancel()
				return
			case "user defined signal 1":
				visitedPages.updateDepthLev(10)

			}

		}
	}

}

func startReaders(cancel context.CancelFunc,
	resultChan chan string,
	errChan chan error,
	shutdownChanForReaders chan struct{},
	readersDone chan struct{},
	maxErrorsBeforeChancel int) {
	// начинаем цикл чтения из каналов
	var errCount int

	for {
		// порядок внутри выбора важен.
		//Сообщение что пора выключаться придёт только после того как другие каналы будут пустыми
		// вернее когда из каналов с буфером в 100 (первые два) все считается
		// select заблокируется как что придет тот кейс и сработает
		select {
		case result := <-resultChan:
			log.Printf("crawling result: %v", result)
		case err := <-errChan:
			log.Printf("when crawling got error: %v", err)
			errCount++
			if errCount == maxErrorsBeforeChancel {
				cancel()
			}
			// если мы дошли до этой части, значит пора прекращать работу из-за количества ошибок
			// заметьте что мы не обратабывам прерывание контекста в уже полученных данных, т.е. не отбрасываем полученную информацию
		case <-shutdownChanForReaders:
			// отправляем сигнал чтение из каналов прекращено
			readersDone <- struct{}{}
			return
		}
	}
}

type Result struct {
	ResultError error
	TextNode    *html.Node
}

// Пинг функция для Parse
// на входе url, timeout в сек.
// на выходе парсенка htmlки
func pingParse(url string, timeout int) (*html.Node, error) {
	//Создаёт новый контекст, который завершится максимум через секунду
	/*
		создается канал, в который в указанный момент времени 1c должно прийти сообщение об успешном завершении функции.
		Канал закрывается после получения первого сообщения либо после вызова cancelFunc.

	*/
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	// т.е через 1 секунду будет завершена по любому эта функция ping
	defer cancelFunc()
	// канал ошибок
	doneCh := make(chan Result)
	go func(ctx context.Context) {
		//pong это функция проверки сервиса возвращает ответ ошибки и результат,
		//которые пишутся в канал ошибок doneCh
		textNode, err := parse(url)
		resultRead := Result{ResultError: err, TextNode: textNode}
		doneCh <- resultRead

	}(ctx)

	var res Result
	// селект блокирует ф которая завершится по ctx.Done (которая приходит по cancel)
	// второй случай pong вернула какое то значение ошибки или нет ошибок
	// ctx.Err это ошибка контекста - сработал таймаут
	select {
	case <-ctx.Done():
		res.ResultError = ctx.Err()
	case res = <-doneCh:
	}
	return res.TextNode, res.ResultError
}

type ResultTitle struct {
	ResultError   error
	ResultMessage string
}

// Пинг функция для pageTitle
// на входе url, timeout в сек.
// на выходе парсенка htmlки
func pingPageTitle(htmlNode *html.Node, timeout int) (string, error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	// т.е через 1 секунду будет завершена по любому эта функция ping
	defer cancelFunc()
	// канал ошибок
	var res ResultTitle
	doneCh := make(chan ResultTitle)
	go func() {
		resMessage, err := pageTitle(htmlNode)
		resultRead := ResultTitle{ResultError: err, ResultMessage: resMessage}
		doneCh <- resultRead
	}()

	select {
	case <-ctx.Done():
		res.ResultError = ctx.Err()
	case res = <-doneCh:
	}
	return res.ResultMessage, res.ResultError
}

type ResultLinks struct {
	ResultError error
	ResultLinks map[string]struct{}
}

// Пинг функция для pageTitle
// на входе url, timeout в сек.
// на выходе парсенка htmlки
func pingPageLinks(links map[string]struct{}, n *html.Node, timeout int) (map[string]struct{}, error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancelFunc()
	var res ResultLinks
	doneCh := make(chan ResultLinks)
	// no need to pass it here
	go func(ctx context.Context) {
		//pong это функция проверки сервиса возвращает ответ ошибки и результат,
		//которые пишутся в канал ошибок doneCh
		resLinks, err := pageLinks(links, n)
		resultRead := ResultLinks{ResultError: err, ResultLinks: resLinks}
		doneCh <- resultRead

	}(ctx)

	// селект блокирует ф которая завершится по ctx.Done (которая приходит по cancel)
	// второй случай pong вернула какое то значение ошибки или нет ошибок
	// ctx.Err это ошибка контекста - сработал таймаут
	select {
	case <-ctx.Done():
		res.ResultError = ctx.Err()
	case res = <-doneCh:
	}
	return res.ResultLinks, res.ResultError
}

// парсим страницу открываем ее гетом и превращаем в структуру html.node
func parse(url string) (*html.Node, error) {
	r, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("can't get page")
	}
	b, err := html.Parse(r.Body)
	if err != nil {
		return nil, fmt.Errorf("can't parse page")
	}
	return b, err
}

// ищем заголовок на странице
// какой то глюк вызывает панику на google.com / может быть из за рекурсии
func pageTitle(n *html.Node) (string, error) {
	var resTitle string
	var err error
	defer func() {
		if v := recover(); v != nil {
			fmt.Println("Действия после наступления паники ...\n", v)
			err = errors.New("pageTitle panicked!!!")
			//return nil, err
		}
	}()

	if n.Type == html.ElementNode && n.Data == "title" {
		return n.FirstChild.Data, nil
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		title, err := pageTitle(c)
		if title != "" {
			resTitle = title
			break

		}
		if err != nil {
			return "", err
		}
	}
	return resTitle, nil
}

// ищем все уникальные ссылки на страницы.
// Используем карту чтобы избежать дубликатов
func pageLinks(links map[string]struct{}, n *html.Node) (map[string]struct{}, error) {
	var err error
	if links == nil {
		links = make(map[string]struct{})
	}

	if n.Type == html.ElementNode && n.Data == "a" {
		for _, a := range n.Attr {
			if a.Key == "href" {
				// если такой нету в мапе то заводим путем занесения указателя на пустую структуру в мапу
				//с ключем - этой гипер сыцлки
				// фильтруем только сцылки имеющие нормальный url формат
				if _, inMap := links[a.Val]; !inMap {
					if strings.HasPrefix(a.Val, "http://") || strings.HasPrefix(a.Val, "https://") {
						links[a.Val] = struct{}{}
						//fmt.Printf("\ngot new link %s\n", a.Val)
					}

				}
			}
		}
	}
	// рекурсивно заходит в под абзацы страницы и сам себя вызывает и процесс идет снова
	// пока в мапе links не будут все сцылки
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		links, err = pageLinks(links, c)
		if err != nil {
			return nil, err
		}
	}
	return links, nil
}
