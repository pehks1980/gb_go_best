package main

import (
	"flag"
	"fmt"
	"github.com/pehks1980/gb_go_best/hw3/app1/fscan"
	logger "github.com/pehks1980/gb_go_best/hw3/app1/logger"
	"github.com/pehks1980/gb_go_best/hw3/app1/mockfs"

	//"log"
	"os"
	"runtime/trace"
)

/*
Добавьте в программу для поиска дубликатов, разработанную в рамках проектной работы на предыдущем модуле логи.
Необходимо использовать пакет zap или logrus.

Разграничить уровни логирования.

Обогатить параметрами по вашему усмотрению.

Вставить вызов panic() в участке коде, в котором осуществляется переход в поддиректорию; удостовериться,
что по логам можно локализовать при каком именно переходе в какую директорию сработала паника.

*/

/*
урок 3 тестирование
1   Рефакторим код в соответствие с рассмотренными принципами (SRP, чистые функции,
интерфейсы, убираем глобальный стэйт) *
(привести в порядок испорченную ветку)
2 Пробуем использовать testify *

3  Делаем стаб/мок (например, для файловой системы) и тестируем свой код без обращений к
внешним компонентам (файловой системе) *

 4 Делаем отдельно 1-2 интеграционных теста, запускаемых с флагом -integration -
не понял что к чему, в методе ничего нет
*/

var (
	// флаги
	deepScan          = flag.Bool("ds", false, "Deep scan check - check contents of the dub files")
	delDubs           = flag.Bool("del", false, "Delete dub files after scan")
	interactiveDelete = flag.Bool("i", false, "Interactive mode delete dub files after scan")
	debug             = flag.Bool("debug", false, "debug info")
	mockFs            = flag.Bool("mockfs", false, "mock filesystem ")
	loglevel          = flag.Int("lev", 1, "level of logging 3-Debug, 2-Warning, 1-Info")

	// флаг запускает трассировку
	// cделать и поглядеть трассировку:
	// GOMAXPROCS=1 go run main.go > trace.out
	// go tool trace trace.out
	//
	TraceOn bool = false
	// mock files system flag
	// mockFs bool = true
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
	//логгер init
	Logger := logger.InitLoggers("log.txt", *debug, *loglevel)

	Logger.Info("1 Starting the application...")

	path := flag.Args()
	if len(path) == 0 {
		pathArg, err := os.Getwd()
		if err != nil {
			Logger.Errorf("path arg error %v ", err)
		}
		path = append(path, pathArg)
	}

	Logger.Infof("Program started with pathDir=%s , Deep Scan is %t", path[0], *deepScan)
	Logger.Infof("Debug=%t, LogLevel=%d, Mocking fs = %t", *debug, *loglevel, *mockFs)
	//python like generator func to use it with mocking fs gives channel with sequences of int numbers
	PyGenCh := mockfs.PyGen()
	// главная структура - инстанс обьекта хеш таблицы поиска дублей
	fileSet := fscan.NewRWSet(*deepScan, Logger, *mockFs, PyGenCh)

	fileSet.WaitGroup.Add(1)
	go fileSet.ScanDir(path[0], "nil")
	//ждем пока все перемножится
	fileSet.WaitGroup.Wait()

	Logger.Warnf("scan created %d go procs...", fileSet.ProcCounter)
	Logger.Warnf("scan found %d unique files with duplicates...", fileSet.FilesHaveDubs)

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
							err := fileSet.DeleteDup(dub)
							if err == nil {
								Logger.Warnf("%d. DUB File: %s DELETED", i, dub)
							}
						}
						break loop

					case delPrompt > 0 && delPrompt <= len(v.DubPaths):
						// keep selected dup, delete anything other
						for i, dub := range v.DubPaths {
							if i+1 != delPrompt {
								err := fileSet.DeleteDup(dub)
								if err == nil {
									Logger.Warnf("%d. DUB File: %s DELETED", i, dub)
								}
							}

						}
						err := fileSet.DeleteDup(v.FullPath)
						if err == nil {
							Logger.Warnf("%d. DUB File: %s DELETED", len(v.DubPaths)-1, v.FullPath)
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
						err := fileSet.DeleteDup(dub)
						if err == nil {
							Logger.Warnf("%d. DUB File: %s DELETED", i, dub)
							i++
						}
					}
				}
			}
		} else {
			Logger.Warnf("DUB Files Not DELETED")
		}
	}
	Logger.Infof("Finishing application...")
}
