package scrub

import (
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
	"pgregory.net/rapid"
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

// Feature: prowiki-gap-analysis, Property 13: JWT token redaction
func TestJWTTokenRedaction(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		header := "eyJ" + rapid.StringMatching(`[A-Za-z0-9\-_]+`).Draw(rt, "header")
		payload := rapid.StringMatching(`[A-Za-z0-9\-_]+`).Draw(rt, "payload")
		signature := rapid.StringMatching(`[A-Za-z0-9\-_]*`).Draw(rt, "signature")
		
		jwt := header + "." + payload + "." + signature
		
		prefix := rapid.StringMatching(`[A-Z]{5,10}`).Draw(rt, "prefix")
		suffix := rapid.StringMatching(`[A-Z]{5,10}`).Draw(rt, "suffix")
		
		input := prefix + " " + jwt + " " + suffix
		
		s := NewRegexScrubber()
		got, hits := s.Scrub(input, domain.Language("go"))
		
		want := prefix + " [REDACTED_SECRET] " + suffix
		if got != want {
			rt.Fatalf("Scrub(%q) = %q, want %q", input, got, want)
		}
		if hits != 1 {
			rt.Fatalf("Scrub hits = %d, want 1", hits)
		}
	})
}

// Feature: prowiki-gap-analysis, Property 14: PEM private key block redaction
func TestPEMPrivateKeyBlockRedaction(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		keyType := rapid.SampledFrom([]string{"", "RSA ", "EC ", "OPENSSH "}).Draw(rt, "keyType")
		body := rapid.StringMatching(`[A-Za-z0-9+/\n=]+`).Draw(rt, "body")
		
		pem := "-----BEGIN " + keyType + "PRIVATE KEY-----\n" + body + "\n-----END " + keyType + "PRIVATE KEY-----"
		
		prefix := rapid.StringMatching(`[A-Z]{5,10}`).Draw(rt, "prefix")
		suffix := rapid.StringMatching(`[A-Z]{5,10}`).Draw(rt, "suffix")
		
		input := prefix + " " + pem + " " + suffix
		
		s := NewRegexScrubber()
		got, hits := s.Scrub(input, domain.Language("go"))
		
		want := prefix + " [REDACTED_SECRET] " + suffix
		if got != want {
			rt.Fatalf("Scrub() = %q, want %q", got, want)
		}
		if hits != 1 {
			rt.Fatalf("Scrub hits = %d, want 1", hits)
		}
	})
}

// Feature: prowiki-gap-analysis, Property 15: GitHub PAT and hex-in-KV redaction
func TestGitHubPATAndHexKVRedaction(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		patType := rapid.SampledFrom([]string{"ghp_", "gho_", "ghu_", "ghs_", "ghr_"}).Draw(rt, "patType")
		patBody := rapid.StringMatching(`[A-Za-z0-9]{36}`).Draw(rt, "patBody")
		pat := patType + patBody
		
		keyName := rapid.SampledFrom([]string{"key", "api_key", "KEY", "API_KEY"}).Draw(rt, "keyName")
		separator := rapid.SampledFrom([]string{":", "=", " := "}).Draw(rt, "separator")
		quote1 := rapid.SampledFrom([]string{"", "\"", "'"}).Draw(rt, "quote1")
		quote2 := rapid.SampledFrom([]string{"", "\"", "'"}).Draw(rt, "quote2")
		hexValue := rapid.StringMatching(`[0-9a-fA-F]{32,64}`).Draw(rt, "hexValue")
		
		hexKV := keyName + separator + quote1 + hexValue + quote2
		hexKVWant := keyName + separator + quote1 + "[REDACTED_SECRET]" + quote2
		
		prefix := rapid.StringMatching(`[A-Z]{5,10}`).Draw(rt, "prefix")
		middle := rapid.StringMatching(`[A-Z]{5,10}`).Draw(rt, "middle")
		suffix := rapid.StringMatching(`[A-Z]{5,10}`).Draw(rt, "suffix")
		
		input := prefix + " " + pat + " " + middle + " " + hexKV + " " + suffix
		want := prefix + " [REDACTED_SECRET] " + middle + " " + hexKVWant + " " + suffix
		
		s := NewRegexScrubber()
		got, hits := s.Scrub(input, domain.Language("go"))
		
		if got != want {
			rt.Fatalf("Scrub() = %q\nwant %q", got, want)
		}
		if hits != 2 {
			rt.Fatalf("Scrub hits = %d, want 2", hits)
		}
	})
}
