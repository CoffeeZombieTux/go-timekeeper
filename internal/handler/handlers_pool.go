package handler

import (
	"go-timekeeper/internal/logger"
	"go-timekeeper/internal/service"
)

// HandlersPool represents a collection of handlers
type HandlersPool struct {
	User       *UserHandler
	Project    *ProjectHandler
	Task       *TaskHandler
	TimeRecord *TimeRecordHandler
	Report     *ReportHandler
}

// NewHandlersPool creates a new HandlersPool instance
func NewHandlersPool(
	userService service.UserServiceInterface,
	projectService service.ProjectServiceInterface,
	taskService service.TaskServiceInterface,
	reportService service.ReportServiceInterface,
	timeRecordService service.TimeRecordServiceInterface,
	logger *logger.Logger,
) *HandlersPool {
	return &HandlersPool{
		User:       NewUserHandler(userService, logger),
		Project:    NewProjectHandler(projectService, logger),
		Task:       NewTaskHandler(taskService, logger),
		TimeRecord: NewTimeRecordHandler(timeRecordService, logger),
		Report:     NewReportHandler(reportService, logger),
	}
}
