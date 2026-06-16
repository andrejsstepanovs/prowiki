package scrub

import (
	"regexp"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

type RegexScrubber struct {
	rules  []*regexp.Regexp
	kvRule *regexp.Regexp // hex-in-KV rule: only capture group 1 is replaced
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
			// JWT: three base64url segments starting with eyJ
			regexp.MustCompile(`eyJ[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]*`),
			// PEM private key block ((?s) makes . match newlines in Go RE2)
			regexp.MustCompile(`(?s)-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----.*?-----END[^\n]*PRIVATE KEY-----`),
			// GitHub Personal Access Token
			regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{36}`),
		},
		// Hex value in key=value assignment; only the hex portion (group 1) is replaced
		kvRule: regexp.MustCompile(`(?i)(?:password|secret|key|token|api_key)["\s:=]+["']?([0-9a-fA-F]{32,})["']?`),
	}
}

func (s *RegexScrubber) Scrub(content string, lang domain.Language) (redacted string, hits int) {
	redacted = content

	// Apply simple full-match rules
	for _, r := range s.rules {
		matches := r.FindAllString(redacted, -1)
		hits += len(matches)
		redacted = r.ReplaceAllString(redacted, "[REDACTED_SECRET]")
	}

	// Apply hex-in-KV rule: replace only the captured hex value (group 1), preserving the key name
	kvMatches := s.kvRule.FindAllStringSubmatchIndex(redacted, -1)
	if len(kvMatches) > 0 {
		hits += len(kvMatches)
		redacted = s.kvRule.ReplaceAllStringFunc(redacted, func(match string) string {
			sub := s.kvRule.FindStringSubmatchIndex(match)
			if len(sub) < 4 || sub[2] < 0 {
				return match
			}
			// sub[2]:sub[3] is the byte range of group 1 within match
			return match[:sub[2]] + "[REDACTED_SECRET]" + match[sub[3]:]
		})
	}

	return redacted, hits
}
