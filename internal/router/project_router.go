package router

import (
	"go-timekeeper/internal/auth"
	"go-timekeeper/internal/handler"
	"go-timekeeper/internal/middleware"

	"github.com/gin-gonic/gin"
)

// setupProjectRoutes sets up the routes for project management.
func setupProjectRoutes(engine *gin.Engine, handlersPool *handler.HandlersPool, tokenManager auth.TokenManagerInterface) {
	projects := engine.Group("/api/project", middleware.AuthMiddleware(tokenManager))
	{
		projects.POST("", handlersPool.Project.CreateProject)
		projects.GET("/:id", handlersPool.Project.GetProject)
		projects.GET("/list", handlersPool.Project.GetUserProjects)
		projects.PATCH("", handlersPool.Project.UpdateProject)
		projects.DELETE("/:id", handlersPool.Project.DeleteProject)
	}
}
