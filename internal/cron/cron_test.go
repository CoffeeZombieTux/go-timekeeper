package cron

import (
	"context"
	"testing"

	"go-timekeeper/internal/config"
	"go-timekeeper/internal/logger"
	apimodel "go-timekeeper/internal/model/api"
	"go-timekeeper/internal/service"
)

type fakeCronUserService struct{}

func (f *fakeCronUserService) Register(ctx context.Context, req apimodel.RegisterRequest) (*apimodel.AuthResponse, error) {
	return nil, nil
}
func (f *fakeCronUserService) Login(ctx context.Context, req apimodel.LoginRequest) (*apimodel.AuthResponse, error) {
	return nil, nil
}
func (f *fakeCronUserService) RefreshToken(ctx context.Context, req apimodel.RefreshRequest) (*apimodel.AuthResponse, error) {
	return nil, nil
}
func (f *fakeCronUserService) Logout(ctx context.Context, req apimodel.RefreshRequest) error {
	return nil
}
func (f *fakeCronUserService) Me(ctx context.Context) (*apimodel.UserPayload, error) { return nil, nil }
func (f *fakeCronUserService) ChangePassword(ctx context.Context, req apimodel.ChangePasswordRequest) error {
	return nil
}
func (f *fakeCronUserService) DeleteMe(ctx context.Context) error { return nil }
func (f *fakeCronUserService) CleanUp(ctx context.Context, interval int) error {
	return nil
}

func TestInitCrons(t *testing.T) {
	log := logger.New("error", "json")
	userService := &fakeCronUserService{}

	cfg := &config.Config{
		Cron: config.CronConfig{
			CronCleanupRevokedTokensSpec:         "@every 1h",
			CronCleanupRevokedTokensIntervalDays: 7,
		},
	}
	c := InitCrons(log, cfg, userService)
	if c == nil {
		t.Fatal("cron should not be nil")
	}
	if len(c.Entries()) != 1 {
		t.Fatalf("expected one cron entry, got %d", len(c.Entries()))
	}

	cfgBad := &config.Config{
		Cron: config.CronConfig{
			CronCleanupRevokedTokensSpec:         "bad spec",
			CronCleanupRevokedTokensIntervalDays: 7,
		},
	}
	cBad := InitCrons(log, cfgBad, userService)
	if cBad == nil {
		t.Fatal("cron should not be nil for bad spec")
	}
}

var _ service.UserServiceInterface = (*fakeCronUserService)(nil)
