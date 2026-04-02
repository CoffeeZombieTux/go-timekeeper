package model

import (
	"time"

	"github.com/google/uuid"
)

// Project represents a project entity in the database.
type Project struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	UserID    uuid.UUID `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
