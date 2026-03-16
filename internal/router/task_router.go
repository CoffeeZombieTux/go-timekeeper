package router

import (
	"go-timekeeper/internal/auth"
	"go-timekeeper/internal/handler"
	"go-timekeeper/internal/middleware"

	"github.com/gin-gonic/gin"
)

// setupTaskRoutes sets up the routes for task management.
func setupTaskRoutes(engine *gin.Engine, handlersPool *handler.HandlersPool, tokenManager auth.TokenManagerInterface) {
	projects := engine.Group("/api/task", middleware.AuthMiddleware(tokenManager))
	{
		projects.POST("", handlersPool.Task.CreateTask)
		projects.GET("/:id", handlersPool.Task.GetTask)
		projects.GET("/list/project/:id", handlersPool.Task.GetProjectTasks)
		projects.PATCH("", handlersPool.Task.UpdateTask)
		projects.DELETE("/:id", handlersPool.Task.DeleteTask)
		projects.PATCH("/:id/start", handlersPool.Task.StartTask)
		projects.PATCH("/:id/stop", handlersPool.Task.StopTask)
		projects.PATCH("/:id/close", handlersPool.Task.CloseTask)

	}
}
