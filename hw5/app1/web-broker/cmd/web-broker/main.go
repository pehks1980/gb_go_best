package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vlslav/web-broker/internal/app/endpoint"
	// сервис сервера ()
	"github.com/vlslav/web-broker/internal/app/service"
	// репозиторий (хранилище) 1 файло 2 память 3 pg sql(db)
	"github.com/vlslav/web-broker/internal/pkg/repository"

	"github.com/vlslav/web-broker/internal/pkg/model"
)


// главная петля
func main() {

	log.Print("Starting the app")
	// настройка порта, настроек хранилища, таймаут при закрытии сервиса
	port := flag.String("port", "8000", "Port")
	storageName := flag.String("storage", "storage.json", "data storage")
	shutdownTimeout := flag.Int64("shutdown_timeout", 3, "shutdown timeout")
	// инициализация файлового хранилища ук на структуру repo
	var repoif repository.RepoIf
	// подстановка в интерфейс соотвествующего хранилища
	repoif = new(repository.FileRepo)
	//repoif = new(repository.MemRepo)
	//repoif = new(repository.PgRepo)
	// вызов доп метода интерфейса - инициализация каким то конфигом
	repoif = repoif.New(*storageName)

	// инициализация сервиса - 'сцепление' с файловым хранилищем
	// через интерфейс :
	// в итоге svc хранит методы и данные структуры файлового хранилища file.go Put Get
	queueSVC := service.New(repoif)
	//создание сервера с таким портом, и обработчиком интерфейс которого связывается а файлохранилищем
	// т.к. инициализация происходит (RegisterPublicHTTP)- в интерфейс endpoint подаетмя структура из file.go
	serv := http.Server{
		Addr:    net.JoinHostPort("", *port),
		Handler: endpoint.RegisterPublicHTTP(queueSVC),
	}
	// запуск сервера
	go func() {
		if err := serv.ListenAndServe(); err != nil {
			log.Fatalf("listen and serve err: %v", err)
		}
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	log.Print("Started app")
	// ждет сигнала
	sig := <-interrupt

	log.Printf("Sig: %v, stopping app", sig)
	// шат даун по контексту с тайм аутом
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*shutdownTimeout)*time.Second)
	defer cancel()
	if err := serv.Shutdown(ctx); err != nil {
		log.Printf("shutdown err: %v", err)
	}
}
