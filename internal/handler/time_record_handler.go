package handler

import (
	"errors"
	"go-timekeeper/internal/logger"
	"go-timekeeper/internal/middleware"
	apimodel "go-timekeeper/internal/model/api"
	"go-timekeeper/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TimeRecordHandler handles time record-related requests.
type TimeRecordHandler struct {
	timeRecordService service.TimeRecordServiceInterface
	logger            *logger.Logger
}

// NewTimeRecordHandler creates a new TimeRecordHandler instance.
func NewTimeRecordHandler(timeRecordService service.TimeRecordServiceInterface, logger *logger.Logger) *TimeRecordHandler {
	return &TimeRecordHandler{
		timeRecordService: timeRecordService,
		logger:            logger,
	}
}

// CreateTimeRecord handles time record creation requests.
func (timeRecordHandler *TimeRecordHandler) CreateTimeRecord(ctx *gin.Context) {
	var req apimodel.CreateTimeRecordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}
	response, err := timeRecordHandler.timeRecordService.CreateTimeRecord(ctx.Request.Context(), &req)
	if err != nil {
		userId, _ := middleware.UserIDFromContext(ctx.Request.Context())
		timeRecordHandler.logger.WithError(err).WithFields(logger.Fields{
			"request_id": requestIDFromContext(ctx),
			"user_id":    userId.String(),
		}).Error(logger.LogMessageFailedToCreateTimeRecord)
		status, code, message, details := mapDomainError(err)
		writeError(ctx, status, message, code, details)
		return
	}
	writeSuccess(ctx, http.StatusOK, "Time record(s) created", response)
}

// UpdateTimeRecord handles time record update requests.
func (timeRecordHandler *TimeRecordHandler) UpdateTimeRecord(ctx *gin.Context) {
	var req apimodel.UpdateTimeRecordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}
	response, err := timeRecordHandler.timeRecordService.UpdateTimeRecord(ctx.Request.Context(), &req)
	if err != nil {
		userId, _ := middleware.UserIDFromContext(ctx.Request.Context())
		timeRecordHandler.logger.WithError(err).WithFields(logger.Fields{
			"request_id": requestIDFromContext(ctx),
			"user_id":    userId.String(),
		}).Error(logger.LogMessageFailedToUpdateTimeRecord)
		status, code, message, details := mapDomainError(err)
		writeError(ctx, status, message, code, details)
		return
	}
	writeSuccess(ctx, http.StatusOK, "Time record updated", response)
}

// DeleteTimeRecord handles time record deletion requests.
func (timeRecordHandler *TimeRecordHandler) DeleteTimeRecord(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		writeBindError(ctx, errors.New("missing or invalid UUID in request"))
	}
	err = timeRecordHandler.timeRecordService.DeleteTimeRecord(ctx.Request.Context(), id)
	if err != nil {
		userId, _ := middleware.UserIDFromContext(ctx.Request.Context())
		timeRecordHandler.logger.WithError(err).WithFields(logger.Fields{
			"request_id": requestIDFromContext(ctx),
			"user_id":    userId.String(),
		}).Error(logger.LogMessageFailedToDeleteTimeRecord)
		status, code, message, details := mapDomainError(err)
		writeError(ctx, status, message, code, details)
		return
	}
	writeSuccess(ctx, http.StatusOK, "Time record", "")
}
