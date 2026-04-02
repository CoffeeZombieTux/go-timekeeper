package service

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-timekeeper/internal/apperror"
	"go-timekeeper/internal/auth"
	"go-timekeeper/internal/middleware"
	"go-timekeeper/internal/model"
	apimodel "go-timekeeper/internal/model/api"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type fakeTimeRecordRepo struct {
	getTaskDayClosedRecordsFn func(
		ctx context.Context,
		userID, taskID uuid.UUID,
		workDate time.Time,
	) ([]*model.TimeRecord, error)
}

func (f *fakeTimeRecordRepo) GetForUpdate(ctx context.Context, id uuid.UUID, tx *sql.Tx) (*model.TimeRecord, error) {
	return nil, sql.ErrNoRows
}
func (f *fakeTimeRecordRepo) GetActiveByUserForUpdate(ctx context.Context, userID uuid.UUID, tx *sql.Tx) (*model.TimeRecord, error) {
	return nil, sql.ErrNoRows
}
func (f *fakeTimeRecordRepo) GetListByTaskForUpdate(ctx context.Context, taskID uuid.UUID, tx *sql.Tx) ([]*model.TimeRecord, error) {
	return nil, nil
}
func (f *fakeTimeRecordRepo) Create(ctx context.Context, rec *model.TimeRecord, tx *sql.Tx) (*model.TimeRecord, error) {
	return rec, nil
}
func (f *fakeTimeRecordRepo) Update(ctx context.Context, rec *model.TimeRecord, tx *sql.Tx) (*model.TimeRecord, error) {
	return rec, nil
}
func (f *fakeTimeRecordRepo) Delete(ctx context.Context, rec *model.TimeRecord, tx *sql.Tx) error {
	return nil
}
func (f *fakeTimeRecordRepo) GetTaskReportRows(
	ctx context.Context,
	userID, taskID uuid.UUID,
	fromDate, toDate time.Time,
) ([]*model.TimeRecordReportRow, error) {
	return nil, nil
}
func (f *fakeTimeRecordRepo) GetTaskDayClosedRecords(
	ctx context.Context,
	userID, taskID uuid.UUID,
	workDate time.Time,
) ([]*model.TimeRecord, error) {
	if f.getTaskDayClosedRecordsFn == nil {
		return []*model.TimeRecord{}, nil
	}
	return f.getTaskDayClosedRecordsFn(ctx, userID, taskID, workDate)
}
func (f *fakeTimeRecordRepo) GetProjectReportRows(
	ctx context.Context,
	userID, projectID uuid.UUID,
	taskIDs *[]uuid.UUID,
	fromDate, toDate time.Time,
) ([]*model.TimeRecordReportRow, error) {
	return nil, nil
}
func (f *fakeTimeRecordRepo) GetGeneralReportRows(
	ctx context.Context,
	userID uuid.UUID,
	projectIDs *[]uuid.UUID,
	fromDate, toDate time.Time,
) ([]*model.TimeRecordReportRow, error) {
	return nil, nil
}

func contextWithAuthenticatedUser(t *testing.T, userID uuid.UUID) context.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	tokenManager := auth.NewTokenManager("test-secret", 15, 24)
	token, err := tokenManager.CreateAccessToken(userID, "u@example.com")
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	router := gin.New()
	var reqCtx context.Context
	router.Use(middleware.AuthMiddleware(tokenManager))
	router.GET("/", func(c *gin.Context) {
		reqCtx = c.Request.Context()
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("unexpected status %d", rec.Code)
	}
	return reqCtx
}

