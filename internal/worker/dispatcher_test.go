package worker

import (
	"context"
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

type MockExtractor struct {
	CalledOverview bool
}

func (m *MockExtractor) ProcessOverview(ctx context.Context, job *domain.Job) error {
	m.CalledOverview = true
	return nil
}

type MockStyle struct{}
type MockGraph struct{}

func TestServiceDispatcher(t *testing.T) {
	extractor := &MockExtractor{}
	d := NewServiceDispatcher(extractor, &MockStyle{}, &MockGraph{})

	job := domain.Job{
		Stage: domain.StageLevel1Overview,
	}

	txFn, err := d.Dispatch(context.Background(), job)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !extractor.CalledOverview {
		t.Fatalf("expected ProcessOverview to be called")
	}

	if txFn == nil {
		t.Fatalf("expected non-nil TxFunc for complete lifecycle")
	}

	// Unknown stage
	_, err = d.Dispatch(context.Background(), domain.Job{Stage: "UNKNOWN"})
	if err == nil {
		t.Fatalf("expected error for unknown stage")
	}
}
