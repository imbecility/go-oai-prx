// handlers.go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

// handleModels отдает статический JSON со списком моделей
func handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, fmt.Sprintf("вместо запросов %s нужно использовать только GET-запросы", r.Method), http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if _, err := w.Write([]byte(modelsResponseJSON)); err != nil {
		log.Printf("ОШИБКА при записи ответа со списком моделей: %v", err)
	}
}

// handleStaticFileOrFallback обслуживает статику или заглушку
// можно положить статический html-файл рядом с программой и он подтянется,
// иначе будет выведен стандартный код html
func handleStaticFileOrFallback(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	if _, err := os.Stat(Html); err == nil {
		http.ServeFile(w, r, Html)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte(StubPage)); err != nil {
		log.Printf("ОШИБКА при записи html-заглушки: %v", err)
	}
}

// loggingMiddleware логирует все входящие запросы
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("получен запрос: %s %s от %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}
