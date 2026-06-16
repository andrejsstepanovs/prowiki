package tokenizer

import "github.com/andrejsstepanovs/prowiki/internal/domain"

// HeuristicCounter implements domain.Counter using a simple character-to-token ratio.
// This is an approximate fallback for a real BPE tokenizer (like tiktoken).
type HeuristicCounter struct{}

func NewHeuristicCounter() *HeuristicCounter {
	return &HeuristicCounter{}
}

// Count estimates tokens based on length/4.
func (c *HeuristicCounter) Count(text string) (int, error) {
	return len(text) / 4, nil
}

// CountMessages estimates tokens for an array of messages, adding standard overhead.
func (c *HeuristicCounter) CountMessages(msgs []domain.Message) int {
	total := 0
	for _, m := range msgs {
		total += len(m.Content) / 4
		total += 4 // overhead per message
	}
	total += 3 // base overhead for the prompt
	return total
}

// FakeCounter is a deterministic counter for testing upstream components.
type FakeCounter struct {
	CountFunc         func(text string) (int, error)
	CountMessagesFunc func(msgs []domain.Message) int
}

func (f *FakeCounter) Count(text string) (int, error) {
	if f.CountFunc != nil {
		return f.CountFunc(text)
	}
	return 0, nil
}

func (f *FakeCounter) CountMessages(msgs []domain.Message) int {
	if f.CountMessagesFunc != nil {
		return f.CountMessagesFunc(msgs)
	}
	return 0
}
