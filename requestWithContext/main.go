package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"
)

var wg = &sync.WaitGroup{}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log.Println("------start------")
	printMemStat()

	wg.Add(1)
	go server(ctx)
	wg.Add(1)
	go monitorMemStat(ctx)
	client()

	// stop server and monitorMemStat
	cancel()
	wg.Wait()

	log.Println("------end------")
	printMemStat()
	runtime.GC()
	log.Println("------GC------")
	printMemStat()
}

func server(ctx context.Context) {
	defer wg.Done()
	mux := http.NewServeMux()
	mux.Handle("/hello", http.HandlerFunc(hello))
	server := &http.Server{Addr: ":9081", Handler: ClearHandler(mux)}
	go func(ctx context.Context) {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}(ctx)
	if err := server.ListenAndServe(); err != nil {
		log.Println(err.Error())
	}
}

func hello(w http.ResponseWriter, r *http.Request) {
	// this line leads to memory leak.
	// we should not use *http.Request as a key of global request variables.
	r = r.WithContext(context.Background())

	authenticate(r)
	// some codes
	// ..
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Hello World %v", GetUserID(r))
}

func authenticate(r *http.Request) {
	// some codes
	// ..
	SetUserID(r, 2)
}

const total = 100000

func client() {
	// request serially
	for i := 0; i < total; i++ {
		func() {
			resp, err := http.Get("http://localhost:9081/hello")
			if err != nil {
				log.Fatalln(err.Error())
			}
			defer resp.Body.Close()
			_, _ = io.ReadAll(resp.Body)
		}()
	}
}

func monitorMemStat(ctx context.Context) {
	defer wg.Done()
	var (
		tick = time.Tick(1 * time.Second)
	)
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick:
			printMemStat()
		}
	}
}

func printMemStat() {
	var (
		ms runtime.MemStats
	)
	runtime.ReadMemStats(&ms)
	log.Printf("Alloc:%v, Sys: %v, NumGC:%v\n", ms.Alloc, ms.Sys, ms.NumGC)
}
