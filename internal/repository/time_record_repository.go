package repository

import (
	"context"
	"database/sql"
	"go-timekeeper/internal/logger"
	"go-timekeeper/internal/model"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// TimeRecordRepositoryInterface represents a time record repository.
type TimeRecordRepositoryInterface interface {
	GetForUpdate(ctx context.Context, id uuid.UUID, tx *sql.Tx) (*model.TimeRecord, error)
	GetActiveByUserForUpdate(ctx context.Context, userID uuid.UUID, tx *sql.Tx) (*model.TimeRecord, error)
	GetListByTaskForUpdate(ctx context.Context, taskID uuid.UUID, tx *sql.Tx) ([]*model.TimeRecord, error)
	Create(ctx context.Context, rec *model.TimeRecord, tx *sql.Tx) (*model.TimeRecord, error)
	Update(ctx context.Context, rec *model.TimeRecord, tx *sql.Tx) (*model.TimeRecord, error)
	Delete(ctx context.Context, rec *model.TimeRecord, tx *sql.Tx) error

	GetTaskReportRows(
		ctx context.Context,
		userID, taskID uuid.UUID,
		fromDate, toDate time.Time,
	) ([]*model.TimeRecordReportRow, error)
	GetTaskDayClosedRecords(
		ctx context.Context,
		userID, taskID uuid.UUID,
		workDate time.Time,
	) ([]*model.TimeRecord, error)

	GetProjectReportRows(
		ctx context.Context,
		userID, projectID uuid.UUID,
		taskIDs *[]uuid.UUID,
		fromDate, toDate time.Time,
	) ([]*model.TimeRecordReportRow, error)

	GetGeneralReportRows(
		ctx context.Context,
		userID uuid.UUID,
		projectIDs *[]uuid.UUID,
		fromDate, toDate time.Time,
	) ([]*model.TimeRecordReportRow, error)
}

// GetTaskDayClosedRecords returns sessions for a user/task on one work date.
func (timeRecordRepo *TimeRecordRepository) GetTaskDayClosedRecords(
	ctx context.Context,
	userID, taskID uuid.UUID,
	workDate time.Time,
) ([]*model.TimeRecord, error) {
	const q = `
		SELECT
			id, user_id, project_id, task_id,
			work_date, work_timezone, started_at, ended_at, total_minutes,
			created_at, updated_at
		FROM time_record
		WHERE user_id = $1
		  AND task_id = $2
		  AND work_date = $3
		ORDER BY started_at, id
	`
	rows, err := timeRecordRepo.db.QueryContext(ctx, q, userID, taskID, workDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]*model.TimeRecord, 0)
	for rows.Next() {
		var rec model.TimeRecord
		var endedAt *time.Time
		var totalMinutes *int
		if err = rows.Scan(
			&rec.ID,
			&rec.UserID,
			&rec.ProjectID,
			&rec.TaskID,
			&rec.WorkDate,
			&rec.Timezone,
			&rec.StartedAt,
			&endedAt,
			&totalMinutes,
			&rec.CreatedAt,
			&rec.UpdatedAt,
		); err != nil {
			return nil, err
		}
		rec.EndedAt = endedAt
		rec.TotalMinutes = totalMinutes
		result = append(result, &rec)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// TimeRecordRepository is a struct that implements the TimeRecordRepositoryInterface.
type TimeRecordRepository struct {
	db     *sql.DB
	logger *logger.Logger
}

// NewTimeRecordRepository creates a new TimeRecordRepository instance.
func NewTimeRecordRepository(db *sql.DB, logger *logger.Logger) TimeRecordRepositoryInterface {
	return &TimeRecordRepository{
		db:     db,
		logger: logger,
	}
}

// GetForUpdate returns a time record by ID.
func (timeRecordRepo *TimeRecordRepository) GetForUpdate(ctx context.Context, id uuid.UUID, tx *sql.Tx) (*model.TimeRecord, error) {
	const q = `
		SELECT
			id, user_id, project_id, task_id,
			work_date, work_timezone, started_at, ended_at, total_minutes,
			created_at, updated_at
		FROM time_record
		WHERE id = $1
		FOR UPDATE
	`
	var rec model.TimeRecord
	err := tx.QueryRowContext(ctx, q, id).Scan(
		&rec.ID,
		&rec.UserID,
		&rec.ProjectID,
		&rec.TaskID,
		&rec.WorkDate,
		&rec.Timezone,
		&rec.StartedAt,
		&rec.EndedAt,
		&rec.TotalMinutes,
		&rec.CreatedAt,
		&rec.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// GetActiveByUserForUpdate returns the active time record for a user.
func (timeRecordRepo *TimeRecordRepository) GetActiveByUserForUpdate(
	ctx context.Context,
	userID uuid.UUID,
	tx *sql.Tx,
) (*model.TimeRecord, error) {
	const q = `
		SELECT
			id, user_id, project_id, task_id,
			work_date, work_timezone, started_at, ended_at, total_minutes,
			created_at, updated_at
		FROM time_record
		WHERE user_id = $1
		  AND ended_at IS NULL
		FOR UPDATE
	`
	var rec model.TimeRecord
	var endedAt *time.Time
	var totalMinutes *int

	err := tx.QueryRowContext(ctx, q, userID).Scan(
		&rec.ID,
		&rec.UserID,
		&rec.ProjectID,
		&rec.TaskID,
		&rec.WorkDate,
		&rec.Timezone,
		&rec.StartedAt,
		&endedAt,
		&totalMinutes,
		&rec.CreatedAt,
		&rec.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	rec.EndedAt = endedAt
	rec.TotalMinutes = totalMinutes
	return &rec, nil
}

// GetListByTaskForUpdate loads all TimeRecord rows assigned to a task.
func (timeRecordRepo *TimeRecordRepository) GetListByTaskForUpdate(
	ctx context.Context,
	taskID uuid.UUID,
	tx *sql.Tx,
) ([]*model.TimeRecord, error) {
	const q = `
		SELECT
			id,
			project_id,
			task_id,
			work_date,
			work_timezone,
			total_minutes
		FROM time_record
		WHERE task_id = $1
		FOR UPDATE
	`

	rows, err := tx.QueryContext(ctx, q, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]*model.TimeRecord, 0)
	for rows.Next() {
		var row model.TimeRecord
		if err := rows.Scan(&row.ID, &row.ProjectID, &row.TaskID, &row.WorkDate, &row.Timezone, &row.TotalMinutes); err != nil {
			return nil, err
		}
		result = append(result, &row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// Create inserts a time record into the database.
func (timeRecordRepo *TimeRecordRepository) Create(
	ctx context.Context,
	rec *model.TimeRecord,
	tx *sql.Tx,
) (*model.TimeRecord, error) {
	const q = `
		INSERT INTO time_record (
			id, user_id, project_id, task_id,
			work_date, work_timezone, started_at, ended_at, total_minutes,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING id
	`
	err := tx.QueryRowContext(
		ctx,
		q,
		uuid.New(),
		rec.UserID,
		rec.ProjectID,
		rec.TaskID,
		rec.WorkDate,
		rec.Timezone,
		rec.StartedAt,
		rec.EndedAt,
		rec.TotalMinutes,
	).Scan(&rec.ProjectID)
	if err != nil {
		return nil, err
	}
	return rec, nil
}

// Update updates a time record in the database.
func (timeRecordRepo *TimeRecordRepository) Update(
	ctx context.Context,
	rec *model.TimeRecord,
	tx *sql.Tx,
) (*model.TimeRecord, error) {
	const q = `
		UPDATE time_record
		SET
			user_id = $2,
			project_id = $3,
			task_id = $4,
			work_date = $5,
			work_timezone = $6,
			started_at = $7,
			ended_at = $8,
			total_minutes = $9,
			updated_at = NOW()
		WHERE id = $1
		RETURNING *
	`
	err := tx.QueryRowContext(
		ctx,
		q,
		rec.ID,
		rec.ProjectID,
		rec.TaskID,
		rec.WorkDate,
		rec.Timezone,
		rec.StartedAt,
		rec.EndedAt,
		rec.TotalMinutes,
	).Scan(
		&rec.ProjectID,
		&rec.UserID,
		&rec.TaskID,
		&rec.WorkDate,
		&rec.Timezone,
		&rec.StartedAt,
		&rec.EndedAt,
		&rec.TotalMinutes,
	)
	if err != nil {
		return nil, err
	}
	return rec, nil
}

// Delete deletes a time record from the database.
func (timeRecordRepo *TimeRecordRepository) Delete(ctx context.Context, rec *model.TimeRecord, tx *sql.Tx) error {
	const q = `
		DELETE FROM time_record
		WHERE id = $1
	`
	_, err := tx.ExecContext(ctx, q, rec.ID)
	return err
}

// GetTaskReportRows returns a list of time records for a given user, task, and date range.
func (timeRecordRepo *TimeRecordRepository) GetTaskReportRows(
	ctx context.Context,
	userID, taskID uuid.UUID,
	fromDate, toDate time.Time,
) ([]*model.TimeRecordReportRow, error) {
	const q = `
		SELECT
			id,
			project_id,
			task_id,
			work_date,
			work_timezone,
			total_minutes
		FROM time_record
		WHERE user_id = $1
		  AND task_id = $2
		  AND ended_at IS NOT NULL
		  AND work_date BETWEEN $3 AND $4
		  AND total_minutes IS NOT NULL
		  AND total_minutes > 0
		ORDER BY work_date, id
	`

	rows, err := timeRecordRepo.db.QueryContext(ctx, q, userID, taskID, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	return getTimeRecordsFromDbRows(rows)
}

// GetProjectReportRows returns a list of time records for a given user, project, and date range.
func (timeRecordRepo *TimeRecordRepository) GetProjectReportRows(
	ctx context.Context,
	userID, projectID uuid.UUID,
	taskIDs *[]uuid.UUID,
	fromDate, toDate time.Time,
) ([]*model.TimeRecordReportRow, error) {
	args := []interface{}{userID, projectID, fromDate, toDate}
	q := `
		SELECT
			id,
			project_id,
			task_id,
			work_date,
			work_timezone,
			total_minutes
		FROM time_record
		WHERE user_id = $1
		  AND project_id = $2
		  AND work_date BETWEEN $3 AND $4
		  AND ended_at IS NOT NULL
		  AND total_minutes IS NOT NULL
		  AND total_minutes > 0`

	if taskIDs != nil {
		taskIDValues := make([]string, 0, len(*taskIDs))
		for _, taskID := range *taskIDs {
			taskIDValues = append(taskIDValues, taskID.String())
		}

		q += `
		  AND task_id = ANY($5)`
		args = append(args, pq.Array(taskIDValues))
	}

	q += `
		ORDER BY task_id, work_date, id
	`

	rows, err := timeRecordRepo.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	return getTimeRecordsFromDbRows(rows)
}

// GetGeneralReportRows returns a list of time records for a given user, project, and date range.
func (timeRecordRepo *TimeRecordRepository) GetGeneralReportRows(
	ctx context.Context,
	userID uuid.UUID,
	projectIDs *[]uuid.UUID,
	fromDate, toDate time.Time,
) ([]*model.TimeRecordReportRow, error) {
	args := []interface{}{userID, fromDate, toDate}
	q := `
		SELECT
			id,
			project_id,
			task_id,
			work_date,
			work_timezone,
			total_minutes
		FROM time_record
		WHERE user_id = $1
		  AND work_date BETWEEN $2 AND $3
		  AND ended_at IS NOT NULL
		  AND total_minutes IS NOT NULL
		  AND total_minutes > 0`

	if projectIDs != nil {
		projectIDValues := make([]string, 0, len(*projectIDs))
		for _, projectID := range *projectIDs {
			projectIDValues = append(projectIDValues, projectID.String())
		}

		q += `
		  AND project_id = ANY($4)`
		args = append(args, pq.Array(projectIDValues))
	}

	q += `
		ORDER BY project_id, task_id, work_date, id
	`

	rows, err := timeRecordRepo.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	return getTimeRecordsFromDbRows(rows)
}

// getTimeRecordsFromDbRows converts a slice of sql.Rows into a slice of TimeRecordReportRow.
func getTimeRecordsFromDbRows(rows *sql.Rows) ([]*model.TimeRecordReportRow, error) {
	defer rows.Close()

	result := make([]*model.TimeRecordReportRow, 0)
	for rows.Next() {
		var row model.TimeRecordReportRow
		if err := rows.Scan(&row.ID, &row.ProjectID, &row.TaskID, &row.WorkDate, &row.WorkTimezone, &row.TotalMinutes); err != nil {
			return nil, err
		}
		result = append(result, &row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
