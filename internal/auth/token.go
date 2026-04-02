package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenManagerInterface defines the interface for token management.
type TokenManagerInterface interface {
	CreateAccessToken(userID uuid.UUID, email string) (string, error)
	ParseAccessToken(tokenStr string) (uuid.UUID, string, error)
	CreateRefreshToken() (rawToken string, tokenHash string, expiresAt time.Time, err error)
}

// TokenManager is a struct that implements the TokenManagerInterface.
type TokenManager struct {
	Secret                []byte
	AccessTokenTTLMinutes int
	RefreshTokenTTLHours  int
}

// NewTokenManager creates a new TokenManager instance.
func NewTokenManager(secret string, accessTTLMinutes, refreshTTLHours int) *TokenManager {
	return &TokenManager{
		Secret:                []byte(secret),
		AccessTokenTTLMinutes: accessTTLMinutes,
		RefreshTokenTTLHours:  refreshTTLHours,
	}
}

// CreateAccessToken creates a new access token.
func (tm *TokenManager) CreateAccessToken(userID uuid.UUID, email string) (string, error) {
	claims := jwt.MapClaims{
		"sub":   userID.String(),
		"email": email,
		"exp":   time.Now().Add(time.Duration(tm.AccessTokenTTLMinutes) * time.Minute).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(tm.Secret)
}

// ParseAccessToken parses user data by provided access token or returns error.
func (tm *TokenManager) ParseAccessToken(tokenStr string) (uuid.UUID, string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return tm.Secret, nil
	})
	if err != nil {
		return uuid.Nil, "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return uuid.Nil, "", jwt.ErrTokenInvalidClaims
	}

	subValue, exists := claims["sub"]
	if !exists {
		return uuid.Nil, "", jwt.ErrTokenInvalidClaims
	}

	sub, ok := subValue.(string)
	if !ok {
		return uuid.Nil, "", jwt.ErrTokenInvalidClaims
	}

	email, _ := claims["email"].(string)

	userID, err := uuid.Parse(sub)
	if err != nil {
		return uuid.Nil, "", err
	}

	return userID, email, nil
}

// CreateRefreshToken creates a new refresh token.
func (tm *TokenManager) CreateRefreshToken() (rawToken string, tokenHash string, expiresAt time.Time, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", time.Time{}, err
	}

	rawToken = hex.EncodeToString(b)
	sum := sha256.Sum256([]byte(rawToken))
	tokenHash = hex.EncodeToString(sum[:])
	expiresAt = time.Now().Add(time.Duration(tm.RefreshTokenTTLHours) * time.Hour)

	return rawToken, tokenHash, expiresAt, nil
}

// HashRefreshToken hashes raw refresh token.
func HashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
