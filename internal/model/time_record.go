package model

import (
	"time"

	"github.com/google/uuid"
)

// TimeRecord represents a time record entity in the database.
type TimeRecord struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	ProjectID    uuid.UUID
	TaskID       uuid.UUID
	WorkDate     time.Time
	Timezone     string
	StartedAt    time.Time
	EndedAt      *time.Time
	TotalMinutes *int
	CreatedAt    time.Time
	UpdatedAt    time.Time
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
