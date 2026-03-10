package model

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken represents a refresh token entity in the database.
type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}
