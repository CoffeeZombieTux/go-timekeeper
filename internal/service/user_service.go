package service

import (
	"context"
	"go-timekeeper/internal/apperror"
	"go-timekeeper/internal/auth"
	"go-timekeeper/internal/middleware"
	"go-timekeeper/internal/model"
	apimodel "go-timekeeper/internal/model/api"
	"go-timekeeper/internal/repository"
	"net/http"

	"github.com/google/uuid"
)

// UserServiceInterface represents a user service.
type UserServiceInterface interface {
	Register(ctx context.Context, req apimodel.RegisterRequest) (*apimodel.AuthResponse, error)
	Login(ctx context.Context, req apimodel.LoginRequest) (*apimodel.AuthResponse, error)
	RefreshToken(ctx context.Context, req apimodel.RefreshRequest) (*apimodel.AuthResponse, error)
	Logout(ctx context.Context, req apimodel.RefreshRequest) error
	Me(ctx context.Context) (*apimodel.UserPayload, error)
	ChangePassword(ctx context.Context, req apimodel.ChangePasswordRequest) error
	DeleteMe(ctx context.Context) error
	CleanUp(ctx context.Context, interval int) error
}

// UserService is a struct that implements the UserServiceInterface.
type UserService struct {
	userRepo     repository.UserRepositoryInterface
	tokenManager auth.TokenManagerInterface
	tokenRepo    repository.RefreshTokenRepositoryInterface
}

// NewUserService creates a new UserService instance.
func NewUserService(
	userRepo repository.UserRepositoryInterface,
	tokenManager auth.TokenManagerInterface,
	tokenRepo repository.RefreshTokenRepositoryInterface,
) *UserService {
	return &UserService{
		userRepo:     userRepo,
		tokenManager: tokenManager,
		tokenRepo:    tokenRepo,
	}
}

// Register process user registration.
func (userService *UserService) Register(
	ctx context.Context,
	req apimodel.RegisterRequest,
) (*apimodel.AuthResponse, error) {
	if err := req.ValidateRegisterRequest(); err != nil {
		return nil, err
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user, err := userService.userRepo.Create(ctx, req.Email, hashedPassword)
	if err != nil {
		return nil, err
	}
	return userService.loginUser(ctx, user)
}

// Login process user login.
func (userService *UserService) Login(
	ctx context.Context,
	req apimodel.LoginRequest,
) (*apimodel.AuthResponse, error) {
	user, err := userService.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}

	err = auth.ComparePasswords(user.PasswordHash, req.Password)
	if err != nil {
		return nil, err
	}
	return userService.loginUser(ctx, user)
}

// RefreshToken process user refresh token.
func (userService *UserService) RefreshToken(
	ctx context.Context,
	req apimodel.RefreshRequest,
) (*apimodel.AuthResponse, error) {
	oldHash := auth.HashRefreshToken(req.RefreshToken)
	stored, err := userService.tokenRepo.GetValidByHash(ctx, oldHash)
	if err != nil {
		return nil, err
	}

	user, err := userService.userRepo.GetById(ctx, stored.UserID)
	if err != nil {
		return nil, err
	}

	if err := userService.tokenRepo.RevokeByHash(ctx, oldHash); err != nil {
		return nil, err
	}
	return userService.loginUser(ctx, user)
}

// Logout process user logout.
func (userService *UserService) Logout(ctx context.Context, req apimodel.RefreshRequest) error {
	if err := userService.tokenRepo.RevokeByHash(ctx, auth.HashRefreshToken(req.RefreshToken)); err != nil {
		return err
	}
	return nil
}

// Me returns the current user's information.'
func (userService *UserService) Me(ctx context.Context) (*apimodel.UserPayload, error) {
	userId, err := getUserIdFromRequest(ctx)
	if err != nil {
		return nil, err
	}

	user, err := userService.userRepo.GetById(ctx, userId)
	if err != nil {
		return nil, err
	}
	return &apimodel.UserPayload{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.UTC().Format(http.TimeFormat),
		UpdatedAt: user.UpdatedAt.UTC().Format(http.TimeFormat),
	}, nil
}

// ChangePassword process changing the user's password.'
func (userService *UserService) ChangePassword(ctx context.Context, req apimodel.ChangePasswordRequest) error {
	userId, err := getUserIdFromRequest(ctx)
	if err != nil {
		return err
	}
	if err := req.ValidateChangePasswordRequest(); err != nil {
		return err
	}
	user, err := userService.userRepo.GetById(ctx, userId)
	if err != nil {
		return err
	}
	if err := auth.ComparePasswords(user.PasswordHash, req.CurrentPassword); err != nil {
		return err
	}
	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}
	err = userService.userRepo.UpdatePassword(ctx, user.ID, newHash)
	if err != nil {
		return err
	}
	return userService.tokenRepo.RevokeAllForUser(ctx, user.ID)
}

// DeleteMe process current user delete.
func (userService *UserService) DeleteMe(ctx context.Context) error {
	userId, err := getUserIdFromRequest(ctx)
	if err != nil {
		return err
	}
	return userService.userRepo.Delete(ctx, &model.User{ID: userId})
}

// CleanUp deletes expired refresh tokens.
func (userService *UserService) CleanUp(ctx context.Context, interval int) error {
	return userService.tokenRepo.CleanUp(ctx, interval)
}

// Helper function to log in a user.
func (userService *UserService) loginUser(ctx context.Context, user *model.User) (*apimodel.AuthResponse, error) {
	accessToken, err := userService.tokenManager.CreateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	refreshRaw, refreshHash, expiresAt, err := userService.tokenManager.CreateRefreshToken()
	if err != nil {
		return nil, err
	}

	if err := userService.tokenRepo.Create(ctx, user.ID, refreshHash, expiresAt); err != nil {
		return nil, err
	}

	return &apimodel.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshRaw,
		User: apimodel.UserPayload{
			ID:        user.ID,
			Email:     user.Email,
			CreatedAt: user.CreatedAt.UTC().Format(http.TimeFormat),
			UpdatedAt: user.UpdatedAt.UTC().Format(http.TimeFormat),
		},
	}, nil
}

func getUserIdFromRequest(ctx context.Context) (uuid.UUID, error) {
	userId, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		return uuid.Nil, apperror.New(apperror.CodeUnauthorizedCode, apperror.CodeUnauthorizedMessage, "User not authenticated")
	}
	return userId, nil
}