func TestValidateTimeRecordsConflict_OverlappingClosedSession(t *testing.T) {
	workDate := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	sessionEnd := time.Date(2026, 3, 31, 11, 0, 0, 0, time.UTC)
	repo := &fakeTimeRecordRepo{
		getTaskDayClosedRecordsFn: func(ctx context.Context, userID, taskID uuid.UUID, d time.Time) ([]*model.TimeRecord, error) {
			return []*model.TimeRecord{
				{
					ID:        uuid.New(),
					WorkDate:  workDate,
					StartedAt: time.Date(2026, 3, 31, 10, 30, 0, 0, time.UTC),
					EndedAt:   &sessionEnd,
				},
			}, nil
		},
	}
	svc := &TimeRecordService{timeRecordRepo: repo}

	candidateEnd := time.Date(2026, 3, 31, 10, 45, 0, 0, time.UTC)
	err := svc.validateTimeRecordsConflict(
		context.Background(),
		uuid.New(),
		uuid.New(),
		model.TimeRecord{
			ID:        uuid.New(),
			WorkDate:  workDate,
			StartedAt: time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC),
			EndedAt:   &candidateEnd,
		},
	)
	if err == nil {
		t.Fatal("expected overlap error, got nil")
	}
	appErr, ok := apperror.As(err)
	if !ok || appErr.Code != apperror.CodeValidationErrorCode {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestValidateTimeRecordsConflict_OverlappingActiveSession(t *testing.T) {
	now := time.Now().UTC()
	workDate := normalizeDate(now, time.UTC)
	repo := &fakeTimeRecordRepo{
		getTaskDayClosedRecordsFn: func(ctx context.Context, userID, taskID uuid.UUID, d time.Time) ([]*model.TimeRecord, error) {
			return []*model.TimeRecord{
				{
					ID:        uuid.New(),
					WorkDate:  workDate,
					StartedAt: now.Add(-45 * time.Minute),
					EndedAt:   nil,
				},
			}, nil
		},
	}
	svc := &TimeRecordService{timeRecordRepo: repo}

	candidateEnd := now.Add(-30 * time.Minute)
	err := svc.validateTimeRecordsConflict(
		context.Background(),
		uuid.New(),
		uuid.New(),
		model.TimeRecord{
			ID:        uuid.New(),
			WorkDate:  workDate,
			StartedAt: now.Add(-2 * time.Hour),
			EndedAt:   &candidateEnd,
		},
	)
	if err == nil {
		t.Fatal("expected active overlap error, got nil")
	}
}

func TestValidateTimeRecordsConflict_NonOverlappingSession(t *testing.T) {
	workDate := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	sessionEnd := time.Date(2026, 3, 31, 9, 0, 0, 0, time.UTC)
	repo := &fakeTimeRecordRepo{
		getTaskDayClosedRecordsFn: func(ctx context.Context, userID, taskID uuid.UUID, d time.Time) ([]*model.TimeRecord, error) {
			return []*model.TimeRecord{
				{
					ID:        uuid.New(),
					WorkDate:  workDate,
					StartedAt: time.Date(2026, 3, 31, 8, 0, 0, 0, time.UTC),
					EndedAt:   &sessionEnd,
				},
			}, nil
		},
	}
	svc := &TimeRecordService{timeRecordRepo: repo}

	candidateEnd := time.Date(2026, 3, 31, 11, 0, 0, 0, time.UTC)
	err := svc.validateTimeRecordsConflict(
		context.Background(),
		uuid.New(),
		uuid.New(),
		model.TimeRecord{
			ID:        uuid.New(),
			WorkDate:  workDate,
			StartedAt: time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC),
			EndedAt:   &candidateEnd,
		},
	)
	if err != nil {
		t.Fatalf("expected nil conflict error, got %v", err)
	}
}

func TestValidateTimeRecordsConflict_InvalidRange(t *testing.T) {
	repo := &fakeTimeRecordRepo{}
	svc := &TimeRecordService{timeRecordRepo: repo}

	end := time.Date(2026, 3, 31, 9, 0, 0, 0, time.UTC)
	err := svc.validateTimeRecordsConflict(
		context.Background(),
		uuid.New(),
		uuid.New(),
		model.TimeRecord{
			ID:        uuid.New(),
			WorkDate:  time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
			StartedAt: time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC),
			EndedAt:   &end,
		},
	)
	if err == nil {
		t.Fatal("expected invalid range error")
	}
}

func TestCreateTimeRecord_RejectsWorkDateMismatch(t *testing.T) {
	userID := uuid.New()
	ctx := contextWithAuthenticatedUser(t, userID)

	repo := &fakeTimeRecordRepo{}
	svc := &TimeRecordService{
		timeRecordRepo: repo,
		uowManager:     nil,
	}

	_, err := svc.CreateTimeRecord(ctx, &apimodel.CreateTimeRecordRequest{
		ProjectID:    uuid.New(),
		TaskID:       uuid.New(),
		WorkDate:     time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC),
		WorkTimezone: "Europe/Prague",
		StartTime:    time.Date(2026, 3, 31, 8, 0, 0, 0, time.UTC),
		EndTime:      time.Date(2026, 3, 31, 9, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected workDate mismatch validation error")
	}
}

func TestSplitByMidnightAndHelpers(t *testing.T) {
	loc, err := getLocation("Europe/Prague")
	if err != nil {
		t.Fatalf("unexpected location error: %v", err)
	}

	start := time.Date(2026, 3, 30, 21, 30, 0, 0, time.UTC)
	end := time.Date(2026, 3, 31, 1, 30, 0, 0, time.UTC)
	days := splitByMidnight(start, end, loc)
	if len(days) != 2 {
		t.Fatalf("expected 2 days, got %d", len(days))
	}
	if days[0].TotalMinutes <= 0 || days[1].TotalMinutes <= 0 {
		t.Fatalf("unexpected minutes: %+v", days)
	}

	if durationToMinutes(-1*time.Minute) != 0 {
		t.Fatal("negative duration should clamp to 0")
	}
	if !sameLocalDate(start, start.Add(10*time.Minute), loc) {
		t.Fatal("sameLocalDate should be true")
	}
}

func TestGetLocationAndUserAccessGuards(t *testing.T) {
	if _, err := getLocation("Not/AZone"); err == nil {
		t.Fatal("expected invalid timezone error")
	}

	userID := uuid.New()
	if err := checkTimeRecordUserAccess(userID, model.TimeRecord{UserID: userID}); err != nil {
		t.Fatalf("expected access success, got %v", err)
	}
	if err := checkTimeRecordUserAccess(uuid.New(), model.TimeRecord{UserID: userID}); err == nil {
		t.Fatal("expected unauthorized error")
	}
}
