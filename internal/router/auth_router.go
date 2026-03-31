package router

import (
	"go-timekeeper/internal/auth"
	"go-timekeeper/internal/handler"
	"go-timekeeper/internal/middleware"

	"github.com/gin-gonic/gin"
)

// setupAuthRoutes sets up the routes for authentication.
func setupAuthRoutes(engine *gin.Engine, handlersPool *handler.HandlersPool, tokenManager auth.TokenManagerInterface) {
	authRoutes := engine.Group("/api/auth")
	{
		authRoutes.POST("/register", handlersPool.User.Register)
		authRoutes.POST("/login", handlersPool.User.Login)
		authRoutes.POST("/logout", middleware.AuthMiddleware(tokenManager), handlersPool.User.Logout)
		authRoutes.POST("/refresh", handlersPool.User.RefreshToken)
		authRoutes.POST("/change-password", middleware.AuthMiddleware(tokenManager), handlersPool.User.ChangePassword)
	}
}
