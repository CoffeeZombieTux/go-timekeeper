package repository

import (
	"context"
	"database/sql"
	"go-timekeeper/internal/logger"
	"go-timekeeper/internal/model"

	"github.com/google/uuid"
)

// ProjectRepositoryInterface represents a project repository.
type ProjectRepositoryInterface interface {
	Save(ctx context.Context, project *model.Project) (*model.Project, error)
	Get(ctx context.Context, id uuid.UUID) (*model.Project, error)
	GetUserProjects(ctx context.Context, userID uuid.UUID) ([]*model.Project, error)
	Delete(ctx context.Context, project *model.Project) error
}

// ProjectRepository is a struct that implements the ProjectRepositoryInterface.
type ProjectRepository struct {
	db     *sql.DB
	logger *logger.Logger
}

// NewProjectRepository creates a new ProjectRepository instance.
func NewProjectRepository(db *sql.DB, logger *logger.Logger) ProjectRepositoryInterface {
	return &ProjectRepository{
		db:     db,
		logger: logger,
	}
}

// Save saves a project to the database.
func (projectRepo ProjectRepository) Save(ctx context.Context, project *model.Project) (*model.Project, error) {
	if project.ID != uuid.Nil {
		query := `
			UPDATE project
			SET name = $1,
				updated_at = NOW()
			WHERE id = $2 AND user_id = $3
			RETURNING id, name, user_id, created_at, updated_at;
		`
		err := projectRepo.db.QueryRowContext(
			ctx,
			query,
			project.Name,
			project.ID,
			project.UserID,
		).Scan(&project.ID, &project.Name, &project.UserID, &project.CreatedAt, &project.UpdatedAt)
		if err != nil {
			return nil, err
		}
		return project, nil
	}

	query := `
		INSERT INTO project (name, user_id)
		VALUES ($1, $2)
		RETURNING id, name, user_id, created_at, updated_at;
	`
	err := projectRepo.db.QueryRowContext(
		ctx,
		query,
		project.Name,
		project.UserID,
	).Scan(&project.ID, &project.Name, &project.UserID, &project.CreatedAt, &project.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return project, nil
}

// Get gets a project by ID.
func (projectRepo ProjectRepository) Get(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	query := `SELECT * FROM project WHERE id = $1`
	var project model.Project
	err := projectRepo.db.QueryRowContext(
		ctx,
		query,
		id,
	).Scan(&project.ID, &project.Name, &project.UserID, &project.CreatedAt, &project.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &project, nil
}

// GetUserProjects gets all projects for a user.
func (projectRepo ProjectRepository) GetUserProjects(ctx context.Context, userID uuid.UUID) ([]*model.Project, error) {
	query := `SELECT * FROM project WHERE user_id = $1`
	rows, err := projectRepo.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*model.Project
	for rows.Next() {
		var project model.Project
		if err := rows.Scan(&project.ID, &project.Name, &project.UserID, &project.CreatedAt, &project.UpdatedAt); err != nil {
		}
		projects = append(projects, &project)
	}
	return projects, nil
}

// Delete deletes a project from the database.
func (projectRepo ProjectRepository) Delete(ctx context.Context, project *model.Project) error {
	query := `DELETE FROM project WHERE id = $1`
	_, err := projectRepo.db.ExecContext(ctx, query, project.ID)
	return err

}
