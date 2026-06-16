package store

import (
	"context"
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

func TestGraphStore(t *testing.T) {
	database := setupTestDB(t)
	projectStore := NewProjectStore(database)
	featureStore := NewFeatureStore(database)
	graphStore := NewGraphStore(database)
	ctx := context.Background()

	p := &domain.Project{Name: "Graph Test"}
	_ = projectStore.Create(ctx, p)

	f1 := &domain.Feature{ProjectID: p.ID, Name: "F1"}
	f2 := &domain.Feature{ProjectID: p.ID, Name: "F2"}
	_ = featureStore.Create(ctx, f1)
	_ = featureStore.Create(ctx, f2)

	interaction := &domain.FeatureInteraction{
		FromFeatureID: f1.ID,
		ToFeatureID:   f2.ID,
		Description:   "f1 calls f2",
	}

	err := graphStore.CreateInteraction(ctx, interaction)
	if err != nil {
		t.Fatalf("failed to create interaction: %v", err)
	}

	pipelines, err := graphStore.DiscoverMacroPipelines(ctx, p.ID)
	if err != nil {
		t.Fatalf("failed to discover macro pipelines: %v", err)
	}
	if len(pipelines) == 0 {
		t.Fatalf("expected to find pipelines")
	}
}
