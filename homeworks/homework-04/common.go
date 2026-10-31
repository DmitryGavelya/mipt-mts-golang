package main

import (
	"net/http"
	"time"
)

const (
	AuthTokenHeader = "X-Auth-Token"
	AuthTokenValue  = "bearer-token"
	TasksFileName   = "tasks_backup.json"
	SaveInterval    = 5 * time.Second
	ShutdownTimeout = 30 * time.Second
	RateLimitRPS    = 10
	RateLimitBurst  = 10
	ServerAddr      = ":8080"
)

type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TaskV2Response struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Meta        TaskMeta  `json:"meta"`
}

type TaskMeta struct {
	Version    string `json:"version"`
	TotalTasks int    `json:"total_tasks"`
}

type Middleware func(http.Handler) http.Handler
