package handler

import (
	"errors"
	"fmt"
	"go-timekeeper/internal/logger"
	"go-timekeeper/internal/middleware"
	apimodel "go-timekeeper/internal/model/api"
	"go-timekeeper/internal/service"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TaskHandler handles task-related requests.
type TaskHandler struct {
	taskService service.TaskServiceInterface
	logger      *logger.Logger
}

// NewTaskHandler creates a new TaskHandler instance.
func NewTaskHandler(taskService service.TaskServiceInterface, logger *logger.Logger) *TaskHandler {
	return &TaskHandler{
		taskService: taskService,
		logger:      logger,
	}
}

// CreateTask handles task creation requests.
func (taskHandler *TaskHandler) CreateTask(ctx *gin.Context) {
	var req apimodel.CreateTaskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}
	response, err := taskHandler.taskService.Create(ctx.Request.Context(), &req)
	if err != nil {
		userId, _ := middleware.UserIDFromContext(ctx.Request.Context())
		taskHandler.logger.WithError(err).WithFields(logger.Fields{
			"request_id": requestIDFromContext(ctx),
			"task_name":  req.Name,
			"user_id":    userId.String(),
		}).Error(logger.LogMessageFailedToCreateTask)
		status, code, message, details := mapDomainError(err)
		writeError(ctx, status, message, code, details)
		return
	}
	writeSuccess(ctx, http.StatusOK, "Task created", response)
}

// UpdateTask handles task update requests.
func (taskHandler *TaskHandler) UpdateTask(ctx *gin.Context) {
	var req apimodel.UpdateTaskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}
	response, err := taskHandler.taskService.Update(ctx.Request.Context(), &req)
	if err != nil {
		userId, _ := middleware.UserIDFromContext(ctx.Request.Context())
		taskHandler.logger.WithError(err).WithFields(logger.Fields{
			"request_id": requestIDFromContext(ctx),
			"task_name":  req.Name,
			"user_id":    userId.String(),
		}).Error(logger.LogMessageFailedToUpdateTask)
		status, code, message, details := mapDomainError(err)
		writeError(ctx, status, message, code, details)
		return
	}
	writeSuccess(ctx, http.StatusOK, "Task updated", response)
}

// GetTask handles task retrieval requests.
func (taskHandler *TaskHandler) GetTask(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		writeBindError(ctx, errors.New("missing or invalid UUID in request"))
		return
	}

	response, err := taskHandler.taskService.Get(ctx.Request.Context(), id, nil)
	if err != nil {
		userId, _ := middleware.UserIDFromContext(ctx.Request.Context())
		taskHandler.logger.WithError(err).WithFields(logger.Fields{
			"request_id": requestIDFromContext(ctx),
			"task_id":    id,
			"user_id":    userId.String(),
		}).Error(logger.LogMessageFailedToGetTask)
		status, code, message, details := mapDomainError(err)
		writeError(ctx, status, message, code, details)
		return
	}
	writeSuccess(ctx, http.StatusOK, "Task get", response)
}

// GetProjectTasks handles task retrieval requests for a specific project.
func (taskHandler *TaskHandler) GetProjectTasks(ctx *gin.Context) {
	projectId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		writeBindError(ctx, errors.New("missing or invalid UUID in request"))
	}

	limitParam := ctx.DefaultQuery("limit", "")
	offsetParam := ctx.DefaultQuery("offset", "")

	requestedLimit := 0
	requestedOffset := 0

	if limitParam != "" {
		if val, err := strconv.Atoi(limitParam); err == nil {
			requestedLimit = val
		}
	}

	if offsetParam != "" {
		if val, err := strconv.Atoi(offsetParam); err == nil {
			requestedOffset = val
		}
	}

	// isActive filter
	isActiveStr := ctx.Query("isActive")
	var isActive *bool

	if isActiveStr != "" {
		value := isActiveStr == "1"
		isActive = &value
	}

	response, pagination, err := taskHandler.taskService.GetByProject(
		ctx.Request.Context(),
		projectId,
		isActive,
		requestedLimit,
		requestedOffset,
	)
	if err != nil {
		userId, _ := middleware.UserIDFromContext(ctx.Request.Context())
		taskHandler.logger.WithError(err).WithFields(logger.Fields{
			"request_id": requestIDFromContext(ctx),
			"project_id": projectId,
			"user_id":    userId.String(),
		}).Error(logger.LogMessageFailedToGetTask)
		status, code, message, details := mapDomainError(err)
		writeError(ctx, status, message, code, details)
		return
	}
	writeSuccess(ctx, http.StatusOK, "Project's tasks", gin.H{
		"tasks":      response,
		"pagination": pagination,
	})
}

