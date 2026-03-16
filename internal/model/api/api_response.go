package api_model

import (
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
