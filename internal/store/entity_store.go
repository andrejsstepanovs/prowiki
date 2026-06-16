package store

import (
	"context"
	"database/sql"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

type EntityStore struct {
	db DBTx
}

func NewEntityStore(db DBTx) *EntityStore {
	return &EntityStore{db: db}
}

func (s *EntityStore) WithTx(tx *sql.Tx) *EntityStore {
	return NewEntityStore(tx)
}

func (s *EntityStore) Create(ctx context.Context, entity *domain.Entity) error {
	query := `INSERT INTO entities (project_id, name, type, description) VALUES (?, ?, ?, ?) RETURNING id, created_at, updated_at`
	return s.db.QueryRowContext(ctx, query, entity.ProjectID, entity.Name, entity.Type, entity.Description).Scan(&entity.ID, &entity.CreatedAt, &entity.UpdatedAt)
}
