package api_model

import (
	"go-timekeeper/internal/model"

	"github.com/google/uuid"
)

// AuthResponse represents the response body for authentication.
type AuthResponse struct {
	AccessToken  string      `json:"accessToken"`
	RefreshToken string      `json:"refreshToken"`
	User         UserPayload `json:"user"`
}

// UserPayload represents the user payload.
type UserPayload struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	CreatedAt string    `json:"createdAt"`
	UpdatedAt string    `json:"updatedAt"`
}

// PaginationResponse represents a response containing pagination metadata
type PaginationResponse struct {
	Limit       int `json:"limit"`
	Offset      int `json:"offset"`
	TotalItems  int `json:"totalItems"`
	CurrentPage int `json:"currentPage"`
	TotalPages  int `json:"totalPages"`
}

// GeneralReportResponse represents the response body for generating a report
type GeneralReportResponse struct {
	Projects     []*ProjectReportResponse `json:"projects"`
	TotalMinutes int                      `json:"totalMinutes"`
	TimeRange    *TimeRangeParams         `json:"timeRange"`
}

// ProjectReportResponse represents the response body for generating a report
type ProjectReportResponse struct {
	ProjectID    uuid.UUID             `json:"projectId"`
	ProjectName  string                `json:"projectName"`
	Tasks        []*TaskReportResponse `json:"tasks"`
	TotalMinutes int                   `json:"totalMinutes"`
	TimeRange    *TimeRangeParams      `json:"timeRange,omitempty"`
}

// TaskReportResponse represents the response body for generating a report
type TaskReportResponse struct {
	TaskId       uuid.UUID            `json:"taskId"`
	TaskName     string               `json:"taskName"`
	DayReports   []*DayReportResponse `json:"dayReports"`
	TotalMinutes int                  `json:"totalMinutes"`
	TimeRange    *TimeRangeParams     `json:"timeRange,omitempty"`
}

// DayReportResponse represents the response body for generating a report
type DayReportResponse struct {
	WorkingDate  string                       `json:"workingDate"`
	WorkTimezone string                       `json:"workTimezone"`
	Sessions     []*model.TimeRecordReportRow `json:"sessions"`
	TotalMinutes int                          `json:"totalMinutes"`
	TimeRange    *TimeRangeParams             `json:"timeRange,omitempty"`
}
