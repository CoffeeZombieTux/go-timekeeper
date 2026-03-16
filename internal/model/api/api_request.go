package api_model

import (
	"go-timekeeper/internal/validator"

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
