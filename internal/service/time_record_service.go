package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go-timekeeper/internal/apperror"
	"go-timekeeper/internal/model"
	apimodel "go-timekeeper/internal/model/api"
	"go-timekeeper/internal/repository"
	"go-timekeeper/internal/uow"
	"time"

	"github.com/google/uuid"
)

// TimeRecordServiceInterface represents a time record service.
type TimeRecordServiceInterface interface {
	StartTask(ctx context.Context, task *model.Task, timezone string, unit *uow.UnitOfWork) error
	StopTask(ctx context.Context, task *model.Task, unit *uow.UnitOfWork) error
	CreateTimeRecord(ctx context.Context, req *apimodel.CreateTimeRecordRequest) (*model.TimeRecord, error)
	UpdateTimeRecord(ctx context.Context, req *apimodel.UpdateTimeRecordRequest) (*model.TimeRecord, error)
	DeleteTimeRecord(ctx context.Context, id uuid.UUID) error
}

// TimeRecordService is a struct that implements the TimeRecordServiceInterface.
type TimeRecordService struct {
	timeRecordRepo repository.TimeRecordRepositoryInterface
	uowManager     *uow.UnitOfWorkManager
}

// NewTimeRecordService creates a new TimeRecordService instance.
func NewTimeRecordService(
	timeRecordRepo repository.TimeRecordRepositoryInterface,
	uowManager *uow.UnitOfWorkManager,
) TimeRecordServiceInterface {
	return &TimeRecordService{
		timeRecordRepo: timeRecordRepo,
		uowManager:     uowManager,
	}
}

// StartTask starts a task.
func (timeRecordService *TimeRecordService) StartTask(
	ctx context.Context,
	task *model.Task,
	timezone string,
	unit *uow.UnitOfWork,
) error {
	tx, err := getTransaction(unit)
	if err != nil {
		return err
	}
	location, err := getLocation(timezone)
	if err != nil {
		return err
	}
	startedAt := time.Now().UTC()
	_, err = timeRecordService.timeRecordRepo.GetActiveByUserForUpdate(ctx, task.UserID, tx)
	if err == nil {
		return apperror.New(
			apperror.CodeDBDuplicateKeyCode,
			apperror.CodeDBDuplicateKeyMessage,
			fmt.Sprintf("active timer already exists for task %s", task.ID),
		)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	localStartedAt := startedAt.In(location)
	workDate := time.Date(
		localStartedAt.Year(),
		localStartedAt.Month(),
		localStartedAt.Day(),
		0, 0, 0, 0,
		time.UTC,
	)

	rec := model.TimeRecord{
		UserID:    task.UserID,
		ProjectID: task.ProjectID,
		TaskID:    task.ID,
		WorkDate:  workDate,
		Timezone:  timezone,
		StartedAt: startedAt.UTC(),
	}

	if _, err = timeRecordService.timeRecordRepo.Create(ctx, &rec, tx); err != nil {
		return err
	}
	return err
}

// StopTask stops a task.
func (timeRecordService *TimeRecordService) StopTask(ctx context.Context, task *model.Task, unit *uow.UnitOfWork) error {
	tx, err := getTransaction(unit)
	if err != nil {
		return err
	}

	active, err := timeRecordService.timeRecordRepo.GetActiveByUserForUpdate(ctx, task.UserID, tx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apperror.New(
				apperror.CodeNotFoundCode,
				apperror.CodeNotFoundMessage,
				"Current user does not have active working session",
			)
		}
		return err
	}

	loc, err := getLocation(active.Timezone)
	if err != nil {
		return err
	}

	stopAt := time.Now().UTC()
	if stopAt.Before(active.StartedAt) {
		return apperror.New(
			apperror.CodeValidationErrorCode,
			apperror.CodeValidationErrorMessage,
			fmt.Sprintf("stop time should be after start. Started at: %sm stop at: %s", active.StartedAt, stopAt),
		)
	}

	daysOfWork := splitByMidnight(active.StartedAt, stopAt, loc)
	if len(daysOfWork) == 0 {
		return apperror.New(
			apperror.CodeValidationErrorCode,
			apperror.CodeValidationErrorMessage,
			fmt.Sprintf("invalid date range %s - %s", active.StartedAt, stopAt),
		)
	}

	first := daysOfWork[0]
	active.WorkDate = first.WorkDate
	active.StartedAt = first.StartedAt
	active.EndedAt = &first.EndedAt
	active.TotalMinutes = &first.TotalMinutes
	if _, err = timeRecordService.timeRecordRepo.Update(ctx, active, tx); err != nil {
		return err
	}

	for i := 1; i < len(daysOfWork); i++ {
		day := daysOfWork[i]
		timeRecordToInsert := model.TimeRecord{
			UserID:       active.UserID,
			ProjectID:    active.ProjectID,
			TaskID:       active.TaskID,
			WorkDate:     day.WorkDate,
			Timezone:     active.Timezone,
			StartedAt:    day.StartedAt,
			EndedAt:      &day.EndedAt,
			TotalMinutes: &day.TotalMinutes,
		}

		if _, err = timeRecordService.timeRecordRepo.Create(ctx, &timeRecordToInsert, tx); err != nil {
			return err
		}
	}
	return nil
}

