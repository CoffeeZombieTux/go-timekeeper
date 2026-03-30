package api_model

import (
	"fmt"
	"go-timekeeper/internal/apperror"
	"go-timekeeper/internal/validator"
	"time"

	"github.com/google/uuid"
)

// RegisterRequest represents the request body for user registration.
type RegisterRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// ValidateRegisterRequest validates the RegisterRequest.
func (input *RegisterRequest) ValidateRegisterRequest() error {
	err := validator.ValidateEmail(input.Email)
	if err != nil {
		return err
	}
	err = validator.ValidatePassword(input.Password)
	if err != nil {
		return err
	}
	return nil
}

// LoginRequest represents the request body for user login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RefreshRequest represents the request body for refreshing the access token or logout.
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// ChangePasswordRequest represents the request body for changing the password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required"`
}

// ValidateChangePasswordRequest validates the ChangePasswordRequest.
func (input *ChangePasswordRequest) ValidateChangePasswordRequest() error {
	err := validator.ValidatePassword(input.NewPassword)
	if err != nil {
		return err
	}
	return nil
}

// CreateProjectRequest represents the request body for creating a new project.
type CreateProjectRequest struct {
	Name string `json:"name" binding:"required"`
}

// UpdateProjectRequest represents the request body for updating a project.
type UpdateProjectRequest struct {
	ID   uuid.UUID `json:"id" binding:"required"`
	Name string    `json:"name" binding:"required"`
}

// CreateTaskRequest represents the request body for creating a new task.
type CreateTaskRequest struct {
	ProjectID uuid.UUID `json:"projectID" binding:"required"`
	Name      string    `json:"name" binding:"required"`
}

// UpdateTaskRequest represents the request body for updating a task.
type UpdateTaskRequest struct {
	ID        uuid.UUID `json:"id" binding:"required"`
	ProjectID uuid.UUID `json:"projectID" binding:"required"`
	Name      string    `json:"name" binding:"required"`
}

// PaginationParams represents pagination parameters for a request
type PaginationParams struct {
	Limit  int
	Offset int
}

// NewPaginationParams creates a new PaginationParams instance
func NewPaginationParams(requestedLimit, requestedOffset int) PaginationParams {
	const (
		DefaultLimit = 50
		MaxLimit     = 10000
		MinLimit     = 1
	)

	// Apply constraints to the limit
	limit := DefaultLimit
	if requestedLimit >= MinLimit {
		limit = requestedLimit
		if limit > MaxLimit {
			// Silent Adjustment
			limit = MaxLimit
		}
	}

	// Ensure the offset is not negative
	offset := 0
	if requestedOffset > 0 {
		offset = requestedOffset
	}

	return PaginationParams{
		Limit:  limit,
		Offset: offset,
	}
}

// TimeRangeParams represents the time range parameters for a request
type TimeRangeParams struct {
	FromDate time.Time `json:"fromDate" binding:"required"`
	ToDate   time.Time `json:"toDate" binding:"required"`
}

// ValidateTimeRangeParams validates the TimeRangeParams.
func (input *TimeRangeParams) ValidateTimeRangeParams() error {
	if input.ToDate.Before(input.FromDate) {
		return apperror.New(
			apperror.CodeValidationErrorCode,
			apperror.CodeValidationErrorMessage,
			fmt.Sprintf(
				"stop time should be after start. Started at: %sm stop at: %s",
				input.ToDate,
				input.FromDate,
			),
		)
	}
	return nil
}

// GeneralReportRequest represents the request body for generating a report
type GeneralReportRequest struct {
	Projects  *[]uuid.UUID     `json:"projects"`
	TimeRange *TimeRangeParams `json:"timeRange" binding:"required"`
}

// ProjectReportRequest represents the request body for generating a report
type ProjectReportRequest struct {
	ProjectID uuid.UUID        `json:"projectID" binding:"required"`
	Tasks     *[]uuid.UUID     `json:"tasks"`
	TimeRange *TimeRangeParams `json:"timeRange" binding:"required"`
}

// TaskReportRequest represents the request body for generating a report
type TaskReportRequest struct {
	TaskID    uuid.UUID        `json:"taskID" binding:"required"`
	TimeRange *TimeRangeParams `json:"timeRange" binding:"required"`
}
