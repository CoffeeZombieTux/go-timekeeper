package handler

import (
	"go-timekeeper/internal/logger"
	"go-timekeeper/internal/service"
)

// HandlersPool represents a collection of handlers
type HandlersPool struct {
	User *UserHandler
}

// NewHandlersPool creates a new HandlersPool instance
func NewHandlersPool(
	userService service.UserServiceInterface,
	logger *logger.Logger,
) *HandlersPool {
	return &HandlersPool{
		User: NewUserHandler(userService, logger),
	}
}
