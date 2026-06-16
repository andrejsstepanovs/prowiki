package store

import (
	"context"
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

func TestEntityStore(t *testing.T) {
	database := setupTestDB(t)
	projectStore := NewProjectStore(database)
	entityStore := NewEntityStore(database)
	ctx := context.Background()

	p := &domain.Project{Name: "Entity Test Project"}
	_ = projectStore.Create(ctx, p)

	e := &domain.Entity{
		ProjectID:   p.ID,
		Name:        "User",
		Type:        "Struct",
		Description: "User model",
	}

	err := entityStore.Create(ctx, e)
	if err != nil {
		t.Fatalf("failed to create entity: %v", err)
	}
	if e.ID == 0 {
		t.Fatalf("expected ID to be set")
	}
}
