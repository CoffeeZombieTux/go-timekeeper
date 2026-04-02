package service

import (
	"context"
	"go-timekeeper/internal/model"
	apimodel "go-timekeeper/internal/model/api"
	"go-timekeeper/internal/repository"

	"github.com/google/uuid"
)

// ReportServiceInterface represents a report service.
type ReportServiceInterface interface {
	GeneralReport(ctx context.Context, req *apimodel.GeneralReportRequest) (*apimodel.GeneralReportResponse, error)
	ProjectReport(ctx context.Context, req *apimodel.ProjectReportRequest) (*apimodel.ProjectReportResponse, error)
	TaskReport(ctx context.Context, req *apimodel.TaskReportRequest) (*apimodel.TaskReportResponse, error)
}

// ReportService is a struct that implements the ReportServiceInterface.
type ReportService struct {
	timeRecordRepo repository.TimeRecordRepositoryInterface
	taskRepo       repository.TaskRepositoryInterface
	projectRepo    repository.ProjectRepositoryInterface
}

// NewReportService creates a new ReportService instance.
func NewReportService(
	timeRecordRepo repository.TimeRecordRepositoryInterface,
	taskRepo repository.TaskRepositoryInterface,
	projectRepo repository.ProjectRepositoryInterface,
) ReportServiceInterface {
	return &ReportService{
		timeRecordRepo: timeRecordRepo,
		taskRepo:       taskRepo,
		projectRepo:    projectRepo,
	}
}

// daySessions is a map of working days to records.
type daySessions map[string][]*model.TimeRecordReportRow

// taskDays is a map of tasks to daySessions.
type taskDays map[uuid.UUID]daySessions

// projectTasks is a map of projects to taskDays.
type projectTasks map[uuid.UUID]taskDays