// CreateTimeRecord services manual time record creation.
func (timeRecordService *TimeRecordService) CreateTimeRecord(ctx context.Context, req *apimodel.CreateTimeRecordRequest) (*model.TimeRecord, error) {
	userId, err := getUserIdFromRequest(ctx)
	if err != nil {
		return nil, err
	}
	loc, err := getLocation(req.WorkTimezone)
	if err != nil {
		return nil, err
	}

	if req.EndTime.Before(req.StartTime) {
		return nil, apperror.New(
			apperror.CodeValidationErrorCode,
			apperror.CodeValidationErrorMessage,
			fmt.Sprintf("stop time should be after start. Started at: %sm stop at: %s", req.StartTime, req.EndTime),
		)
	}

	startLocal := req.StartTime.In(loc)
	endLocal := req.EndTime.In(loc)

	if !sameLocalDate(startLocal, endLocal, loc) {
		return nil, apperror.New(
			apperror.CodeValidationErrorCode,
			apperror.CodeValidationErrorMessage,
			fmt.Sprintf(
				"start and end time should be on the same day. Started at: %sm stop at: %s. Location: %s",
				startLocal,
				endLocal,
				req.WorkTimezone,
			),
		)
	}

	duration := endLocal.Sub(startLocal)
	minutes := durationToMinutes(duration)

	timeRecord := &model.TimeRecord{
		UserID:       userId,
		ProjectID:    req.ProjectID,
		TaskID:       req.TaskID,
		WorkDate:     req.WorkDate,
		Timezone:     req.WorkTimezone,
		StartedAt:    req.StartTime,
		EndedAt:      &req.EndTime,
		TotalMinutes: &minutes,
	}
	if err = timeRecordService.validateTimeRecordsConflict(ctx, userId, req.TaskID, *timeRecord); err != nil {
		return nil, err
	}
	err = uow.WithUnitOfWork(ctx, timeRecordService.uowManager, func(unit *uow.UnitOfWork) error {
		timeRecord, err = timeRecordService.timeRecordRepo.Create(ctx, timeRecord, unit.GetTransaction())
		return err
	})
	if err != nil {
		return nil, err
	}
	return timeRecord, nil
}

// UpdateTimeRecord services manual time record edit.
func (timeRecordService *TimeRecordService) UpdateTimeRecord(ctx context.Context, req *apimodel.UpdateTimeRecordRequest) (*model.TimeRecord, error) {
	userId, err := getUserIdFromRequest(ctx)
	if err != nil {
		return nil, err
	}
	var timeRecord *model.TimeRecord
	err = uow.WithUnitOfWork(ctx, timeRecordService.uowManager, func(unit *uow.UnitOfWork) error {
		tx := unit.GetTransaction()
		timeRecord, err = timeRecordService.timeRecordRepo.GetForUpdate(ctx, req.ID, tx)
		if err != nil {
			return err
		}
		if err = checkTimeRecordUserAccess(userId, *timeRecord); err != nil {
			return err
		}
		timeRecord.ProjectID = req.ProjectID
		timeRecord.TaskID = req.TaskID
		timeRecord.WorkDate = req.WorkDate
		timeRecord.Timezone = req.WorkTimezone
		timeRecord.StartedAt = req.StartTime
		timeRecord.EndedAt = &req.EndTime
		if err = timeRecordService.validateTimeRecordsConflict(ctx, userId, req.TaskID, *timeRecord); err != nil {
			return err
		}

		timeRecord, err = timeRecordService.timeRecordRepo.Update(ctx, timeRecord, tx)
		return err
	})
	if err != nil {
		return nil, err
	}
	return timeRecord, nil
}

// DeleteTimeRecord services manual time record deletion.
func (timeRecordService *TimeRecordService) DeleteTimeRecord(ctx context.Context, id uuid.UUID) error {
	userId, err := getUserIdFromRequest(ctx)
	if err != nil {
		return err
	}
	return uow.WithUnitOfWork(ctx, timeRecordService.uowManager, func(unit *uow.UnitOfWork) error {
		tx := unit.GetTransaction()
		timeRecord, err := timeRecordService.timeRecordRepo.GetForUpdate(ctx, id, tx)
		if err != nil {
			return err
		}
		if err = checkTimeRecordUserAccess(userId, *timeRecord); err != nil {
			return err
		}
		return timeRecordService.timeRecordRepo.Delete(ctx, timeRecord, tx)
	})
}

