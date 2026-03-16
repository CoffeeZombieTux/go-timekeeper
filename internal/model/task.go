package model

import (
	"time"

	"github.com/google/uuid"
)

// TaskStatus represents the status of a task.
type TaskStatus string

const (
	StatusCreated   TaskStatus = "CREATED"
	StatusWorkingOn TaskStatus = "WORKING_ON"
	StatusClosed    TaskStatus = "CLOSED"
	DefaultStatus              = StatusCreated
)

// Task represents a task entity in the database.
type Task struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	ProjectID uuid.UUID  `json:"project_id"`
	Name      string     `json:"name"`
	Status    TaskStatus `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// IsValidTaskStatus checks if the provided status is valid.
func (task *Task) IsValidTaskStatus(inputStatus string) bool {
	switch TaskStatus(inputStatus) {
	case StatusCreated, StatusWorkingOn, StatusClosed:
		return true
	default:
		return false
	}
}
