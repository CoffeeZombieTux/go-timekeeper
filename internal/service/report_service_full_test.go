package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"go-timekeeper/internal/auth"
	"go-timekeeper/internal/logger"
	"go-timekeeper/internal/middleware"
	apimodel "go-timekeeper/internal/model/api"
	"go-timekeeper/internal/repository"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func reportAuthContext(t *testing.T, userID uuid.UUID) context.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	tm := auth.NewTokenManager("test-secret", 15, 24)
	token, err := tm.CreateAccessToken(userID, "report@example.com")
	if err != nil {
		t.Fatalf("token create failed: %v", err)
	}
	r := gin.New()
	var reqCtx context.Context
	r.Use(middleware.AuthMiddleware(tm))
	r.GET("/", func(c *gin.Context) {
		reqCtx = c.Request.Context()
		c.Status(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return reqCtx
}

func TestReportService_ProjectAndTaskReports(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	log := logger.New("error", "json")
	timeRecordRepo := repository.NewTimeRecordRepository(db, log)
	taskRepo := repository.NewTaskRepository(db, log)
	projectRepo := repository.NewProjectRepository(db, log)
	svc := NewReportService(timeRecordRepo, taskRepo, projectRepo)

	userID := uuid.New()
	projectID := uuid.New()
	taskID := uuid.New()
	ctx := reportAuthContext(t, userID)

	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC)
	workDate := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery("FROM time_record").
		WillReturnRows(sqlmock.NewRows([]string{"id", "project_id", "task_id", "work_date", "work_timezone", "total_minutes"}).
			AddRow(uuid.New(), projectID, taskID, workDate, "Europe/Prague", 30).
			AddRow(uuid.New(), projectID, taskID, workDate, "Europe/Prague", 45))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, user_id, project_id, name, status, created_at, updated_at FROM task WHERE id = $1`)).
		WithArgs(taskID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "project_id", "name", "status", "created_at", "updated_at"}).
			AddRow(taskID, userID, projectID, "Task 1", "CREATED", workDate, workDate))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name, user_id, created_at, updated_at FROM project WHERE id = $1`)).
		WithArgs(projectID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "user_id", "created_at", "updated_at"}).
			AddRow(projectID, "Project 1", userID, workDate, workDate))

	projectResp, err := svc.ProjectReport(ctx, &apimodel.ProjectReportRequest{
		ProjectID: projectID,
		TimeRange: &apimodel.TimeRangeParams{FromDate: from, ToDate: to},
	})
	if err != nil {
		t.Fatalf("ProjectReport failed: %v", err)
	}
	if projectResp.TotalMinutes != 75 {
		t.Fatalf("unexpected project total %d", projectResp.TotalMinutes)
	}

	mock.ExpectQuery("FROM time_record").
		WillReturnRows(sqlmock.NewRows([]string{"id", "project_id", "task_id", "work_date", "work_timezone", "total_minutes"}).
			AddRow(uuid.New(), projectID, taskID, workDate, "Europe/Prague", 15).
			AddRow(uuid.New(), projectID, taskID, workDate, "Europe/Prague", 20))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, user_id, project_id, name, status, created_at, updated_at FROM task WHERE id = $1`)).
		WithArgs(taskID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "project_id", "name", "status", "created_at", "updated_at"}).
			AddRow(taskID, userID, projectID, "Task 1", "CREATED", workDate, workDate))

	taskResp, err := svc.TaskReport(ctx, &apimodel.TaskReportRequest{
		TaskID:    taskID,
		TimeRange: &apimodel.TimeRangeParams{FromDate: from, ToDate: to},
	})
	if err != nil {
		t.Fatalf("TaskReport failed: %v", err)
	}
	if taskResp.TotalMinutes != 35 {
		t.Fatalf("unexpected task total %d", taskResp.TotalMinutes)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

