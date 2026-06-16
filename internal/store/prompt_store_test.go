package store

import (
	"context"
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

func TestPromptStore(t *testing.T) {
	database := setupTestDB(t)
	store := NewPromptStore(database)
	ctx := context.Background()

	p := &domain.PromptTemplate{
		Stage:    "TEST_STAGE",
		Template: "System: You are an agent.",
		Version:  1,
		IsActive: true,
	}

	err := store.Create(ctx, p)
	if err != nil {
		t.Fatalf("failed to create prompt: %v", err)
	}

	active, err := store.Active(ctx, "TEST_STAGE")
	if err != nil {
		t.Fatalf("failed to get active prompt: %v", err)
	}
	if active.Template != "System: You are an agent." {
		t.Fatalf("unexpected template: %s", active.Template)
	}
}
