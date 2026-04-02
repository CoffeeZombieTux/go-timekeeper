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

type fakeTaskService struct {
	createFn       func(ctx context.Context, req *apimodel.CreateTaskRequest) (*model.Task, error)
	updateFn       func(ctx context.Context, req *apimodel.UpdateTaskRequest) (*model.Task, error)
	getFn          func(ctx context.Context, id uuid.UUID, unit *uow.UnitOfWork) (*model.Task, error)
	getByProjectFn func(ctx context.Context, projectID uuid.UUID, isActive *bool, requestedLimit, requestedOffset int) ([]*model.Task, *apimodel.PaginationResponse, error)
	deleteFn       func(ctx context.Context, id uuid.UUID) error
	startFn        func(ctx context.Context, id uuid.UUID, timezone string) error
	stopFn         func(ctx context.Context, id uuid.UUID) error
	closeFn        func(ctx context.Context, id uuid.UUID) error
}

func (f *fakeTaskService) Create(ctx context.Context, req *apimodel.CreateTaskRequest) (*model.Task, error) {
	return f.createFn(ctx, req)
}
func (f *fakeTaskService) Update(ctx context.Context, req *apimodel.UpdateTaskRequest) (*model.Task, error) {
	return f.updateFn(ctx, req)
}
func (f *fakeTaskService) Get(ctx context.Context, id uuid.UUID, unit *uow.UnitOfWork) (*model.Task, error) {
	return f.getFn(ctx, id, unit)
}
func (f *fakeTaskService) GetByProject(
	ctx context.Context,
	projectID uuid.UUID,
	isActive *bool,
	requestedLimit, requestedOffset int,
) ([]*model.Task, *apimodel.PaginationResponse, error) {
	return f.getByProjectFn(ctx, projectID, isActive, requestedLimit, requestedOffset)
}
func (f *fakeTaskService) Delete(ctx context.Context, id uuid.UUID) error { return f.deleteFn(ctx, id) }
func (f *fakeTaskService) Start(ctx context.Context, id uuid.UUID, timezone string) error {
	return f.startFn(ctx, id, timezone)
}
func (f *fakeTaskService) Stop(ctx context.Context, id uuid.UUID) error  { return f.stopFn(ctx, id) }
func (f *fakeTaskService) Close(ctx context.Context, id uuid.UUID) error { return f.closeFn(ctx, id) }

type fakeProjectService struct {
	createFn func(ctx context.Context, request apimodel.CreateProjectRequest) (*model.Project, error)
	updateFn func(ctx context.Context, request apimodel.UpdateProjectRequest) (*model.Project, error)
	getFn    func(ctx context.Context, id uuid.UUID) (*model.Project, error)
	listFn   func(ctx context.Context) ([]*model.Project, error)
	deleteFn func(ctx context.Context, id uuid.UUID) error
}

func (f *fakeProjectService) Create(ctx context.Context, request apimodel.CreateProjectRequest) (*model.Project, error) {
	return f.createFn(ctx, request)
}
func (f *fakeProjectService) Update(ctx context.Context, request apimodel.UpdateProjectRequest) (*model.Project, error) {
	return f.updateFn(ctx, request)
}
func (f *fakeProjectService) Get(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	return f.getFn(ctx, id)
}
func (f *fakeProjectService) GetUserProjects(ctx context.Context) ([]*model.Project, error) {
	return f.listFn(ctx)
}
func (f *fakeProjectService) Delete(ctx context.Context, id uuid.UUID) error {
	return f.deleteFn(ctx, id)
}

type fakeReportService struct {
	generalFn func(ctx context.Context, req *apimodel.GeneralReportRequest) (*apimodel.GeneralReportResponse, error)
	projectFn func(ctx context.Context, req *apimodel.ProjectReportRequest) (*apimodel.ProjectReportResponse, error)
	taskFn    func(ctx context.Context, req *apimodel.TaskReportRequest) (*apimodel.TaskReportResponse, error)
}

func (f *fakeReportService) GeneralReport(ctx context.Context, req *apimodel.GeneralReportRequest) (*apimodel.GeneralReportResponse, error) {
	return f.generalFn(ctx, req)
}
func (f *fakeReportService) ProjectReport(ctx context.Context, req *apimodel.ProjectReportRequest) (*apimodel.ProjectReportResponse, error) {
	return f.projectFn(ctx, req)
}
func (f *fakeReportService) TaskReport(ctx context.Context, req *apimodel.TaskReportRequest) (*apimodel.TaskReportResponse, error) {
	return f.taskFn(ctx, req)
}

