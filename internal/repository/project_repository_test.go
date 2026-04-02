package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"go-timekeeper/internal/logger"
	"go-timekeeper/internal/model"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestProjectRepository_CRUD(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	repo := NewProjectRepository(db, logger.New("error", "json"))
	ctx := context.Background()

	projectID := uuid.New()
	userID := uuid.New()
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO project (name, user_id)
			VALUES ($1, $2)
			RETURNING id, name, user_id, created_at, updated_at;`)).
		WithArgs("p1", userID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "user_id", "created_at", "updated_at"}).
			AddRow(projectID, "p1", userID, now, now))
	project, err := repo.Save(ctx, &model.Project{Name: "p1", UserID: userID})
	if err != nil {
		t.Fatalf("insert Save: %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE project
				SET name = $1,
					updated_at = NOW()
				WHERE id = $2 AND user_id = $3
				RETURNING id, name, user_id, created_at, updated_at;`)).
		WithArgs("p2", projectID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "user_id", "created_at", "updated_at"}).
			AddRow(projectID, "p2", userID, now, now))
	project.Name = "p2"
	if _, err := repo.Save(ctx, project); err != nil {
		t.Fatalf("update Save: %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM project WHERE id = $1`)).
		WithArgs(projectID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "user_id", "created_at", "updated_at"}).
			AddRow(projectID, "p2", userID, now, now))
	if _, err := repo.Get(ctx, projectID); err != nil {
		t.Fatalf("Get: %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM project WHERE user_id = $1`)).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "user_id", "created_at", "updated_at"}).
			AddRow(projectID, "p2", userID, now, now))
	list, err := repo.GetUserProjects(ctx, userID)
	if err != nil {
		t.Fatalf("GetUserProjects: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("unexpected list len %d", len(list))
	}

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM project WHERE id = $1`)).
		WithArgs(projectID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.Delete(ctx, project); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
