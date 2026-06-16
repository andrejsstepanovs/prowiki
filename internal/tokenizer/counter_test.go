package tokenizer

import (
	"testing"

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
	msgCount := c.CountMessages(msgs)
	
	expected := (len("You are an AI.")/4 + 4) + (len("Hello!")/4 + 4) + 3
	if msgCount != expected {
		t.Errorf("expected %d, got %d", expected, msgCount)
	}
}
