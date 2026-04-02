package config

import "testing"

func TestLoadAndDSN(t *testing.T) {
	t.Setenv("DB_HOST", "db.local")
	t.Setenv("DB_PORT", "5544")
	t.Setenv("DB_USER", "u1")
	t.Setenv("DB_PASSWORD", "p1")
	t.Setenv("DB_NAME", "name1")
	t.Setenv("DB_SSL_MODE", "disable")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "text")
	t.Setenv("JWT_SECRET", "x")
	t.Setenv("ACCESS_TOKEN_TTL_MINUTES", "30")
	t.Setenv("REFRESH_TOKEN_TTL_HOURS", "72")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if cfg.Database.Host != "db.local" || cfg.Database.Port != 5544 {
		t.Fatalf("unexpected db config: %+v", cfg.Database)
	}
	if cfg.Server.Port != 9090 {
		t.Fatalf("unexpected server port %d", cfg.Server.Port)
	}
	dsn := cfg.GetDatabaseDSN()
	if dsn == "" {
		t.Fatal("dsn should not be empty")
	}
}

func TestGetEnvAndGetEnvIntDefaults(t *testing.T) {
	if got := getEnv("UNSET_ENV_X", "default"); got != "default" {
		t.Fatalf("expected default, got %s", got)
	}
	t.Setenv("INT_BAD", "x")
	if got := getEnvInt("INT_BAD", 5); got != 5 {
		t.Fatalf("expected fallback int, got %d", got)
	}
}
