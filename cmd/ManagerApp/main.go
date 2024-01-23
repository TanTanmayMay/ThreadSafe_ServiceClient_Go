// connectionManager.go
package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	pool         *pgxpool.Pool
	poolCapacity = 10
	mutex        sync.RWMutex
)

func main() {
	initializeConnectionPool()
	defer pool.Close()	

	r := chi.NewRouter()
	r.Get("/acquire/{num}", getConnectionHandler)

	port := "8001" 
	log.Printf("connectionManager is running on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func initializeConnectionPool() {
	connStr := getConnectionStringFromClientService()

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		log.Fatal("Error parsing connection string:", err)
	}

	config.MaxConns = int32(poolCapacity)
	pool, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatal("Error establishing connection pool:", err)
	}

	log.Printf("Connection pool initialized with a capacity of %d connections", poolCapacity)
}

func getConnectionStringFromClientService() string {
	clientServiceURL := "http://localhost:8000/connection-string"

	resp, err := http.Get(clientServiceURL)
	if err != nil {
		log.Fatal("Error getting connection string:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Error: Unexpected status code %d", resp.StatusCode)
	}

	connStr, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Error reading connection string:", err)
	}
	return string(connStr)
}

func getConnectionHandler(w http.ResponseWriter, r *http.Request) {
	mutex.RLock()
	defer mutex.RUnlock()
	NoOfCo := chi.URLParam(r, "num")

	NoOfCon, _ := strconv.Atoi(NoOfCo)

	if pool == nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Connection pool is not initialized"))
		return
	}

	var connList []*pgxpool.Conn

	for i := 0; i < NoOfCon; i++ {
		conn, err := pool.Acquire(context.Background())
		if err != nil {
			render.JSON(w, r, err)
			return
		}
		connList = append(connList, conn)
	}

	time.Sleep(5 * time.Second)

	for _, con := range connList {
		con.Release()
	}

	render.JSON(w, r, connList)
}

