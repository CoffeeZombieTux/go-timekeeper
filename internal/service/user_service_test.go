package service

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-timekeeper/internal/auth"
	"go-timekeeper/internal/middleware"
	"go-timekeeper/internal/model"
	apimodel "go-timekeeper/internal/model/api"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type fakeUserRepo struct {
	usersByID    map[uuid.UUID]*model.User
	usersByEmail map[string]*model.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		usersByID:    make(map[uuid.UUID]*model.User),
		usersByEmail: make(map[string]*model.User),
	}
}

func (f *fakeUserRepo) GetById(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user, ok := f.usersByID[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	cp := *user
	return &cp, nil
}
func (f *fakeUserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	user, ok := f.usersByEmail[email]
	if !ok {
		return nil, sql.ErrNoRows
	}
	cp := *user
	return &cp, nil
}
func (f *fakeUserRepo) Create(ctx context.Context, email string, passwordHash string) (*model.User, error) {
	if _, exists := f.usersByEmail[email]; exists {
		return nil, sql.ErrNoRows
	}
	user := &model.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	f.usersByID[user.ID] = user
	f.usersByEmail[user.Email] = user
	cp := *user
	return &cp, nil
}
func (f *fakeUserRepo) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	user, ok := f.usersByID[id]
	if !ok {
		return sql.ErrNoRows
	}
	user.PasswordHash = passwordHash
	user.UpdatedAt = time.Now().UTC()
	return nil
}
func (f *fakeUserRepo) Delete(ctx context.Context, user *model.User) error {
	existing, ok := f.usersByID[user.ID]
	if !ok {
		return sql.ErrNoRows
	}
	delete(f.usersByEmail, existing.Email)
	delete(f.usersByID, user.ID)
	return nil
}

type fakeRefreshRepo struct {
	items map[string]*model.RefreshToken
}

func newFakeRefreshRepo() *fakeRefreshRepo {
	return &fakeRefreshRepo{items: make(map[string]*model.RefreshToken)}
}

func (f *fakeRefreshRepo) Create(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	f.items[tokenHash] = &model.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now().UTC(),
	}
	return nil
}
func (f *fakeRefreshRepo) GetValidByHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	token, ok := f.items[tokenHash]
	if !ok || token.RevokedAt != nil || token.ExpiresAt.Before(time.Now().UTC()) {
		return nil, sql.ErrNoRows
	}
	cp := *token
	return &cp, nil
}
func (f *fakeRefreshRepo) RevokeByHash(ctx context.Context, tokenHash string) error {
	token, ok := f.items[tokenHash]
	if !ok {
		return sql.ErrNoRows
	}
	now := time.Now().UTC()
	token.RevokedAt = &now
	return nil
}
func (f *fakeRefreshRepo) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	now := time.Now().UTC()
	for _, token := range f.items {
		if token.UserID == userID {
			token.RevokedAt = &now
		}
	}
	return nil
}
func (f *fakeRefreshRepo) CleanUp(ctx context.Context, interval int) error { return nil }

func authContextForUserService(t *testing.T, userID uuid.UUID) context.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	tm := auth.NewTokenManager("test-secret", 15, 24)
	token, err := tm.CreateAccessToken(userID, "u@example.com")
	if err != nil {
		t.Fatalf("token creation failed: %v", err)
	}

	r := gin.New()
	var reqCtx context.Context
	r.Use(middleware.AuthMiddleware(tm))
	r.GET("/", func(c *gin.Context) {
		reqCtx = c.Request.Context()
		c.Status(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return reqCtx
}

func TestUserService_FullFlow(t *testing.T) {
	userRepo := newFakeUserRepo()
	refreshRepo := newFakeRefreshRepo()
	tm := auth.NewTokenManager("test-secret", 15, 24)
	svc := NewUserService(userRepo, tm, refreshRepo)

	registerResp, err := svc.Register(context.Background(), apimodel.RegisterRequest{
		Email:    "service-flow@example.com",
		Password: "Valid@123",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if registerResp.AccessToken == "" || registerResp.RefreshToken == "" {
		t.Fatal("register should return tokens")
	}

	_, err = svc.Login(context.Background(), apimodel.LoginRequest{
		Email:    "service-flow@example.com",
		Password: "Valid@123",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	refreshResp, err := svc.RefreshToken(context.Background(), apimodel.RefreshRequest{
		RefreshToken: registerResp.RefreshToken,
	})
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if refreshResp.RefreshToken == "" {
		t.Fatal("refresh should return new refresh token")
	}

	userCtx := authContextForUserService(t, registerResp.User.ID)
	me, err := svc.Me(userCtx)
	if err != nil {
		t.Fatalf("me failed: %v", err)
	}
	if me.ID != registerResp.User.ID {
		t.Fatalf("unexpected me user id %s", me.ID)
	}

	err = svc.ChangePassword(userCtx, apimodel.ChangePasswordRequest{
		CurrentPassword: "Valid@123",
		NewPassword:     "NewValid@123",
	})
	if err != nil {
		t.Fatalf("change password failed: %v", err)
	}

	err = svc.Logout(context.Background(), apimodel.RefreshRequest{
		RefreshToken: refreshResp.RefreshToken,
	})
	if err != nil {
		t.Fatalf("logout failed: %v", err)
	}

	if err := svc.CleanUp(context.Background(), 7); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	if err := svc.DeleteMe(userCtx); err != nil {
		t.Fatalf("delete me failed: %v", err)
	}
}
