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
