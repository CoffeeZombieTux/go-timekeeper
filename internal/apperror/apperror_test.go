package apperror

import (
	"database/sql"
	"errors"
	"testing"
)

func TestErrorFormattingAndUnwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := &Error{
		Code:    CodeInternalErrorCode,
		Message: "something failed",
		Cause:   cause,
	}

	if err.Error() != "something failed: root cause" {
		t.Fatalf("unexpected error string: %q", err.Error())
	}
	if !errors.Is(err, cause) {
		t.Fatal("expected wrapped cause")
	}
}

func TestNewWrapAndAs(t *testing.T) {
	base := New(CodeValidationErrorCode, CodeValidationErrorMessage, "d1")
	appErr, ok := As(base)
	if !ok {
		t.Fatal("expected app error")
	}
	if appErr.Code != CodeValidationErrorCode {
		t.Fatalf("unexpected code %s", appErr.Code)
	}

	cause := errors.New("boom")
	wrapped := Wrap(cause, CodeConflictCode, CodeConflictMessage, "d2")
	wrappedErr, ok := As(wrapped)
	if !ok {
		t.Fatal("expected wrapped app error")
	}
	if wrappedErr.Cause == nil {
		t.Fatal("expected wrapped cause")
	}
}

func TestFromDBMappings(t *testing.T) {
	if err := FromDB(nil, "", ""); err != nil {
		t.Fatal("expected nil error for nil input")
	}

	noRows := FromDB(sql.ErrNoRows, "", "")
	noRowsErr, _ := As(noRows)
	if noRowsErr.Code != CodeDBNoRowsCode {
		t.Fatalf("unexpected code %s", noRowsErr.Code)
	}

	dup := FromDB(errors.New("duplicate key value violates unique constraint"), "", "")
	dupErr, _ := As(dup)
	if dupErr.Code != CodeDBDuplicateKeyCode {
		t.Fatalf("unexpected duplicate code %s", dupErr.Code)
	}

	fk := FromDB(errors.New("violates foreign key constraint"), "", "")
	fkErr, _ := As(fk)
	if fkErr.Code != CodeDBForeignKeyConstraintCode {
		t.Fatalf("unexpected foreign key code %s", fkErr.Code)
	}

	chk := FromDB(errors.New("violates check constraint"), "", "")
	chkErr, _ := As(chk)
	if chkErr.Code != CodeDBCheckConstraintCode {
		t.Fatalf("unexpected check constraint code %s", chkErr.Code)
	}

	def := FromDB(errors.New("unmapped db error"), "custom", CodeConflictCode)
	defErr, _ := As(def)
	if defErr.Code != CodeConflictCode {
		t.Fatalf("unexpected default code %s", defErr.Code)
	}
}
