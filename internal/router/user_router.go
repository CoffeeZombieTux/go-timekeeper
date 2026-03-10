package router

import (
	"go-timekeeper/internal/auth"
	"go-timekeeper/internal/handler"
	"go-timekeeper/internal/middleware"

	"github.com/gin-gonic/gin"
)

// setupUserRoutes sets up the routes for user management.
func setupUserRoutes(engine *gin.Engine, handlersPool *handler.HandlersPool, tokenManager auth.TokenManagerInterface) {
	user := engine.Group("/api/user")
	user.Use(middleware.AuthMiddleware(tokenManager))
	{
		user.GET("/me", handlersPool.User.GetMe)
		user.DELETE("/me", handlersPool.User.DeleteAccount)
	}
}
