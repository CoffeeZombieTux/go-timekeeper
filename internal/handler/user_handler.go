package handler

import (
	"go-timekeeper/internal/logger"
	apimodel "go-timekeeper/internal/model/api"
	"go-timekeeper/internal/service"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// UserHandler handles user-related and auth-related requests.
type UserHandler struct {
	UserService service.UserServiceInterface
	logger      *logger.Logger
}

// NewUserHandler creates a new UserHandler instance.
func NewUserHandler(userService service.UserServiceInterface, logger *logger.Logger) *UserHandler {
	return &UserHandler{
		UserService: userService,
		logger:      logger,
	}
}

// Register handles user registration requests.
func (userHandler *UserHandler) Register(ctx *gin.Context) {
	var req apimodel.RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}
	req.Email = sanitizeEmail(req.Email)

	response, err := userHandler.UserService.Register(ctx.Request.Context(), req)
	if err != nil {
		userHandler.logger.WithError(err).WithFields(logger.Fields{
			"request_id": requestIDFromContext(ctx),
			"email":      req.Email,
		}).Error(logger.LogMessageFailedToRegisterUser)
		status, code, message, details := mapDomainError(err)
		writeError(ctx, status, message, code, details)
		return
	}
	writeSuccess(ctx, http.StatusOK, "User registered", response)
}

// Login handles user login requests.
func (userHandler *UserHandler) Login(ctx *gin.Context) {
	var req apimodel.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}
	req.Email = sanitizeEmail(req.Email)

	response, err := userHandler.UserService.Login(ctx.Request.Context(), req)
	if err != nil {
		userHandler.logger.WithError(err).WithFields(logger.Fields{
			"request_id": requestIDFromContext(ctx),
			"email":      req.Email,
		}).Error(logger.LogMessageFailedToLoginUser)
		status, code, message, details := mapDomainError(err)
		writeError(ctx, status, message, code, details)
		return
	}
	writeSuccess(ctx, http.StatusOK, "User is logged-in", response)
}

// RefreshToken handles user token refresh requests.
func (userHandler *UserHandler) RefreshToken(ctx *gin.Context) {
	var req apimodel.RefreshRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}
	response, err := userHandler.UserService.RefreshToken(ctx.Request.Context(), req)
	if err != nil {
		userHandler.logger.WithError(err).WithFields(logger.Fields{
			"request_id": requestIDFromContext(ctx),
			"operation":  "refresh_token",
		}).Error(logger.LogMessageFailedToRefreshUserToken)
		status, code, message, details := mapDomainError(err)
		writeError(ctx, status, message, code, details)
		return
	}
	writeSuccess(ctx, http.StatusOK, "Tokens refreshed", response)
}

// Logout handles user logout requests.
func (userHandler *UserHandler) Logout(ctx *gin.Context) {
	var req apimodel.RefreshRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}

	err := userHandler.UserService.Logout(ctx.Request.Context(), req)
	if err != nil {
		userHandler.logger.WithError(err).WithFields(logger.Fields{
			"request_id": requestIDFromContext(ctx),
			"operation":  "logout",
		}).Error(logger.LogMessageFailedToLogoutUser)
		status, code, message, details := mapDomainError(err)
		writeError(ctx, status, message, code, details)
		return
	}
	writeSuccess(ctx, http.StatusOK, "User logged-out", "")
}

// GetMe handles user profile requests.
func (userHandler *UserHandler) GetMe(ctx *gin.Context) {
	response, err := userHandler.UserService.Me(ctx.Request.Context())
	if err != nil {
		userHandler.logger.WithError(err).WithFields(logger.Fields{
			"request_id": requestIDFromContext(ctx),
		}).Error(logger.LogMessageFailedToGetUser)
		status, code, message, details := mapDomainError(err)
		writeError(ctx, status, message, code, details)
		return
	}
	writeSuccess(ctx, http.StatusOK, "User get success", response)
}

// ChangePassword handles user password change requests.
func (userHandler *UserHandler) ChangePassword(ctx *gin.Context) {
	var req apimodel.ChangePasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}
	err := userHandler.UserService.ChangePassword(ctx.Request.Context(), req)
	if err != nil {
		userHandler.logger.WithError(err).WithFields(logger.Fields{
			"request_id": requestIDFromContext(ctx),
		}).Error(logger.LogMessageFailedToChangePassword)
		status, code, message, details := mapDomainError(err)
		writeError(ctx, status, message, code, details)
		return
	}
	writeSuccess(ctx, http.StatusOK, "Password is changed", "")
}

// DeleteAccount handles user account deletion requests.
func (userHandler *UserHandler) DeleteAccount(ctx *gin.Context) {
	err := userHandler.UserService.DeleteMe(ctx.Request.Context())
	if err != nil {
		userHandler.logger.WithError(err).WithFields(logger.Fields{
			"request_id": requestIDFromContext(ctx),
		}).Error(logger.LogMessageFailedToDeleteUser)
		status, code, message, details := mapDomainError(err)
		writeError(ctx, status, message, code, details)
		return
	}
	writeSuccess(ctx, http.StatusOK, "User deleted", "")
}

// sanitizeEmail sanitizes the email address.
func sanitizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
