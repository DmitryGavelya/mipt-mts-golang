package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"
)

func TestTokenBucket(t *testing.T) {
	tb := NewTokenBucket(10, 10)
	for i := 0; i < 10; i++ {
		if !tb.Allow() {
			t.Fatalf("request %d should be allowed", i)
		}
	}
	if tb.Allow() {
		t.Fatal("11th request should be rate limited")
	}
}

func TestTokenBucketRefill(t *testing.T) {
	tb := NewTokenBucket(10, 5)
	for i := 0; i < 5; i++ {
		if !tb.Allow() {
			t.Fatalf("request %d should be allowed within burst", i)
		}
	}
	if tb.Allow() {
		t.Fatal("6th request should be rate limited after burst exhausted")
	}
	time.Sleep(210 * time.Millisecond)
	if !tb.Allow() {
		t.Fatal("after 200ms 2 tokens should be available")
	}
	if !tb.Allow() {
		t.Fatal("2nd token should be available after 200ms")
	}
	if tb.Allow() {
		t.Fatal("3rd token should not be available yet")
	}
}

func TestTaskStoreCRUD(t *testing.T) {
	store := NewTaskStore()

	created := store.Create(Task{Title: "Test", Description: "Desc", Status: "pending"})
	if created.ID == "" {
		t.Fatal("task ID should be generated")
	}
	if created.CreatedAt.IsZero() {
		t.Fatal("CreatedAt should be set")
	}

	found, ok := store.Get(created.ID)
	if !ok {
		t.Fatal("created task should be found")
	}
	if found.Title != "Test" {
		t.Fatalf("expected title Test, got %s", found.Title)
	}

	tasks := store.List()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	found.Status = "done"
	found.Title = "Updated"
	updated, ok := store.Update(created.ID, found)
	if !ok {
		t.Fatal("update should succeed")
	}
	if updated.Status != "done" {
		t.Fatalf("expected status done, got %s", updated.Status)
	}
	if updated.Title != "Updated" {
		t.Fatalf("expected title Updated, got %s", updated.Title)
	}

	if !store.Delete(created.ID) {
		t.Fatal("delete should succeed")
	}

	_, ok = store.Get(created.ID)
	if ok {
		t.Fatal("task should be deleted")
	}

	if store.Len() != 0 {
		t.Fatalf("expected store len 0 after delete, got %d", store.Len())
	}
}

func TestTaskStoreConcurrency(t *testing.T) {
	store := NewTaskStore()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			store.Create(Task{Title: "Task"})
		}(i)
	}
	wg.Wait()

	if store.Len() != 100 {
		t.Fatalf("expected 100 tasks, got %d", store.Len())
	}

	ids := make(map[string]bool)
	for _, task := range store.List() {
		if ids[task.ID] {
			t.Fatal("duplicate task ID detected")
		}
		ids[task.ID] = true
	}
}

func TestTaskStoreSaveLoad(t *testing.T) {
	store := NewTaskStore()
	store.Create(Task{Title: "Test", Status: "pending", Description: "Description"})

	tmpfile, err := os.CreateTemp("", "tasks-*.json")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	if err := store.SaveToFile(tmpfile.Name()); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	store2 := NewTaskStore()
	if err := store2.LoadFromFile(tmpfile.Name()); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if store2.Len() != 1 {
		t.Fatalf("expected 1 task after load, got %d", store2.Len())
	}

	tasks := store2.List()
	if tasks[0].Title != "Test" {
		t.Fatalf("expected title Test, got %s", tasks[0].Title)
	}
	if tasks[0].Status != "pending" {
		t.Fatalf("expected status pending, got %s", tasks[0].Status)
	}
}

func TestMiddlewareChain(t *testing.T) {
	var calls []string

	m1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "m1-before")
			next.ServeHTTP(w, r)
			calls = append(calls, "m1-after")
		})
	}

	m2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "m2-before")
			next.ServeHTTP(w, r)
			calls = append(calls, "m2-after")
		})
	}

	handler := Chain(m1, m2)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, "handler")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	expected := []string{"m1-before", "m2-before", "handler", "m2-after", "m1-after"}
	if len(calls) != len(expected) {
		t.Fatalf("expected %d calls, got %d: %v", len(expected), len(calls), calls)
	}
	for i, exp := range expected {
		if calls[i] != exp {
			t.Fatalf("call %d: expected %s, got %s", i, exp, calls[i])
		}
	}
}

func TestAuthMiddleware(t *testing.T) {
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}

	req.Header.Set(AuthTokenHeader, AuthTokenValue)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid token, got %d", rec.Code)
	}
}

func TestAuthMiddlewareWrongToken(t *testing.T) {
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set(AuthTokenHeader, "wrong-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong token, got %d", rec.Code)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	handler := RecoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for panic, got %d", rec.Code)
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	handler := RateLimitMiddleware(2, 2)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d should pass, got %d", i, rec.Code)
		}
	}

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
}

func TestRateLimitByIP(t *testing.T) {
	handler := RateLimitMiddleware(1, 1)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "1.2.3.4:1234"
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("first req from 1.2.3.4 should pass, got %d", rec1.Code)
	}

	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "5.6.7.8:5678"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("first req from 5.6.7.8 should pass, got %d", rec2.Code)
	}

	req3 := httptest.NewRequest("GET", "/", nil)
	req3.RemoteAddr = "1.2.3.4:1234"
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusTooManyRequests {
		t.Fatalf("second req from 1.2.3.4 should be rate limited, got %d", rec3.Code)
	}
}

