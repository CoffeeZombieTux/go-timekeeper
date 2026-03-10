package repository

import (
	"context"
	"database/sql"
	"go-timekeeper/internal/logger"
	"go-timekeeper/internal/model"
	"time"

	"github.com/google/uuid"
)

// RefreshTokenRepositoryInterface represents a repository for refresh tokens.
type RefreshTokenRepositoryInterface interface {
	Create(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	GetValidByHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error)
	RevokeByHash(ctx context.Context, tokenHash string) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
	CleanUp(ctx context.Context, interval int) error
}

// RefreshTokenRepository is a struct that implements the RefreshTokenRepositoryInterface.
type RefreshTokenRepository struct {
	db     *sql.DB
	logger *logger.Logger
}

// NewRefreshTokenRepository creates a new RefreshTokenRepository instance.
func NewRefreshTokenRepository(db *sql.DB, logger *logger.Logger) RefreshTokenRepositoryInterface {
	return &RefreshTokenRepository{
		db:     db,
		logger: logger,
	}
}

// Create creates a new refresh token.
func (tokenRepo *RefreshTokenRepository) Create(
	ctx context.Context,
	userID uuid.UUID,
	tokenHash string,
	expiresAt time.Time,
) error {
	query := `
		INSERT INTO refresh_token (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`
	_, err := tokenRepo.db.ExecContext(ctx, query, userID, tokenHash, expiresAt)
	return err
}

// GetValidByHash gets a valid refresh token by its hash.
func (tokenRepo *RefreshTokenRepository) GetValidByHash(
	ctx context.Context,
	tokenHash string,
) (*model.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		FROM refresh_token
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
	`
	var token model.RefreshToken
	err := tokenRepo.db.QueryRowContext(ctx, query, tokenHash).
		Scan(&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &token.RevokedAt, &token.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// RevokeByHash revokes a refresh token by its hash.
func (tokenRepo *RefreshTokenRepository) RevokeByHash(ctx context.Context, tokenHash string) error {
	query := `
		UPDATE refresh_token
		SET revoked_at = NOW()
		WHERE token_hash = $1 AND revoked_at IS NULL
	`
	_, err := tokenRepo.db.ExecContext(ctx, query, tokenHash)
	return err
}

// RevokeAllForUser revokes all refresh tokens for a user.
func (tokenRepo *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE refresh_token
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`
	_, err := tokenRepo.db.ExecContext(ctx, query, userID)
	return err
}

// CleanUp deletes expired and revoked refresh tokens.
func (tokenRepo *RefreshTokenRepository) CleanUp(ctx context.Context, interval int) error {
	query := `
		DELETE FROM refresh_token
		WHERE
			(revoked_at IS NOT NULL AND revoked_at < NOW() - ($1 * INTERVAL '1 days'))
			OR
			(expires_at < NOW())
	`
	_, err := tokenRepo.db.ExecContext(ctx, query, interval)
	return err
}
