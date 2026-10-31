package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"sync"
	"syscall"
	"time"
)

type TokenBucket struct {
	mu     sync.Mutex
	tokens float64
	last   time.Time
	rate   float64
	burst  float64
}

func NewTokenBucket(rate float64, burst int) *TokenBucket {
	return &TokenBucket{
		tokens: float64(burst),
		last:   time.Now(),
		rate:   rate,
		burst:  float64(burst),
	}
}

func (b *TokenBucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	b.tokens = min(b.burst, b.tokens+now.Sub(b.last).Seconds()*b.rate)
	b.last = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

type TaskStore struct {
	mu    sync.RWMutex
	tasks map[string]Task
}

func NewTaskStore() *TaskStore {
	return &TaskStore{tasks: make(map[string]Task)}
}

func (s *TaskStore) Create(task Task) Task {
	now := time.Now().UTC()
	task.ID = rand.Text()
	task.CreatedAt = now
	task.UpdatedAt = now

	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasks[task.ID] = task
	return task
}

func (s *TaskStore) Get(id string) (Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[id]
	return task, ok
}

// GetWithTotal читает задачу и размер хранилища под одной блокировкой,
// чтобы метаданные v2 соответствовали возвращаемой задаче.
func (s *TaskStore) GetWithTotal(id string) (Task, int, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[id]
	return task, len(s.tasks), ok
}

func (s *TaskStore) List() []Task {
	s.mu.RLock()
	tasks := make([]Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}
	s.mu.RUnlock()

	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].CreatedAt.Equal(tasks[j].CreatedAt) {
			return tasks[i].ID < tasks[j].ID
		}
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})
	return tasks
}

func (s *TaskStore) Update(id string, task Task) (Task, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.tasks[id]
	if !ok {
		return Task{}, false
	}

	task.ID = current.ID
	task.CreatedAt = current.CreatedAt
	task.UpdatedAt = time.Now().UTC()
	s.tasks[id] = task
	return task, true
}

func (s *TaskStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[id]; !ok {
		return false
	}
	delete(s.tasks, id)
	return true
}

func (s *TaskStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.tasks)
}

// SaveToFile пишет через временный файл и rename, чтобы сбой посреди записи
// не испортил предыдущую копию.
func (s *TaskStore) SaveToFile(path string) error {
	data, err := json.MarshalIndent(s.List(), "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func (s *TaskStore) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var loaded []Task
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}

	tasks := make(map[string]Task, len(loaded))
	for _, task := range loaded {
		tasks[task.ID] = task
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasks = tasks
	return nil
}

func StartPeriodicSave(store *TaskStore, path string, interval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := store.SaveToFile(path); err != nil {
				log.Printf("periodic save failed: %v", err)
			}
		case <-stop:
			return
		}
	}
}

func Chain(middlewares ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(status int) {
	if !r.wroteHeader {
		r.status = status
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(status)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(recorder, r)

		log.Printf("%s %s %d %s", r.Method, r.URL.Path, recorder.status, time.Since(start))
	})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func RateLimitMiddleware(rate float64, burst int) Middleware {
	var mu sync.Mutex
	buckets := make(map[string]*TokenBucket)

	bucketFor := func(ip string) *TokenBucket {
		mu.Lock()
		defer mu.Unlock()

		bucket, ok := buckets[ip]
		if !ok {
			bucket = NewTokenBucket(rate, burst)
			buckets[ip] = bucket
		}
		return bucket
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !bucketFor(clientIP(r)).Allow() {
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic in %s %s: %v", r.Method, r.URL.Path, err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(AuthTokenHeader) != AuthTokenValue {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write response failed: %v", err)
	}
}

func writeNotFound(w http.ResponseWriter) {
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}

func decodeTask(w http.ResponseWriter, r *http.Request) (Task, bool) {
	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return Task{}, false
	}
	return task, true
}

func createTaskHandler(store *TaskStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		task, ok := decodeTask(w, r)
		if !ok {
			return
		}
		writeJSON(w, http.StatusCreated, store.Create(task))
	}
}

func listTasksHandler(store *TaskStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, store.List())
	}
}

func getTaskHandler(store *TaskStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		task, ok := store.Get(r.PathValue("id"))
		if !ok {
			writeNotFound(w)
			return
		}
		writeJSON(w, http.StatusOK, task)
	}
}

func updateTaskHandler(store *TaskStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		task, ok := decodeTask(w, r)
		if !ok {
			return
		}

		updated, ok := store.Update(r.PathValue("id"), task)
		if !ok {
			writeNotFound(w)
			return
		}
		writeJSON(w, http.StatusOK, updated)
	}
}

func deleteTaskHandler(store *TaskStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !store.Delete(r.PathValue("id")) {
			writeNotFound(w)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func getTaskV2Handler(store *TaskStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		task, total, ok := store.GetWithTotal(r.PathValue("id"))
		if !ok {
			writeNotFound(w)
			return
		}

		writeJSON(w, http.StatusOK, TaskV2Response{
			ID:          task.ID,
			Title:       task.Title,
			Description: task.Description,
			Status:      task.Status,
			CreatedAt:   task.CreatedAt,
			UpdatedAt:   task.UpdatedAt,
			Meta:        TaskMeta{Version: "v2", TotalTasks: total},
		})
	}
}

func BuildRouter(store *TaskStore) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/tasks", createTaskHandler(store))
	mux.HandleFunc("GET /v1/tasks", listTasksHandler(store))
	mux.HandleFunc("GET /v1/tasks/{id}", getTaskHandler(store))
	mux.HandleFunc("PUT /v1/tasks/{id}", updateTaskHandler(store))
	mux.HandleFunc("DELETE /v1/tasks/{id}", deleteTaskHandler(store))
	mux.HandleFunc("GET /v2/tasks/{id}", getTaskV2Handler(store))

	return Chain(
		LoggingMiddleware,
		RateLimitMiddleware(RateLimitRPS, RateLimitBurst),
		RecoveryMiddleware,
		AuthMiddleware,
	)(mux)
}

func run() error {
	store := NewTaskStore()
	if err := store.LoadFromFile(TasksFileName); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("load tasks: %w", err)
	}

	stopSave := make(chan struct{})
	saveDone := make(chan struct{})
	go func() {
		defer close(saveDone)
		StartPeriodicSave(store, TasksFileName, SaveInterval, stopSave)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	server := &http.Server{Addr: ServerAddr, Handler: BuildRouter(store)}
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.ListenAndServe()
	}()
	log.Printf("server started on %s", ServerAddr)

	var runErr error
	select {
	case err := <-serveErr:
		runErr = fmt.Errorf("serve: %w", err)
	case sig := <-quit:
		log.Printf("received %s, shutting down", sig)

		ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			runErr = fmt.Errorf("shutdown: %w", err)
			server.Close()
		}
	}

	close(stopSave)
	<-saveDone

	if err := store.SaveToFile(TasksFileName); err != nil {
		runErr = errors.Join(runErr, fmt.Errorf("final save: %w", err))
	}
	return runErr
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
