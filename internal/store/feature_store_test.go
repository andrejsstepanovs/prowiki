package store

import (
	"context"
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

func TestFeatureStore(t *testing.T) {
	database := setupTestDB(t)
	projectStore := NewProjectStore(database)
	featureStore := NewFeatureStore(database)
	fileStore := NewFileStore(database)
	fvStore := NewFileVersionStore(database)
	ctx := context.Background()

	p := &domain.Project{Name: "Feature Test"}
	_ = projectStore.Create(ctx, p)

	f := &domain.Feature{ProjectID: p.ID, Name: "Login", Description: "User auth"}
	err := featureStore.Create(ctx, f)
	if err != nil {
		t.Fatalf("failed to create feature: %v", err)
	}

	features, err := featureStore.GetByProjectID(ctx, p.ID)
	if err != nil {
		t.Fatalf("failed to get features: %v", err)
	}
	if len(features) != 1 || features[0].Name != "Login" {
		t.Fatalf("expected 1 feature with name Login")
	}

	// Test junction
	file := &domain.File{ProjectID: p.ID, Path: "main.go"}
	_ = fileStore.Create(ctx, file)
	fv := &domain.FileVersion{FileID: file.ID, AstHash: "xyz", IsLatest: true}
	_ = fvStore.InsertVersion(ctx, fv)

	err = featureStore.AddToFileVersion(ctx, fv.ID, f.ID)
	if err != nil {
		t.Fatalf("failed to add file feature junction: %v", err)
	}
}
