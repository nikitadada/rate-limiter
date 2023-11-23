package main

import (
	"fmt"
	"io"
	"net/http"
	"rate-limiter/internal"
	"sync"
	"time"
)

func rateLimiterMiddleware(handler func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	// Разрешено 5 запросов за 10 секунд
	limiter := internal.NewRateLimiter(5, time.Second*10)

	return func(w http.ResponseWriter, r *http.Request) {
		ip := getUserIP(r)
		if limiter.Allow(ip) {
			handler(w, r)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			_, err := io.WriteString(w, "request limit has been reached\n")
			if err != nil {
				w.WriteHeader(http.StatusTooManyRequests)
			}
		}
	}
}

func apiHandler(w http.ResponseWriter, _ *http.Request) {
	_, err := io.WriteString(w, "request completed successfully\n")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func getUserIP(r *http.Request) string {
	IPAddress := r.Header.Get("X-Real-Ip")
	if IPAddress == "" {
		IPAddress = r.Header.Get("X-Forwarded-For")
	}
	if IPAddress == "" {
		IPAddress = r.RemoteAddr
	}
	return IPAddress
}

func main() {
	http.HandleFunc("/api", rateLimiterMiddleware(apiHandler))

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			panic(fmt.Sprintf("FATAL ERROR: %s", err))
		}
	}()

	time.Sleep(time.Second * 1)

	// Тестовая отправка клиентских запросов
	for i := 0; i < 20; i++ {
		resp, err := http.Get("http://localhost:8080/api")
		if err != nil {
			panic(fmt.Sprintf("FATAL ERROR: %s", err))
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(fmt.Sprintf("FATAL ERROR: %s", err))
		}
		bodyString := string(bodyBytes)
		fmt.Printf("response from server: %s", bodyString)

		time.Sleep(time.Second)
	}

	wg.Wait()
}
