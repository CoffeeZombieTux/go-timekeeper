package repository

import (
	"context"
	"database/sql"
)

// SQLExecutor represents a generic SQL executor
// Used for run queries with `*sql.DB` and `*sql.Tx`
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
