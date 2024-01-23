// serverA.go
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
)

var (
	client     = &http.Client{}
	connectionList []string
	mutex      sync.Mutex
)

func main() {
	r := chi.NewRouter()

	r.Get("/acquire/{numConnections}", acquireConnectionsHandler)

	port := "8002"
	log.Printf("serverA is running on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func acquireConnectionsHandler(w http.ResponseWriter, r *http.Request) {
	numConnectionsStr := chi.URLParam(r, "numConnections")
	numConnections, err := strconv.Atoi(numConnectionsStr)
	if err != nil {
		http.Error(w, "Invalid number of connections", http.StatusBadRequest)
		return
	}

	url := fmt.Sprintf("http://localhost:8001/acquire/%d", numConnections)

	resp, err := http.Get(url)
	if err != nil {
		log.Print(err)
	}
	defer resp.Body.Close()

	connections, _ := io.ReadAll(resp.Body)

	connectionList = append(connectionList, string(connections))
	log.Printf("Successfully acquired %d connections", numConnections)
	render.JSON(w, r, connectionList)
}

