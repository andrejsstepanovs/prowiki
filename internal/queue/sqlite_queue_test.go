package queue

import (
	"context"
	"os"
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/db"
	"github.com/andrejsstepanovs/prowiki/internal/domain"
	"github.com/andrejsstepanovs/prowiki/internal/migrate"
	"github.com/andrejsstepanovs/prowiki/internal/store"
)

func TestSQLiteQueue(t *testing.T) {
	tmpFile, _ := os.CreateTemp("", "prowiki-queue-test-*.sqlite")
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

	jStore := store.NewJobStore(database)
	dlqStore := store.NewDLQStore(database)

	q := NewSQLiteQueue(database, jStore, dlqStore)

	// Enqueue
	q.Enqueue(ctx, domain.Job{
		ProjectID:  project.ID,
		TargetID:   1,
		TargetType: "DUMMY",
		Stage:      "TEST",
		Priority:   10,
	})

	// Claim
	jobs, err := q.ClaimBatch(ctx, 5)
	if err != nil || len(jobs) != 1 {
		t.Fatalf("expected 1 job claimed, got %d", len(jobs))
	}

	// Fail x3
	jobID := jobs[0].ID
	q.Fail(ctx, jobID, "err 1")
	q.Fail(ctx, jobID, "err 2")
	
	// Check retry count
	var retry int
	database.QueryRow("SELECT retry_count FROM job_queue WHERE id = ?", jobID).Scan(&retry)
	if retry != 2 {
		t.Fatalf("expected 2 retries, got %d", retry)
	}

	q.Fail(ctx, jobID, "err 3") // now retry_count is 3
	q.Fail(ctx, jobID, "err 4") // exceeds limit, moves to DLQ

	var status domain.JobStatus
	database.QueryRow("SELECT status FROM job_queue WHERE id = ?", jobID).Scan(&status)
	if status != domain.JobStatusFailed {
		t.Fatalf("expected job to be FAILED")
	}

	var dlqCount int
	database.QueryRow("SELECT COUNT(*) FROM dead_letter_queue WHERE job_id = ?", jobID).Scan(&dlqCount)
	if dlqCount != 1 {
		t.Fatalf("expected 1 DLQ entry")
	}

	// Complete a job
	q.Enqueue(ctx, domain.Job{
		ProjectID:  project.ID,
		TargetID:   2,
		TargetType: "DUMMY",
		Stage:      "TEST",
	})
	jobs, _ = q.ClaimBatch(ctx, 1)
	q.Complete(ctx, jobs[0].ID, nil)

	var count int
	database.QueryRow("SELECT COUNT(*) FROM job_queue WHERE id = ?", jobs[0].ID).Scan(&count)
	if count != 0 {
		t.Fatalf("expected job to be deleted on completion")
	}
}
