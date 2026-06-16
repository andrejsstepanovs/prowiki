package handlers

import (
	"context"
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

func TestDispatcher(t *testing.T) {
	d := NewDispatcher(NoOpHandler)

	ctx := context.Background()

	job := domain.Job{Stage: "PARSE"}
	fn, err := d.Dispatch(ctx, job)
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if fn == nil {
		t.Fatalf("expected valid TxFunc")
	}

	job2 := domain.Job{Stage: "UNKNOWN"}
	_, err2 := d.Dispatch(ctx, job2)
	if err2 == nil {
		t.Fatalf("expected err for unknown stage")
	}
}
