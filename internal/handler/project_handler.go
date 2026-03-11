package handler

import (
	"errors"
	"go-timekeeper/internal/logger"
	"go-timekeeper/internal/middleware"
	apimodel "go-timekeeper/internal/model/api"
	"go-timekeeper/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ProjectHandler handles project-related requests.
type ProjectHandler struct {
	projectService service.ProjectServiceInterface
	logger         *logger.Logger
}

// NewProjectHandler creates a new ProjectHandler instance.
func NewProjectHandler(projectService service.ProjectServiceInterface, logger *logger.Logger) *ProjectHandler {
	return &ProjectHandler{projectService: projectService, logger: logger}
}

// CreateProject handles project creation requests.
func (projectHandler *ProjectHandler) CreateProject(ctx *gin.Context) {
	var req apimodel.CreateProjectRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}
	response, err := projectHandler.projectService.Create(ctx.Request.Context(), req)
	if err != nil {
		userId, _ := middleware.UserIDFromContext(ctx.Request.Context())
		projectHandler.logger.WithError(err).WithFields(logger.Fields{
			"request_id":   requestIDFromContext(ctx),
			"project_name": req.Name,
			"user_id":      userId.String(),
		}).Error(logger.LogMessageFailedToCreateProject)
		status, code, message, details := mapDomainError(err)
		writeError(ctx, status, message, code, details)
		return
	}
	writeSuccess(ctx, http.StatusOK, "Project created", response)
}

// GetProject handles project retrieval requests.
func (projectHandler *ProjectHandler) GetProject(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		writeBindError(ctx, errors.New("missing or invalid ID in request"))
		return
	}
	response, err := projectHandler.projectService.Get(ctx.Request.Context(), id)
	if err != nil {
		userId, _ := middleware.UserIDFromContext(ctx.Request.Context())
		projectHandler.logger.WithError(err).WithFields(logger.Fields{
			"request_id": requestIDFromContext(ctx),
			"project_id": id,
			"user_id":    userId.String(),
		}).Error(logger.LogMessageFailedToGetProject)
		status, code, message, details := mapDomainError(err)
		writeError(ctx, status, message, code, details)
		return
	}
	writeSuccess(ctx, http.StatusOK, "Project loaded", response)
}

// GetUserProjects handles project retrieval requests.
func (projectHandler *ProjectHandler) GetUserProjects(ctx *gin.Context) {
	response, err := projectHandler.projectService.GetUserProjects(ctx.Request.Context())
	if err != nil {
		userId, _ := middleware.UserIDFromContext(ctx.Request.Context())
		projectHandler.logger.WithError(err).WithFields(logger.Fields{
			"request_id": requestIDFromContext(ctx),
			"user_id":    userId.String(),
		}).Error(logger.LogMessageFailedToGetProject)
		status, code, message, details := mapDomainError(err)
		writeError(ctx, status, message, code, details)
		return
	}
	writeSuccess(ctx, http.StatusOK, "User projects loaded", response)
}

// UpdateProject handles project update requests.
func (projectHandler *ProjectHandler) UpdateProject(ctx *gin.Context) {
	var req apimodel.UpdateProjectRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}
	response, err := projectHandler.projectService.Update(ctx.Request.Context(), req)
	if err != nil {
		userId, _ := middleware.UserIDFromContext(ctx.Request.Context())
		projectHandler.logger.WithError(err).WithFields(logger.Fields{
			"request_id": requestIDFromContext(ctx),
			"project_id": req.ID,
			"user_id":    userId.String(),
		}).Error(logger.LogMessageFailedToUpdateProject)
		status, code, message, details := mapDomainError(err)
		writeError(ctx, status, message, code, details)
		return
	}
	writeSuccess(ctx, http.StatusOK, "Project updated", response)
}

// DeleteProject handles project deletion requests.
func (projectHandler *ProjectHandler) DeleteProject(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		writeBindError(ctx, errors.New("missing or invalid ID in request"))
	}
	err = projectHandler.projectService.Delete(ctx.Request.Context(), id)
	if err != nil {
		userId, _ := middleware.UserIDFromContext(ctx.Request.Context())
		projectHandler.logger.WithError(err).WithFields(logger.Fields{
			"request_id": requestIDFromContext(ctx),
			"project_id": id,
			"user_id":    userId.String(),
		}).Error(logger.LogMessageFailedToDeleteProject)
		status, code, message, details := mapDomainError(err)
		writeError(ctx, status, message, code, details)
		return
	}
	writeSuccess(ctx, http.StatusOK, "Project deleted", "")
}
