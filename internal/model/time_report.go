package model

import (
	"time"

	"github.com/google/uuid"
)

// TimeRecordReportRow represents a row in the time record report.
type TimeRecordReportRow struct {
	ID           uuid.UUID
	ProjectID    uuid.UUID
	TaskID       uuid.UUID
	WorkDate     time.Time
	WorkTimezone string
	TotalMinutes int
}
