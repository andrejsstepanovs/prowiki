package llm

import "github.com/andrejsstepanovs/prowiki/internal/domain"

// History manages conversation history with token-aware trimming.
type History struct {
	messages []domain.Message
	counter  domain.Counter
}

func NewHistory(counter domain.Counter) *History {
	return &History{
		messages: make([]domain.Message, 0),
		counter:  counter,
	}
}

func (h *History) Append(role, content string) {
	h.messages = append(h.messages, domain.Message{Role: role, Content: content})
}

func (h *History) Messages() []domain.Message {
	return h.messages
}

// TrimToBudget drops the oldest non-system messages until the total token count
// is within the maxTokens limit, or until only the system message is left.
func (h *History) TrimToBudget(maxTokens int) {
	for len(h.messages) > 1 {
		tokens, err := h.counter.CountMessages(h.messages)
		if err != nil || tokens <= maxTokens {
			break
		}

		// Find first non-system message
		removeIdx := -1
		for i := 0; i < len(h.messages); i++ {
			if h.messages[i].Role != "system" {
				removeIdx = i
				break
			}
		}

		// If no non-system messages found, we can't trim further
		if removeIdx == -1 {
			break
		}

		// Remove message at removeIdx
		h.messages = append(h.messages[:removeIdx], h.messages[removeIdx+1:]...)
	}
}
