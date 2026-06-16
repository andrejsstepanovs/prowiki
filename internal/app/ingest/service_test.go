package ingest

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/ast"
	"github.com/andrejsstepanovs/prowiki/internal/db"
	"github.com/andrejsstepanovs/prowiki/internal/domain"
	"github.com/andrejsstepanovs/prowiki/internal/migrate"
	"github.com/andrejsstepanovs/prowiki/internal/store"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "prowiki-ingest-test-*.sqlite")
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

func TestProcessFile_ASTBypass(t *testing.T) {
	database := setupTestDB(t)
	fvStore := store.NewFileVersionStore(database)
	jStore := store.NewJobStore(database)
	pStore := store.NewProjectStore(database)
	fStore := store.NewFileStore(database)
	parser := ast.NewHeuristicParser()

	service := NewService(fvStore, jStore, parser)
	ctx := context.Background()

	p := &domain.Project{Name: "Bypass Test"}
	_ = pStore.Create(ctx, p)
	f := &domain.File{ProjectID: p.ID, Path: "test.go"}
	_ = fStore.Create(ctx, f)

	// First ingest
	content1 := []byte("func main() {\n\t// comment\n}")
	err := service.ProcessFile(ctx, p.ID, f, content1)
	if err != nil {
		t.Fatalf("failed to process file: %v", err)
	}

	stats, _ := jStore.GetStats(ctx, p.ID)
	if stats.Pending != 1 {
		t.Fatalf("expected 1 job, got %d", stats.Pending)
	}

	jobs, _ := jStore.ClaimBatch(ctx, 1)
	_ = jStore.UpdateStatus(ctx, jobs[0].ID, domain.JobStatusCompleted)

	// Second ingest: change only comments
	content2 := []byte("func main() {\n\t// modified comment\n}")
	err = service.ProcessFile(ctx, p.ID, f, content2)
	if err != nil {
		t.Fatalf("failed to process file: %v", err)
	}

	stats2, _ := jStore.GetStats(ctx, p.ID)
	if stats2.Pending != 0 {
		t.Fatalf("expected 0 pending jobs due to bypass, got %d", stats2.Pending)
	}

	// Third ingest: structural change
	content3 := []byte("func main() {\n\tfmt.Println(1)\n}")
	err = service.ProcessFile(ctx, p.ID, f, content3)
	if err != nil {
		t.Fatalf("failed to process file: %v", err)
	}

	stats3, _ := jStore.GetStats(ctx, p.ID)
	if stats3.Pending != 1 {
		t.Fatalf("expected 1 pending job after structural change, got %d", stats3.Pending)
	}
}
