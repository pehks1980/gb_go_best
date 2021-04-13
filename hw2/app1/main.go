package main

import (
	"flag"
	"fmt"
	"github.com/pehks1980/gb_go_best/hw2/app1/fscan"
	"github.com/pehks1980/gb_go_best/hw2/app1/logger"
	"github.com/sirupsen/logrus"
	//"log"
	"os"
	"runtime/trace"
	"sync"
	"sync/atomic"
)
/*
Добавьте в программу для поиска дубликатов, разработанную в рамках проектной работы на предыдущем модуле логи.
Необходимо использовать пакет zap или logrus.

Разграничить уровни логирования.

Обогатить параметрами по вашему усмотрению.

Вставить вызов panic() в участке коде, в котором осуществляется переход в поддиректорию; удостовериться,
что по логам можно локализовать при каком именно переходе в какую директорию сработала паника.

 */
var (
	// флаги
	deepScan          = flag.Bool("ds", false, "Deep scan check - check contents of the dub files")
	delDubs           = flag.Bool("del", false, "Delete dub files after scan")
	interactiveDelete = flag.Bool("i", false, "Interactive mode delete dub files after scan")
	debug 			  = flag.Bool("debug", false, "debug info")
	loglevel 		  = flag.Int("lev", 1, "level of logging 3-Debug, 2-Warning, 1-Info")
	// waitgroup
	wg = sync.WaitGroup{}
	// хештаблица структур файлов
	fileSet = fscan.NewRWSet()
	// счетчик гоу поточков
	goProcCounter int64 = 1
	// флаг запускает трассировку
	// cделать и поглядеть трассировку:
	// GOMAXPROCS=1 go run main.go > trace.out
	// go tool trace trace.out
	//
	TraceOn bool = false
)

