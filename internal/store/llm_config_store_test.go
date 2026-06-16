package store

import (
	"context"
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

func TestLLMConfigStore(t *testing.T) {
	database := setupTestDB(t)
	projectStore := NewProjectStore(database)
	llmStore := NewLLMConfigStore(database)
	ctx := context.Background()

	p := &domain.Project{Name: "LLM Config Test"}
	_ = projectStore.Create(ctx, p)

	query := `INSERT INTO llm_configs (project_id, model_tier, safe_token_limit) VALUES (?, ?, ?)`
	_, err := database.Exec(query, p.ID, domain.ModelTier1, 4000)
	if err != nil {
		t.Fatalf("failed to insert mock config: %v", err)
	}

	cfg, err := llmStore.GetByTier(ctx, p.ID, domain.ModelTier1)
	if err != nil {
		t.Fatalf("failed to get config: %v", err)
	}
	if cfg.SafeTokenLimit != 4000 {
		t.Fatalf("expected 4000 limit, got %d", cfg.SafeTokenLimit)
	}
}
