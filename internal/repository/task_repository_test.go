package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"go-timekeeper/internal/logger"
	"go-timekeeper/internal/model"
	"go-timekeeper/internal/uow"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestTaskRepository_Flow(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	repo := NewTaskRepository(db, logger.New("error", "json"))
	ctx := context.Background()

	taskID := uuid.New()
	userID := uuid.New()
	projectID := uuid.New()
	now := time.Now().UTC()

	task := &model.Task{
		UserID:    userID,
		ProjectID: projectID,
		Name:      "task-1",
		Status:    model.StatusCreated,
	}

	mock.ExpectQuery(regexp.QuoteMeta(`
				INSERT INTO task (user_id, project_id, name, status)
				VALUES ($1, $2, $3, $4)
				RETURNING *;
			`)).
		WithArgs(userID, projectID, "task-1", model.StatusCreated).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "project_id", "name", "status", "created_at", "updated_at"}).
			AddRow(taskID, userID, projectID, "task-1", model.StatusCreated, now, now))
	if _, err := repo.Save(ctx, task, nil); err != nil {
		t.Fatalf("Save insert: %v", err)
	}

	task.Name = "task-2"
	mock.ExpectQuery(regexp.QuoteMeta(`
			UPDATE task SET project_id = $2, name = $3, status = $4 , updated_at = NOW()
			WHERE id = $1 RETURNING project_id, name, status, updated_at;
		`)).
		WithArgs(taskID, projectID, "task-2", model.StatusCreated).
		WillReturnRows(sqlmock.NewRows([]string{"project_id", "name", "status", "updated_at"}).
			AddRow(projectID, "task-2", model.StatusCreated, now))
	if _, err := repo.Save(ctx, task, nil); err != nil {
		t.Fatalf("Save update: %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, user_id, project_id, name, status, created_at, updated_at FROM task WHERE id = $1`)).
		WithArgs(taskID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "project_id", "name", "status", "created_at", "updated_at"}).
			AddRow(taskID, userID, projectID, "task-2", model.StatusCreated, now, now))
	if _, err := repo.Get(ctx, taskID, nil); err != nil {
		t.Fatalf("Get: %v", err)
	}

	isActive := true
	mock.ExpectQuery(`SELECT id, user_id, project_id, name, status, created_at, updated_at FROM task WHERE project_id = \$1 AND user_id = \$2 AND status IN \(\$3, \$4\) LIMIT \$5 OFFSET \$6`).
		WithArgs(projectID, userID, model.StatusCreated, model.StatusWorkingOn, 10, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "project_id", "name", "status", "created_at", "updated_at"}).
			AddRow(taskID, userID, projectID, "task-2", model.StatusCreated, now, now))
	if _, err := repo.GetByProjectAndUserId(ctx, projectID, userID, &isActive, 10, 0); err != nil {
		t.Fatalf("GetByProjectAndUserId: %v", err)
	}

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM task WHERE project_id = \$1 AND user_id = \$2 AND status IN \(\$3, \$4\)`).
		WithArgs(projectID, userID, model.StatusCreated, model.StatusWorkingOn).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	if _, err := repo.CountByProjectAndUserId(ctx, projectID, userID, &isActive); err != nil {
		t.Fatalf("CountByProjectAndUserId: %v", err)
	}

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM task WHERE id = $1`)).
		WithArgs(taskID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.Delete(ctx, task, nil); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Get with FOR UPDATE path (inside transaction)
	uowManager := uow.NewUnitOfWorkManager(db)
	mock.ExpectBegin()
	unit, err := uowManager.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, user_id, project_id, name, status, created_at, updated_at FROM task WHERE id = $1 FOR UPDATE`)).
		WithArgs(taskID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "project_id", "name", "status", "created_at", "updated_at"}).
			AddRow(taskID, userID, projectID, "task-2", model.StatusCreated, now, now))
	if _, err := repo.Get(ctx, taskID, unit); err != nil {
		t.Fatalf("Get FOR UPDATE: %v", err)
	}
	mock.ExpectRollback()
	if err := unit.Rollback(); err != nil {
		t.Fatalf("Rollback: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