// DeleteTask handles task deletion requests.
func (taskHandler *TaskHandler) DeleteTask(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		writeBindError(ctx, errors.New("missing or invalid UUID in request"))
	}
	err = taskHandler.taskService.Delete(ctx.Request.Context(), id)
	if err != nil {
		userId, _ := middleware.UserIDFromContext(ctx.Request.Context())
		taskHandler.logger.WithError(err).WithFields(logger.Fields{
			"request_id": requestIDFromContext(ctx),
			"task_id":    id,
			"user_id":    userId.String(),
		}).Error(logger.LogMessageFailedToDeleteTask)
		status, code, message, details := mapDomainError(err)
		writeError(ctx, status, message, code, details)
		return
	}
	writeSuccess(ctx, http.StatusOK, "Task deleted", "")
}

// StartTask handles task start requests.
func (taskHandler *TaskHandler) StartTask(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		writeBindError(ctx, errors.New("missing or invalid UUID in request"))
	}

	workTimezone := ctx.GetHeader("X-Timezone")
	err = validateTimezone(workTimezone)
	if err != nil {
		writeBindError(ctx, errors.New("invalid or missing X-Timezone header"))
	}

	err = taskHandler.taskService.Start(ctx.Request.Context(), id, workTimezone)
	if err != nil {
		userId, _ := middleware.UserIDFromContext(ctx.Request.Context())
		taskHandler.logger.WithError(err).WithFields(logger.Fields{
			"request_id": requestIDFromContext(ctx),
			"task_id":    id,
			"user_id":    userId.String(),
		}).Error(logger.LogMessageFailedToStartTask)
		status, code, message, details := mapDomainError(err)
		writeError(ctx, status, message, code, details)
		return
	}
	writeSuccess(ctx, http.StatusOK, "Task started", "")
}

// StopTask handles task stop requests.
func (taskHandler *TaskHandler) StopTask(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		writeBindError(ctx, errors.New("missing or invalid UUID in request"))
	}
	err = taskHandler.taskService.Stop(ctx.Request.Context(), id)
	if err != nil {
		userId, _ := middleware.UserIDFromContext(ctx.Request.Context())
		taskHandler.logger.WithError(err).WithFields(logger.Fields{
			"request_id": requestIDFromContext(ctx),
			"task_id":    id,
			"user_id":    userId.String(),
		}).Error(logger.LogMessageFailedToStopTask)
		status, code, message, details := mapDomainError(err)
		writeError(ctx, status, message, code, details)
		return
	}
	writeSuccess(ctx, http.StatusOK, "Task stopped", "")
}

// CloseTask handles task close requests.
func (taskHandler *TaskHandler) CloseTask(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		writeBindError(ctx, errors.New("missing or invalid UUID in request"))
	}
	err = taskHandler.taskService.Close(ctx.Request.Context(), id)
	if err != nil {
		userId, _ := middleware.UserIDFromContext(ctx.Request.Context())
		taskHandler.logger.WithError(err).WithFields(logger.Fields{
			"request_id": requestIDFromContext(ctx),
			"task_id":    id,
			"user_id":    userId.String(),
		}).Error(logger.LogMessageFailedToCloseTask)
		status, code, message, details := mapDomainError(err)
		writeError(ctx, status, message, code, details)
		return
	}
	writeSuccess(ctx, http.StatusOK, "Task closed", "")
}

// validateTimezone validates the timezone string.
func validateTimezone(tz string) error {
	if tz == "" {
		return errors.New("timezone is required")
	}

	_, err := time.LoadLocation(tz)
	if err != nil {
		return fmt.Errorf("invalid timezone: %s", tz)
	}
	return nil
}
