package scrub

import (
	"regexp"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

type RegexScrubber struct {
	rules []*regexp.Regexp
}

func NewRegexScrubber() *RegexScrubber {
	return &RegexScrubber{
		rules: []*regexp.Regexp{
			// Basic AWS Key
			regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
			// Basic Bearer token
			regexp.MustCompile(`Bearer\s+[a-zA-Z0-9\-\._~+/]+=*`),
			// Basic password fields
			regexp.MustCompile(`(?i)(password|secret|token)["\s:=]+[^\s",}]+`),
		},
	}
}

func (s *RegexScrubber) Scrub(content string, lang domain.Language) (redacted string, hits int) {
	redacted = content
	for _, r := range s.rules {
		matches := r.FindAllString(redacted, -1)
		hits += len(matches)
		redacted = r.ReplaceAllString(redacted, "[REDACTED_SECRET]")
	}
	return redacted, hits
}
