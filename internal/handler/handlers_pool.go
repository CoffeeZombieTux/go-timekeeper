package handler

import (
	"go-timekeeper/internal/logger"
	"go-timekeeper/internal/service"
)

// HandlersPool represents a collection of handlers
type HandlersPool struct {
	User    *UserHandler
	Project *ProjectHandler
}

// NewHandlersPool creates a new HandlersPool instance
func NewHandlersPool(
	userService service.UserServiceInterface,
	projectService service.ProjectServiceInterface,
	logger *logger.Logger,
) *HandlersPool {
	return &HandlersPool{
		User:    NewUserHandler(userService, logger),
		Project: NewProjectHandler(projectService, logger),
	}
}