// GeneralReport generates a general report for the given user. Filter by project IDs and time range.
func (reportService *ReportService) GeneralReport(
	ctx context.Context,
	req *apimodel.GeneralReportRequest,
) (*apimodel.GeneralReportResponse, error) {
	// Validate time range params
	err := req.TimeRange.ValidateTimeRangeParams()
	if err != nil {
		return nil, err
	}

	userId, err := getUserIdFromRequest(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := reportService.timeRecordRepo.GetGeneralReportRows(
		ctx,
		userId,
		req.Projects,
		req.TimeRange.FromDate,
		req.TimeRange.ToDate,
	)
	if err != nil {
		return nil, err
	}

	// Make 3 dimension matrix (map of maps of maps) of projects + tasks + working days to records
	// [project1][task1][02.03.2026] => [sessions]
	// [project1][task1][03.03.2026] => [sessions]
	// [project2][task2][02.03.2026] => [sessions]
	// [project2][task2][03.03.2026] => [sessions]
	grouped := make(projectTasks, len(rows))
	for _, row := range rows {
		dateKey := row.WorkDate.Format("2006-01-02")

		// init inner map for project if missing
		if grouped[row.ProjectID] == nil {
			grouped[row.ProjectID] = make(taskDays, 10)
		}

		// init inner map for task if missing
		if grouped[row.ProjectID][row.TaskID] == nil {
			grouped[row.ProjectID][row.TaskID] = make(daySessions, len(rows)/10)
		}

		grouped[row.ProjectID][row.TaskID][dateKey] = append(
			grouped[row.ProjectID][row.TaskID][dateKey],
			row,
		)
	}

	projectsResponse := make([]*apimodel.ProjectReportResponse, 0, len(grouped))
	for projectID, tasks := range grouped {
		tasksResponse := make([]*apimodel.TaskReportResponse, 0, len(tasks))
		for taskID, days := range tasks {
			taskDaysResponse := make([]*apimodel.DayReportResponse, 0, len(days))
			for workDate, day := range days {
				taskDaysResponse = append(taskDaysResponse, reportService.getDaySessionsReportFromRows(workDate, day))
			}
			taskToAppend, err := reportService.getTaskReportResponseFromDaySessions(ctx, taskID, taskDaysResponse)
			if err != nil {
				return nil, err
			}
			tasksResponse = append(tasksResponse, taskToAppend)
		}
		projectToAppend, err := reportService.getProjectReportResponseFromTasks(ctx, projectID, tasksResponse)
		if err != nil {
			return nil, err
		}
		projectsResponse = append(projectsResponse, projectToAppend)
	}

	generalReport := reportService.getGeneralReportResponseFromProjects(projectsResponse)
	generalReport.TimeRange = req.TimeRange
	return generalReport, nil
}

// ProjectReport generates a project report for the given project. Filter by tasks ids and time range.
func (reportService *ReportService) ProjectReport(
	ctx context.Context,
	req *apimodel.ProjectReportRequest,
) (*apimodel.ProjectReportResponse, error) {
	// Validate time range params
	err := req.TimeRange.ValidateTimeRangeParams()
	if err != nil {
		return nil, err
	}

	userId, err := getUserIdFromRequest(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := reportService.timeRecordRepo.GetProjectReportRows(
		ctx,
		userId,
		req.ProjectID,
		req.Tasks,
		req.TimeRange.FromDate,
		req.TimeRange.ToDate,
	)
	if err != nil {
		return nil, err
	}

	// Make matrix (map of maps) of tasks + working days to records
	// [task1][02.03.2026] => [sessions]
	// [task1][03.03.2026] => [sessions]
	// [task2][02.03.2026] => [sessions]
	// [task2][03.03.2026] => [sessions]
	grouped := make(taskDays, len(rows))
	for _, row := range rows {
		dateKey := row.WorkDate.Format("2006-01-02")

		// init inner map for task if missing
		if grouped[row.TaskID] == nil {
			grouped[row.TaskID] = make(map[string][]*model.TimeRecordReportRow)
		}

		grouped[row.TaskID][dateKey] = append(
			grouped[row.TaskID][dateKey],
			row,
		)
	}

	tasks := make([]*apimodel.TaskReportResponse, 0, len(grouped))
	for taskID, days := range grouped {
		taskDays := make([]*apimodel.DayReportResponse, 0, len(days))
		for workDate, day := range days {
			taskDays = append(taskDays, reportService.getDaySessionsReportFromRows(workDate, day))
		}
		taskToAppend, err := reportService.getTaskReportResponseFromDaySessions(ctx, taskID, taskDays)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, taskToAppend)
	}

	projectReport, err := reportService.getProjectReportResponseFromTasks(ctx, req.ProjectID, tasks)
	if err != nil {
		return nil, err
	}
	projectReport.TimeRange = req.TimeRange
	return projectReport, nil
}

// TaskReport generates a task report for the given task. Filter by time range.
func (reportService *ReportService) TaskReport(
	ctx context.Context,
	req *apimodel.TaskReportRequest,
) (*apimodel.TaskReportResponse, error) {
	// Validate time range params
	err := req.TimeRange.ValidateTimeRangeParams()
	if err != nil {
		return nil, err
	}

	userId, err := getUserIdFromRequest(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := reportService.timeRecordRepo.GetTaskReportRows(
		ctx,
		userId,
		req.TaskID,
		req.TimeRange.FromDate,
		req.TimeRange.ToDate,
	)
	if err != nil {
		return nil, err
	}

	// Make map of working days to records
	// 02.03.2026 => [sessions]
	// 03.03.2026 => [sessions]
	grouped := make(daySessions, len(rows))
	for _, row := range rows {
		dateKey := row.WorkDate.Format("2006-01-02")
		grouped[dateKey] = append(grouped[dateKey], row)
	}

	var days []*apimodel.DayReportResponse
	for workDate, records := range grouped {
		days = append(days, reportService.getDaySessionsReportFromRows(workDate, records))
	}

	taskReport, err := reportService.getTaskReportResponseFromDaySessions(ctx, req.TaskID, days)
	if err != nil {
		return nil, err
	}
	taskReport.TimeRange = req.TimeRange
	return taskReport, nil
}

// getDaySessionsReportFromRows converts a list of time record rows to a day report response.
func (reportService *ReportService) getDaySessionsReportFromRows(
	workingDay string,
	rows []*model.TimeRecordReportRow,
) *apimodel.DayReportResponse {
	return &apimodel.DayReportResponse{
		WorkingDate:  workingDay,
		WorkTimezone: rows[0].WorkTimezone,
		Sessions:     rows,
		TotalMinutes: SumBy(rows, func(r *model.TimeRecordReportRow) int { return r.TotalMinutes }),
	}
}

// getTaskReportResponseFromDaySessions converts a list of day report responses to a task report response.
func (reportService *ReportService) getTaskReportResponseFromDaySessions(
	ctx context.Context,
	taskId uuid.UUID,
	rows []*apimodel.DayReportResponse,
) (*apimodel.TaskReportResponse, error) {
	total := SumBy(rows, func(r *apimodel.DayReportResponse) int {
		return r.TotalMinutes
	})
	task, err := reportService.taskRepo.Get(ctx, taskId, nil)
	if err != nil {
		return nil, err
	}
	return &apimodel.TaskReportResponse{
		TaskId:       taskId,
		TaskName:     task.Name,
		DayReports:   rows,
		TotalMinutes: total,
	}, nil
}

// getProjectReportResponseFromTasks converts a list of task report responses to a project report response.
func (reportService *ReportService) getProjectReportResponseFromTasks(
	ctx context.Context,
	projectID uuid.UUID,
	rows []*apimodel.TaskReportResponse,
) (*apimodel.ProjectReportResponse, error) {
	total := SumBy(rows, func(r *apimodel.TaskReportResponse) int {
		return r.TotalMinutes
	})
	project, err := reportService.projectRepo.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &apimodel.ProjectReportResponse{
		ProjectID:    projectID,
		ProjectName:  project.Name,
		Tasks:        rows,
		TotalMinutes: total,
	}, nil
}

// getGeneralReportResponseFromProjects converts a list of project report responses to a general report response.
func (reportService *ReportService) getGeneralReportResponseFromProjects(
	rows []*apimodel.ProjectReportResponse,
) *apimodel.GeneralReportResponse {
	total := SumBy(rows, func(r *apimodel.ProjectReportResponse) int {
		return r.TotalMinutes
	})
	return &apimodel.GeneralReportResponse{
		Projects:     rows,
		TotalMinutes: total,
	}
}

// SumBy sums the values of a slice of items using the provided selector function.
func SumBy[T any](items []T, selector func(T) int) int {
	var total int
	for _, item := range items {
		total += selector(item)
	}
	return total
}
