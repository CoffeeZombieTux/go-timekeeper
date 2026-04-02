package handler

import (
	"testing"

	"go-timekeeper/internal/logger"
)

func TestNewHandlersPool(t *testing.T) {
	userSvc := &fakeUserHandlerService{}
	projectSvc := &fakeProjectService{}
	taskSvc := &fakeTaskService{}
	reportSvc := &fakeReportService{}
	timeRecordSvc := &fakeTimeRecordService{}
	log := logger.New("error", "json")

	pool := NewHandlersPool(userSvc, projectSvc, taskSvc, reportSvc, timeRecordSvc, log)
	if pool == nil {
		t.Fatal("handlers pool should not be nil")
	}
	if pool.User == nil || pool.Project == nil || pool.Task == nil || pool.Report == nil || pool.TimeRecord == nil {
		t.Fatal("all handlers should be initialized")
	}
	if pool.User.UserService != userSvc {
		t.Fatal("user service was not wired")
	}
	if pool.Project.projectService != projectSvc {
		t.Fatal("project service was not wired")
	}
	if pool.Task.taskService != taskSvc {
		t.Fatal("task service was not wired")
	}
	if pool.Report.reportService != reportSvc {
		t.Fatal("report service was not wired")
	}
	if pool.TimeRecord.timeRecordService != timeRecordSvc {
		t.Fatal("time record service was not wired")
	}
}
