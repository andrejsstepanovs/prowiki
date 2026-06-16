package scrub

import (
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

func TestRegexScrubber(t *testing.T) {
	s := NewRegexScrubber()
	
	tests := []struct{
		input string
		want string
		hits int
	}{
		{"No secrets here", "No secrets here", 0},
		{"My key is AKIA1234567890ABCDEF", "My key is [REDACTED_SECRET]", 1},
		{"Authorization: Bearer abcdef.123456", "Authorization: [REDACTED_SECRET]", 1},
		{"password: super_secret_password\n", "[REDACTED_SECRET]\n", 1},
	}

	for _, tt := range tests {
		got, hits := s.Scrub(tt.input, domain.Language("go"))
		if got != tt.want {
			t.Errorf("Scrub(%q) = %q, want %q", tt.input, got, tt.want)
		}
		if hits != tt.hits {
			t.Errorf("Scrub(%q) hits = %d, want %d", tt.input, hits, tt.hits)
		}
	}
}
