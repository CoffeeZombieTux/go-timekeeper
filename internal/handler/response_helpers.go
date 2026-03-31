package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"go-timekeeper/internal/apperror"
	apimodel "go-timekeeper/internal/model/api"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

const requestIDContextKey = "request_id"

// requestIDFromContext returns request id from Gin context when available.
func requestIDFromContext(ctx *gin.Context) string {
	val, ok := ctx.Get(requestIDContextKey)
	if !ok {
		return ""
	}
	requestID, ok := val.(string)
	if !ok {
		return ""
	}
	return requestID
}

// writeSuccess writes a standardized successful API response.
func writeSuccess(ctx *gin.Context, status int, message string, data interface{}) {
	ctx.JSON(status, apimodel.APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// writeError writes a standardized error API response.
func writeError(ctx *gin.Context, status int, message string, code string, details []apimodel.ErrorDetail) {
	ctx.JSON(status, apimodel.APIResponse{
		Success: false,
		Message: message,
		Error: &apimodel.ErrorObject{
			Code:      code,
			Details:   details,
			RequestID: requestIDFromContext(ctx),
		},
	})
}

// writeBindError converts request binding errors into validation response format.
func writeBindError(ctx *gin.Context, err error) {
	details := make([]apimodel.ErrorDetail, 0)
	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		for _, fieldErr := range validationErrs {
			details = append(details, apimodel.ErrorDetail{
				Field:  fieldErr.Field(),
				Reason: fieldErr.Tag(),
			})
		}
	}
	if len(details) == 0 && err != nil {
		details = append(details, apimodel.ErrorDetail{Reason: err.Error()})
	}

	writeError(
		ctx,
		http.StatusBadRequest,
		apperror.CodeValidationErrorMessage,
		apperror.CodeValidationErrorCode,
		details,
	)
}

// mapDomainError maps domain/service errors to HTTP status and API error code.
func mapDomainError(err error) (status int, code string, message string, details []apimodel.ErrorDetail) {
	if err == nil {
		return http.StatusOK, "", "", nil
	}

	if appErr, ok := apperror.As(err); ok {
		detailItems := make([]apimodel.ErrorDetail, 0, len(appErr.Details))
		for _, d := range appErr.Details {
			detailItems = append(detailItems, apimodel.ErrorDetail{Reason: d})
		}
		if len(detailItems) == 0 && appErr.Cause != nil {
			detailItems = append(detailItems, apimodel.ErrorDetail{Reason: appErr.Cause.Error()})
		}

		switch appErr.Code {
		case apperror.CodeUnauthorizedCode:
			return http.StatusUnauthorized, appErr.Code, appErr.Message, detailItems
		case apperror.CodeValidationErrorCode, apperror.CodeDBCheckConstraintCode:
			return http.StatusUnprocessableEntity, appErr.Code, appErr.Message, detailItems
		case apperror.CodeDBNoRowsCode, apperror.CodeNotFoundCode:
			return http.StatusNotFound, appErr.Code, appErr.Message, detailItems
		case apperror.CodeDBDuplicateKeyCode, apperror.CodeDBForeignKeyConstraintCode, apperror.CodeConflictCode:
			return http.StatusConflict, appErr.Code, appErr.Message, detailItems
		default:
			return http.StatusInternalServerError, appErr.Code, appErr.Message, detailItems
		}
	}

	errMsg := err.Error()
	errMsgLower := strings.ToLower(errMsg)

	if errors.Is(err, sql.ErrNoRows) ||
		strings.Contains(errMsgLower, "no rows in result set") ||
		strings.Contains(errMsgLower, "not found") {
		return http.StatusNotFound, apperror.CodeDBNoRowsCode, apperror.CodeDBNoRowsMessage,
			[]apimodel.ErrorDetail{{Reason: errMsg}}
	}

	if strings.Contains(errMsgLower, "duplicate key value violates unique constraint") ||
		strings.Contains(errMsgLower, "violates unique constraint") {
		return http.StatusConflict, apperror.CodeDBDuplicateKeyCode, apperror.CodeDBDuplicateKeyMessage,
			[]apimodel.ErrorDetail{{Reason: errMsg}}
	}

	if strings.Contains(errMsgLower, "violates foreign key constraint") {
		return http.StatusConflict, apperror.CodeDBForeignKeyConstraintCode, apperror.CodeDBForeignKeyConstraintMessage,
			[]apimodel.ErrorDetail{{Reason: errMsg}}
	}

	if strings.Contains(errMsgLower, "violates check constraint") {
		return http.StatusUnprocessableEntity, apperror.CodeDBCheckConstraintCode, apperror.CodeDBCheckConstraintMessage,
			[]apimodel.ErrorDetail{{Reason: errMsg}}
	}

	switch {
	case strings.Contains(errMsg, "mail: ") ||
		strings.Contains(errMsg, "password must"):
		return http.StatusBadRequest, apperror.CodeValidationErrorCode, apperror.CodeValidationErrorMessage,
			[]apimodel.ErrorDetail{{Reason: errMsg}}
	default:
		return http.StatusInternalServerError, apperror.CodeInternalErrorCode, apperror.CodeInternalErrorMessage,
			[]apimodel.ErrorDetail{{Reason: errMsg}}
	}
}
