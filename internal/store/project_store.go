package store

import (
	"context"
	"database/sql"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

// DBTx is an interface that matches both *sql.DB and *sql.Tx
type DBTx interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type ProjectStore struct {
	db DBTx
}

func NewProjectStore(db DBTx) *ProjectStore {
	return &ProjectStore{db: db}
}

// WithTx returns a new ProjectStore bound to a transaction.
func (s *ProjectStore) WithTx(tx *sql.Tx) *ProjectStore {
	return NewProjectStore(tx)
}

func (s *ProjectStore) Create(ctx context.Context, project *domain.Project) error {
	query := `INSERT INTO projects (name) VALUES (?) RETURNING id, created_at, updated_at`
	err := s.db.QueryRowContext(ctx, query, project.Name).Scan(&project.ID, &project.CreatedAt, &project.UpdatedAt)
	return err
}

func (s *ProjectStore) GetByID(ctx context.Context, id int64) (*domain.Project, error) {
	query := `SELECT id, name, created_at, updated_at FROM projects WHERE id = ?`
	var p domain.Project
	err := s.db.QueryRowContext(ctx, query, id).Scan(&p.ID, &p.Name, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (s *ProjectStore) Update(ctx context.Context, project *domain.Project) error {
	query := `UPDATE projects SET name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? RETURNING updated_at`
	err := s.db.QueryRowContext(ctx, query, project.Name, project.ID).Scan(&project.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *ProjectStore) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM projects WHERE id = ?`
	res, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}
