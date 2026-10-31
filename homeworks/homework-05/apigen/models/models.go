// Package models — модели таблиц. Не изменять, кроме директивы go:generate.
package models

//go:generate go run ../gen models.go

//gen:api users
type User struct {
	ID    int64  `db:"id" json:"id"`
	Name  string `db:"name" json:"name"`
	Email string `db:"email" json:"email"`
}

//gen:api notes
type Note struct {
	ID     int64  `db:"id" json:"id"`
	UserID int64  `db:"user_id" json:"user_id"`
	Title  string `db:"title" json:"title"`
	Done   bool   `db:"done" json:"done"`
}
