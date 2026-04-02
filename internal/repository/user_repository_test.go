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

func TestUserRepository_CRUD(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewUserRepository(db, logger.New("error", "json"))
	ctx := context.Background()

	now := time.Now().UTC()
	userID := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "user" (email, password_hash) 
				VALUES ($1, $2) RETURNING id, email, password_hash, created_at, updated_at`)).
		WithArgs("u@example.com", "hash").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash", "created_at", "updated_at"}).
			AddRow(userID, "u@example.com", "hash", now, now))

	created, err := repo.Create(ctx, "u@example.com", "hash")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID != userID {
		t.Fatalf("unexpected user id %s", created.ID)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, email, password_hash, created_at, updated_at FROM "user" WHERE id = $1`)).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash", "created_at", "updated_at"}).
			AddRow(userID, "u@example.com", "hash", now, now))
	if _, err := repo.GetById(ctx, userID); err != nil {
		t.Fatalf("GetById: %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, email, password_hash, created_at, updated_at FROM "user" WHERE email = $1`)).
		WithArgs("u@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash", "created_at", "updated_at"}).
			AddRow(userID, "u@example.com", "hash", now, now))
	if _, err := repo.GetByEmail(ctx, "u@example.com"); err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "user"
			SET password_hash = $1, updated_at = NOW()
			WHERE id = $2`)).
		WithArgs("hash2", userID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.UpdatePassword(ctx, userID, "hash2"); err != nil {
		t.Fatalf("UpdatePassword: %v", err)
	}

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "user" WHERE id = $1`)).
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.Delete(ctx, created); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
