package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var wg = &sync.WaitGroup{}

func main() {
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go waitSignal(i)
	}
	go killMyself()
	wg.Wait()
	log.Println("all of task get signal")
}

func waitSignal(idx int) {
	defer wg.Done()
	termCh := make(chan os.Signal, 1)
	signal.Notify(termCh, syscall.SIGINT, syscall.SIGTERM)
	<-termCh
	if idx == 5 {
		time.Sleep(time.Second * 3)
	}
	log.Printf("%v goroutine catch signal\n", idx)
}

func killMyself() {
	time.Sleep(time.Second * 3)
	process, err := os.FindProcess(os.Getpid())
	if err != nil {
		log.Fatalln("fail to get process")
	}
	process.Signal(syscall.SIGINT)
}
