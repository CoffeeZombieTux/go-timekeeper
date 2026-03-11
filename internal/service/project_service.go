package service

import (
	"context"
	"go-timekeeper/internal/apperror"
	"go-timekeeper/internal/middleware"
	"go-timekeeper/internal/model"
	apimodel "go-timekeeper/internal/model/api"
	"go-timekeeper/internal/repository"

	"github.com/google/uuid"
)

// ProjectServiceInterface represents the interface for the project service.
type ProjectServiceInterface interface {
	Create(ctx context.Context, request apimodel.CreateProjectRequest) (*model.Project, error)
	Update(ctx context.Context, request apimodel.UpdateProjectRequest) (*model.Project, error)
	Get(ctx context.Context, id int64) (*model.Project, error)
	GetUserProjects(ctx context.Context) ([]*model.Project, error)
	Delete(ctx context.Context, id int64) error
}

// ProjectService is a struct that implements the ProjectServiceInterface.
type ProjectService struct {
	projectRepo repository.ProjectRepositoryInterface
}

// NewProjectService creates a new ProjectService instance.
func NewProjectService(projectRepo repository.ProjectRepositoryInterface) *ProjectService {
	return &ProjectService{projectRepo: projectRepo}
}

// Create process a new project creation.
func (projectService *ProjectService) Create(ctx context.Context, request apimodel.CreateProjectRequest) (*model.Project, error) {
	userId, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		return nil, apperror.New(apperror.CodeUnauthorizedCode, apperror.CodeUnauthorizedMessage, "User not authenticated")
	}
	var project = &model.Project{
		UserID: userId,
		Name:   request.Name,
	}
	return projectService.projectRepo.Save(ctx, project)
}

// Update process a project update.
func (projectService *ProjectService) Update(ctx context.Context, request apimodel.UpdateProjectRequest) (*model.Project, error) {
	userId, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		return nil, apperror.New(apperror.CodeUnauthorizedCode, apperror.CodeUnauthorizedMessage, "User not authenticated")
	}

	project, err := projectService.projectRepo.Get(ctx, request.ID)
	if err != nil {
		return nil, err
	}
	if err := checkUserAccess(userId, *project); err != nil {
		return nil, err
	}
	project.Name = request.Name
	return projectService.projectRepo.Save(ctx, project)
}

// Get process get a project by ID and including validation the user's access'.
func (projectService *ProjectService) Get(ctx context.Context, id int64) (*model.Project, error) {
	userId, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		return nil, apperror.New(apperror.CodeUnauthorizedCode, apperror.CodeUnauthorizedMessage, "User not authenticated")
	}

	project, err := projectService.projectRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := checkUserAccess(userId, *project); err != nil {
		return nil, err
	}
	return project, nil
}

// GetUserProjects process listing all user's projects.
func (projectService *ProjectService) GetUserProjects(ctx context.Context) ([]*model.Project, error) {
	userId, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		return nil, apperror.New(apperror.CodeUnauthorizedCode, apperror.CodeUnauthorizedMessage, "User not authenticated")
	}
	return projectService.projectRepo.GetUserProjects(ctx, userId)
}

// Delete process project deletion.
func (projectService *ProjectService) Delete(ctx context.Context, id int64) error {
	userId, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		return apperror.New(apperror.CodeUnauthorizedCode, apperror.CodeUnauthorizedMessage, "User not authenticated")
	}

	project, err := projectService.projectRepo.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := checkUserAccess(userId, *project); err != nil {
		return err
	}
	return projectService.projectRepo.Delete(ctx, project)
}

// checkUserAccess checks if the user is allowed to access the project.
func checkUserAccess(userId uuid.UUID, project model.Project) error {
	if userId == project.UserID {
		return nil
	}
	return apperror.New(apperror.CodeUnauthorizedCode, apperror.CodeUnauthorizedMessage, "User not authenticated")
}
