package model

import (
	"time"

	"github.com/google/uuid"
)

// TimeRecord represents a time record entity in the database.
type TimeRecord struct {
	ID           uuid.UUID  `json:"id"`
	UserID       uuid.UUID  `json:"userId"`
	ProjectID    uuid.UUID  `json:"projectId"`
	TaskID       uuid.UUID  `json:"taskId"`
	WorkDate     time.Time  `json:"workDate"`
	Timezone     string     `json:"timezone"`
	StartedAt    time.Time  `json:"startedAt"`
	EndedAt      *time.Time `json:"endedAt"`
	TotalMinutes *int       `json:"totalMinutes"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
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
