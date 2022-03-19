package main

import (
	"net/http"
	"sync"
)

var (
	mu   sync.Mutex
	data = make(map[*http.Request]int)
)

func SetUserID(r *http.Request, userID int) {
	mu.Lock()
	defer mu.Unlock()
	data[r] = userID
}

func GetUserID(r *http.Request) int {
	mu.Lock()
	defer mu.Unlock()
	return data[r]
}

func Clear(r *http.Request) {
	delete(data, r)
}

func ClearHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer Clear(r)
		h.ServeHTTP(w, r)
	})
}
