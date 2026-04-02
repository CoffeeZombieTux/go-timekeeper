package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-timekeeper/internal/logger"

	"github.com/gin-gonic/gin"
)

func TestHTTPLoggingMiddleware_AndHelpers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := logger.New("debug", "json")
	r := gin.New()
	r.Use(RequestID())
	r.Use(HTTPLogging(log))
	r.POST("/log-test/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/log-test/123?x=1", bytes.NewBufferString(`{"x":1}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	textBody, truncated := normalizeBodyForLog("application/json", []byte(`{"a":1}`))
	if textBody == "" || truncated {
		t.Fatalf("unexpected normalizeBodyForLog output: body=%q truncated=%v", textBody, truncated)
	}

	omittedBody, _ := normalizeBodyForLog("application/octet-stream", []byte{0x01, 0x02})
	if omittedBody == "" {
		t.Fatal("expected omitted marker for binary body")
	}

	if !isTextLikeContentType("text/plain") {
		t.Fatal("text/plain should be text-like")
	}
	if isTextLikeContentType("application/octet-stream") {
		t.Fatal("octet-stream should not be text-like")
	}
}
