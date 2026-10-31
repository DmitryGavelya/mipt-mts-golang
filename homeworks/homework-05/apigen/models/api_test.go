// Не изменять.
package models_test

import (
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"homework05/apitest"
	"homework05/models"
)

func TestUsers(t *testing.T) {
	apitest.Run(t, apitest.Users(), models.RegisterUser)
}

func TestNotes(t *testing.T) {
	apitest.Run(t, apitest.Notes(), models.RegisterNote)
}

func TestGeneratedFile(t *testing.T) {
	src, err := os.ReadFile("models_api.go")
	if err != nil {
		t.Fatalf("models_api.go не найден — запустите go generate ./...: %v", err)
	}
	header := regexp.MustCompile(`(?m)^// Code generated .* DO NOT EDIT\.$`)
	if !header.Match(src) {
		t.Fatal(`models_api.go: нет заголовка "// Code generated ... DO NOT EDIT."`)
	}
	f, err := parser.ParseFile(token.NewFileSet(), "models_api.go", src, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, imp := range f.Imports {
		if imp.Path.Value == `"reflect"` {
			t.Fatal("сгенерированный код не должен использовать reflect")
		}
	}
}

// gen запускает генератор на файле dir/models.go с содержимым src
// и возвращает то, что он записал в dir/models_api.go.
func gen(t *testing.T, dir string, src []byte) string {
	t.Helper()
	in, err := filepath.Abs(filepath.Join(dir, "models.go"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(in, src, 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", ".", in)
	cmd.Dir = "../gen"
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("генератор упал: %v\n%s", err, out)
	}
	out, err := os.ReadFile(filepath.Join(dir, "models_api.go"))
	if err != nil {
		t.Fatalf("генератор не создал models_api.go рядом с входным файлом: %v", err)
	}
	return string(out)
}

func TestGeneratedUpToDate(t *testing.T) {
	src, err := os.ReadFile("models.go")
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("models_api.go")
	if err != nil {
		t.Fatal(err)
	}
	if gen(t, t.TempDir(), src) != string(want) {
		t.Fatal("models_api.go отличается от вывода генератора: его правили руками или забыли go generate")
	}
}

const tagModel = `package tagmodel

type Skipped struct {
	ID int64 ` + "`db:\"id\"`" + `
}

//gen:api tags
type Tag struct {
	Weight int32  ` + "`db:\"weight\" json:\"weight\"`" + `
	ID     int64  ` + "`db:\"id\" json:\"id\"`" + `
	Label  string ` + "`db:\"label\" json:\"label\"`" + `
	Cached string ` + "`json:\"-\"`" + `
}
`

const tagTest = `package tagmodel

import (
	"testing"

	"homework05/apitest"
)

func TestTags(t *testing.T) {
	apitest.Run(t, apitest.Table{
		Path:   "/tags",
		Schema: "CREATE TABLE tags (id BIGSERIAL PRIMARY KEY, label TEXT NOT NULL, weight INTEGER NOT NULL);",
		Seed:   "INSERT INTO tags (label, weight) VALUES ('go', 5), ('sql', 3), ('ast', 5)",
		Rows: []map[string]any{
			{"id": 1, "label": "go", "weight": 5},
			{"id": 2, "label": "sql", "weight": 3},
			{"id": 3, "label": "ast", "weight": 5},
		},
		Filters: []apitest.Filter{
			{Query: "weight=5", Rows: []int{0, 2}},
			{Query: "weight=5&label=ast", Rows: []int{2}},
			{Query: "weight=heavy", Error: "weight must be int"},
			{Query: "cached=x", Error: "unknown filter cached"},
		},
	}, RegisterTag)
}
`

// Новый тип в новом пакете: генератор без правок должен дать рабочий API,
// который проходит тот же набор тестов.
func TestGeneratorNewType(t *testing.T) {
	// Внутри модуля, чтобы пакет видел homework05/common и homework05/apitest.
	dir, err := os.MkdirTemp("..", "gentest")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

	out := gen(t, dir, []byte(tagModel))
	if strings.Contains(out, "RegisterSkipped") {
		t.Error("сгенерирован API для структуры без //gen:api")
	}
	if err := os.WriteFile(filepath.Join(dir, "tag_test.go"), []byte(tagTest), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test", ".")
	cmd.Dir = dir
	if res, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("API, сгенерированный для нового типа, не прошёл тесты: %v\n%s\n--- models_api.go ---\n%s", err, res, out)
	}
}
