package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTokenManager_AccessTokenRoundTrip(t *testing.T) {
	tm := NewTokenManager("secret", 15, 24)
	userID := uuid.New()

	token, err := tm.CreateAccessToken(userID, "user@example.com")
	if err != nil {
		t.Fatalf("create access token failed: %v", err)
	}

	parsedID, email, err := tm.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("parse token failed: %v", err)
	}
	if parsedID != userID {
		t.Fatalf("unexpected user id %s", parsedID)
	}
	if email != "user@example.com" {
		t.Fatalf("unexpected email %s", email)
	}
}

func TestTokenManager_ParseInvalidToken(t *testing.T) {
	tm := NewTokenManager("secret", 15, 24)
	if _, _, err := tm.ParseAccessToken("not-a-token"); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestTokenManager_RefreshToken(t *testing.T) {
	tm := NewTokenManager("secret", 15, 24)
	raw, hash, expiresAt, err := tm.CreateRefreshToken()
	if err != nil {
		t.Fatalf("create refresh token failed: %v", err)
	}
	if raw == "" || hash == "" {
		t.Fatal("raw/hash should not be empty")
	}
	if len(hash) != 64 {
		t.Fatalf("unexpected hash length %d", len(hash))
	}
	if !expiresAt.After(time.Now().Add(23 * time.Hour)) {
		t.Fatal("refresh token expiry is too short")
	}

	rehashed := HashRefreshToken(raw)
	if !strings.EqualFold(hash, rehashed) {
		t.Fatal("hashed refresh token mismatch")
	}
}
