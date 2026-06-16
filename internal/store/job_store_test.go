package store

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

func TestJobStore(t *testing.T) {
	database := setupTestDB(t)
	projectStore := NewProjectStore(database)
	jobStore := NewJobStore(database)
	ctx := context.Background()

	p := &domain.Project{Name: "Job Test Project"}
	_ = projectStore.Create(ctx, p)

	jobs := []domain.Job{
		{ProjectID: p.ID, TargetID: 1, TargetType: "File", Stage: domain.StageLevel1Overview, Priority: 1},
		{ProjectID: p.ID, TargetID: 2, TargetType: "File", Stage: domain.StageLevel1Overview, Priority: 2},
	}

	err := jobStore.EnqueueMany(ctx, jobs)
	if err != nil {
		t.Fatalf("failed to enqueue jobs: %v", err)
	}

	stats, err := jobStore.GetStats(ctx, p.ID)
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}
	if stats.Pending != 2 {
		t.Fatalf("expected 2 pending jobs, got %d", stats.Pending)
	}

	claimed, err := jobStore.ClaimBatch(ctx, 1)
	if err != nil {
		t.Fatalf("failed to claim batch: %v", err)
	}
	if len(claimed) != 1 {
		t.Fatalf("expected 1 claimed job, got %d", len(claimed))
	}
	if claimed[0].Priority != 2 {
		t.Fatalf("expected job with priority 2 to be claimed first, got %d", claimed[0].Priority)
	}

	err = jobStore.UpdateStatus(ctx, claimed[0].ID, domain.JobStatusCompleted)
	if err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	stats, _ = jobStore.GetStats(ctx, p.ID)
	if stats.Completed != 1 || stats.Pending != 1 {
		t.Fatalf("expected 1 completed and 1 pending job, got stats: %+v", stats)
	}
}

func TestJobStore_Concurrency(t *testing.T) {
	database := setupTestDB(t)
	projectStore := NewProjectStore(database)
	jobStore := NewJobStore(database)
	ctx := context.Background()

	p := &domain.Project{Name: "Job Concurrency Project"}
	_ = projectStore.Create(ctx, p)

	// Enqueue 100 jobs
	var jobs []domain.Job
	for i := 0; i < 100; i++ {
		jobs = append(jobs, domain.Job{
			ProjectID:  p.ID,
			TargetID:   int64(i + 1),
			TargetType: "File",
			Stage:      domain.StageLevel1Overview,
			Priority:   1,
		})
	}
	_ = jobStore.EnqueueMany(ctx, jobs)

	// Concurrently claim jobs using 10 goroutines
	var wg sync.WaitGroup
	claimedJobs := make(chan domain.Job, 200)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				batch, err := jobStore.ClaimBatch(context.Background(), 2)
				if err != nil {
					t.Errorf("claim error: %v", err)
					return
				}
				if len(batch) == 0 {
					break // No more jobs
				}
				for _, j := range batch {
					claimedJobs <- j
				}
				time.Sleep(1 * time.Millisecond) // Yield
			}
		}()
	}

	wg.Wait()
	close(claimedJobs)

	// Verify no double claims
	claimedMap := make(map[int64]bool)
	count := 0
	for j := range claimedJobs {
		if claimedMap[j.ID] {
			t.Fatalf("job %d was claimed multiple times!", j.ID)
		}
		claimedMap[j.ID] = true
		count++
	}

	if count != 100 {
		t.Fatalf("expected 100 unique jobs claimed, got %d", count)
	}
}