func TestAPIV1CreateTask(t *testing.T) {
	store := NewTaskStore()
	router := BuildRouter(store)

	body := `{"title":"Test Task","description":"Test Desc","status":"pending"}`
	req := httptest.NewRequest("POST", "/v1/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(AuthTokenHeader, AuthTokenValue)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	var task Task
	if err := json.Unmarshal(rec.Body.Bytes(), &task); err != nil {
		t.Fatal(err)
	}
	if task.Title != "Test Task" {
		t.Fatalf("expected title 'Test Task', got %s", task.Title)
	}
	if task.ID == "" {
		t.Fatal("ID should be set")
	}
}

func TestAPIV1Unauthorized(t *testing.T) {
	store := NewTaskStore()
	router := BuildRouter(store)

	body := `{"title":"Test"}`
	req := httptest.NewRequest("POST", "/v1/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAPIV1GetTask(t *testing.T) {
	store := NewTaskStore()
	created := store.Create(Task{Title: "Test", Status: "pending"})
	router := BuildRouter(store)

	req := httptest.NewRequest("GET", "/v1/tasks/"+created.ID, nil)
	req.Header.Set(AuthTokenHeader, AuthTokenValue)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var task Task
	if err := json.Unmarshal(rec.Body.Bytes(), &task); err != nil {
		t.Fatal(err)
	}
	if task.ID != created.ID {
		t.Fatalf("expected ID %s, got %s", created.ID, task.ID)
	}
}

func TestAPIV1ListTasks(t *testing.T) {
	store := NewTaskStore()
	store.Create(Task{Title: "Task1", Status: "pending"})
	store.Create(Task{Title: "Task2", Status: "done"})
	router := BuildRouter(store)

	req := httptest.NewRequest("GET", "/v1/tasks", nil)
	req.Header.Set(AuthTokenHeader, AuthTokenValue)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var tasks []Task
	if err := json.Unmarshal(rec.Body.Bytes(), &tasks); err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
}

func TestAPIV1UpdateTask(t *testing.T) {
	store := NewTaskStore()
	created := store.Create(Task{Title: "Old", Status: "pending"})
	router := BuildRouter(store)

	body := `{"title":"New","description":"Updated","status":"done"}`
	req := httptest.NewRequest("PUT", "/v1/tasks/"+created.ID, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(AuthTokenHeader, AuthTokenValue)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var task Task
	if err := json.Unmarshal(rec.Body.Bytes(), &task); err != nil {
		t.Fatal(err)
	}
	if task.Title != "New" {
		t.Fatalf("expected title 'New', got %s", task.Title)
	}
	if task.Status != "done" {
		t.Fatalf("expected status 'done', got %s", task.Status)
	}
}

func TestAPIV1DeleteTask(t *testing.T) {
	store := NewTaskStore()
	created := store.Create(Task{Title: "ToDelete", Status: "pending"})
	router := BuildRouter(store)

	req := httptest.NewRequest("DELETE", "/v1/tasks/"+created.ID, nil)
	req.Header.Set(AuthTokenHeader, AuthTokenValue)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent && rec.Code != http.StatusOK {
		t.Fatalf("expected 204 or 200, got %d", rec.Code)
	}

	req2 := httptest.NewRequest("GET", "/v1/tasks/"+created.ID, nil)
	req2.Header.Set(AuthTokenHeader, AuthTokenValue)
	rec2 := httptest.NewRecorder()

	router.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", rec2.Code)
	}
}

func TestAPINotFound(t *testing.T) {
	store := NewTaskStore()
	router := BuildRouter(store)

	req := httptest.NewRequest("GET", "/v1/tasks/nonexistent", nil)
	req.Header.Set(AuthTokenHeader, AuthTokenValue)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestAPIV2Format(t *testing.T) {
	store := NewTaskStore()
	created := store.Create(Task{Title: "V2Test", Status: "pending", Description: "V2 Desc"})
	router := BuildRouter(store)

	req := httptest.NewRequest("GET", "/v2/tasks/"+created.ID, nil)
	req.Header.Set(AuthTokenHeader, AuthTokenValue)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var v2 TaskV2Response
	if err := json.Unmarshal(rec.Body.Bytes(), &v2); err != nil {
		t.Fatal(err)
	}
	if v2.ID != created.ID {
		t.Fatalf("expected ID %s, got %s", created.ID, v2.ID)
	}
	if v2.Meta.Version != "v2" {
		t.Fatalf("expected version v2, got %s", v2.Meta.Version)
	}
	if v2.Meta.TotalTasks != 1 {
		t.Fatalf("expected total_tasks 1, got %d", v2.Meta.TotalTasks)
	}
	if v2.Title != "V2Test" {
		t.Fatalf("expected title V2Test, got %s", v2.Title)
	}
}

func TestPeriodicSave(t *testing.T) {
	store := NewTaskStore()
	store.Create(Task{Title: "Periodic", Status: "pending"})

	tmpfile, err := os.CreateTemp("", "tasks-*.json")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	stop := make(chan struct{})
	go StartPeriodicSave(store, tmpfile.Name(), 100*time.Millisecond, stop)

	time.Sleep(250 * time.Millisecond)
	close(stop)

	time.Sleep(100 * time.Millisecond)

	data, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("file should exist after periodic save: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("file should not be empty")
	}

	var raw []map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("file should contain valid JSON array: %v", err)
	}
	if len(raw) != 1 {
		t.Fatalf("expected 1 task in file, got %d", len(raw))
	}
}
