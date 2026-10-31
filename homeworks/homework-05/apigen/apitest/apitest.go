// Package apitest — набор тестов, которым проверяется сгенерированный API.
// Не изменять.
package apitest

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Schema — схема БД задания.
const Schema = `
CREATE TABLE users (
	id    BIGSERIAL PRIMARY KEY,
	name  TEXT NOT NULL,
	email TEXT NOT NULL
);
CREATE TABLE notes (
	id      BIGSERIAL PRIMARY KEY,
	user_id BIGINT NOT NULL,
	title   TEXT NOT NULL,
	done    BOOLEAN NOT NULL DEFAULT FALSE
);`

// DefaultDSN — база из docker-compose.yml. Другую можно задать через DATABASE_URL.
const DefaultDSN = "postgres://hw5:hw5@localhost:5432/hw5?sslmode=disable"

// OpenDB создаёт для теста отдельную пустую схему Postgres с таблицами
// из Schema и дополнительным SQL extra и удаляет её после теста.
func OpenDB(tb testing.TB, extra string) *sql.DB {
	tb.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = DefaultDSN
	}
	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { admin.Close() })
	if err := admin.Ping(); err != nil {
		tb.Fatalf("нет связи с Postgres по %s: %v\nзапустите базу из папки apigen: docker compose up -d --wait", dsn, err)
	}

	schema := fmt.Sprintf("hw5_test_%x", rand.Uint64())
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { admin.Exec("DROP SCHEMA " + schema + " CASCADE") })

	u, err := url.Parse(dsn)
	if err != nil {
		tb.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema) // все соединения пула видят только свою схему
	u.RawQuery = q.Encode()
	db, err := sql.Open("pgx", u.String())
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { db.Close() })
	if _, err := db.Exec(Schema + extra); err != nil {
		tb.Fatal(err)
	}
	return db
}

// Table описывает таблицу для проверки.
type Table struct {
	Path    string           // "/users"; имя таблицы — Path без "/"
	Schema  string           // дополнительный CREATE TABLE, если таблицы нет в Schema
	Seed    string           // INSERT нескольких строк напрямую в БД
	Rows    []map[string]any // как эти строки выглядят в JSON, по порядку id
	Filters []Filter
}

// Filter — запрос списка с фильтрами и ожидаемый ответ.
type Filter struct {
	Query string // "name=Ada&email=ada@example.com"
	Rows  []int  // индексы ожидаемых строк из Table.Rows
	Error string // если не пусто — ждём 400 с этой ошибкой
}

func Users() Table {
	return Table{
		Path: "/users",
		Seed: `INSERT INTO users (name, email) VALUES
			('Ada', 'ada@example.com'), ('Rob', 'rob@example.com'), ('Ada', 'ada2@example.com')`,
		Rows: []map[string]any{
			{"id": 1, "name": "Ada", "email": "ada@example.com"},
			{"id": 2, "name": "Rob", "email": "rob@example.com"},
			{"id": 3, "name": "Ada", "email": "ada2@example.com"},
		},
		Filters: []Filter{
			{Query: "name=Ada", Rows: []int{0, 2}},
			{Query: "name=Ada&email=ada2@example.com", Rows: []int{2}},
			{Query: "name=Nobody", Rows: []int{}},
			{Query: "name=" + url.QueryEscape("' OR '1'='1"), Rows: []int{}},
			{Query: "age=30", Error: "unknown filter age"},
			{Query: "id=1", Error: "unknown filter id"},
		},
	}
}

func Notes() Table {
	return Table{
		Path: "/notes",
		Seed: `INSERT INTO notes (user_id, title, done) VALUES
			(7, 'buy milk', TRUE), (7, 'write hw5', FALSE), (3, 'sleep', TRUE)`,
		Rows: []map[string]any{
			{"id": 1, "user_id": 7, "title": "buy milk", "done": true},
			{"id": 2, "user_id": 7, "title": "write hw5", "done": false},
			{"id": 3, "user_id": 3, "title": "sleep", "done": true},
		},
		Filters: []Filter{
			{Query: "user_id=7", Rows: []int{0, 1}},
			{Query: "done=true", Rows: []int{0, 2}},
			{Query: "user_id=7&done=false", Rows: []int{1}},
			{Query: "title=sleep&user_id=3&done=true", Rows: []int{2}},
			{Query: "user_id=seven", Error: "user_id must be int"},
			{Query: "done=maybe", Error: "done must be bool"},
			{Query: "user_id=7&owner=me", Error: "unknown filter owner"},
		},
	}
}

// Register — то, что проверяется: регистрирует обработчики таблицы в mux.
type Register func(mux *http.ServeMux, db *sql.DB)

// Run проверяет GET списка, GET по id и фильтры списка по таблице tbl.
func Run(t *testing.T, tbl Table, register Register) {
	setup := func(t *testing.T, seed bool) http.Handler {
		db := OpenDB(t, tbl.Schema)
		if seed {
			if _, err := db.Exec(tbl.Seed); err != nil {
				t.Fatal(err)
			}
		}
		mux := http.NewServeMux()
		register(mux, db)
		return mux
	}

	t.Run("list empty", func(t *testing.T) {
		h := setup(t, false)
		code, body := do(t, h, tbl.Path)
		if code != http.StatusOK || strings.TrimSpace(body) != "[]" {
			t.Fatalf("GET %s on empty table: want 200 [], got %d %s", tbl.Path, code, body)
		}
	})

	t.Run("list", func(t *testing.T) {
		code, body := do(t, setup(t, true), tbl.Path)
		wantJSON(t, code, body, http.StatusOK, tbl.Rows)
	})

	t.Run("get", func(t *testing.T) {
		h := setup(t, true)
		for i, row := range tbl.Rows {
			code, body := do(t, h, fmt.Sprintf("%s/%d", tbl.Path, i+1))
			wantJSON(t, code, body, http.StatusOK, row)
		}
	})

	t.Run("errors", func(t *testing.T) {
		h := setup(t, true)
		for _, c := range []struct {
			path, err string
			code      int
		}{
			{tbl.Path + "/42", "not found", http.StatusNotFound},
			{tbl.Path + "/abc", "bad id", http.StatusBadRequest},
		} {
			code, body := do(t, h, c.path)
			wantJSON(t, code, body, c.code, map[string]any{"error": c.err})
		}
	})

	t.Run("filters", func(t *testing.T) {
		h := setup(t, true)
		for _, f := range tbl.Filters {
			path := tbl.Path + "?" + f.Query
			code, body := do(t, h, path)
			if f.Error != "" {
				wantJSON(t, code, body, http.StatusBadRequest, map[string]any{"error": f.Error})
				continue
			}
			want := []map[string]any{}
			for _, i := range f.Rows {
				want = append(want, tbl.Rows[i])
			}
			wantJSON(t, code, body, http.StatusOK, want)
		}
	})
}

func do(t *testing.T, h http.Handler, path string) (int, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
	return rec.Code, rec.Body.String()
}

// wantJSON сравнивает тело с want после приведения обоих к any через JSON.
func wantJSON(t *testing.T, code int, body string, wantCode int, want any) {
	t.Helper()
	wantRaw, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if code != wantCode {
		t.Fatalf("want %d %s, got %d %s", wantCode, wantRaw, code, strings.TrimSpace(body))
	}
	var got, exp any
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("response is not JSON: %v: %s", err, body)
	}
	json.Unmarshal(wantRaw, &exp)
	if !reflect.DeepEqual(got, exp) {
		t.Fatalf("want %s, got %s", wantRaw, strings.TrimSpace(body))
	}
}
