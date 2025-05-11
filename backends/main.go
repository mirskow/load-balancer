// Запуск простых бекендов для тестирования балансировщика и лимитера
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT is required")
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from backend on port %s\n", port)
	})

	log.Printf("Backend is running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
