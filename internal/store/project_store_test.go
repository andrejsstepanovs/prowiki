package store

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/db"
	"github.com/andrejsstepanovs/prowiki/internal/domain"
	"github.com/andrejsstepanovs/prowiki/internal/migrate"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "prowiki-store-test-*.sqlite")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	database, err := db.Open(db.Config{Path: tmpPath})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	if err := migrate.Up(database); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	t.Cleanup(func() {
		database.Close()
		os.Remove(tmpPath)
	})

	return database
}

func TestProjectStore(t *testing.T) {
	database := setupTestDB(t)
	store := NewProjectStore(database)
	ctx := context.Background()

	p := &domain.Project{Name: "Test Project"}
	err := store.Create(ctx, p)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	if p.ID == 0 {
		t.Fatalf("expected ID to be set")
	}

	fetched, err := store.GetByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("failed to get project: %v", err)
	}
	if fetched.Name != "Test Project" {
		t.Fatalf("expected name to be Test Project, got %s", fetched.Name)
	}

	err = store.Delete(ctx, p.ID)
	if err != nil {
		t.Fatalf("failed to delete project: %v", err)
	}

	_, err = store.GetByID(ctx, p.ID)
	if err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
