package model

import (
	"time"

	"github.com/google/uuid"
)

// Project represents a project entity in the database.
type Project struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	UserID    uuid.UUID `json:"user_id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
