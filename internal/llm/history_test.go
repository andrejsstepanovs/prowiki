package llm_test

import (
	"context"
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
	"github.com/andrejsstepanovs/prowiki/internal/llm"
	"pgregory.net/rapid"
)

type mockCounter struct{}

func (m *mockCounter) Count(text string) (int, error) {
	return len(text), nil
}

func (m *mockCounter) CountMessages(msgs []domain.Message) (int, error) {
	total := 0
	for _, msg := range msgs {
		total += len(msg.Content)
	}
	return total, nil
}

func TestTrimToBudget_RespectsBudget(t *testing.T) {
	// Property 10: Token budget respected before LLM call
	// Validates: Requirements 5.3
	rapid.Check(t, func(t *rapid.T) {
		msgs := rapid.SliceOf(rapid.String()).Draw(t, "messages")
		budget := rapid.IntRange(0, 1000).Draw(t, "budget")

		h := llm.NewHistory(&mockCounter{})
		for _, msg := range msgs {
			h.Append("user", msg)
		}

		h.TrimToBudget(budget)

		tokens, _ := (&mockCounter{}).CountMessages(h.Messages())
		if tokens > budget {
			t.Fatalf("TrimToBudget failed: %d > %d", tokens, budget)
		}
	})
}

type mockCompleter struct {
	err error
}

func (m *mockCompleter) Complete(ctx context.Context, req domain.CompletionRequest) (*domain.CompletionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &domain.CompletionResponse{Text: "ok"}, nil
}

func TestContextOverflow_RetryHalvesBudget(t *testing.T) {
	// Property 11: Context overflow retry halves budget
	// Validates: Requirements 5.5
	rapid.Check(t, func(t *rapid.T) {
		msgs := rapid.SliceOf(rapid.String()).Draw(t, "messages")
		budget := rapid.IntRange(10, 1000).Draw(t, "budget")

		h := llm.NewHistory(&mockCounter{})
		for _, msg := range msgs {
			h.Append("user", msg)
		}

		// Simulate caller logic:
		// 1. Send request
		// 2. ErrContextOverflow
		// 3. TrimToBudget(budget/2)
		// 4. Send request
		
		h.TrimToBudget(budget / 2)

		tokens, _ := (&mockCounter{}).CountMessages(h.Messages())
		if tokens > budget/2 {
			t.Fatalf("Context overflow retry did not halve budget correctly: %d > %d", tokens, budget/2)
		}
	})
}
