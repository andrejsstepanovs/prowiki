package handlers

import (
	"context"
	"os"
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/db"
	"github.com/andrejsstepanovs/prowiki/internal/domain"
	"github.com/andrejsstepanovs/prowiki/internal/migrate"
	"github.com/andrejsstepanovs/prowiki/internal/prompt"
	"github.com/andrejsstepanovs/prowiki/internal/store"
)

type MockCompleter struct {
	ResponseText string
}

func (m *MockCompleter) Complete(ctx context.Context, req domain.CompletionRequest) (*domain.CompletionResponse, error) {
	return &domain.CompletionResponse{Text: m.ResponseText}, nil
}

func TestParseHandler(t *testing.T) {
	tmpFile, _ := os.CreateTemp("", "prowiki-parse-handler-*.sqlite")
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	database, _ := db.Open(db.Config{Path: tmpPath})
	defer database.Close()
	migrate.Up(database)

	ctx := context.Background()
	pStore := store.NewProjectStore(database)
	p := &domain.Project{Name: "Test"}
	pStore.Create(ctx, p)

	fStore := store.NewFileStore(database)
	f := &domain.File{ProjectID: p.ID, Path: "test.go"}
	fStore.Create(ctx, f)

	vStore := store.NewFileVersionStore(database)
	v := &domain.FileVersion{FileID: f.ID, Content: "func main(){}", AstHash: "hash"}
	vStore.InsertVersion(ctx, v)

	featStore := store.NewFeatureStore(database)
	jobStore := store.NewJobStore(database)

	completer := &MockCompleter{
		ResponseText: `{"summary": "A dummy file", "features": [{"name": "Main", "description": "Entry point"}]}`,
	}
	registry := prompt.NewHardcodedRegistry()

	handler := NewParseHandler(completer, registry, vStore, featStore, jobStore)

	job := domain.Job{
		ProjectID:  p.ID,
		TargetID:   v.ID,
		TargetType: "FILE_VERSION",
		Stage:      "PARSE",
	}

	txFn, err := handler.Handle(ctx, job)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	tx, _ := database.Begin()
	err = txFn(tx)
	if err != nil {
		t.Fatalf("unexpected tx error: %v", err)
	}
	tx.Commit()

	// Verify summary updated
	updatedV, _ := vStore.GetByID(ctx, v.ID)
	if updatedV.Summary != "A dummy file" {
		t.Fatalf("expected summary to be updated")
	}

	// Verify features inserted
	var count int
	database.QueryRow("SELECT COUNT(*) FROM features WHERE project_id = ?", p.ID).Scan(&count)
	if count != 1 {
		t.Fatalf("expected 1 feature, got %d", count)
	}

	// Verify downstream job enqueued
	jobs, _ := jobStore.ClaimBatch(ctx, 10)
	if len(jobs) != 1 || jobs[0].Stage != "ANALYZE_STYLE" {
		t.Fatalf("expected 1 downstream job, got %v", jobs)
	}
}
