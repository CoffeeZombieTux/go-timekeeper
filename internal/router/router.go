package router

import (
	"go-timekeeper/internal/config"
	"go-timekeeper/internal/logger"
	"go-timekeeper/internal/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// SetupRoutes sets up the application routes
func SetupRoutes(engine *gin.Engine, cfg *config.Config, log *logger.Logger) {
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-Id"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	engine.Use(cors.New(corsConfig))

	engine.Use(middleware.RequestID())
	engine.Use(middleware.HTTPLogging(log))

	setupAppHealthRoutes(engine)
}
