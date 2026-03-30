package handler

import (
	"go-timekeeper/internal/logger"
	"go-timekeeper/internal/middleware"
	apimodel "go-timekeeper/internal/model/api"
	"go-timekeeper/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ReportHandler handles report-related requests.
type ReportHandler struct {
	reportService service.ReportServiceInterface
	logger        *logger.Logger
}

// NewReportHandler creates a new ReportHandler instance.
func NewReportHandler(reportService service.ReportServiceInterface, logger *logger.Logger) *ReportHandler {
	return &ReportHandler{
		reportService: reportService,
		logger:        logger,
	}
}

// GeneralReport handles general report requests.
func (reportHandler *ReportHandler) GeneralReport(ctx *gin.Context) {
	var req apimodel.GeneralReportRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}
	response, err := reportHandler.reportService.GeneralReport(ctx.Request.Context(), &req)
	if err != nil {
		userId, _ := middleware.UserIDFromContext(ctx.Request.Context())
		reportHandler.logger.WithError(err).WithFields(logger.Fields{
			"request_id": requestIDFromContext(ctx),
			"user_id":    userId.String(),
		}).Error(logger.LogMessageFailedToGetReport)
		status, code, message, details := mapDomainError(err)
		writeError(ctx, status, message, code, details)
		return
	}
	writeSuccess(ctx, http.StatusOK, "General report generated", response)
}

// ProjectReport handles project-specific report requests.
func (reportHandler *ReportHandler) ProjectReport(ctx *gin.Context) {
	var req apimodel.ProjectReportRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}
	response, err := reportHandler.reportService.ProjectReport(ctx.Request.Context(), &req)
	if err != nil {
		userId, _ := middleware.UserIDFromContext(ctx.Request.Context())
		reportHandler.logger.WithError(err).WithFields(logger.Fields{
			"request_id": requestIDFromContext(ctx),
			"user_id":    userId.String(),
		}).Error(logger.LogMessageFailedToGetReport)
		status, code, message, details := mapDomainError(err)
		writeError(ctx, status, message, code, details)
		return
	}
	writeSuccess(ctx, http.StatusOK, "Project report generated", response)
}

// TaskReport handles task-specific report requests.
func (reportHandler *ReportHandler) TaskReport(ctx *gin.Context) {
	var req apimodel.TaskReportRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}
	response, err := reportHandler.reportService.TaskReport(ctx.Request.Context(), &req)
	if err != nil {
		userId, _ := middleware.UserIDFromContext(ctx.Request.Context())
		reportHandler.logger.WithError(err).WithFields(logger.Fields{
			"request_id": requestIDFromContext(ctx),
			"user_id":    userId.String(),
		}).Error(logger.LogMessageFailedToGetReport)
		status, code, message, details := mapDomainError(err)
		writeError(ctx, status, message, code, details)
		return
	}
	writeSuccess(ctx, http.StatusOK, "Task report generated", response)
}
