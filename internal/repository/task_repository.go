package repository

import (
	"context"
	"database/sql"
	"fmt"
	"go-timekeeper/internal/apperror"
	"go-timekeeper/internal/logger"
	"go-timekeeper/internal/model"
	"go-timekeeper/internal/uow"

	"github.com/google/uuid"
)

// TaskRepositoryInterface represents a task repository.
type TaskRepositoryInterface interface {
	Save(ctx context.Context, task *model.Task, unit *uow.UnitOfWork) (*model.Task, error)
	Get(ctx context.Context, id uuid.UUID, unit *uow.UnitOfWork) (*model.Task, error)
	GetByProjectAndUserId(
		ctx context.Context,
		projectID uuid.UUID,
		userID uuid.UUID,
		isActive *bool,
		limit,
		offset int,
	) ([]*model.Task, error)
	CountByProjectAndUserId(
		ctx context.Context,
		projectID uuid.UUID,
		userID uuid.UUID,
		isActive *bool,
	) (int, error)
	Delete(ctx context.Context, task *model.Task, unit *uow.UnitOfWork) error
	getExecutor(unit *uow.UnitOfWork) (SQLExecutor, bool)
}

// TaskRepository represents a task repository.
type TaskRepository struct {
	db     *sql.DB
	logger *logger.Logger
}

// NewTaskRepository creates a new TaskRepository instance.
func NewTaskRepository(db *sql.DB, logger *logger.Logger) TaskRepositoryInterface {
	return &TaskRepository{
		db:     db,
		logger: logger,
	}
}

// Save saves a task to the database.
func (taskRepo TaskRepository) Save(ctx context.Context, task *model.Task, unit *uow.UnitOfWork) (*model.Task, error) {
	exec, _ := taskRepo.getExecutor(unit)

	if task.ID == uuid.Nil {
		query := `
			INSERT INTO task (user_id, project_id, name, status)
			VALUES ($1, $2, $3, $4)
			RETURNING *;
		`

		err := exec.QueryRowContext(
			ctx,
			query,
			task.UserID,
			task.ProjectID,
			task.Name,
			task.Status,
		).Scan(&task.ID, &task.UserID, &task.ProjectID, &task.Name, &task.Status, &task.CreatedAt, &task.UpdatedAt)
		if err != nil {
			return nil, err
		}
		return task, nil
	}
	query := `
		UPDATE task SET project_id = $2, name = $3, status = $4 , updated_at = NOW()
		WHERE id = $1 RETURNING project_id, name, status, updated_at;
	`
	err := exec.QueryRowContext(
		ctx,
		query,
		task.ID,
		task.ProjectID,
		task.Name,
		task.Status,
	).Scan(&task.ProjectID, &task.Name, &task.Status, &task.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return task, nil
}

// Get gets a task by ID.
func (taskRepo TaskRepository) Get(ctx context.Context, id uuid.UUID, unit *uow.UnitOfWork) (*model.Task, error) {
	exec, isTx := taskRepo.getExecutor(unit)
	query := `SELECT * FROM task WHERE id = $1`
	if isTx {
		query += ` FOR UPDATE`
	}
	var task model.Task
	err := exec.QueryRowContext(
		ctx,
		query,
		id,
	).Scan(&task.ID, &task.UserID, &task.ProjectID, &task.Name, &task.Status, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// GetByProjectAndUserId gets tasks by project ID, user ID, and status.
func (taskRepo TaskRepository) GetByProjectAndUserId(
	ctx context.Context,
	projectID uuid.UUID,
	userID uuid.UUID,
	isActive *bool,
	limit,
	offset int,
) ([]*model.Task, error) {
	query := `SELECT * FROM task WHERE project_id = $1 AND user_id = $2`
	args := []any{projectID, userID}

	if isActive != nil {
		if *isActive {
			query += ` AND status IN ($3, $4)`
			args = append(args, model.StatusCreated, model.StatusWorkingOn)
		} else {
			query += ` AND status = $3`
			args = append(args, model.StatusClosed)
		}
	}
	limitPlaceholder := len(args) + 1
	offsetPlaceholder := len(args) + 2
	query += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, limitPlaceholder, offsetPlaceholder)
	args = append(args, limit, offset)

	rows, err := taskRepo.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && taskRepo.logger != nil {
			taskRepo.logger.WithError(closeErr).Error(logger.LogMessageFailedToCloseRows)
		}
	}()
	var tasks []*model.Task
	for rows.Next() {
		var task model.Task
		if err := rows.Scan(
			&task.ID,
			&task.UserID,
			&task.ProjectID,
			&task.Name,
			&task.Status,
			&task.CreatedAt,
			&task.UpdatedAt,
		); err != nil {
			return nil, err
		}
		tasks = append(tasks, &task)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

// CountByProjectAndUserId counts tasks by project ID, user ID, and status.
func (taskRepo TaskRepository) CountByProjectAndUserId(
	ctx context.Context,
	projectID uuid.UUID,
	userID uuid.UUID,
	isActive *bool,
) (int, error) {
	var count int

	query := `SELECT COUNT(*) FROM task WHERE project_id = $1 AND user_id = $2`
	args := []any{projectID, userID}

	if isActive != nil {
		if *isActive {
			query += ` AND status IN ($3, $4)`
			args = append(args, model.StatusCreated, model.StatusWorkingOn)
		} else {
			query += ` AND status = $3`
			args = append(args, model.StatusClosed)
		}
	}

	err := taskRepo.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, apperror.FromDB(
			err,
			"Failed to count project tasks records", apperror.CodeInternalErrorCode,
		)
	}

	return count, nil
}

// Delete deletes a task from the database.
func (taskRepo TaskRepository) Delete(ctx context.Context, task *model.Task, unit *uow.UnitOfWork) error {
	exec, _ := taskRepo.getExecutor(unit)
	query := `DELETE FROM task WHERE id = $1`
	_, err := exec.ExecContext(ctx, query, task.ID)
	return err
}

// getExecutor returns the SQLExecutor to use for the current transaction or the database connection.
func (taskRepo TaskRepository) getExecutor(unit *uow.UnitOfWork) (SQLExecutor, bool) {
	if unit != nil && unit.GetTransaction() != nil {
		return unit.GetTransaction(), true
	}

	return taskRepo.db, false
}
