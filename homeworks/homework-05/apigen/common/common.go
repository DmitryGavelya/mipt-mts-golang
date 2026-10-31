// Package common — готовые HTTP-помощники для сгенерированного кода. Не изменять.
package common

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// WriteJSON пишет v как JSON с заданным статусом.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// WriteError пишет {"error": msg} с заданным статусом.
func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}

// PathID достаёт {id} из пути вида /users/{id}.
func PathID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}
