package llm

import (
	"context"
	"fmt"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

// DiscoverBoundary finds the safe token limit (T_safe) for a model by provoking 413s.
func DiscoverBoundary(ctx context.Context, model string, completer domain.Completer) (int, error) {
	tInit := 1000000
	step := 250000
	currentT := tInit
	epsilon := 5000

	convergedThreshold := 0

	// Binary search for the boundary
	for step > epsilon {
		req := domain.CompletionRequest{
			Model: model,
			Messages: []domain.Message{
				{Role: "user", Content: generateDummyPayload(currentT)},
			},
			MaxTokens: 10,
		}

		_, err := completer.Complete(ctx, req)
		
		if err == domain.ErrContextOverflow {
			// Failed due to overflow, step back and halve
			currentT -= step
			step /= 2
		} else if err != nil {
			return 0, fmt.Errorf("unexpected error during discovery: %w", err)
		} else {
			// Success, record threshold, step forward, halve step
			convergedThreshold = currentT
			currentT += step
			step /= 2
		}
	}

	if convergedThreshold == 0 {
		return 0, fmt.Errorf("could not find a safe token threshold")
	}

	tSafe := int(float64(convergedThreshold) * 0.9)
	return tSafe, nil
}

// generateDummyPayload generates a string of approximate token length.
// Assuming HeuristicCounter uses len/4, we generate a string of length T*4.
func generateDummyPayload(tokens int) string {
	bytes := make([]byte, tokens*4)
	for i := range bytes {
		bytes[i] = 'a'
	}
	return string(bytes)
}
