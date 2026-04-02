package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-timekeeper/internal/apperror"
	"go-timekeeper/internal/auth"
	"go-timekeeper/internal/logger"
	"go-timekeeper/internal/middleware"
	"go-timekeeper/internal/model"
	apimodel "go-timekeeper/internal/model/api"
	"go-timekeeper/internal/uow"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type fakeTimeRecordService struct {
	updateFn     func(ctx context.Context, req *apimodel.UpdateTimeRecordRequest) (*model.TimeRecord, error)
	deleteCalled bool
}

func (f *fakeTimeRecordService) StartTask(ctx context.Context, task *model.Task, timezone string, unit *uow.UnitOfWork) error {
	return nil
}

func (f *fakeTimeRecordService) StopTask(ctx context.Context, task *model.Task, unit *uow.UnitOfWork) error {
	return nil
}

func (f *fakeTimeRecordService) CreateTimeRecord(ctx context.Context, req *apimodel.CreateTimeRecordRequest) (*model.TimeRecord, error) {
	return &model.TimeRecord{}, nil
}

func (f *fakeTimeRecordService) UpdateTimeRecord(ctx context.Context, req *apimodel.UpdateTimeRecordRequest) (*model.TimeRecord, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return &model.TimeRecord{}, nil
}

func (f *fakeTimeRecordService) DeleteTimeRecord(ctx context.Context, id uuid.UUID) error {
	f.deleteCalled = true
	return nil
}

func TestUpdateTimeRecord_UnauthorizedAttempt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeTimeRecordService{
		updateFn: func(ctx context.Context, req *apimodel.UpdateTimeRecordRequest) (*model.TimeRecord, error) {
			return nil, apperror.New(apperror.CodeUnauthorizedCode, apperror.CodeUnauthorizedMessage, "unauthorized edit attempt")
		},
	}
	h := NewTimeRecordHandler(service, logger.New("error", "json"))

	tokenManager := auth.NewTokenManager("test-secret", 15, 24)
	token, err := tokenManager.CreateAccessToken(uuid.New(), "user@example.com")
	if err != nil {
		t.Fatalf("token create failed: %v", err)
	}

	router := gin.New()
	router.Use(middleware.RequestID())
	router.PATCH("/api/task/session", middleware.AuthMiddleware(tokenManager), h.UpdateTimeRecord)

	body := apimodel.UpdateTimeRecordRequest{
		ID:           uuid.New(),
		ProjectID:    uuid.New(),
		TaskID:       uuid.New(),
		WorkDate:     time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
		WorkTimezone: "Europe/Prague",
		StartTime:    time.Date(2026, 3, 31, 8, 0, 0, 0, time.UTC),
		EndTime:      time.Date(2026, 3, 31, 9, 0, 0, 0, time.UTC),
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/api/task/session", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d, body=%s", rec.Code, rec.Body.String())
	}
}

func TestDeleteTimeRecord_InvalidUUIDStopsExecution(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeTimeRecordService{}
	h := NewTimeRecordHandler(service, logger.New("error", "json"))

	tokenManager := auth.NewTokenManager("test-secret", 15, 24)
	token, err := tokenManager.CreateAccessToken(uuid.New(), "user@example.com")
	if err != nil {
		t.Fatalf("token create failed: %v", err)
	}

	router := gin.New()
	router.Use(middleware.RequestID())
	router.DELETE("/api/task/session/:id", middleware.AuthMiddleware(tokenManager), h.DeleteTimeRecord)

	req := httptest.NewRequest(http.MethodDelete, "/api/task/session/not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", rec.Code, rec.Body.String())
	}
	if service.deleteCalled {
		t.Fatal("DeleteTimeRecord service method should not be called on invalid UUID")
	}
}
