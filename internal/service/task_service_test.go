package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"go-timekeeper/internal/apperror"
	"go-timekeeper/internal/logger"
	"go-timekeeper/internal/model"
	apimodel "go-timekeeper/internal/model/api"
	"go-timekeeper/internal/repository"
	"go-timekeeper/internal/uow"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

type fakeTaskTimeRecordService struct {
	startFn func(ctx context.Context, task *model.Task, timezone string, unit *uow.UnitOfWork) error
	stopFn  func(ctx context.Context, task *model.Task, unit *uow.UnitOfWork) error
}

func (f *fakeTaskTimeRecordService) StartTask(
	ctx context.Context,
	task *model.Task,
	timezone string,
	unit *uow.UnitOfWork,
) error {
	if f.startFn != nil {
		return f.startFn(ctx, task, timezone, unit)
	}
	return nil
}

func (f *fakeTaskTimeRecordService) StopTask(ctx context.Context, task *model.Task, unit *uow.UnitOfWork) error {
	if f.stopFn != nil {
		return f.stopFn(ctx, task, unit)
	}
	return nil
}

func (f *fakeTaskTimeRecordService) CreateTimeRecord(
	ctx context.Context,
	req *apimodel.CreateTimeRecordRequest,
) (*model.TimeRecord, error) {
	return nil, nil
}

func (f *fakeTaskTimeRecordService) UpdateTimeRecord(
	ctx context.Context,
	req *apimodel.UpdateTimeRecordRequest,
) (*model.TimeRecord, error) {
	return nil, nil
}

func (f *fakeTaskTimeRecordService) DeleteTimeRecord(ctx context.Context, id uuid.UUID) error {
	return nil
}

func TestCheckStatusValidator(t *testing.T) {
	tests := []struct {
		name      string
		oldStatus model.TaskStatus
		newStatus model.TaskStatus
		want      bool
	}{
		{
			name:      "created to working is allowed",
			oldStatus: model.StatusCreated,
			newStatus: model.StatusWorkingOn,
			want:      true,
		},
		{
			name:      "working to created is allowed",
			oldStatus: model.StatusWorkingOn,
			newStatus: model.StatusCreated,
			want:      true,
		},
		{
			name:      "created to closed is allowed",
			oldStatus: model.StatusCreated,
			newStatus: model.StatusClosed,
			want:      true,
		},
		{
			name:      "working to closed is not allowed",
			oldStatus: model.StatusWorkingOn,
			newStatus: model.StatusClosed,
			want:      false,
		},
		{
			name:      "closed to working is not allowed",
			oldStatus: model.StatusClosed,
			newStatus: model.StatusWorkingOn,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkStatusValidator(tt.oldStatus, tt.newStatus)
			if got != tt.want {
				t.Fatalf("checkStatusValidator(%s -> %s) = %v, want %v", tt.oldStatus, tt.newStatus, got, tt.want)
			}
		})
	}
}

