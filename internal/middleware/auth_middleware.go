package middleware

import (
	"context"
	"go-timekeeper/internal/auth"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type contextKey string

const userIDKey contextKey = "userID"

// UserIDFromContext returns the user ID from the context.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	v := ctx.Value(userIDKey)
	id, ok := v.(uuid.UUID)
	return id, ok
}

// AuthMiddleware is a middleware that extracts the user ID from the Authorization header.
func AuthMiddleware(tm auth.TokenManagerInterface) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			unauthorizedResponse(ctx, "There is no Authorization header in request")
			return
		}
		const prefix = "Bearer "
		if !strings.HasPrefix(authHeader, prefix) {
			unauthorizedResponse(ctx, "Authorization header must use Bearer token")
			return
		}

		providedToken := strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
		if providedToken == "" {
			unauthorizedResponse(ctx, "There is invalid Authorization header in request")
			return
		}

		userID, _, err := tm.ParseAccessToken(providedToken)

		if err != nil {
			unauthorizedResponse(ctx, "There is invalid or expired Authorization header in request")
			return
		}

		ctxWithUserID := context.WithValue(ctx.Request.Context(), userIDKey, userID)
		ctx.Request = ctx.Request.WithContext(ctxWithUserID)

		ctx.Next()
	}
}
