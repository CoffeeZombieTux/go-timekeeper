package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go-timekeeper/internal/apperror"
	"go-timekeeper/internal/model"
	"go-timekeeper/internal/repository"
	"go-timekeeper/internal/uow"
	"time"

	"github.com/google/uuid"
)

// TimeRecordServiceInterface represents a time record service.
type TimeRecordServiceInterface interface {
	StartTask(ctx context.Context, task *model.Task, timezone string, unit *uow.UnitOfWork) error
	StopTask(ctx context.Context, task *model.Task, unit *uow.UnitOfWork) error
}

// TimeRecordService is a struct that implements the TimeRecordServiceInterface.
type TimeRecordService struct {
	timeRecordRepo repository.TimeRecordRepositoryInterface
}

// NewTimeRecordService creates a new TimeRecordService instance.
func NewTimeRecordService(timeRecordRepo repository.TimeRecordRepositoryInterface) TimeRecordServiceInterface {
	return &TimeRecordService{
		timeRecordRepo: timeRecordRepo,
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
		ID:        uuid.New(),
		UserID:    task.UserID,
		ProjectID: task.ProjectID,
		TaskID:    task.ID,
		WorkDate:  workDate,
		Timezone:  timezone,
		StartedAt: startedAt.UTC(),
	}

	if err = timeRecordService.timeRecordRepo.InsertActive(ctx, rec, tx); err != nil {
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
	if err = timeRecordService.timeRecordRepo.UpdateClosedRecord(ctx, model.ClosedTimeRecordInput{
		ID:           active.ID,
		UserID:       active.UserID,
		ProjectID:    active.ProjectID,
		TaskID:       active.TaskID,
		WorkDate:     first.WorkDate,
		Timezone:     active.Timezone,
		StartedAt:    first.StartedAt,
		EndedAt:      first.EndedAt,
		TotalMinutes: first.TotalMinutes,
	}, tx); err != nil {
		return err
	}

	for i := 1; i < len(daysOfWork); i++ {
		day := daysOfWork[i]
		if err = timeRecordService.timeRecordRepo.InsertClosedRecord(ctx, model.ClosedTimeRecordInput{
			ID:           uuid.New(),
			UserID:       active.UserID,
			ProjectID:    active.ProjectID,
			TaskID:       active.TaskID,
			WorkDate:     day.WorkDate,
			Timezone:     active.Timezone,
			StartedAt:    day.StartedAt,
			EndedAt:      day.EndedAt,
			TotalMinutes: day.TotalMinutes,
		}, tx); err != nil {
			return err
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