func authTokenForHandlerTests(t *testing.T) string {
	t.Helper()
	tokenManager := auth.NewTokenManager("test-secret", 15, 24)
	token, err := tokenManager.CreateAccessToken(uuid.New(), "u@example.com")
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}
	return token
}

func performHandlerRequest(t *testing.T, r *gin.Engine, method, path string, body any, accessToken string) *httptest.ResponseRecorder {
	t.Helper()
	var raw []byte
	if body != nil {
		var err error
		raw, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestTaskHandler_FlowAndBreakLogic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	taskID := uuid.New()
	projectID := uuid.New()
	token := authTokenForHandlerTests(t)
	tm := auth.NewTokenManager("test-secret", 15, 24)
	log := logger.New("error", "json")

	svc := &fakeTaskService{
		createFn: func(ctx context.Context, req *apimodel.CreateTaskRequest) (*model.Task, error) {
			return &model.Task{ID: taskID, ProjectID: req.ProjectID, Name: req.Name}, nil
		},
		updateFn: func(ctx context.Context, req *apimodel.UpdateTaskRequest) (*model.Task, error) {
			return &model.Task{ID: req.ID, ProjectID: req.ProjectID, Name: req.Name}, nil
		},
		getFn: func(ctx context.Context, id uuid.UUID, unit *uow.UnitOfWork) (*model.Task, error) {
			return &model.Task{ID: id, ProjectID: projectID, Name: "t1"}, nil
		},
		getByProjectFn: func(ctx context.Context, projectID uuid.UUID, isActive *bool, requestedLimit, requestedOffset int) ([]*model.Task, *apimodel.PaginationResponse, error) {
			return []*model.Task{{ID: taskID, ProjectID: projectID, Name: "t1"}}, &apimodel.PaginationResponse{Limit: 10, Offset: 0, TotalItems: 1, CurrentPage: 1, TotalPages: 1}, nil
		},
		deleteFn: func(ctx context.Context, id uuid.UUID) error { return nil },
		startFn: func(ctx context.Context, id uuid.UUID, timezone string) error {
			return nil
		},
		stopFn:  func(ctx context.Context, id uuid.UUID) error { return nil },
		closeFn: func(ctx context.Context, id uuid.UUID) error { return nil },
	}

	h := NewTaskHandler(svc, log)
	r := gin.New()
	r.Use(middleware.RequestID())
	grp := r.Group("/api/task", middleware.AuthMiddleware(tm))
	{
		grp.POST("", h.CreateTask)
		grp.PATCH("", h.UpdateTask)
		grp.GET("/:id", h.GetTask)
		grp.GET("/list/project/:id", h.GetProjectTasks)
		grp.DELETE("/:id", h.DeleteTask)
		grp.PATCH("/:id/start", h.StartTask)
		grp.PATCH("/:id/stop", h.StopTask)
		grp.PATCH("/:id/close", h.CloseTask)
	}

	rec := performHandlerRequest(t, r, http.MethodPost, "/api/task", apimodel.CreateTaskRequest{ProjectID: projectID, Name: "t1"}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("create task failed: %d %s", rec.Code, rec.Body.String())
	}

	rec = performHandlerRequest(t, r, http.MethodPatch, "/api/task", apimodel.UpdateTaskRequest{ID: taskID, ProjectID: projectID, Name: "t2"}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("update task failed: %d %s", rec.Code, rec.Body.String())
	}

	rec = performHandlerRequest(t, r, http.MethodGet, "/api/task/"+taskID.String(), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("get task failed: %d %s", rec.Code, rec.Body.String())
	}

	rec = performHandlerRequest(t, r, http.MethodGet, "/api/task/list/project/"+projectID.String()+"?isActive=1&limit=10&offset=0", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("list project tasks failed: %d %s", rec.Code, rec.Body.String())
	}

	rec = performHandlerRequest(t, r, http.MethodPatch, "/api/task/"+taskID.String()+"/start", nil, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("start should fail without timezone header: %d %s", rec.Code, rec.Body.String())
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/task/"+taskID.String()+"/start", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Timezone", "Europe/Prague")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("start task failed: %d %s", res.Code, res.Body.String())
	}

	rec = performHandlerRequest(t, r, http.MethodPatch, "/api/task/"+taskID.String()+"/stop", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("stop task failed: %d %s", rec.Code, rec.Body.String())
	}

	rec = performHandlerRequest(t, r, http.MethodPatch, "/api/task/"+taskID.String()+"/close", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("close task failed: %d %s", rec.Code, rec.Body.String())
	}

	rec = performHandlerRequest(t, r, http.MethodDelete, "/api/task/"+taskID.String(), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete task failed: %d %s", rec.Code, rec.Body.String())
	}
}

