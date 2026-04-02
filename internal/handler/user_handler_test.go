package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-timekeeper/internal/apperror"
	"go-timekeeper/internal/auth"
	"go-timekeeper/internal/logger"
	"go-timekeeper/internal/middleware"
	apimodel "go-timekeeper/internal/model/api"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type fakeUserHandlerService struct {
	registerFn       func(ctx context.Context, req apimodel.RegisterRequest) (*apimodel.AuthResponse, error)
	loginFn          func(ctx context.Context, req apimodel.LoginRequest) (*apimodel.AuthResponse, error)
	refreshFn        func(ctx context.Context, req apimodel.RefreshRequest) (*apimodel.AuthResponse, error)
	logoutFn         func(ctx context.Context, req apimodel.RefreshRequest) error
	meFn             func(ctx context.Context) (*apimodel.UserPayload, error)
	changePasswordFn func(ctx context.Context, req apimodel.ChangePasswordRequest) error
	deleteMeFn       func(ctx context.Context) error
}

func (f *fakeUserHandlerService) Register(
	ctx context.Context,
	req apimodel.RegisterRequest,
) (*apimodel.AuthResponse, error) {
	if f.registerFn != nil {
		return f.registerFn(ctx, req)
	}
	return &apimodel.AuthResponse{}, nil
}

func (f *fakeUserHandlerService) Login(
	ctx context.Context,
	req apimodel.LoginRequest,
) (*apimodel.AuthResponse, error) {
	if f.loginFn != nil {
		return f.loginFn(ctx, req)
	}
	return &apimodel.AuthResponse{}, nil
}

func (f *fakeUserHandlerService) RefreshToken(
	ctx context.Context,
	req apimodel.RefreshRequest,
) (*apimodel.AuthResponse, error) {
	if f.refreshFn != nil {
		return f.refreshFn(ctx, req)
	}
	return &apimodel.AuthResponse{}, nil
}

func (f *fakeUserHandlerService) Logout(ctx context.Context, req apimodel.RefreshRequest) error {
	if f.logoutFn != nil {
		return f.logoutFn(ctx, req)
	}
	return nil
}

func (f *fakeUserHandlerService) Me(ctx context.Context) (*apimodel.UserPayload, error) {
	if f.meFn != nil {
		return f.meFn(ctx)
	}
	return &apimodel.UserPayload{}, nil
}

func (f *fakeUserHandlerService) ChangePassword(ctx context.Context, req apimodel.ChangePasswordRequest) error {
	if f.changePasswordFn != nil {
		return f.changePasswordFn(ctx, req)
	}
	return nil
}

func (f *fakeUserHandlerService) DeleteMe(ctx context.Context) error {
	if f.deleteMeFn != nil {
		return f.deleteMeFn(ctx)
	}
	return nil
}

func (f *fakeUserHandlerService) CleanUp(ctx context.Context, interval int) error { return nil }

func setupUserHandlerRouter(t *testing.T, service *fakeUserHandlerService) (*gin.Engine, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	tokenManager := auth.NewTokenManager("test-secret", 15, 24)
	token, err := tokenManager.CreateAccessToken(uuid.New(), "u@example.com")
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	h := NewUserHandler(service, logger.New("error", "json"))
	r := gin.New()
	r.Use(middleware.RequestID())

	authRoutes := r.Group("/api/auth")
	{
		authRoutes.POST("/register", h.Register)
		authRoutes.POST("/login", h.Login)
		authRoutes.POST("/refresh", h.RefreshToken)
		authRoutes.POST("/logout", middleware.AuthMiddleware(tokenManager), h.Logout)
		authRoutes.POST("/change-password", middleware.AuthMiddleware(tokenManager), h.ChangePassword)
	}
	userRoutes := r.Group("/api/user")
	userRoutes.Use(middleware.AuthMiddleware(tokenManager))
	{
		userRoutes.GET("/me", h.GetMe)
		userRoutes.DELETE("/me", h.DeleteAccount)
	}

	return r, token
}

