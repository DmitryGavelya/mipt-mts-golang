// Не изменять.
package i2s

import (
	"encoding/json"
	"reflect"
	"testing"
)

type Simple struct {
	ID       int
	Username string
	Active   bool
}

type IDBlock struct {
	ID int
}

// unpack превращает JSON в то, что получается при распаковке в any:
// map[string]any, []any, float64, string, bool.
func unpack(t *testing.T, v any) any {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestSimple(t *testing.T) {
	expected := &Simple{ID: 42, Username: "rvasily", Active: true}

	result := new(Simple)
	if err := i2s(unpack(t, expected), result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(expected, result) {
		t.Errorf("results not match\nGot:\n%#v\nExpected:\n%#v", result, expected)
	}
}

type Complex struct {
	SubSimple  Simple
	ManySimple []Simple
	Blocks     []IDBlock
}

func TestComplex(t *testing.T) {
	smpl := Simple{ID: 42, Username: "rvasily", Active: true}
	expected := &Complex{
		SubSimple:  smpl,
		ManySimple: []Simple{smpl, smpl},
		Blocks:     []IDBlock{{42}, {42}},
	}

	result := new(Complex)
	if err := i2s(unpack(t, expected), result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(expected, result) {
		t.Errorf("results not match\nGot:\n%#v\nExpected:\n%#v", result, expected)
	}
}

func TestSlice(t *testing.T) {
	smpl := Simple{ID: 42, Username: "rvasily", Active: true}
	expected := []Simple{smpl, smpl}

	result := []Simple{}
	if err := i2s(unpack(t, expected), &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(expected, result) {
		t.Errorf("results not match\nGot:\n%#v\nExpected:\n%#v", result, expected)
	}
}

// Поля, которых нет в данных, не трогаются.
func TestMissingFields(t *testing.T) {
	result := &Simple{ID: 1, Username: "old", Active: true}
	if err := i2s(map[string]any{"Username": "new"}, result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := &Simple{ID: 1, Username: "new", Active: true}
	if !reflect.DeepEqual(expected, result) {
		t.Errorf("results not match\nGot:\n%#v\nExpected:\n%#v", result, expected)
	}
}

func TestErrors(t *testing.T) {
	cases := []struct {
		name   string
		result any
		json   string
	}{
		{"string вместо bool", &Simple{}, `{"ID":42,"Username":"rvasily","Active":"DA"}`},
		{"string вместо int", &Simple{}, `{"ID":"42","Username":"rvasily","Active":true}`},
		{"int вместо string", &Simple{}, `{"ID":42,"Username":100500,"Active":true}`},
		{"ждём слайс, пришла структура", &Complex{}, `{"SubSimple":{"ID":42,"Username":"rvasily","Active":true},"ManySimple":{}}`},
		{"ждём структуру, пришёл bool", &Complex{}, `{"SubSimple":true,"ManySimple":[{"ID":42,"Username":"rvasily","Active":true}]}`},
		{"ждём структуру, пришёл массив", &Simple{}, `[{"ID":42,"Username":"rvasily","Active":true}]`},
		{"ошибка внутри элемента слайса", &Complex{}, `{"ManySimple":[{"ID":"x"}]}`},
		// Структура по значению, а не указатель: записать результат некуда.
		{"не указатель", Simple{}, `{"ID":42,"Username":"rvasily","Active":true}`},
		{"nil-указатель", (*Simple)(nil), `{"ID":42}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var data any
			if err := json.Unmarshal([]byte(c.json), &data); err != nil {
				t.Fatal(err)
			}
			if err := i2s(data, c.result); err == nil {
				t.Error("expected error here")
			}
		})
	}
}
