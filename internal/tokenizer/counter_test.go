package tokenizer

import (
	"testing"

	"pgregory.net/rapid"
	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

func TestHeuristicCounter(t *testing.T) {
	c := NewHeuristicCounter()
	text := "This is a simple test string with thirty-eight chars." // 53 chars

	count, _ := c.Count(text)
	if count != len(text)/4 {
		t.Errorf("expected %d, got %d", len(text)/4, count)
	}

	msgs := []domain.Message{
		{Role: "system", Content: "You are an AI."},
		{Role: "user", Content: "Hello!"},
	}
	msgCount, err := c.CountMessages(msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	expected := (len("You are an AI.")/4 + 4) + (len("Hello!")/4 + 4) + 3
	if msgCount != expected {
		t.Errorf("expected %d, got %d", expected, msgCount)
	}
}

// Feature: prowiki-gap-analysis, Property 19: Heuristic counter within factor of 2 of BPE
// Validates: Requirements 16.5
func TestPropertyHeuristicCounterFactor(t *testing.T) {
	bpe, err := NewTikTokenCounter()
	if err != nil {
		t.Skip("skipping test: cl100k_base not available")
	}
	heur := NewHeuristicCounter()
	rapid.Check(t, func(t *rapid.T) {
		// generate realistic words to avoid BPE edge cases with random characters
		words := rapid.SliceOfN(rapid.StringMatching(`cat|dog|if|else|func|var|int|string|return|for`), 10, 200).Draw(t, "words")
		text := ""
		for _, w := range words {
			text += w + " "
		}
		
		bpeCount, _ := bpe.Count(text)
		heurCount, _ := heur.Count(text)
		
		if bpeCount == 0 {
			return
		}
		
		ratio := float64(heurCount) / float64(bpeCount)
		if ratio < 0.5 || ratio > 2.0 {
			t.Fatalf("ratio out of bounds: heur=%d, bpe=%d, ratio=%.2f", heurCount, bpeCount, ratio)
		}
	})
}

// Feature: prowiki-gap-analysis, Property 20: CountMessages sum invariant
// Validates: Requirements 16.6
func TestPropertyCountMessagesInvariant(t *testing.T) {
	heur := NewHeuristicCounter()
	rapid.Check(t, func(t *rapid.T) {
		msgs := rapid.SliceOf(rapid.Custom(func(t *rapid.T) domain.Message {
			return domain.Message{
				Role:    rapid.StringMatching(`system|user|assistant`).Draw(t, "role"),
				Content: rapid.StringMatching(`[a-zA-Z0-9 _,.!?\n\t]{1,100}`).Draw(t, "content"),
			}
		})).Draw(t, "msgs")
		
		msgCount, err := heur.CountMessages(msgs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		
		expected := 0
		for _, m := range msgs {
			c, _ := heur.Count(m.Content)
			expected += c + 4
		}
		expected += 3
		
		if msgCount != expected {
			t.Fatalf("msgCount mismatch: expected %d, got %d", expected, msgCount)
		}
	})
}