func TestProjectAndReportHandlers_UnauthorizedAndValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	token := authTokenForHandlerTests(t)
	tm := auth.NewTokenManager("test-secret", 15, 24)
	log := logger.New("error", "json")

	projectSvc := &fakeProjectService{
		createFn: func(ctx context.Context, request apimodel.CreateProjectRequest) (*model.Project, error) {
			return nil, apperror.New(apperror.CodeUnauthorizedCode, apperror.CodeUnauthorizedMessage, "no access")
		},
		updateFn: func(ctx context.Context, request apimodel.UpdateProjectRequest) (*model.Project, error) {
			return &model.Project{ID: request.ID, Name: request.Name}, nil
		},
		getFn: func(ctx context.Context, id uuid.UUID) (*model.Project, error) {
			return &model.Project{ID: id, Name: "p1"}, nil
		},
		listFn: func(ctx context.Context) ([]*model.Project, error) {
			return []*model.Project{{ID: uuid.New(), Name: "p1"}}, nil
		},
		deleteFn: func(ctx context.Context, id uuid.UUID) error { return nil },
	}
	reportSvc := &fakeReportService{
		generalFn: func(ctx context.Context, req *apimodel.GeneralReportRequest) (*apimodel.GeneralReportResponse, error) {
			return &apimodel.GeneralReportResponse{}, nil
		},
		projectFn: func(ctx context.Context, req *apimodel.ProjectReportRequest) (*apimodel.ProjectReportResponse, error) {
			return &apimodel.ProjectReportResponse{}, nil
		},
		taskFn: func(ctx context.Context, req *apimodel.TaskReportRequest) (*apimodel.TaskReportResponse, error) {
			return &apimodel.TaskReportResponse{}, nil
		},
	}

	projectHandler := NewProjectHandler(projectSvc, log)
	reportHandler := NewReportHandler(reportSvc, log)
	r := gin.New()
	r.Use(middleware.RequestID())

	pg := r.Group("/api/project", middleware.AuthMiddleware(tm))
	{
		pg.POST("", projectHandler.CreateProject)
		pg.PATCH("", projectHandler.UpdateProject)
		pg.GET("/:id", projectHandler.GetProject)
		pg.GET("/list", projectHandler.GetUserProjects)
		pg.DELETE("/:id", projectHandler.DeleteProject)
	}
	rg := r.Group("/api/report", middleware.AuthMiddleware(tm))
	{
		rg.POST("/general", reportHandler.GeneralReport)
		rg.POST("/project", reportHandler.ProjectReport)
		rg.POST("/task", reportHandler.TaskReport)
	}

	rec := performHandlerRequest(t, r, http.MethodPost, "/api/project", apimodel.CreateProjectRequest{Name: "p1"}, token)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized create project, got %d %s", rec.Code, rec.Body.String())
	}

	rec = performHandlerRequest(t, r, http.MethodGet, "/api/project/not-a-uuid", nil, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request for invalid uuid, got %d", rec.Code)
	}

	projectID := uuid.New()
	rec = performHandlerRequest(t, r, http.MethodPatch, "/api/project", apimodel.UpdateProjectRequest{
		ID:   projectID,
		Name: "p2",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected project update success, got %d %s", rec.Code, rec.Body.String())
	}

	rec = performHandlerRequest(t, r, http.MethodGet, "/api/project/"+projectID.String(), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected project get success, got %d %s", rec.Code, rec.Body.String())
	}

	rec = performHandlerRequest(t, r, http.MethodGet, "/api/project/list", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected project list success, got %d %s", rec.Code, rec.Body.String())
	}

	rec = performHandlerRequest(t, r, http.MethodDelete, "/api/project/"+projectID.String(), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected project delete success, got %d %s", rec.Code, rec.Body.String())
	}

	rec = performHandlerRequest(t, r, http.MethodPost, "/api/report/general", map[string]any{}, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request for invalid report payload, got %d", rec.Code)
	}

	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC)
	rec = performHandlerRequest(t, r, http.MethodPost, "/api/report/project", apimodel.ProjectReportRequest{
		ProjectID: projectID,
		TimeRange: &apimodel.TimeRangeParams{FromDate: from, ToDate: to},
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected project report success, got %d %s", rec.Code, rec.Body.String())
	}

	rec = performHandlerRequest(t, r, http.MethodPost, "/api/report/task", apimodel.TaskReportRequest{
		TaskID:    uuid.New(),
		TimeRange: &apimodel.TimeRangeParams{FromDate: from, ToDate: to},
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected task report success, got %d %s", rec.Code, rec.Body.String())
	}
}