// main основная функция работы утилиты
func main() {

	// trace code
	if TraceOn {
		trace.Start(os.Stderr)
		defer trace.Stop()
	}

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage\n %s [options] <path>\n\nOptions:\n\t<path>\tpath for scan to start with (default 'working dir')\n ", os.Args[0])

		flag.VisitAll(func(f *flag.Flag) {
			fmt.Fprintf(os.Stderr, "\t-%v\t%v (default '%v')\n", f.Name, f.Usage, f.Value)
		})
		fmt.Fprintf(os.Stderr, "\nExamples:\n\n"+
			" '%s -ds -del -i /home/user/go'  \n"+
			"\t- find duplicates in /home/user/go using md5 hash calculation,\n"+
			"\tdelete files in interactive mode - one occurence by one (by pressing Y key).\n\n"+
			" '%s -del /home/user/go'  \n"+
			"\t- find duplicates in /home/user/go using file name and file size\n"+
			"\tdelete all files duplicates after scan is finished.\n\n", os.Args[0], os.Args[0])
	}
	flag.Parse()
	//logrus init
	logger.InitLoggers("log.txt", *debug, *loglevel)

	logger.Logger.Info("1 Starting the application...")

	path := flag.Args()
	if len(path) == 0 {
		pathArg, err := os.Getwd()
		if err != nil {
			logger.Logger.Errorf("path arg error %v ", err)
		}
		path = append(path, pathArg)
	}

	logger.Logger.Infof("Program started with pathDir=%s , Deep Scan is %t, Debug=%t, LogLevel=%d", path[0], *deepScan, *debug, *loglevel)
	wg.Add(1)
	go ScanDir(path[0], "nil")
	//ждем пока все перемножится
	wg.Wait()

	logger.Logger.Warnf("scan created %d go procs...", goProcCounter)
	logger.Logger.Warnf("scan found %d unique files with duplicates...", fileSet.FilesHaveDubs)

	fnum := 1
	for _, v := range fileSet.MM {
		if v.DubPaths != nil {
			switch *deepScan {
			case true:
				fmt.Printf("\n%d. File: %s Size: %d (B) Number of Dubs: %d md5: %s \n", fnum, v.FullPath, v.Filesize, len(v.DubPaths), v.FileHash)
			case false:
				fmt.Printf("\n%d. File: %s Size: %d (B) Number of Dubs: %d \n", fnum, v.FullPath, v.Filesize, len(v.DubPaths))
			}

			for i, dub := range v.DubPaths {
				fmt.Printf("%d.%d (DUB) File: %s \n", fnum, i+1, dub)
			}
			fnum++
			// обработка в интерактивном режиме
			if *delDubs && *interactiveDelete && fileSet.FilesHaveDubs != 0 {
			loop:
				for {
					fmt.Printf("\nWhich one you want to KEEP? Press number from 0 to %d, 0 - Keep original file (%s)\n", len(v.DubPaths), v.FullPath)
					var delPrompt int
					_, err := fmt.Scanf("%d", &delPrompt)
					if err != nil {
						// error here
						fmt.Printf("Error enter")
						return
					}
					switch {
					case delPrompt == 0:
						// delete all but original (0)
						for i, dub := range v.DubPaths {
							err := fscan.DeleteDup(dub)
							if err == nil {
								logger.Logger.Warnf("%d. DUB File: %s DELETED", i, dub)
							}
						}
						break loop

					case delPrompt > 0 && delPrompt <= len(v.DubPaths):
						// keep selected dup, delete anything other
						for i, dub := range v.DubPaths {
							if i+1 != delPrompt {
								err := fscan.DeleteDup(dub)
								if err == nil {
									logger.Logger.Warnf("%d. DUB File: %s DELETED", i, dub)
								}
							}

						}
						err := fscan.DeleteDup(v.FullPath)
						if err == nil {
							logger.Logger.Warnf("%d. DUB File: %s DELETED", len(v.DubPaths)-1, v.FullPath)
						}
						break loop
					}

				}
			}
		}
	}
	// обработка в основном режиме
	if *delDubs && !*interactiveDelete && fileSet.FilesHaveDubs != 0 {
		var delPrompt string
		fmt.Println("\nConfirm delete of All duplicates 'Y' (then enter)?")

		_, err := fmt.Scanf("%s", &delPrompt)
		if err != nil {
			// error here
			fmt.Printf("Error enter")
			return
		}

		if delPrompt == "Y" || delPrompt == "y" {
			// delete all Dubs
			i := 0
			for _, v := range fileSet.MM {
				if v.DubPaths != nil {
					for _, dub := range v.DubPaths {
						err := fscan.DeleteDup(dub)
						if err == nil {
							logger.Logger.Warnf("%d. DUB File: %s DELETED", i, dub)
							i++
						}
					}
				}
			}
		} else {
			logger.Logger.Warnf("DUB Files Not DELETED")
		}
	}
	logger.Logger.Infof("Finishing application...")
}

// ScanDir - принимает начальную папку и сканирует все подпапки
// для каждой подпапки запускает саму себя, выделяя новый поточек
func ScanDir(pathDir string, rootDir string) {
	defer wg.Done()
	defer func() {
		err := recover()
		if err != nil {
			entry := err.(*logrus.Entry)
			logger.Logger.WithFields(logrus.Fields{
				"dir_root":  rootDir, // рут папка
				"dir_err":    pathDir,
				"err_level":   entry.Level,
				"err_message": entry.Message,
			}).Error("Ошибка!!! Доступ к папке!!!")
		}
	}()

	dirs, err := fscan.IOReadDir(pathDir, fileSet, deepScan)
	if err != nil {
		logger.Logger.Panicf("Error reading dirs: %v", err)
		//logger.Logger.Errorf("Error reading dirs: %v", err)
		return
	}

	for _, dir := range dirs {
		wg.Add(1)
		atomic.AddInt64(&goProcCounter, 1)
		sDir := pathDir + "/" + dir
		go ScanDir(sDir, pathDir)
	}

}
