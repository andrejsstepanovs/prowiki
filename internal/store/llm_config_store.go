package store

import (
	"context"
	"database/sql"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

type LLMConfigStore struct {
	db DBTx
}

func NewLLMConfigStore(db DBTx) *LLMConfigStore {
	return &LLMConfigStore{db: db}
}

func (s *LLMConfigStore) WithTx(tx *sql.Tx) *LLMConfigStore {
	return NewLLMConfigStore(tx)
}

func (s *LLMConfigStore) GetByTier(ctx context.Context, projectID int64, tier domain.ModelTier) (*domain.LLMConfig, error) {
	query := `SELECT id, project_id, model_tier, safe_token_limit, created_at, updated_at FROM llm_configs WHERE project_id = ? AND model_tier = ?`
	var c domain.LLMConfig
	err := s.db.QueryRowContext(ctx, query, projectID, tier).Scan(&c.ID, &c.ProjectID, &c.ModelTier, &c.SafeTokenLimit, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &c, err
}
