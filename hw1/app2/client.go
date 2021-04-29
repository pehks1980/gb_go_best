package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

const addr = "127.0.0.1:5433"

type Counts struct {
	Requests       uint32
	TotalSuccesses uint32
	TotalFailures  uint32
}

func (c *Counts) onRequest() {
	c.Requests++
}

func (c *Counts) onSuccess() {
	c.TotalSuccesses++
}

func (c *Counts) onFailure() {
	c.TotalFailures++
}

func (c *Counts) clear() {
	c.Requests = 0
	c.TotalSuccesses = 0
	c.TotalFailures = 0
}

type CircuitBreaker struct {
	name         string
	interval     time.Duration // check interval
	numberoftrys uint32
	timeout      time.Duration //time out of one try
	mutex        sync.Mutex
	currstate    State
	prevstate    State
	counts       Counts
	addr         string
}

const defaultInterval = time.Duration(1) * time.Second
const defaultTimeout = time.Duration(1) * time.Second
const defaultNumberofTrys = 6

// State is a type that represents a state of CircuitBreaker.
type State int

// These constants are states of CircuitBreaker.
const (
	StateClosed State = iota
	StateHalfOpen
	StateOpen
)

// Counts returns internal counters
func (cb *CircuitBreaker) Counts() Counts {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	return cb.counts
}

// В случае удачи состояние переходит в сост open
func (cb *CircuitBreaker) onSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.prevstate = cb.currstate
	switch cb.currstate {
	case StateClosed:
		//needs to open
		cb.currstate = StateOpen
	case StateHalfOpen:
		//same
		cb.currstate = StateOpen
	}
}

// В случае неудачи состояние переходит в сост closed
func (cb *CircuitBreaker) onFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.prevstate = cb.currstate
	switch cb.currstate {
	case StateOpen:
		//needs to closed
		cb.currstate = StateClosed
	case StateHalfOpen:
		//same
		cb.currstate = StateClosed

	}
}

//если были и удачные и неудачные попытки - состояние half open
func (cb *CircuitBreaker) onSomeFail() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.prevstate = cb.currstate
	switch cb.currstate {
	case StateOpen:
		//needs to closed
		cb.currstate = StateHalfOpen
	case StateClosed:
		//same
		cb.currstate = StateHalfOpen

	}
}

// получить текущее состояние
func (cb *CircuitBreaker) getCurrState() State {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	return cb.currstate
}

// Конструктор cct breaker
func NewCircuitBreaker(addr string) *CircuitBreaker {
	cb := new(CircuitBreaker)
	cb.interval = defaultInterval
	cb.timeout = defaultTimeout
	cb.numberoftrys = defaultNumberofTrys
	cb.currstate = StateClosed
	cb.prevstate = StateClosed
	cb.addr = addr
	return cb
}

// основная функция breaker
func (cb *CircuitBreaker) Run(done <-chan interface{}) {
	// делаем тест коннекшена и получаем в канал состояние
	// по факту меняем состояние брейкера
	cb.mutex.Lock()
	timeout := cb.timeout
	numberoftrys := cb.numberoftrys
	address := cb.addr
	cb.mutex.Unlock()

	for stateChannel := range cb.mainWholeCheckConn(done, address, timeout, numberoftrys) {
		// ok
		if stateChannel == 1 {
			cb.onSuccess()
		}
		// not ok
		if stateChannel == 0 {
			cb.onFailure()
		}
		// so-so 1/2
		if stateChannel == -1 {
			cb.onSomeFail()
		}
	}

}

// функция шайба для wholeCheckConn которая подсчитывает удачи неудачи соединения
func (cb *CircuitBreaker) mainWholeCheckConn(done <-chan interface{}, address string, timeout time.Duration, numberoftrys uint32) <-chan int {
	stateChannel := make(chan int)
	go func() {
		defer close(stateChannel)
		for {
			result := cb.wholeCheckConn(address, timeout, numberoftrys)

			select {
			case stateChannel <- result:
			case <-done:
				return
			}
		}
	}()
	return stateChannel
}

