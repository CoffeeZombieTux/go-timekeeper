package service

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-timekeeper/internal/auth"
	"go-timekeeper/internal/middleware"
	"go-timekeeper/internal/model"
	apimodel "go-timekeeper/internal/model/api"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type fakeReportTimeRecordRepo struct {
	generalRows []*model.TimeRecordReportRow
}

func (f *fakeReportTimeRecordRepo) GetForUpdate(ctx context.Context, id uuid.UUID, tx *sql.Tx) (*model.TimeRecord, error) {
	return nil, sql.ErrNoRows
}
func (f *fakeReportTimeRecordRepo) GetActiveByUserForUpdate(ctx context.Context, userID uuid.UUID, tx *sql.Tx) (*model.TimeRecord, error) {
	return nil, sql.ErrNoRows
}
func (f *fakeReportTimeRecordRepo) GetListByTaskForUpdate(ctx context.Context, taskID uuid.UUID, tx *sql.Tx) ([]*model.TimeRecord, error) {
	return nil, nil
}
func (f *fakeReportTimeRecordRepo) Create(ctx context.Context, rec *model.TimeRecord, tx *sql.Tx) (*model.TimeRecord, error) {
	return rec, nil
}
func (f *fakeReportTimeRecordRepo) Update(ctx context.Context, rec *model.TimeRecord, tx *sql.Tx) (*model.TimeRecord, error) {
	return rec, nil
}
func (f *fakeReportTimeRecordRepo) Delete(ctx context.Context, rec *model.TimeRecord, tx *sql.Tx) error {
	return nil
}
func (f *fakeReportTimeRecordRepo) GetTaskReportRows(
	ctx context.Context,
	userID, taskID uuid.UUID,
	fromDate, toDate time.Time,
) ([]*model.TimeRecordReportRow, error) {
	return nil, nil
}
func (f *fakeReportTimeRecordRepo) GetTaskDayClosedRecords(
	ctx context.Context,
	userID, taskID uuid.UUID,
	workDate time.Time,
) ([]*model.TimeRecord, error) {
	return nil, nil
}
func (f *fakeReportTimeRecordRepo) GetProjectReportRows(
	ctx context.Context,
	userID, projectID uuid.UUID,
	taskIDs *[]uuid.UUID,
	fromDate, toDate time.Time,
) ([]*model.TimeRecordReportRow, error) {
	return nil, nil
}
func (f *fakeReportTimeRecordRepo) GetGeneralReportRows(
	ctx context.Context,
	userID uuid.UUID,
	projectIDs *[]uuid.UUID,
	fromDate, toDate time.Time,
) ([]*model.TimeRecordReportRow, error) {
	return f.generalRows, nil
}

func authCtxForReportTests(t *testing.T, userID uuid.UUID) context.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	tokenManager := auth.NewTokenManager("test-secret", 15, 24)
	token, err := tokenManager.CreateAccessToken(userID, "u@example.com")
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}
	r := gin.New()
	var reqCtx context.Context
	r.Use(middleware.AuthMiddleware(tokenManager))
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

func TestReportService_GeneralReportEmptyRows(t *testing.T) {
	userID := uuid.New()
	ctx := authCtxForReportTests(t, userID)
	svc := &ReportService{
		timeRecordRepo: &fakeReportTimeRecordRepo{generalRows: []*model.TimeRecordReportRow{}},
	}
	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC)

	resp, err := svc.GeneralReport(ctx, &apimodel.GeneralReportRequest{
		TimeRange: &apimodel.TimeRangeParams{FromDate: from, ToDate: to},
	})
	if err != nil {
		t.Fatalf("general report should pass: %v", err)
	}
	if resp.TotalMinutes != 0 || len(resp.Projects) != 0 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestReportService_GeneralReportInvalidRange(t *testing.T) {
	userID := uuid.New()
	ctx := authCtxForReportTests(t, userID)
	svc := &ReportService{
		timeRecordRepo: &fakeReportTimeRecordRepo{generalRows: []*model.TimeRecordReportRow{}},
	}
	from := time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC)
	to := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	if _, err := svc.GeneralReport(ctx, &apimodel.GeneralReportRequest{
		TimeRange: &apimodel.TimeRangeParams{FromDate: from, ToDate: to},
	}); err == nil {
		t.Fatal("expected validation error for invalid range")
	}
}

func TestSumBy(t *testing.T) {
	got := SumBy([]int{1, 2, 3}, func(v int) int { return v })
	if got != 6 {
		t.Fatalf("expected 6, got %d", got)
	}
}
