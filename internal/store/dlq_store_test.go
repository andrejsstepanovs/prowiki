package store

import (
	"context"
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

func TestDLQStore(t *testing.T) {
	database := setupTestDB(t)
	dlqStore := NewDLQStore(database)
	ctx := context.Background()

	projectStore := NewProjectStore(database)
	jobStore := NewJobStore(database)
	p := &domain.Project{Name: "DLQ Test"}
	_ = projectStore.Create(ctx, p)
	
	jobs := []domain.Job{{ProjectID: p.ID, TargetID: 1, TargetType: "File", Stage: domain.StageLevel1Overview, Priority: 1}}
	_ = jobStore.EnqueueMany(ctx, nil, jobs)
	claimed, _ := jobStore.ClaimBatch(ctx, 1)

	item := &domain.DeadLetterItem{
		JobID:   claimed[0].ID,
		Payload: "{}",
		Reason:  "max retries",
	}

	err := dlqStore.Create(ctx, item)
	if err != nil {
		t.Fatalf("failed to create dlq item: %v", err)
	}
}
