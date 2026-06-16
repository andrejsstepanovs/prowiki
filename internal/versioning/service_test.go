package versioning

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/ast"
	"github.com/andrejsstepanovs/prowiki/internal/db"
	"github.com/andrejsstepanovs/prowiki/internal/domain"
	"github.com/andrejsstepanovs/prowiki/internal/migrate"
	"github.com/andrejsstepanovs/prowiki/internal/scanner"
	"github.com/andrejsstepanovs/prowiki/internal/store"
)

func TestIngestionService(t *testing.T) {
	// Setup DB
	tmpFile, _ := os.CreateTemp("", "prowiki-versioning-test-*.sqlite")
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	database, _ := db.Open(db.Config{Path: tmpPath})
	defer database.Close()
	migrate.Up(database)

	ctx := context.Background()

	pStore := store.NewProjectStore(database)
	project := &domain.Project{Name: "Test"}
	pStore.Create(ctx, project)

	tmpDir, _ := os.MkdirTemp("", "prowiki-fs")
	defer os.RemoveAll(tmpDir)
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\n\nfunc main() {}"), 0644)

	walker := scanner.NewDefaultWalker()
	parser := ast.NewHeuristicParser()
	fStore := store.NewFileStore(database)
	vStore := store.NewFileVersionStore(database)
	jStore := store.NewJobStore(database)

	svc := NewIngestionService(database, walker, parser, fStore, vStore, jStore, project.ID, tmpDir)

	err := svc.Run(ctx)
	if err != nil {
		t.Fatalf("ingestion 1 failed: %v", err)
	}

	jobs, err := jStore.ClaimBatch(ctx, 10)
	if err != nil || len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %v", len(jobs))
	}

	err = svc.Run(ctx)
	if err != nil {
		t.Fatalf("ingestion 2 failed: %v", err)
	}

	jobs2, _ := jStore.ClaimBatch(ctx, 10)
	if len(jobs2) != 0 {
		t.Fatalf("expected 0 jobs, got %v", len(jobs2))
	}

	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\n\nfunc main() { fmt.Println(\"hi\") }"), 0644)
	err = svc.Run(ctx)
	if err != nil {
		t.Fatalf("ingestion 3 failed: %v", err)
	}

	jobs3, _ := jStore.ClaimBatch(ctx, 10)
	if len(jobs3) != 1 {
		t.Fatalf("expected 1 new job for modification, got %v", len(jobs3))
	}
}
