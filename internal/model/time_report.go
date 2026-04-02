package model

import (
	"time"

	"github.com/google/uuid"
)

// TimeRecordReportRow represents a row in the time record report.
type TimeRecordReportRow struct {
	ID           uuid.UUID `json:"id"`
	ProjectID    uuid.UUID `json:"projectId"`
	TaskID       uuid.UUID `json:"taskId"`
	WorkDate     time.Time `json:"workDate"`
	WorkTimezone string    `json:"workTimezone"`
	TotalMinutes int       `json:"totalMinutes"`
}