func TestCheckTaskUserAccess(t *testing.T) {
	userID := uuid.New()
	err := checkTaskUserAccess(userID, model.Task{UserID: userID})
	if err != nil {
		t.Fatalf("expected nil for same user, got %v", err)
	}

	err = checkTaskUserAccess(uuid.New(), model.Task{UserID: userID})
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
	appErr, ok := apperror.As(err)
	if !ok || appErr.Code != apperror.CodeUnauthorizedCode {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTaskService_Flow_StartStopCloseDelete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	userID := uuid.New()
	projectID := uuid.New()
	taskID := uuid.New()
	timeRecordID := uuid.New()
	createdAt := time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC)
	ctx := authenticatedContextForServiceTests(t, userID)

	log := logger.New("error", "json")
	taskRepo := repository.NewTaskRepository(db, log)
	timeRecordRepo := repository.NewTimeRecordRepository(db, log)
	timeRecordService := &TimeRecordService{timeRecordRepo: timeRecordRepo}
	svc := NewTaskService(taskRepo, timeRecordRepo, timeRecordService, uow.NewUnitOfWorkManager(db))

	taskCols := []string{"id", "user_id", "project_id", "name", "status", "created_at", "updated_at"}
	taskUpdateCols := []string{"project_id", "name", "status", "updated_at"}
	trCols := []string{
		"id", "user_id", "project_id", "task_id",
		"work_date", "work_timezone", "started_at", "ended_at", "total_minutes",
		"created_at", "updated_at",
	}

	mock.ExpectQuery("INSERT INTO task").
		WithArgs(userID, projectID, "task-flow", model.StatusCreated).
		WillReturnRows(sqlmock.NewRows(taskCols).
			AddRow(taskID, userID, projectID, "task-flow", model.StatusCreated, createdAt, createdAt))

	created, err := svc.Create(ctx, &apimodel.CreateTaskRequest{
		ProjectID: projectID,
		Name:      "task-flow",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if created.ID == uuid.Nil {
		t.Fatal("Create returned empty ID")
	}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT \\* FROM task WHERE id = \\$1 FOR UPDATE").
		WithArgs(taskID).
		WillReturnRows(sqlmock.NewRows(taskCols).
			AddRow(taskID, userID, projectID, "task-flow", model.StatusCreated, createdAt, createdAt))
	mock.ExpectQuery("UPDATE task SET").
		WithArgs(taskID, projectID, "task-flow", model.StatusWorkingOn).
		WillReturnRows(sqlmock.NewRows(taskUpdateCols).
			AddRow(projectID, "task-flow", model.StatusWorkingOn, createdAt))
	mock.ExpectQuery("FROM time_record[\\s\\S]*ended_at IS NULL").
		WithArgs(userID).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("INSERT INTO time_record").
		WithArgs(
			sqlmock.AnyArg(),
			userID,
			projectID,
			taskID,
			sqlmock.AnyArg(),
			"Europe/Prague",
			sqlmock.AnyArg(),
			nil,
			nil,
		).
		WillReturnRows(sqlmock.NewRows(trCols).
			AddRow(
				timeRecordID,
				userID,
				projectID,
				taskID,
				createdAt,
				"Europe/Prague",
				createdAt,
				nil,
				nil,
				createdAt,
				createdAt,
			))
	mock.ExpectCommit()

	if err = svc.Start(ctx, taskID, "Europe/Prague"); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	activeStarted := time.Now().UTC().Add(-40 * time.Minute)
	activeWorkDate := normalizeDate(activeStarted, time.UTC)
	closedAt := time.Now().UTC().Add(-5 * time.Minute)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT \\* FROM task WHERE id = \\$1 FOR UPDATE").
		WithArgs(taskID).
		WillReturnRows(sqlmock.NewRows(taskCols).
			AddRow(taskID, userID, projectID, "task-flow", model.StatusWorkingOn, createdAt, createdAt))
	mock.ExpectQuery("UPDATE task SET").
		WithArgs(taskID, projectID, "task-flow", model.StatusCreated).
		WillReturnRows(sqlmock.NewRows(taskUpdateCols).
			AddRow(projectID, "task-flow", model.StatusCreated, createdAt))
	mock.ExpectQuery("FROM time_record[\\s\\S]*ended_at IS NULL").
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows(trCols).
			AddRow(
				timeRecordID,
				userID,
				projectID,
				taskID,
				activeWorkDate,
				"Europe/Prague",
				activeStarted,
				nil,
				nil,
				createdAt,
				createdAt,
			))
	mock.ExpectQuery("UPDATE time_record").
		WithArgs(
			timeRecordID,
			userID,
			projectID,
			taskID,
			sqlmock.AnyArg(),
			"Europe/Prague",
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).
		WillReturnRows(sqlmock.NewRows(trCols).
			AddRow(
				timeRecordID,
				userID,
				projectID,
				taskID,
				activeWorkDate,
				"Europe/Prague",
				activeStarted,
				closedAt,
				35,
				createdAt,
				createdAt,
			))
	mock.ExpectCommit()

	if err = svc.Stop(ctx, taskID); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT \\* FROM task WHERE id = \\$1 FOR UPDATE").
		WithArgs(taskID).
		WillReturnRows(sqlmock.NewRows(taskCols).
			AddRow(taskID, userID, projectID, "task-flow", model.StatusCreated, createdAt, createdAt))
	mock.ExpectQuery("UPDATE task SET").
		WithArgs(taskID, projectID, "task-flow", model.StatusClosed).
		WillReturnRows(sqlmock.NewRows(taskUpdateCols).
			AddRow(projectID, "task-flow", model.StatusClosed, createdAt))
	mock.ExpectCommit()

	if err = svc.Close(ctx, taskID); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT \\* FROM task WHERE id = \\$1 FOR UPDATE").
		WithArgs(taskID).
		WillReturnRows(sqlmock.NewRows(taskCols).
			AddRow(taskID, userID, projectID, "task-flow", model.StatusClosed, createdAt, createdAt))
	mock.ExpectExec("DELETE FROM task WHERE id = \\$1").
		WithArgs(taskID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err = svc.Delete(ctx, taskID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestTaskService_Update_PropagatesProjectToTimeRecords(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	userID := uuid.New()
	oldProjectID := uuid.New()
	newProjectID := uuid.New()
	taskID := uuid.New()
	recordID := uuid.New()
	createdAt := time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC)
	ctx := authenticatedContextForServiceTests(t, userID)

	log := logger.New("error", "json")
	taskRepo := repository.NewTaskRepository(db, log)
	timeRecordRepo := repository.NewTimeRecordRepository(db, log)
	svc := NewTaskService(taskRepo, timeRecordRepo, &fakeTaskTimeRecordService{}, uow.NewUnitOfWorkManager(db))

	taskCols := []string{"id", "user_id", "project_id", "name", "status", "created_at", "updated_at"}
	taskUpdateCols := []string{"project_id", "name", "status", "updated_at"}
	trCols := []string{
		"id", "user_id", "project_id", "task_id",
		"work_date", "work_timezone", "started_at", "ended_at", "total_minutes",
		"created_at", "updated_at",
	}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT \\* FROM task WHERE id = \\$1 FOR UPDATE").
		WithArgs(taskID).
		WillReturnRows(sqlmock.NewRows(taskCols).
			AddRow(taskID, userID, oldProjectID, "task", model.StatusCreated, createdAt, createdAt))
	mock.ExpectQuery("UPDATE task SET").
		WithArgs(taskID, newProjectID, "task-new-name", model.StatusCreated).
		WillReturnRows(sqlmock.NewRows(taskUpdateCols).
			AddRow(newProjectID, "task-new-name", model.StatusCreated, createdAt))
	mock.ExpectQuery("FROM time_record[\\s\\S]*WHERE task_id = \\$1[\\s\\S]*FOR UPDATE").
		WithArgs(taskID).
		WillReturnRows(sqlmock.NewRows(trCols).
			AddRow(
				recordID,
				userID,
				oldProjectID,
				taskID,
				createdAt,
				"Europe/Prague",
				createdAt,
				createdAt,
				20,
				createdAt,
				createdAt,
			))
	mock.ExpectQuery("UPDATE time_record").
		WithArgs(
			recordID,
			userID,
			newProjectID,
			taskID,
			createdAt,
			"Europe/Prague",
			createdAt,
			createdAt,
			20,
		).
		WillReturnRows(sqlmock.NewRows(trCols).
			AddRow(
				recordID,
				userID,
				newProjectID,
				taskID,
				createdAt,
				"Europe/Prague",
				createdAt,
				createdAt,
				20,
				createdAt,
				createdAt,
			))
	mock.ExpectCommit()

	updated, err := svc.Update(ctx, &apimodel.UpdateTaskRequest{
		ID:        taskID,
		ProjectID: newProjectID,
		Name:      "task-new-name",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.ProjectID != newProjectID {
		t.Fatalf("expected project %s, got %s", newProjectID, updated.ProjectID)
	}

	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestTaskService_Start_InvalidStatus_BreakLogic(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	userID := uuid.New()
	projectID := uuid.New()
	taskID := uuid.New()
	createdAt := time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC)
	ctx := authenticatedContextForServiceTests(t, userID)

	log := logger.New("error", "json")
	taskRepo := repository.NewTaskRepository(db, log)
	timeRecordRepo := repository.NewTimeRecordRepository(db, log)
	svc := NewTaskService(taskRepo, timeRecordRepo, &fakeTaskTimeRecordService{}, uow.NewUnitOfWorkManager(db))

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT \\* FROM task WHERE id = \\$1 FOR UPDATE").
		WithArgs(taskID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "project_id", "name", "status", "created_at", "updated_at"}).
			AddRow(taskID, userID, projectID, "closed-task", model.StatusClosed, createdAt, createdAt))
	mock.ExpectRollback()

	err = svc.Start(ctx, taskID, "Europe/Prague")
	if err == nil {
		t.Fatal("expected validation error on invalid transition")
	}
	appErr, ok := apperror.As(err)
	if !ok || appErr.Code != apperror.CodeValidationErrorCode {
		t.Fatalf("expected validation error, got %v", err)
	}

	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestTaskService_Get_Unauthorized(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	userID := uuid.New()
	taskOwnerID := uuid.New()
	projectID := uuid.New()
	taskID := uuid.New()
	createdAt := time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC)
	ctx := authenticatedContextForServiceTests(t, userID)

	log := logger.New("error", "json")
	taskRepo := repository.NewTaskRepository(db, log)
	timeRecordRepo := repository.NewTimeRecordRepository(db, log)
	svc := NewTaskService(taskRepo, timeRecordRepo, &fakeTaskTimeRecordService{}, uow.NewUnitOfWorkManager(db))

	mock.ExpectQuery("SELECT \\* FROM task WHERE id = \\$1").
		WithArgs(taskID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "project_id", "name", "status", "created_at", "updated_at"}).
			AddRow(taskID, taskOwnerID, projectID, "foreign", model.StatusCreated, createdAt, createdAt))

	_, err = svc.Get(ctx, taskID, nil)
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
	appErr, ok := apperror.As(err)
	if !ok || appErr.Code != apperror.CodeUnauthorizedCode {
		t.Fatalf("expected unauthorized error, got %v", err)
	}
}

func TestTaskService_GetByProject_OutOfRange_BreakLogic(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	userID := uuid.New()
	projectID := uuid.New()
	ctx := authenticatedContextForServiceTests(t, userID)

	log := logger.New("error", "json")
	taskRepo := repository.NewTaskRepository(db, log)
	timeRecordRepo := repository.NewTimeRecordRepository(db, log)
	svc := NewTaskService(taskRepo, timeRecordRepo, &fakeTaskTimeRecordService{}, uow.NewUnitOfWorkManager(db))

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM task WHERE project_id = \\$1 AND user_id = \\$2").
		WithArgs(projectID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("SELECT \\* FROM task WHERE project_id = \\$1 AND user_id = \\$2 LIMIT \\$3 OFFSET \\$4").
		WithArgs(projectID, userID, 10, 30).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "project_id", "name", "status", "created_at", "updated_at"}))

	_, _, err = svc.GetByProject(ctx, projectID, nil, 10, 30)
	if err == nil {
		t.Fatal("expected out-of-range pagination error")
	}
	appErr, ok := apperror.As(err)
	if !ok || appErr.Code != apperror.CodeDBNoRowsCode {
		t.Fatalf("expected db no rows error, got %v", err)
	}
}

