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

type fakeProjectRepo struct {
	saveFn            func(ctx context.Context, project *model.Project) (*model.Project, error)
	getFn             func(ctx context.Context, id uuid.UUID) (*model.Project, error)
	getUserProjectsFn func(ctx context.Context, userID uuid.UUID) ([]*model.Project, error)
	deleteFn          func(ctx context.Context, project *model.Project) error
}

func (f *fakeProjectRepo) Save(ctx context.Context, project *model.Project) (*model.Project, error) {
	if f.saveFn != nil {
		return f.saveFn(ctx, project)
	}
	return project, nil
}
func (f *fakeProjectRepo) Get(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	if f.getFn != nil {
		return f.getFn(ctx, id)
	}
	return nil, sql.ErrNoRows
}
func (f *fakeProjectRepo) GetUserProjects(ctx context.Context, userID uuid.UUID) ([]*model.Project, error) {
	if f.getUserProjectsFn != nil {
		return f.getUserProjectsFn(ctx, userID)
	}
	return []*model.Project{}, nil
}
func (f *fakeProjectRepo) Delete(ctx context.Context, project *model.Project) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, project)
	}
	return nil
}

func authenticatedContextForServiceTests(t *testing.T, userID uuid.UUID) context.Context {
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

func TestProjectService_CRUDFlow(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()
	ctx := authenticatedContextForServiceTests(t, userID)

	repo := &fakeProjectRepo{
		saveFn: func(ctx context.Context, project *model.Project) (*model.Project, error) {
			if project.ID == uuid.Nil {
				project.ID = projectID
			}
			project.UserID = userID
			project.CreatedAt = time.Now().UTC()
			project.UpdatedAt = time.Now().UTC()
			return project, nil
		},
		getFn: func(ctx context.Context, id uuid.UUID) (*model.Project, error) {
			return &model.Project{
				ID:        id,
				UserID:    userID,
				Name:      "p1",
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}, nil
		},
		getUserProjectsFn: func(ctx context.Context, userID uuid.UUID) ([]*model.Project, error) {
			return []*model.Project{{ID: projectID, UserID: userID, Name: "p1"}}, nil
		},
		deleteFn: func(ctx context.Context, project *model.Project) error {
			return nil
		},
	}
	svc := NewProjectService(repo)

	created, err := svc.Create(ctx, apimodel.CreateProjectRequest{Name: "p1"})
	if err != nil || created.ID == uuid.Nil {
		t.Fatalf("create failed: created=%+v err=%v", created, err)
	}

	updated, err := svc.Update(ctx, apimodel.UpdateProjectRequest{ID: created.ID, Name: "p2"})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if updated.Name != "p2" {
		t.Fatalf("expected updated name, got %s", updated.Name)
	}

	got, err := svc.Get(ctx, created.ID)
	if err != nil || got.ID != created.ID {
		t.Fatalf("get failed: %+v err=%v", got, err)
	}

	list, err := svc.GetUserProjects(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("list failed: len=%d err=%v", len(list), err)
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}
