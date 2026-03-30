package router

import (
	"go-timekeeper/internal/auth"
	"go-timekeeper/internal/handler"
	"go-timekeeper/internal/middleware"

	"github.com/gin-gonic/gin"
)

func setupReportRoutes(engine *gin.Engine, handlersPool *handler.HandlersPool, tokenManager auth.TokenManagerInterface) {
	projects := engine.Group("/api/report", middleware.AuthMiddleware(tokenManager))
	{
		projects.POST("/general", handlersPool.Report.GeneralReport)
		projects.POST("/project", handlersPool.Report.ProjectReport)
		projects.POST("/task", handlersPool.Report.TaskReport)
	}
}
