package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"go-timekeeper/internal/logger"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestRefreshTokenRepository_Flow(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	repo := NewRefreshTokenRepository(db, logger.New("error", "json"))
	ctx := context.Background()

	userID := uuid.New()
	hash := "hash-1"
	exp := time.Now().UTC().Add(24 * time.Hour)
	now := time.Now().UTC()

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO refresh_token (user_id, token_hash, expires_at)
			VALUES ($1, $2, $3)`)).
		WithArgs(userID, hash, exp).
		WillReturnResult(sqlmock.NewResult(1, 1))
	if err := repo.Create(ctx, userID, hash, exp); err != nil {
		t.Fatalf("Create: %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
			FROM refresh_token
			WHERE token_hash = $1
			  AND revoked_at IS NULL
			  AND expires_at > NOW()`)).
		WithArgs(hash).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "token_hash", "expires_at", "revoked_at", "created_at"}).
			AddRow(uuid.New(), userID, hash, exp, nil, now))
	if _, err := repo.GetValidByHash(ctx, hash); err != nil {
		t.Fatalf("GetValidByHash: %v", err)
	}

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE refresh_token
			SET revoked_at = NOW()
			WHERE token_hash = $1 AND revoked_at IS NULL`)).
		WithArgs(hash).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.RevokeByHash(ctx, hash); err != nil {
		t.Fatalf("RevokeByHash: %v", err)
	}

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE refresh_token
			SET revoked_at = NOW()
			WHERE user_id = $1 AND revoked_at IS NULL`)).
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.RevokeAllForUser(ctx, userID); err != nil {
		t.Fatalf("RevokeAllForUser: %v", err)
	}

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM refresh_token
			WHERE
				(revoked_at IS NOT NULL AND revoked_at < NOW() - ($1 * INTERVAL '1 days'))
				OR
				(expires_at < NOW())`)).
		WithArgs(7).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.CleanUp(ctx, 7); err != nil {
		t.Fatalf("CleanUp: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
