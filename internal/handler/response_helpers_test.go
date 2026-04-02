package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"testing"

	"go-timekeeper/internal/apperror"
)

func TestMapDomainError_DuplicateKey(t *testing.T) {
	err := errors.New("duplicate key value violates unique constraint \"uq_x\"")
	status, code, message, _ := mapDomainError(err)

	if status != http.StatusConflict {
		t.Fatalf("expected 409, got %d", status)
	}
	if code != apperror.CodeDBDuplicateKeyCode {
		t.Fatalf("expected code %s, got %s", apperror.CodeDBDuplicateKeyCode, code)
	}
	if message != apperror.CodeDBDuplicateKeyMessage {
		t.Fatalf("expected message %q, got %q", apperror.CodeDBDuplicateKeyMessage, message)
	}
}

func TestMapDomainError_NoRows(t *testing.T) {
	status, code, _, _ := mapDomainError(sql.ErrNoRows)

	if status != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", status)
	}
	if code != apperror.CodeDBNoRowsCode {
		t.Fatalf("expected code %s, got %s", apperror.CodeDBNoRowsCode, code)
	}
}
