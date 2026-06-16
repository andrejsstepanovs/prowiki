package store

import (
	"context"
	"database/sql"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

type StyleStore struct {
	db DBTx
}

func NewStyleStore(db DBTx) *StyleStore {
	return &StyleStore{db: db}
}

func (s *StyleStore) WithTx(tx *sql.Tx) *StyleStore {
	return NewStyleStore(tx)
}

func (s *StyleStore) CreateCodeStyle(ctx context.Context, style *domain.CodeStyle) error {
	query := `INSERT INTO code_styles (project_id, rule) VALUES (?, ?) RETURNING id, created_at, updated_at`
	return s.db.QueryRowContext(ctx, query, style.ProjectID, style.Rule).Scan(&style.ID, &style.CreatedAt, &style.UpdatedAt)
}

func (s *StyleStore) CreateStyleAnomaly(ctx context.Context, anomaly *domain.StyleAnomaly) error {
	query := `INSERT INTO style_anomalies (file_version_id, code_style_id, rationale) VALUES (?, ?, ?) RETURNING id, created_at`
	return s.db.QueryRowContext(ctx, query, anomaly.FileVersionID, anomaly.CodeStyleID, anomaly.Rationale).Scan(&anomaly.ID, &anomaly.CreatedAt)
}
