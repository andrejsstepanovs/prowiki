package domain_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
	"pgregory.net/rapid"
)

// Feature: prowiki-gap-analysis, Property 3: Language detection from file path
// Validates: Requirements 1.5, 7.6
func TestLanguageFromPath(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Generate random file paths with specific extensions
		extGen := rapid.SampledFrom([]string{".go", ".py", ".js", ".ts", ".txt", ".md", ".java", "", ".GO", ".Py", ".Js", ".tS"})
		ext := extGen.Draw(rt, "ext")

		// Generate a random base name
		baseGen := rapid.StringMatching(`^[a-zA-Z0-9_-]+$`)
		base := baseGen.Draw(rt, "base")

		// Add optional directories
		dirGen := rapid.SampledFrom([]string{"", "src/", "foo/bar/"})
		dir := dirGen.Draw(rt, "dir")

		// Construct path
		path := filepath.Join(dir, base+ext)

		// Test LanguageFromPath
		lang := domain.LanguageFromPath(path)

		// Verify correctness
		extLower := strings.ToLower(ext)
		switch extLower {
		case ".go":
			if lang != domain.Language("go") {
				rt.Fatalf("expected go, got %v for path %q", lang, path)
			}
		case ".py":
			if lang != domain.Language("python") {
				rt.Fatalf("expected python, got %v for path %q", lang, path)
			}
		case ".js":
			if lang != domain.Language("javascript") {
				rt.Fatalf("expected javascript, got %v for path %q", lang, path)
			}
		case ".ts":
			if lang != domain.Language("typescript") {
				rt.Fatalf("expected typescript, got %v for path %q", lang, path)
			}
		default:
			if lang != domain.Language("") {
				rt.Fatalf("expected empty string, got %v for path %q", lang, path)
			}
		}
	})
}
