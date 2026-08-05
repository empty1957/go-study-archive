// Package task contains the domain model and use cases for the learning API.
// It intentionally knows nothing about HTTP, JSON, or a concrete database.
package task

import "time"

// Task is a small domain value. Its fields are safe to copy.
type Task struct {
	ID        string
	Title     string
	Done      bool
	CreatedAt time.Time
}
