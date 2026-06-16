package ingest

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/ast"
	"github.com/andrejsstepanovs/prowiki/internal/db"
	"github.com/andrejsstepanovs/prowiki/internal/domain"
	"github.com/andrejsstepanovs/prowiki/internal/migrate"
	"github.com/andrejsstepanovs/prowiki/internal/store"
	"pgregory.net/rapid"
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

// Feature: prowiki-gap-analysis, Property 1: Ingest clone-on-unchanged-hash
func TestIngestCloneOnUnchangedHash_Property(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		database := setupTestDB(t)
		fvStore := store.NewFileVersionStore(database)
		jStore := store.NewJobStore(database)
		pStore := store.NewProjectStore(database)
		fStore := store.NewFileStore(database)
		featStore := store.NewFeatureStore(database)
		parser := ast.NewHeuristicParser()

		projectRoot := t.TempDir()
		service := NewService(database, nil, parser, fStore, fvStore, jStore, 1, projectRoot)
		ctx := context.Background()

		p := &domain.Project{Name: "Bypass Property Test"}
		_ = pStore.Create(ctx, p)

		fileName := rapid.StringMatching(`^[a-z]+\.go$`).Draw(rt, "fileName")
		filePath := filepath.Join(projectRoot, fileName)

		content1 := []byte("func main() {\n\t// comment 1\n}")
		os.WriteFile(filePath, content1, 0644)

		scanned := domain.ScannedFile{Path: filePath}

		err := service.ProcessFile(ctx, scanned)
		if err != nil {
			t.Fatalf("first ingest failed: %v", err)
		}

		domainFile, _ := fStore.GetOrCreate(ctx, nil, 1, fileName)
		latest1, _ := fvStore.LatestByFileID(ctx, domainFile.ID)
		
		feat := &domain.Feature{ProjectID: 1, Name: "Feat1"}
		_ = featStore.Create(ctx, feat)
		_ = featStore.AddToFileVersion(ctx, latest1.ID, feat.ID)

		stats1, _ := jStore.GetStats(ctx, 1)
		
		content2 := []byte("func main() {\n\t// comment 2\n}")
		os.WriteFile(filePath, content2, 0644)
		
		err = service.ProcessFile(ctx, scanned)
		if err != nil {
			t.Fatalf("second ingest failed: %v", err)
		}

		latest2, _ := fvStore.LatestByFileID(ctx, domainFile.ID)
		
		if latest1.ID == latest2.ID {
			t.Fatalf("expected new file version ID")
		}

		stats2, _ := jStore.GetStats(ctx, 1)
		if stats2.Pending != stats1.Pending {
			t.Fatalf("expected pending jobs to remain %d, got %d", stats1.Pending, stats2.Pending)
		}
		
		var junctionCount int
		database.QueryRowContext(ctx, "SELECT count(*) FROM file_features WHERE file_version_id = ?", latest2.ID).Scan(&junctionCount)
		if junctionCount != 1 {
			t.Fatalf("expected 1 junction cloned, got %d", junctionCount)
		}
	})
}

// Feature: prowiki-gap-analysis, Property 2: Ingest cascade invalidation on changed hash
func TestIngestCascadeInvalidation_Property(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		database := setupTestDB(t)
		fvStore := store.NewFileVersionStore(database)
		jStore := store.NewJobStore(database)
		pStore := store.NewProjectStore(database)
		fStore := store.NewFileStore(database)
		parser := ast.NewHeuristicParser()

		projectRoot := t.TempDir()
		service := NewService(database, nil, parser, fStore, fvStore, jStore, 1, projectRoot)
		ctx := context.Background()

		p := &domain.Project{Name: "Cascade Property Test"}
		_ = pStore.Create(ctx, p)

		fileName := rapid.StringMatching(`^[a-z]+\.go$`).Draw(rt, "fileName")
		filePath := filepath.Join(projectRoot, fileName)

		content1 := []byte("func main() {\n\t// original\n}")
		os.WriteFile(filePath, content1, 0644)

		scanned := domain.ScannedFile{Path: filePath}

		err := service.ProcessFile(ctx, scanned)
		if err != nil {
			t.Fatalf("first ingest failed: %v", err)
		}

		domainFile, _ := fStore.GetOrCreate(ctx, nil, 1, fileName)
		latest1, _ := fvStore.LatestByFileID(ctx, domainFile.ID)

		jobs, _ := jStore.ClaimBatch(ctx, 1)
		if len(jobs) > 0 {
			jStore.UpdateStatus(ctx, jobs[0].ID, domain.JobStatusCompleted)
		}

		content2 := []byte("func main() {\n\tfmt.Println(1)\n}")
		os.WriteFile(filePath, content2, 0644)
		
		err = service.ProcessFile(ctx, scanned)
		if err != nil {
			t.Fatalf("second ingest failed: %v", err)
		}

		latest2, _ := fvStore.LatestByFileID(ctx, domainFile.ID)
		if latest1.ID == latest2.ID {
			t.Fatalf("expected new file version ID")
		}

		stats, _ := jStore.GetStats(ctx, 1)
		
		if stats.Pending != 2 {
			t.Fatalf("expected 2 pending jobs, got %d", stats.Pending)
		}
	})
}
