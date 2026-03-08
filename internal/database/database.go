package database

import (
	"database/sql"
	"fmt"
	applogger "go-timekeeper/internal/logger"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// Database represents a database connection
type Database struct {
	DB     *sql.DB
	logger *logrus.Logger
}

// New creates a new Database instance
func New(dsn string, logger *logrus.Logger) (*Database, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Database connection established successfully")

	return &Database{
		DB:     db,
		logger: logger,
	}, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	if d.DB != nil {
		d.logger.Info("Closing database connection")
		return d.DB.Close()
	}
	return nil
}

// Ping checks if the database connection is alive
func (d *Database) Ping() error {
	return d.DB.Ping()
}

// HealthCheck runs a simple query to check if the database connection is alive
func (d *Database) HealthCheck() error {
	var result int
	err := d.DB.QueryRow("SELECT 1").Scan(&result)
	if err != nil {
		d.logger.WithError(err).Error(applogger.LogMessageDatabaseHealthCheckFailed)
		return err
	}
	return nil
}
