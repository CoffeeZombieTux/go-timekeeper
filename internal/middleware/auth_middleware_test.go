package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go-timekeeper/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tm := auth.NewTokenManager("secret", 15, 24)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/secure", AuthMiddleware(tm), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_ValidTokenSetsContextUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tm := auth.NewTokenManager("secret", 15, 24)
	userID := uuid.New()
	token, err := tm.CreateAccessToken(userID, "u@example.com")
	if err != nil {
		t.Fatalf("create token failed: %v", err)
	}

	r := gin.New()
	r.Use(RequestID())
	r.GET("/secure", AuthMiddleware(tm), func(c *gin.Context) {
		id, ok := UserIDFromContext(c.Request.Context())
		if !ok {
			t.Fatal("expected user id in context")
		}
		if id != userID {
			t.Fatalf("unexpected user id %s", id)
		}
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}
