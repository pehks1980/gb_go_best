package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/pehks1980/gb_go_best/kurs/app1/config"
	"github.com/pehks1980/gb_go_best/kurs/app1/filter"
	"github.com/pehks1980/gb_go_best/kurs/app1/iter"
	"github.com/pehks1980/gb_go_best/kurs/app1/logger"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)
/*

1. При запуске программа печатает путь до исполняемого файла и версию последнего коммита
2. Программа должна корректно обрабатывать выход по сигналу SIGINT, прерывая поиск, если он запущен
3. Программа должна получать настройки из текстового конфигурационного файла (например, в TOML формате) при старте
4. Программа должна завершать исполнение запроса, если он занимает слишком продолжительное время
(значение таймаута задается в конфигурационном файле)
5 Программа должна логировать все запросы в файл access.log, логировать все ошибки
(например, остановку пользователем или прерывания по таймауту, невалидные запросы пользователя) в error.log
5.1 Код должен быть покрыт тестами (test coverage хотя бы 30%)
6. Код должен быть организован согласно выбранным принципам, например
можно использовать project-layout для вдохновения
7. Должен быть создан конфигурационный файл для golangci-lint
8. При коммите в локальный репозиторий в автоматическом режиме должно
происходить следующее:
a. make test - должен запускать тесты и печатать отчет о coverage
b. make check - должен запускать все линтеры

 */
var (
	//флаги
	debug    = flag.Bool("debug", false, "debug info")
	loglevel = flag.Int("lev", 1, "level of logging 3-Debug, 2-Warning, 1-Info")
	flCols   = flag.String("cols", "", "columns list, separated by comma(not given = all columns)")
)

type Row struct {
	Cols map[string]string
}

func main() {
	//sets for testing files

	basepath := "/Users/user/go/gb/gb_bp/gb_go_best/"

	//args := `third > 0.4 AND first = "www" OR second = "Kesha"`
	//pathfile := basepath + "kurs/app1/test.csv"

	args := `crimedescr = "459 PC  BURGLARY VEHICLE"`
	pathfile := basepath + "kurs/app1/test1.csv"

	// init flag / loggers
	flag.Parse()

	//access.log error.log init
	access_logger, err := logger.InitLogger("access.log", *debug, *loglevel)
	if err != nil {
		log.Fatalf("cannot init access.log. exit ")
	}

	error_logger, err := logger.InitLogger("error.log", *debug, *loglevel)
	if err != nil {
		log.Fatalf("cannot init error.log. exit ")
	}

	access_logger.Info("1 Starting the application...")

	error_logger.Infof("1 Starting the application...args string : %s", args)

	// load config
	c, err := config.New(basepath+"kurs/app1/config/.env")
	if err != nil {
		log.Println(err)
		return
	}
	access_logger.Infof("config struct : %+v", c)

	resulttext, err := pingAnalize(pathfile, args, *flCols, c.Timeout)
	if err != nil {
		log.Printf("AnalizeError: %v",err)
	}

	for _, _ = range resulttext {
		//fmt.Println(str)
	}

	error_logger.Infof("Finished the application..")
}

// ловим сигналы выключения
func catchOs(cancel context.CancelFunc) {
	osSignalChan := make(chan os.Signal)

	signal.Notify(osSignalChan, syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		)
	for {// нужен ли тут фор?
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

			}
		}
	}
}

//main func which makes the job reads and analizes csv
func readAnalizeFile(pathfile string, args string, flcols string) ([]string,error) {

	//check delimeters in file
	delims := iter.CheckDelimiters(pathfile)

	line_del, err := iter.CheckEndLineDelimiters(pathfile)
	if err != nil {
		return nil, err
	}

	fmt.Println(delims)
	reader, err := iter.ReadLinesReadString(pathfile,line_del)
	if err != nil {
		return nil, err
	}

	filerowidx := 0

	var filter *filter.Filter

	var textout []string

	for line := range reader {
		time.Sleep(5*time.Millisecond)
		//fmt.Println(line)
		//line from file remove \n or \r if we have it
		line = strings.TrimRight(line, "\r\n")
		if filerowidx == 0 {
			// first row get columns names in file
			fileCols := strings.Split(line, delims[0])
			// process get columns name from cmd
			flgCols := strings.Split(flcols, ",")
			// NEW we get map colsMask of keys columns, and matched cols will have 1 s values
			// colsIdx has key of columns and val index in row
			// add cols to struct init filter
			if flcols != ""{
				err = validateCols(fileCols,flgCols)
				if err != nil {
					return nil, err
				}
			}

			filter, err = filter.New(args, fileCols, flgCols)
			if err != nil {
				log.Printf("error creating filter: ( filerow=%d ) %v", filerowidx, err)
				return nil, err
			}
			filerowidx++
			continue
		}
		// line of data here
		filerow := strings.Split(line, delims[0])
		// execute method filter if condition is ok then 'out' will be 'filerow' (only masked values)
		out, err := filter.Filter(filerow)
		if err != nil {
			//log.Printf("message when filter: ( filerow=%d ) %v\n", filerowidx, err)
		}
		if out != ""  {
			fmt.Printf("result: %s\n",out)
			textout = append(textout,out)
		}
		filerowidx++
	}

	return textout, nil
}

// результат понг функции ввиде структуры
type Result struct {
	ResultError error
	TextOut    []string
}

// Пинг функция для readAnalize
// на входе парамсы, timeout в сек.
// на выходе строки результата и ошибки
func pingAnalize(pathfile string, args string, flcols string, timeout int) ([]string,error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	// т.е через таймаут будет завершена по любому эта функция ping
	defer cancelFunc()
	// check for OS signal ctrl+c
	go catchOs(cancelFunc)
	// канал ошибок и данных
	doneCh := make(chan Result)
	go func(ctx context.Context) {
		//pong это функция проверки сервиса возвращает ответ ошибки и результат,
		//которые пишутся в канал doneCh
		textout, err := readAnalizeFile(pathfile,args,flcols)
		resultRead := Result{ResultError: err, TextOut: textout}
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
	return res.TextOut, res.ResultError
}

//validate file cols and flgcols if any specified
func validateCols(fileCols []string, flgCols []string) error {
	var er int = 0

		for _, flgcls := range flgCols{
			notfound := true
			for _,filecls := range fileCols{
				if flgcls == filecls{
					notfound = false
					break
				}
			}
			if notfound {
				log.Printf("column %s not found in file columns (%s) \n",flgcls, fileCols)
				er++
			}
		}
		if er>0 {
			err:=fmt.Errorf("errors when validating -columns flag")
			return err
		}
	return nil
}
