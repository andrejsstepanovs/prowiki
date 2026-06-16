package tokenizer

import (
	"github.com/andrejsstepanovs/prowiki/internal/domain"
	tiktoken "github.com/pkoukk/tiktoken-go"
)

// TikTokenCounter implements domain.Counter using the cl100k_base BPE encoding
// from the tiktoken-go library, giving accurate token counts for GPT-3.5/GPT-4 models.
type TikTokenCounter struct {
	enc *tiktoken.Tiktoken
}

// NewTikTokenCounter returns a TikTokenCounter backed by cl100k_base encoding.
// Returns an error if the encoding cannot be loaded.
func NewTikTokenCounter() (*TikTokenCounter, error) {
	enc, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		return nil, err
	}
	return &TikTokenCounter{enc: enc}, nil
}

// Count returns the number of BPE tokens in the given text.
func (c *TikTokenCounter) Count(text string) (int, error) {
	tokens := c.enc.Encode(text, nil, nil)
	return len(tokens), nil
}

// CountMessages estimates tokens for an array of messages using BPE and standard overhead.
func (c *TikTokenCounter) CountMessages(msgs []domain.Message) (int, error) {
	total := 0
	for _, m := range msgs {
		count, err := c.Count(m.Content)
		if err != nil {
			return 0, err
		}
		total += count
		total += 4 // overhead per message
	}
	total += 3 // base overhead for the prompt
	return total, nil
}
