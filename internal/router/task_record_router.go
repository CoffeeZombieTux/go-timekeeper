package router

import (
	"go-timekeeper/internal/auth"
	"go-timekeeper/internal/handler"
	"go-timekeeper/internal/middleware"

	"github.com/gin-gonic/gin"
)

// setupTimeRecordRoutes sets up the routes for working sessions management.
func setupTimeRecordRoutes(engine *gin.Engine, handlersPool *handler.HandlersPool, tokenManager auth.TokenManagerInterface) {
	projects := engine.Group("/api/task/session", middleware.AuthMiddleware(tokenManager))
	{
		projects.POST("", handlersPool.TimeRecord.CreateTimeRecord)
		projects.PATCH("", handlersPool.TimeRecord.UpdateTimeRecord)
		projects.DELETE("/:id", handlersPool.TimeRecord.DeleteTimeRecord)
	}
}
