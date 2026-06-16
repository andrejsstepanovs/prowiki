package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

type Dispatcher interface {
	Dispatch(ctx context.Context, job domain.Job) (domain.TxFunc, error)
}

type Daemon struct {
	queue      domain.Queue
	dispatcher Dispatcher
	pollDelay  time.Duration
}

func NewDaemon(q domain.Queue, d Dispatcher, delay time.Duration) *Daemon {
	if delay == 0 {
		delay = 1 * time.Second
	}
	return &Daemon{
		queue:      q,
		dispatcher: d,
		pollDelay:  delay,
	}
}

func (d *Daemon) Start(ctx context.Context) {
	log.Println("Worker daemon started")
	
	// 9.5 Implement DiscoverBoundary startup call in daemon
	// Since we don't have LLMConfigStore injected directly, we log a warning for now
	log.Println("WARN: safe_token_limit defaulting to 4096. DiscoverBoundary not fully wired.")

	for {
		select {
		case <-ctx.Done():
			log.Println("Worker daemon stopping due to context cancellation")
			return
		default:
			d.processBatch(ctx)
			time.Sleep(d.pollDelay)
		}
	}
}

func (d *Daemon) processBatch(ctx context.Context) {
	// Panic recovery for the batch
	defer func() {
		if r := recover(); r != nil {
			log.Printf("CRITICAL: Panic caught in worker loop: %v", r)
		}
	}()

	jobs, err := d.queue.ClaimBatch(ctx, 10)
	if err != nil {
		log.Printf("Error claiming batch: %v", err)
		return
	}

	for _, job := range jobs {
		d.processJobSafe(ctx, job)
	}
}

func (d *Daemon) processJobSafe(ctx context.Context, job domain.Job) {
	// Use a background context for cleanup operations so they succeed even if main ctx is canceled
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			errPayload := fmt.Sprintf("panic: %v", r)
			log.Printf("Panic processing job %d: %v", job.ID, r)
			_ = d.queue.Fail(cleanupCtx, job.ID, errPayload)
		}
	}()

	txFn, err := d.dispatcher.Dispatch(ctx, job)
	if err != nil {
		log.Printf("Job %d failed: %v", job.ID, err)
		_ = d.queue.Fail(cleanupCtx, job.ID, err.Error())
		return
	}

	err = d.queue.Complete(cleanupCtx, job.ID, txFn)
	if err != nil {
		log.Printf("Failed to complete job %d: %v", job.ID, err)
		_ = d.queue.Fail(cleanupCtx, job.ID, fmt.Sprintf("completion error: %v", err))
	}
}