// wholeCheckConn подсчитывает удачи неудачи соединения
// return 1 - ok 0 bad -1 so-so
func (cb *CircuitBreaker) wholeCheckConn(address string, timeout time.Duration, numberoftrys uint32) int {
	var (
		okCtr uint32 = 0
		noCtr uint32 = 0
	)

	for checkResult := range cb.checkConn(address, timeout, numberoftrys) {
		if checkResult == true {
			okCtr++
		}
		if checkResult == false {
			noCtr++
		}
	}

	cb.mutex.Lock()
	cb.counts.Requests += numberoftrys
	cb.counts.TotalSuccesses += okCtr
	cb.counts.TotalFailures += noCtr
	defer cb.mutex.Unlock()

	if okCtr > 0 && noCtr == 0 {
		return 1
	}
	if noCtr > 0 && okCtr == 0 {
		return 0
	}
	return -1
}

// делаем несколько попыток и смотрим результат в канале
func (cb *CircuitBreaker) checkConn(address string, timeout time.Duration, numberoftrys uint32) <-chan bool {
	checkResults := make(chan bool)
	go func() {
		defer close(checkResults)
		var i uint32
		for i = 0; i < numberoftrys; i++ {
			result := cb.isServerAlive(address, timeout)
			fmt.Printf("test resilt %t \n", result)
			// одна попытка в секунду
			time.Sleep(1 * time.Second)
			select {
			case checkResults <- result:
			}
		}
	}()
	return checkResults
}

// ф. оригинал - тестит коннект сети до сервера
// true - был коннект false нет
func (cb *CircuitBreaker) isServerAlive(address string, timeout time.Duration) bool {
	// проверяем можем ли подключиться к серверу
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		//log.Printf("can't connect to server, error: %s", err)
		return false
	}
	defer conn.Close()
	return true
}

func main() {
	// управлялка
	done := make(chan interface{})
	defer close(done)
	// инстанс
	cb := NewCircuitBreaker(addr)
	// запуск cb
	go cb.Run(done)

	orders := []string{
		"123432-234",
		"123432-234",
		"123432-234",
		"853432-332",
		"853432-332",
		"853432-332",
		"254432-341",
		"254432-341",
		"254432-341",
		"254432-341",
		"853432-332",
		"853432-332",
		"853432-332",
		"254432-341",
		"123432-234",
		"123432-234",
		"853432-332",
		"123432-234",
		"123432-234",
		"123432-234",
		"123432-234",
		"853432-332",
		"853432-332",
		"853432-332",
		"254432-341",
		"254432-341",
		"254432-341",
		"254432-341",
		"853432-332",
		"853432-332",
		"853432-332",
		"254432-341",
		"123432-234",
		"123432-234",
		"853432-332",
		"123432-234",
	}

	for _, order := range orders {
		// паттерн circuit breaker в простейшей реализации
		// в реальности за функцией может скрываться объёмная и сложная логика
		// лишнее выполнение которой мы и хотим прервать
		// бизнес логика читает состояние breaker и решает что делать:

		state := cb.getCurrState()
		switch state {
		case StateOpen:
			// в случае open - платежка уходит в сервер
			fmt.Println("state open")
			pay(order, addr, cb.timeout)

		case StateHalfOpen:
			// случае 1/2 open - платежка уходит в сервер через 2 секунды
			fmt.Println("state 1/2 open")
			// три попытки ухода платежки
			//success := false
			for i := 0; i < 3; i++ {
				time.Sleep(2 * time.Second)
				err := pay(order, addr, cb.timeout)
				if err == nil {
					//success = true
					break
				}
			}
			/*
				if !success{
					// если попытки провалились то ждем пока состояние breaker не станет open
					for {
						fmt.Println("state half open stalled.")
						stat := cb.getCurrState()
						time.Sleep(1 * time.Second)
						if stat == StateOpen {
							break
						}
					}
				}
			*/
		case StateClosed:
			// в случае closed - ждем ничего не отправляем, пока состяние не поменяется на какое-то другое
			for {
				fmt.Println("state closed")
				stat := cb.getCurrState()
				time.Sleep(1 * time.Second)
				if stat != StateClosed {
					break
				}
			}

		}

		time.Sleep(1 * time.Second)
	}

	counts := cb.Counts()
	fmt.Printf("All payments processed, Requests = %d, Success = %d, Failures = %d",
		counts.Requests, counts.TotalSuccesses, counts.TotalFailures)
}

func pay(order string, address string, timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		log.Printf("error payment: can't connect to server, error: %s", err)
		return err
	}
	fmt.Fprintf(conn, "please, process order %q\n", order)
	fmt.Printf("Connex is ok! sending order %s ...\n", order)
	message, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return err
	}
	log.Printf("Message from server: %q", message)

	return nil
}
