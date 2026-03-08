package apperror

import (
	"database/sql"
	"errors"
	"strings"
)

const (
	CodeValidationErrorCode            = "VALIDATION_ERROR"
	CodeUnauthorizedCode               = "UNAUTHORIZED"
	CodeNotFoundCode                   = "NOT_FOUND"
	CodeConflictCode                   = "CONFLICT"
	CodeInsufficientStockCode          = "INSUFFICIENT_STOCK"
	CodeReservationVersionConflictCode = "RESERVATION_VERSION_CONFLICT"
	CodeDBNoRowsCode                   = "DB_NO_ROWS"
	CodeDBDuplicateKeyCode             = "DB_DUPLICATE_KEY"
	CodeDBForeignKeyConstraintCode     = "DB_FOREIGN_KEY_CONSTRAINT"
	CodeDBCheckConstraintCode          = "DB_CHECK_CONSTRAINT"
	CodeInternalErrorCode              = "INTERNAL_ERROR"
)

const (
	CodeValidationErrorMessage            = "Validation failed"
	CodeUnauthorizedMessage               = "Unauthorized"
	CodeNotFoundMessage                   = "Resource not found"
	CodeConflictMessage                   = "Conflict"
	CodeInsufficientStockMessage          = "Insufficient stock for requested items"
	CodeReservationVersionConflictMessage = "Reservation was updated concurrently. Please retry."
	CodeDBNoRowsMessage                   = "Resource not found"
	CodeDBDuplicateKeyMessage             = "Duplicate key violation"
	CodeDBForeignKeyConstraintMessage     = "Foreign key constraint violation"
	CodeDBCheckConstraintMessage          = "Check constraint violation"
	CodeInternalErrorMessage              = "Internal server error"
)

// Backward-compatible aliases.
const (
	CodeValidationError            = CodeValidationErrorCode
	CodeUnauthorized               = CodeUnauthorizedCode
	CodeNotFound                   = CodeNotFoundCode
	CodeConflict                   = CodeConflictCode
	CodeInsufficientStock          = CodeInsufficientStockCode
	CodeReservationVersionConflict = CodeReservationVersionConflictCode
	CodeDBNoRows                   = CodeDBNoRowsCode
	CodeDBDuplicateKey             = CodeDBDuplicateKeyCode
	CodeDBForeignKeyConstraint     = CodeDBForeignKeyConstraintCode
	CodeDBCheckConstraint          = CodeDBCheckConstraintCode
	CodeInternalError              = CodeInternalErrorCode
)

// Error is a structured application error with machine-readable code and details.
type Error struct {
	Code    string
	Message string
	Details []string
	Cause   error
}

// Error returns a human-readable error string.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// Unwrap returns the wrapped cause.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// New creates a new structured application error.
func New(code, message string, details ...string) error {
	return &Error{Code: code, Message: message, Details: details}
}

// Wrap wraps cause with a structured application error.
func Wrap(cause error, code, message string, details ...string) error {
	if cause == nil {
		return nil
	}
	return &Error{Code: code, Message: message, Details: details, Cause: cause}
}

// As extracts structured application error from err.
func As(err error) (*Error, bool) {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

// FromDB maps database error to structured database/application error.
func FromDB(err error, defaultMessage string, defaultCode string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return New(CodeDBNoRows, CodeDBNoRowsMessage, err.Error())
	}

	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "duplicate key value violates unique constraint"),
		strings.Contains(msg, "violates unique constraint"):
		return Wrap(err, CodeDBDuplicateKey, CodeDBDuplicateKeyMessage)
	case strings.Contains(msg, "violates foreign key constraint"):
		return Wrap(err, CodeDBForeignKeyConstraint, CodeDBForeignKeyConstraintMessage)
	case strings.Contains(msg, "violates check constraint"):
		return Wrap(err, CodeDBCheckConstraint, CodeDBCheckConstraintMessage)
	default:
		if defaultCode == "" {
			defaultCode = CodeInternalError
		}
		if defaultMessage == "" {
			defaultMessage = CodeInternalErrorMessage
		}
		return Wrap(err, defaultCode, defaultMessage)
	}
}
