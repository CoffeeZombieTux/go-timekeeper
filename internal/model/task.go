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
	UserID    uuid.UUID  `json:"userId"`
	ProjectID uuid.UUID  `json:"projectId"`
	Name      string     `json:"name"`
	Status    TaskStatus `json:"status"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
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
