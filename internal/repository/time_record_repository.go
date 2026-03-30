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
	InsertActive(ctx context.Context, rec model.TimeRecord, tx *sql.Tx) error
	GetActiveByUserForUpdate(ctx context.Context, userID uuid.UUID, tx *sql.Tx) (*model.TimeRecord, error)
	GetListByTaskForUpdate(ctx context.Context, taskID uuid.UUID, tx *sql.Tx) ([]*model.TimeRecord, error)
	UpdateProjectReference(ctx context.Context, rec model.TimeRecord, tx *sql.Tx) error
	UpdateClosedRecord(ctx context.Context, rec model.ClosedTimeRecordInput, tx *sql.Tx) error
	InsertClosedRecord(ctx context.Context, rec model.ClosedTimeRecordInput, tx *sql.Tx) error

	GetTaskReportRows(
		ctx context.Context,
		userID, taskID uuid.UUID,
		fromDate, toDate time.Time,
	) ([]*model.TimeRecordReportRow, error)

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

// InsertActive inserts a new active time record into the database.
func (timeRecordRepo *TimeRecordRepository) InsertActive(
	ctx context.Context,
	rec model.TimeRecord,
	tx *sql.Tx,
) error {
	const q = `
		INSERT INTO time_record (
			id, user_id, project_id, task_id,
			work_date, work_timezone, started_at, ended_at, total_minutes,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULL, NULL, NOW(), NOW())
	`
	_, err := tx.ExecContext(ctx, q,
		rec.ID,
		rec.UserID,
		rec.ProjectID,
		rec.TaskID,
		rec.WorkDate,
		rec.Timezone,
		rec.StartedAt,
	)
	return err
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

// UpdateClosedRecord updates a closed time record in the database.
func (timeRecordRepo *TimeRecordRepository) UpdateClosedRecord(
	ctx context.Context,
	rec model.ClosedTimeRecordInput,
	tx *sql.Tx,
) error {
	const q = `
		UPDATE time_record
		SET
			work_date = $2,
			started_at = $3,
			ended_at = $4,
			total_minutes = $5,
			updated_at = NOW()
		WHERE id = $1
	`
	_, err := tx.ExecContext(ctx, q,
		rec.ID,
		rec.WorkDate,
		rec.StartedAt,
		rec.EndedAt,
		rec.TotalMinutes,
	)
	if err != nil {
		return err
	}
	return nil
}

// InsertClosedRecord inserts a closed time record into the database.
func (timeRecordRepo *TimeRecordRepository) InsertClosedRecord(
	ctx context.Context,
	rec model.ClosedTimeRecordInput,
	tx *sql.Tx,
) error {
	const q = `
		INSERT INTO time_record (
			id, user_id, project_id, task_id,
			work_date, work_timezone, ended_at, total_minutes,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
	`
	_, err := tx.ExecContext(ctx, q,
		rec.ID,
		rec.UserID,
		rec.ProjectID,
		rec.TaskID,
		rec.WorkDate,
		rec.Timezone,
		rec.StartedAt,
		rec.EndedAt,
		rec.TotalMinutes,
	)
	return err
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

func (timeRecordRepo *TimeRecordRepository) UpdateProjectReference(
	ctx context.Context,
	entity model.TimeRecord,
	tx *sql.Tx,
) error {
	const q = `
		UPDATE time_record
		SET
			project_id = $2,
			updated_at = NOW()
		WHERE id = $1
	`
	_, err := tx.ExecContext(ctx, q,
		entity.ID,
		entity.ProjectID,
	)
	if err != nil {
		return err
	}
	return nil
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
	//defer rows.Close()
	//
	//result := make([]*model.TimeRecordReportRow, 0)
	//for rows.Next() {
	//	var row model.TimeRecordReportRow
	//	if err := rows.Scan(&row.ID, &row.ProjectID, &row.TaskID, &row.WorkDate, &row.WorkTimezone, &row.TotalMinutes); err != nil {
	//		return nil, err
	//	}
	//	result = append(result, &row)
	//}
	//
	//if err := rows.Err(); err != nil {
	//	return nil, err
	//}
	//
	//return result, nil
}

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
