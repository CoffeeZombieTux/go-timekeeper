package uow

import (
	"context"
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestWithUnitOfWork_Commit(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	manager := NewUnitOfWorkManager(db)
	mock.ExpectBegin()
	mock.ExpectCommit()

	err = WithUnitOfWork(context.Background(), manager, func(unit *UnitOfWork) error {
		if unit.GetTransaction() == nil {
			t.Fatal("transaction should not be nil")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithUnitOfWork commit flow failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestWithUnitOfWork_RollbackOnError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	manager := NewUnitOfWorkManager(db)
	mock.ExpectBegin()
	mock.ExpectRollback()

	boom := errors.New("boom")
	err = WithUnitOfWork(context.Background(), manager, func(unit *UnitOfWork) error {
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("expected rollback error to bubble up, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
