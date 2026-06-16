package store

import (
	"context"
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

func TestFileStore(t *testing.T) {
	database := setupTestDB(t)
	projectStore := NewProjectStore(database)
	fileStore := NewFileStore(database)
	ctx := context.Background()

	p := &domain.Project{Name: "File Test Project"}
	_ = projectStore.Create(ctx, p)

	f := &domain.File{
		ProjectID: p.ID,
		Path:      "main.go",
	}

	err := fileStore.Create(ctx, f)
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	files, err := fileStore.GetByProjectID(ctx, p.ID)
	if err != nil {
		t.Fatalf("failed to get files: %v", err)
	}

	if len(files) != 1 || files[0].Path != "main.go" {
		t.Fatalf("expected 1 file with path main.go, got %v", files)
	}
}

func TestFileVersionStore(t *testing.T) {
	database := setupTestDB(t)
	projectStore := NewProjectStore(database)
	fileStore := NewFileStore(database)
	fvStore := NewFileVersionStore(database)
	ctx := context.Background()

	p := &domain.Project{Name: "FV Test Project"}
	_ = projectStore.Create(ctx, p)

	f := &domain.File{ProjectID: p.ID, Path: "main.go"}
	_ = fileStore.Create(ctx, f)

	fv1 := &domain.FileVersion{
		FileID:   f.ID,
		Content:  "fmt.Println(\"v1\")",
		AstHash:  "hash1",
		IsLatest: true,
	}
	err := fvStore.InsertVersion(ctx, nil, fv1)
	if err != nil {
		t.Fatalf("failed to insert version: %v", err)
	}

	latest, err := fvStore.LatestByFileID(ctx, f.ID)
	if err != nil {
		t.Fatalf("failed to get latest version: %v", err)
	}
	if latest.AstHash != "hash1" {
		t.Fatalf("expected hash1, got %s", latest.AstHash)
	}

	fv2 := &domain.FileVersion{
		FileID:   f.ID,
		Content:  "fmt.Println(\"v2\")",
		AstHash:  "hash2",
		IsLatest: false,
	}
	_ = fvStore.InsertVersion(ctx, nil, fv2)

	// Test Atomic swap
	err = fvStore.SetLatest(ctx, nil, fv2.ID, fv1.ID)
	if err != nil {
		t.Fatalf("failed to set latest: %v", err)
	}

	latest, _ = fvStore.LatestByFileID(ctx, f.ID)
	if latest.AstHash != "hash2" {
		t.Fatalf("expected hash2, got %s", latest.AstHash)
	}

	// GetByID
	fetched, err := fvStore.GetByID(ctx, fv1.ID)
	if err != nil {
		t.Fatalf("failed to get by id: %v", err)
	}
	if fetched.IsLatest != false {
		t.Fatalf("expected old version to not be latest")
	}

	// Update Summary
	err = fvStore.UpdateSummary(ctx, fv2.ID, "A nice update")
	if err != nil {
		t.Fatalf("failed to update summary: %v", err)
	}
	fetched2, _ := fvStore.GetByID(ctx, fv2.ID)
	if fetched2.Summary != "A nice update" {
		t.Fatalf("expected summary to be updated")
	}
}
