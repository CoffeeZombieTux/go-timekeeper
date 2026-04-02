package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"go-timekeeper/internal/logger"
	"go-timekeeper/internal/model"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestTimeRecordRepository_MainMethods(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	repo := NewTimeRecordRepository(db, logger.New("error", "json"))
	ctx := context.Background()

	recID := uuid.New()
	userID := uuid.New()
	projectID := uuid.New()
	taskID := uuid.New()
	workDate := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	start := time.Date(2026, 3, 31, 8, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 31, 9, 0, 0, 0, time.UTC)
	minutes := 60
	now := time.Now().UTC()

	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			id, user_id, project_id, task_id,
			work_date, work_timezone, started_at, ended_at, total_minutes,
			created_at, updated_at
		FROM time_record
		WHERE id = $1
		FOR UPDATE
	`)).
		WithArgs(recID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "project_id", "task_id", "work_date", "work_timezone", "started_at", "ended_at", "total_minutes", "created_at", "updated_at"}).
			AddRow(recID, userID, projectID, taskID, workDate, "Europe/Prague", start, end, minutes, now, now))
	if _, err := repo.GetForUpdate(ctx, recID, tx); err != nil {
		t.Fatalf("GetForUpdate: %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			id, user_id, project_id, task_id,
			work_date, work_timezone, started_at, ended_at, total_minutes,
			created_at, updated_at
		FROM time_record
		WHERE user_id = $1
		  AND ended_at IS NULL
		FOR UPDATE
	`)).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "project_id", "task_id", "work_date", "work_timezone", "started_at", "ended_at", "total_minutes", "created_at", "updated_at"}).
			AddRow(recID, userID, projectID, taskID, workDate, "Europe/Prague", start, nil, nil, now, now))
	if _, err := repo.GetActiveByUserForUpdate(ctx, userID, tx); err != nil {
		t.Fatalf("GetActiveByUserForUpdate: %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			id, user_id, project_id, task_id,
			work_date, work_timezone, started_at, ended_at, total_minutes,
			created_at, updated_at
		FROM time_record
		WHERE task_id = $1
		FOR UPDATE
	`)).
		WithArgs(taskID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "project_id", "task_id", "work_date", "work_timezone", "started_at", "ended_at", "total_minutes", "created_at", "updated_at"}).
			AddRow(recID, userID, projectID, taskID, workDate, "Europe/Prague", start, end, minutes, now, now))
	if _, err := repo.GetListByTaskForUpdate(ctx, taskID, tx); err != nil {
		t.Fatalf("GetListByTaskForUpdate: %v", err)
	}

	mock.ExpectExec(regexp.QuoteMeta(`
		DELETE FROM time_record
		WHERE id = $1
	`)).
		WithArgs(recID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.Delete(ctx, &model.TimeRecord{ID: recID}, tx); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback tx: %v", err)
	}

	record := &model.TimeRecord{
		UserID:       userID,
		ProjectID:    projectID,
		TaskID:       taskID,
		WorkDate:     workDate,
		Timezone:     "Europe/Prague",
		StartedAt:    start,
		EndedAt:      &end,
		TotalMinutes: &minutes,
	}
	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO time_record (
			id, user_id, project_id, task_id,
			work_date, work_timezone, started_at, ended_at, total_minutes,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING
			id, user_id, project_id, task_id,
			work_date, work_timezone, started_at, ended_at, total_minutes,
			created_at, updated_at
	`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "project_id", "task_id", "work_date", "work_timezone", "started_at", "ended_at", "total_minutes", "created_at", "updated_at"}).
			AddRow(recID, userID, projectID, taskID, workDate, "Europe/Prague", start, end, minutes, now, now))
	created, err := repo.Create(ctx, record, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	updatedEnd := end.Add(30 * time.Minute)
	updatedMinutes := 90
	created.EndedAt = &updatedEnd
	created.TotalMinutes = &updatedMinutes
	mock.ExpectQuery(regexp.QuoteMeta(`
		UPDATE time_record
		SET
			user_id = $2,
			project_id = $3,
			task_id = $4,
			work_date = $5,
			work_timezone = $6,
			started_at = $7,
			ended_at = $8,
			total_minutes = $9,
			updated_at = NOW()
		WHERE id = $1
		RETURNING
			id, user_id, project_id, task_id,
			work_date, work_timezone, started_at, ended_at, total_minutes,
			created_at, updated_at
	`)).
		WithArgs(created.ID, userID, projectID, taskID, workDate, "Europe/Prague", start, updatedEnd, updatedMinutes).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "project_id", "task_id", "work_date", "work_timezone", "started_at", "ended_at", "total_minutes", "created_at", "updated_at"}).
			AddRow(recID, userID, projectID, taskID, workDate, "Europe/Prague", start, updatedEnd, updatedMinutes, now, now))
	if _, err := repo.Update(ctx, created, nil); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestTimeRecordRepository_ReportQueries(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	repo := NewTimeRecordRepository(db, logger.New("error", "json"))
	ctx := context.Background()

	userID := uuid.New()
	projectID := uuid.New()
	taskID := uuid.New()
	workDate := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)

	reportRows := sqlmock.NewRows([]string{"id", "project_id", "task_id", "work_date", "work_timezone", "total_minutes"}).
		AddRow(uuid.New(), projectID, taskID, workDate, "Europe/Prague", 30)

	mock.ExpectQuery("FROM time_record").
		WillReturnRows(reportRows)
	if rows, err := repo.GetTaskReportRows(ctx, userID, taskID, workDate, workDate); err != nil || len(rows) != 1 {
		t.Fatalf("GetTaskReportRows failed len=%d err=%v", len(rows), err)
	}

	reportRows2 := sqlmock.NewRows([]string{"id", "project_id", "task_id", "work_date", "work_timezone", "total_minutes"}).
		AddRow(uuid.New(), projectID, taskID, workDate, "Europe/Prague", 45)
	mock.ExpectQuery("FROM time_record").
		WillReturnRows(reportRows2)
	if rows, err := repo.GetProjectReportRows(ctx, userID, projectID, nil, workDate, workDate); err != nil || len(rows) != 1 {
		t.Fatalf("GetProjectReportRows failed len=%d err=%v", len(rows), err)
	}

	reportRows3 := sqlmock.NewRows([]string{"id", "project_id", "task_id", "work_date", "work_timezone", "total_minutes"}).
		AddRow(uuid.New(), projectID, taskID, workDate, "Europe/Prague", 55)
	mock.ExpectQuery("FROM time_record").
		WillReturnRows(reportRows3)
	if rows, err := repo.GetGeneralReportRows(ctx, userID, nil, workDate, workDate); err != nil || len(rows) != 1 {
		t.Fatalf("GetGeneralReportRows failed len=%d err=%v", len(rows), err)
	}

	mock.ExpectQuery("FROM time_record").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "project_id", "task_id", "work_date", "work_timezone", "started_at", "ended_at", "total_minutes", "created_at", "updated_at"}).
			AddRow(uuid.New(), userID, projectID, taskID, workDate, "Europe/Prague", workDate, nil, nil, workDate, workDate))
	if rows, err := repo.GetTaskDayClosedRecords(ctx, userID, taskID, workDate); err != nil || len(rows) != 1 {
		t.Fatalf("GetTaskDayClosedRecords failed len=%d err=%v", len(rows), err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
