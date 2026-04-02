package repository

import (
	"context"
	"database/sql"
	"go-timekeeper/internal/logger"
	"go-timekeeper/internal/model"

	"github.com/google/uuid"
)

// UserRepositoryInterface represents a user repository.
type UserRepositoryInterface interface {
	GetById(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	Create(ctx context.Context, email string, passwordHash string) (*model.User, error)
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	Delete(ctx context.Context, user *model.User) error
}

// UserRepository is a struct that implements the UserRepositoryInterface.
type UserRepository struct {
	db     *sql.DB
	logger *logger.Logger
}

// NewUserRepository creates a new UserRepository instance.
func NewUserRepository(db *sql.DB, logger *logger.Logger) UserRepositoryInterface {
	return &UserRepository{
		db:     db,
		logger: logger,
	}
}

// GetById returns a user by ID.
func (userRepo *UserRepository) GetById(ctx context.Context, id uuid.UUID) (*model.User, error) {
	query := `SELECT id, email, password_hash, created_at, updated_at FROM "user" WHERE id = $1`

	var user model.User
	err := userRepo.db.QueryRowContext(
		ctx,
		query,
		id,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmail returns a user by email.
func (userRepo *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `SELECT id, email, password_hash, created_at, updated_at FROM "user" WHERE email = $1`

	var user model.User
	err := userRepo.db.QueryRowContext(
		ctx,
		query,
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Create creates a new user.
func (userRepo *UserRepository) Create(ctx context.Context, email string, passwordHash string) (*model.User, error) {
	query := `
			INSERT INTO "user" (email, password_hash) 
			VALUES ($1, $2) RETURNING id, email, password_hash, created_at, updated_at
	`

	var user model.User
	err := userRepo.db.QueryRowContext(
		ctx,
		query,
		email,
		passwordHash,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// UpdatePassword updates the password of a user.
func (userRepo *UserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	query := `
		UPDATE "user"
		SET password_hash = $1, updated_at = NOW()
		WHERE id = $2
	`
	_, err := userRepo.db.ExecContext(ctx, query, passwordHash, id)
	if err != nil {
		return err
	}
	return nil
}

// Delete deletes a user.
func (userRepo *UserRepository) Delete(ctx context.Context, user *model.User) error {
	query := `DELETE FROM "user" WHERE id = $1`
	_, err := userRepo.db.ExecContext(ctx, query, user.ID)
	if err != nil {
		return err
	}
	return nil
}