func userHandlerRequest(t *testing.T, r *gin.Engine, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	var raw []byte
	if body != nil {
		var err error
		raw, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestUserHandler_PublicFlowAndSanitize(t *testing.T) {
	createdAt := time.Now().UTC().Format(http.TimeFormat)
	updatedAt := createdAt
	userID := uuid.New()

	var capturedRegisterEmail string
	var capturedLoginEmail string
	service := &fakeUserHandlerService{
		registerFn: func(ctx context.Context, req apimodel.RegisterRequest) (*apimodel.AuthResponse, error) {
			capturedRegisterEmail = req.Email
			return &apimodel.AuthResponse{
				AccessToken:  "access",
				RefreshToken: "refresh",
				User: apimodel.UserPayload{
					ID:        userID,
					Email:     req.Email,
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
			}, nil
		},
		loginFn: func(ctx context.Context, req apimodel.LoginRequest) (*apimodel.AuthResponse, error) {
			capturedLoginEmail = req.Email
			return &apimodel.AuthResponse{
				AccessToken:  "access2",
				RefreshToken: "refresh2",
				User: apimodel.UserPayload{
					ID:        userID,
					Email:     req.Email,
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
			}, nil
		},
		refreshFn: func(ctx context.Context, req apimodel.RefreshRequest) (*apimodel.AuthResponse, error) {
			return &apimodel.AuthResponse{
				AccessToken:  "new-access",
				RefreshToken: "new-refresh",
				User: apimodel.UserPayload{
					ID:        userID,
					Email:     "user@example.com",
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
			}, nil
		},
	}
	router, _ := setupUserHandlerRouter(t, service)

	registerRec := userHandlerRequest(t, router, http.MethodPost, "/api/auth/register", apimodel.RegisterRequest{
		Email:    "  USER@Example.COM  ",
		Password: "Valid@123",
	}, "")
	if registerRec.Code != http.StatusOK {
		t.Fatalf("register expected 200, got %d body=%s", registerRec.Code, registerRec.Body.String())
	}
	if capturedRegisterEmail != "user@example.com" {
		t.Fatalf("expected sanitized register email, got %q", capturedRegisterEmail)
	}

	loginRec := userHandlerRequest(t, router, http.MethodPost, "/api/auth/login", apimodel.LoginRequest{
		Email:    "  USER@Example.COM  ",
		Password: "Valid@123",
	}, "")
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login expected 200, got %d body=%s", loginRec.Code, loginRec.Body.String())
	}
	if capturedLoginEmail != "user@example.com" {
		t.Fatalf("expected sanitized login email, got %q", capturedLoginEmail)
	}

	refreshRec := userHandlerRequest(t, router, http.MethodPost, "/api/auth/refresh", apimodel.RefreshRequest{
		RefreshToken: "refresh-token",
	}, "")
	if refreshRec.Code != http.StatusOK {
		t.Fatalf("refresh expected 200, got %d body=%s", refreshRec.Code, refreshRec.Body.String())
	}
}

func TestUserHandler_AuthFlowAndErrorCases(t *testing.T) {
	service := &fakeUserHandlerService{
		logoutFn: func(ctx context.Context, req apimodel.RefreshRequest) error {
			return apperror.New(
				apperror.CodeUnauthorizedCode,
				apperror.CodeUnauthorizedMessage,
				"unauthorized",
			)
		},
		meFn: func(ctx context.Context) (*apimodel.UserPayload, error) {
			return &apimodel.UserPayload{ID: uuid.New(), Email: "u@example.com"}, nil
		},
		changePasswordFn: func(ctx context.Context, req apimodel.ChangePasswordRequest) error {
			return apperror.New(
				apperror.CodeValidationErrorCode,
				apperror.CodeValidationErrorMessage,
				"invalid password",
			)
		},
		deleteMeFn: func(ctx context.Context) error { return nil },
	}
	router, token := setupUserHandlerRouter(t, service)

	logoutRec := userHandlerRequest(t, router, http.MethodPost, "/api/auth/logout", apimodel.RefreshRequest{
		RefreshToken: "r",
	}, token)
	if logoutRec.Code != http.StatusUnauthorized {
		t.Fatalf("logout expected 401, got %d body=%s", logoutRec.Code, logoutRec.Body.String())
	}

	meRec := userHandlerRequest(t, router, http.MethodGet, "/api/user/me", nil, token)
	if meRec.Code != http.StatusOK {
		t.Fatalf("get me expected 200, got %d body=%s", meRec.Code, meRec.Body.String())
	}

	changePassRec := userHandlerRequest(t, router, http.MethodPost, "/api/auth/change-password", apimodel.ChangePasswordRequest{
		CurrentPassword: "Old@1234",
		NewPassword:     "New@1234",
	}, token)
	if changePassRec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("change password expected 422, got %d body=%s", changePassRec.Code, changePassRec.Body.String())
	}

	deleteRec := userHandlerRequest(t, router, http.MethodDelete, "/api/user/me", nil, token)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete account expected 200, got %d body=%s", deleteRec.Code, deleteRec.Body.String())
	}
}

func TestUserHandler_BindErrors(t *testing.T) {
	service := &fakeUserHandlerService{}
	router, token := setupUserHandlerRouter(t, service)

	tests := []struct {
		method string
		path   string
		body   any
		token  string
	}{
		{method: http.MethodPost, path: "/api/auth/register", body: map[string]any{}, token: ""},
		{method: http.MethodPost, path: "/api/auth/login", body: map[string]any{}, token: ""},
		{method: http.MethodPost, path: "/api/auth/refresh", body: map[string]any{}, token: ""},
		{method: http.MethodPost, path: "/api/auth/logout", body: map[string]any{}, token: token},
		{method: http.MethodPost, path: "/api/auth/change-password", body: map[string]any{}, token: token},
	}

	for _, tt := range tests {
		rec := userHandlerRequest(t, router, tt.method, tt.path, tt.body, tt.token)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s %s expected 400, got %d body=%s", tt.method, tt.path, rec.Code, rec.Body.String())
		}
	}
}

func TestSanitizeEmail(t *testing.T) {
	got := sanitizeEmail("  USER@Example.COM  ")
	if got != "user@example.com" {
		t.Fatalf("unexpected sanitized email: %s", got)
	}
}
