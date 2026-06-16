package extract

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/db"
	"github.com/andrejsstepanovs/prowiki/internal/domain"
	"github.com/andrejsstepanovs/prowiki/internal/migrate"
	"github.com/andrejsstepanovs/prowiki/internal/prompt"
	"github.com/andrejsstepanovs/prowiki/internal/scrub"
	"github.com/andrejsstepanovs/prowiki/internal/store"
)

type mockCompleter struct {
	text string
}

func (m *mockCompleter) Complete(ctx context.Context, req domain.CompletionRequest) (*domain.CompletionResponse, error) {
	return &domain.CompletionResponse{Text: m.text}, nil
}

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "prowiki-extract-test-*.sqlite")
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

func TestProcessOverview(t *testing.T) {
	database := setupTestDB(t)

	fvs := store.NewFileVersionStore(database)
	fs := store.NewFeatureStore(database)
	es := store.NewEntityStore(database)
	js := store.NewJobStore(database)
	pStore := store.NewProjectStore(database)
	fileStore := store.NewFileStore(database)
	promptStore := store.NewPromptStore(database)

	registry := prompt.NewDBRegistry(promptStore)
	scrubber := scrub.NewRegexScrubber()

	completer := &mockCompleter{
		text: `{"summary": "Test file", "features": [{"name": "Feat1", "description": "Desc1"}]}`,
	}

	svc := NewService(completer, registry, fvs, fs, es, js, scrubber)
	ctx := context.Background()

	p := &domain.Project{Name: "Extract Test"}
	_ = pStore.Create(ctx, p)
	
	f := &domain.File{ProjectID: p.ID, Path: "test.go"}
	_ = fileStore.Create(ctx, f)

	fv := &domain.FileVersion{FileID: f.ID, Content: "code", IsLatest: true}
	_ = fvs.InsertVersion(ctx, fv)

	job := &domain.Job{
		ProjectID:  p.ID,
		TargetID:   fv.ID,
		TargetType: "File",
		Stage:      domain.StageLevel1Overview,
	}
	_ = js.EnqueueMany(ctx, []domain.Job{*job})

	// we need the job id, let's claim
	claimed, _ := js.ClaimBatch(ctx, 1)
	if len(claimed) == 0 {
		t.Fatalf("expected to claim a job")
	}

	err := svc.ProcessOverview(ctx, &claimed[0])
	if err != nil {
		t.Fatalf("ProcessOverview failed: %v", err)
	}

	// Verify Feature created
	features, _ := fs.GetByProjectID(ctx, p.ID)
	if len(features) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(features))
	}
	if features[0].Name != "Feat1" {
		t.Errorf("expected Feat1, got %s", features[0].Name)
	}

	// Verify Stage 2 Job enqueued
	stats, _ := js.GetStats(ctx, p.ID)
	if stats.Pending != 1 {
		t.Fatalf("expected 1 pending job for stage 2, got %d", stats.Pending)
	}
}
