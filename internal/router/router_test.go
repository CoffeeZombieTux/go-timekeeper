package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go-timekeeper/internal/auth"
	"go-timekeeper/internal/handler"
	"go-timekeeper/internal/logger"

	"github.com/gin-gonic/gin"
)

func TestSetupRoutes_RegistersExpectedRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	handlersPool := &handler.HandlersPool{}
	tokenManager := auth.NewTokenManager("secret", 15, 24)

	SetupRoutes(engine, handlersPool, tokenManager, logger.New("error", "json"))

	routes := engine.Routes()
	if len(routes) == 0 {
		t.Fatal("expected routes to be registered")
	}

	required := map[string]bool{
		"GET /ping":                    false,
		"POST /api/auth/register":      false,
		"POST /api/auth/login":         false,
		"GET /api/user/me":             false,
		"POST /api/project":            false,
		"POST /api/task":               false,
		"POST /api/task/session":       false,
		"POST /api/report/general":     false,
		"PATCH /api/task/:id/start":    false,
		"DELETE /api/task/session/:id": false,
	}
	for _, r := range routes {
		key := r.Method + " " + r.Path
		if _, ok := required[key]; ok {
			required[key] = true
		}
	}
	for key, ok := range required {
		if !ok {
			t.Fatalf("required route not registered: %s", key)
		}
	}
}

func TestPingRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	setupAppHealthRoutes(engine)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
