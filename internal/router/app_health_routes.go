package router

import (
	apimodel "go-timekeeper/internal/model/api"
	"net/http"

	"github.com/gin-gonic/gin"
)

// setupAppHealthRoutes sets up the health check routes
func setupAppHealthRoutes(engine *gin.Engine) {
	engine.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, apimodel.APIResponse{
			Success: true,
			Message: "pong",
		})
	})

}
