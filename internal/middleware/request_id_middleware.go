package middleware

import (
	"net/http"
	"strings"

	"go-timekeeper/internal/apperror"
	apimodel "go-timekeeper/internal/model/api"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const requestIDContextKey = "request_id"

// RequestID adds request id to context and response header.
func RequestID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		requestID := strings.TrimSpace(ctx.GetHeader("X-Request-Id"))
		if requestID == "" {
			requestID = uuid.NewString()
		}

		ctx.Set(requestIDContextKey, requestID)
		ctx.Writer.Header().Set("X-Request-Id", requestID)

		ctx.Next()
	}
}

// requestIDFromContext extracts request id from Gin context.
func requestIDFromContext(ctx *gin.Context) string {
	val, ok := ctx.Get(requestIDContextKey)
	if !ok {
		return ""
	}
	requestID, ok := val.(string)
	if !ok {
		return ""
	}
	return requestID
}

// unauthorizedResponse writes a standardized unauthorized API response.
func unauthorizedResponse(ctx *gin.Context, message string) {
	ctx.AbortWithStatusJSON(http.StatusUnauthorized, apimodel.APIResponse{
		Success: false,
		Message: message,
		Error: &apimodel.ErrorObject{
			Code:      apperror.CodeUnauthorizedCode,
			RequestID: requestIDFromContext(ctx),
		},
	})
}
