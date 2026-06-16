package llm

import (
	"context"
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

type mockCompleter struct {
	err error
}

func (m *mockCompleter) Complete(ctx context.Context, req domain.CompletionRequest) (*domain.CompletionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &domain.CompletionResponse{Text: "{}"}, nil
}

func TestDiscoverBoundary(t *testing.T) {
	ctx := context.Background()

	completer := &mockCompleter{err: domain.ErrContextOverflow}
	_, err := DiscoverBoundary(ctx, "test-model", completer)
	if err == nil {
		t.Fatalf("expected error when always overflowing")
	}

	completer.err = nil
	tSafe, err := DiscoverBoundary(ctx, "test-model", completer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tSafe == 0 {
		t.Fatalf("expected non-zero safe boundary")
	}
}


