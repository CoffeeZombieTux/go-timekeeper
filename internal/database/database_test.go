package database

import (
	"testing"

	"go-timekeeper/internal/logger"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestDatabase_ClosePingHealthCheck(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	log := logger.New("error", "json")
	d := &Database{DB: db, logger: log.Logger}

	mock.ExpectPing()
	if err := d.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}

	mock.ExpectQuery("SELECT 1").
		WillReturnRows(sqlmock.NewRows([]string{"?column?"}).AddRow(1))
	if err := d.HealthCheck(); err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}

	mock.ExpectClose()
	if err := d.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestDatabase_NewPingFailure(t *testing.T) {
	log := logger.New("error", "json")
	_, err := New("host=127.0.0.1 port=1 user=x password=x dbname=x sslmode=disable connect_timeout=1", log.Logger)
	if err == nil {
		t.Fatal("expected New to fail for unreachable DB")
	}
}
