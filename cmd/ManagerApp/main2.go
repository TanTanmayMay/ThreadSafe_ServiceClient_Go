package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CircularQueue struct {
	size     int
	capacity int
	front    int
	rear     int
	elements []interface{}
}

func NewCircularQueue(capacity int) *CircularQueue {
	return &CircularQueue{
		size:     0,
		capacity: capacity,
		front:    0,
		rear:     -1,
		elements: make([]interface{}, capacity),
	}
}

func (cq *CircularQueue) Enqueue(element interface{}) error {
	if cq.IsFull() {
		return errors.New("queue is full")
	}

	cq.rear = (cq.rear + 1) % cq.capacity
	cq.elements[cq.rear] = element
	cq.size++

	return nil
}

func (cq *CircularQueue) Dequeue() (interface{}, error) {
	if cq.IsEmpty() {
		return nil, errors.New("queue is empty")
	}

	element := cq.elements[cq.front]
	cq.front = (cq.front + 1) % cq.capacity
	cq.size--

	return element, nil
}

func (cq *CircularQueue) Front() (interface{}, error) {
	if cq.IsEmpty() {
		return nil, errors.New("queue is empty")
	}

	return cq.elements[cq.front], nil
}

func (cq *CircularQueue) IsEmpty() bool {
	return cq.size == 0
}

func (cq *CircularQueue) IsFull() bool {
	return cq.size == cq.capacity
}

func (cq *CircularQueue) Size() int {
	return cq.size
}

// --------------> DONE CIRCULAR QUEUE <-----------

type Semaphore struct {
	cnt int
}

func NewSemaphore(cnt int) *Semaphore {
	return &Semaphore{
		cnt: cnt,
	}
}

func (sem *Semaphore) Wait() {
	for sem.cnt <= 0 {
	}
	sem.cnt = (sem.cnt - 1)
}

func (sem *Semaphore) Signal() {
	sem.cnt = (sem.cnt + 1)
}

//-----------> DONE SEMAPHORES <--------------

type Manager struct {
	pool *pgxpool.Pool
	sem  *Semaphore
	qu   *CircularQueue
	mut  sync.RWMutex
}

func NewManager(pool *pgxpool.Pool, sem *Semaphore, qu *CircularQueue) *Manager {
	return &Manager{
		pool: pool,
		sem:  sem,
		qu:   qu,
	}
}

func (mgr *Manager) getConnectionHandler(w http.ResponseWriter, r *http.Request) {
	mgr.sem.Wait()
	defer mgr.sem.Signal()
	mgr.mut.RLock()
	defer mgr.mut.RUnlock()

	con, _ := mgr.qu.Dequeue()
	render.JSON(w, r, con)
}

func (mgr *Manager) releaseConnectionHandler(w http.ResponseWriter, r *http.Request) {
	var con *pgxpool.Conn
	if err := render.Decode(r, &con); err != nil {
		http.Error(w, "Failed to decode JSON", http.StatusBadRequest)
		return
	}

	mgr.sem.Wait()
	defer mgr.sem.Signal()
	mgr.mut.RLock()
	defer mgr.mut.RUnlock()

	err := mgr.qu.Enqueue(con)
	if err != nil {
		log.Print("Could not enqueue because it violates the thread safety rule")
	}
	render.JSON(w, r, con)
}

// -------------> DONE MANAGER <----------------

var myMgr *Manager
var mySem *Semaphore
var myCqu *CircularQueue

func initConMgr() {
	conStr := getConStr()
	config, err := pgxpool.ParseConfig(conStr)
	if err != nil {
		log.Print("Error")
	}

	config.MaxConns = int32(10)
	pool, _ := pgxpool.NewWithConfig(context.Background(), config)

	mySem = NewSemaphore(10)
	myCqu = NewCircularQueue(10)
	myMgr = NewManager(pool, mySem, myCqu)

	for i := 0; i < 10; i++ {
		con, _ := myMgr.pool.Acquire(context.Background())
		myMgr.qu.Enqueue(con)
	}
}

func getConStr() string {
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

func main2() {
	initConMgr()
	defer myMgr.pool.Close()
	r := chi.NewRouter()
	r.Get("/getConnection", myMgr.getConnectionHandler)
	r.Post("/releaseConnection", myMgr.releaseConnectionHandler)

	port := "8001"
	log.Fatal(http.ListenAndServe(":"+port, r))
}
