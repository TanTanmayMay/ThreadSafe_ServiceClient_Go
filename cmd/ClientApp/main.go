// clientService.go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	r.Get("/connection-string", func(w http.ResponseWriter, r *http.Request) {
		host := "localhost"
		port := "5432"
		user := "nishant"
		password := "om"
		dbname := "tanmay"

		connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(connStr))
	})

	port := "8000" // Set the port for the clientService
	log.Printf("clientService is running on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
