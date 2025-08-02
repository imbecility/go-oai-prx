// main.go
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
)

func main() {
	port := flag.Int("port", 7860, "порт для прослушивания")
	quiet := flag.Bool("quiet", false, "тихий режим (подавляет логирование  запуска и запросов)")
	html := flag.String("html", "index.html", "путь до странички-заглушки, которая будет отображаться на главной")

	flag.Parse()

	if *quiet {
		log.SetOutput(io.Discard)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/models", handleModels)
	mux.HandleFunc("/api/v1/chat/completions", handleChatCompletions)
	Html = *html
	mux.HandleFunc("/", handleStaticFileOrFallback)

	server := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", *port),
		Handler: loggingMiddleware(mux),
	}

	log.Printf("прокси запущен: http://%s", server.Addr)

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("не удалось проднять сервер: %s\n", err)
	}
}