// validateTimeRecordsConflict checks if the time record conflicts with existing time records.
func (timeRecordService *TimeRecordService) validateTimeRecordsConflict(
	ctx context.Context,
	userID uuid.UUID,
	taskID uuid.UUID,
	timeRecordCandidate model.TimeRecord,
) error {
	if timeRecordCandidate.EndedAt == nil {
		return nil
	}
	candidateStart := timeRecordCandidate.StartedAt.UTC()
	candidateEnd := timeRecordCandidate.EndedAt.UTC()
	if !candidateEnd.After(candidateStart) {
		return apperror.New(
			apperror.CodeValidationErrorCode,
			apperror.CodeValidationErrorMessage,
			fmt.Sprintf("stop time should be after start. Started at: %s stop at: %s", candidateStart, candidateEnd),
		)
	}

	workDate := normalizeDate(timeRecordCandidate.WorkDate, time.UTC)
	taskDaySessions, err := timeRecordService.timeRecordRepo.GetTaskDayClosedRecords(ctx, userID, taskID, workDate)
	if err != nil {
		return err
	}

	for _, session := range taskDaySessions {
		if session.ID == timeRecordCandidate.ID {
			continue
		}

		sessionStart := session.StartedAt.UTC()
		sessionEnd := time.Now().UTC()
		if session.EndedAt != nil {
			sessionEnd = session.EndedAt.UTC()
		}
		overlaps := candidateStart.Before(sessionEnd) && sessionStart.Before(candidateEnd)
		if overlaps {
			return apperror.New(
				apperror.CodeValidationErrorCode,
				apperror.CodeValidationErrorMessage,
				fmt.Sprintf(
					"time record conflict with existing session %s on %s for task %s",
					session.ID,
					workDate.Format("2006-01-02"),
					taskID,
				),
			)
		}
	}
	return nil
}

// dayRecord represents a record of a day.
type dayRecord struct {
	WorkDate     time.Time
	StartedAt    time.Time
	EndedAt      time.Time
	TotalMinutes int
	Timezone     string
}

// splitByMidnight splits a time range into records of a day.
func splitByMidnight(startUTC, endUTC time.Time, loc *time.Location) []dayRecord {
	if endUTC.Before(startUTC) {
		return nil
	}

	startLocal := startUTC.In(loc)
	endLocal := endUTC.In(loc)

	result := make([]dayRecord, 0, 4)
	currentStartLocal := startLocal

	for {
		currentDate := normalizeDate(currentStartLocal, loc)

		var currentEndLocal time.Time
		if sameLocalDate(currentStartLocal, endLocal, loc) {
			currentEndLocal = endLocal
		} else {
			currentEndLocal = nextMidnight(currentStartLocal, loc)
		}

		duration := currentEndLocal.Sub(currentStartLocal)
		minutes := durationToMinutes(duration)

		result = append(result, dayRecord{
			WorkDate:     currentDate,
			StartedAt:    currentStartLocal.UTC(),
			EndedAt:      currentEndLocal.UTC(),
			TotalMinutes: minutes,
		})

		if !currentEndLocal.Before(endLocal) {
			break
		}

		currentStartLocal = currentEndLocal
	}
	return result
}

// normalizeDate normalizes a time.Time to the beginning of the day.
func normalizeDate(t time.Time, loc *time.Location) time.Time {
	lt := t.In(loc)
	y, m, d := lt.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// nextMidnight returns the time of the next midnight.
func nextMidnight(t time.Time, loc *time.Location) time.Time {
	lt := t.In(loc)
	y, m, d := lt.Date()
	return time.Date(y, m, d+1, 0, 0, 0, 0, loc)
}

// sameLocalDate checks if two times are on the same day.
func sameLocalDate(a, b time.Time, loc *time.Location) bool {
	la := a.In(loc)
	lb := b.In(loc)

	ay, am, ad := la.Date()
	by, bm, bd := lb.Date()

	return ay == by && am == bm && ad == bd
}

// durationToMinutes converts a time.Duration to minutes.
func durationToMinutes(d time.Duration) int {
	if d < 0 {
		return 0
	}
	return int(d / time.Minute)
}

// getTransaction returns the transaction from the unit of work.
func getTransaction(unit *uow.UnitOfWork) (*sql.Tx, error) {
	if unit == nil || unit.GetTransaction() == nil {
		return nil, apperror.New(
			apperror.CodeInternalErrorCode,
			apperror.CodeInternalErrorMessage,
			"missing transaction",
		)
	}
	return unit.GetTransaction(), nil
}

// getLocation returns the location from the timezone.
func getLocation(timezone string) (*time.Location, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("failed to get location from timezone %s", timezone)
	}
	return loc, nil
}

// checkTimeRecordUserAccess checks if the user is allowed to access the task.
func checkTimeRecordUserAccess(userId uuid.UUID, timeRecord model.TimeRecord) error {
	if userId == timeRecord.UserID {
		return nil
	}
	return apperror.New(apperror.CodeUnauthorizedCode, apperror.CodeUnauthorizedMessage, "User not authenticated")
}