func TestTaskService_Update_Unauthorized(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	userID := uuid.New()
	taskOwnerID := uuid.New()
	oldProjectID := uuid.New()
	newProjectID := uuid.New()
	taskID := uuid.New()
	createdAt := time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC)
	ctx := authenticatedContextForServiceTests(t, userID)

	log := logger.New("error", "json")
	taskRepo := repository.NewTaskRepository(db, log)
	timeRecordRepo := repository.NewTimeRecordRepository(db, log)
	svc := NewTaskService(taskRepo, timeRecordRepo, &fakeTaskTimeRecordService{}, uow.NewUnitOfWorkManager(db))

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT \\* FROM task WHERE id = \\$1 FOR UPDATE").
		WithArgs(taskID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "project_id", "name", "status", "created_at", "updated_at"}).
			AddRow(taskID, taskOwnerID, oldProjectID, "foreign", model.StatusCreated, createdAt, createdAt))
	mock.ExpectRollback()

	_, err = svc.Update(ctx, &apimodel.UpdateTaskRequest{
		ID:        taskID,
		ProjectID: newProjectID,
		Name:      "new-name",
	})
	if err == nil {
		t.Fatal("expected unauthorized update error")
	}
	appErr, ok := apperror.As(err)
	if !ok || appErr.Code != apperror.CodeUnauthorizedCode {
		t.Fatalf("expected unauthorized error, got %v", err)
	}

	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
