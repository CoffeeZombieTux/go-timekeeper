package model

import (
	"time"

	"github.com/google/uuid"
)

// TimeRecord represents a time record entity in the database.
type TimeRecord struct {
	ID           uuid.UUID  `json:"id"`
	UserID       uuid.UUID  `json:"user_id"`
	ProjectID    uuid.UUID  `json:"project_id"`
	TaskID       uuid.UUID  `json:"task_id"`
	WorkDate     time.Time  `json:"work_date"`
	Timezone     string     `json:"timezone"`
	StartedAt    time.Time  `json:"started_at"`
	EndedAt      *time.Time `json:"ended_at"`
	TotalMinutes *int       `json:"total_minutes"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// ClosedTimeRecordInput represents the input for closing a time record.
// It is not an API input but used in the service layer.
type ClosedTimeRecordInput struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	ProjectID    uuid.UUID
	TaskID       uuid.UUID
	WorkDate     time.Time
	Timezone     string
	StartedAt    time.Time
	EndedAt      time.Time
	TotalMinutes int
}
