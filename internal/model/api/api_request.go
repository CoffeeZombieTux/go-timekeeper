package api_model

import (
	"errors"
	"go-timekeeper/internal/validator"
)

// RegisterRequest represents the request body for user registration.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ValidateRegisterRequest validates the RegisterRequest.
func (input *RegisterRequest) ValidateRegisterRequest() error {
	err := validator.ValidateEmail(input.Email)
	if err != nil {
		return err
	}
	err = validator.ValidatePassword(input.Password)
	if err != nil {
		return err
	}
	return nil
}

// LoginRequest represents the request body for user login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RefreshRequest represents the request body for refreshing the access token or logout.
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// ValidateRefreshTokenRequest validates the RefreshRequest.
func (input *RefreshRequest) ValidateRefreshTokenRequest() error {
	if input.RefreshToken == "" {
		return errors.New("refresh token is required")
	}
	return nil
}

// ChangePasswordRequest represents the request body for changing the password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// ValidateChangePasswordRequest validates the ChangePasswordRequest.
func (input *ChangePasswordRequest) ValidateChangePasswordRequest() error {
	err := validator.ValidatePassword(input.NewPassword)
	if err != nil {
		return err
	}
	return nil
}
