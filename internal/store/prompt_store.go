package store

import (
	"context"
	"database/sql"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

type PromptStore struct {
	db DBTx
}

func NewPromptStore(db DBTx) *PromptStore {
	return &PromptStore{db: db}
}

func (s *PromptStore) WithTx(tx *sql.Tx) *PromptStore {
	return NewPromptStore(tx)
}

func (s *PromptStore) Active(ctx context.Context, stage domain.Stage) (domain.PromptTemplate, error) {
	query := `SELECT id, stage, template, version, is_active, created_at, updated_at FROM prompt_registry WHERE stage = ? AND is_active = 1 LIMIT 1`
	var p domain.PromptTemplate
	err := s.db.QueryRowContext(ctx, query, stage).Scan(&p.ID, &p.Stage, &p.Template, &p.Version, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return p, domain.ErrNotFound
	}
	return p, err
}

func (s *PromptStore) Create(ctx context.Context, prompt *domain.PromptTemplate) error {
	query := `INSERT INTO prompt_registry (stage, template, version, is_active) VALUES (?, ?, ?, ?) RETURNING id, created_at, updated_at`
	return s.db.QueryRowContext(ctx, query, prompt.Stage, prompt.Template, prompt.Version, prompt.IsActive).Scan(&prompt.ID, &prompt.CreatedAt, &prompt.UpdatedAt)
}
