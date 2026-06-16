package store

import (
	"context"
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

func TestStyleStore(t *testing.T) {
	database := setupTestDB(t)
	projectStore := NewProjectStore(database)
	fileStore := NewFileStore(database)
	fvStore := NewFileVersionStore(database)
	styleStore := NewStyleStore(database)
	ctx := context.Background()

	p := &domain.Project{Name: "Style Test"}
	_ = projectStore.Create(ctx, p)

	file := &domain.File{ProjectID: p.ID, Path: "main.go"}
	_ = fileStore.Create(ctx, file)
	fv := &domain.FileVersion{FileID: file.ID, AstHash: "hash"}
	err := fvStore.InsertVersion(ctx, nil, fv)

	style := &domain.CodeStyle{
		ProjectID: p.ID,
		Rule:      "use camelCase",
	}
	err = styleStore.CreateCodeStyle(ctx, style)
	if err != nil {
		t.Fatalf("failed to create code style: %v", err)
	}

	anomaly := &domain.StyleAnomaly{
		FileVersionID: fv.ID,
		CodeStyleID:   style.ID,
		Rationale:     "used snake_case",
	}
	err = styleStore.CreateStyleAnomaly(ctx, anomaly)
	if err != nil {
		t.Fatalf("failed to create style anomaly: %v", err)
	}
}
