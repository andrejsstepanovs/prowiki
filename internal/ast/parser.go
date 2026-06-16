package ast

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

// blockCommentRe matches /* ... */ block comments, including multi-line.
var blockCommentRe = regexp.MustCompile(`/\*[\s\S]*?\*/`)

// tripleDoubleQuoteRe matches """...""" docstrings (including multi-line).
var tripleDoubleQuoteRe = regexp.MustCompile(`"""[\s\S]*?"""`)

// tripleSingleQuoteRe matches ”'...”' docstrings (including multi-line).
var tripleSingleQuoteRe = regexp.MustCompile(`'''[\s\S]*?'''`)

type HeuristicParser struct{}

func NewHeuristicParser() *HeuristicParser {
	return &HeuristicParser{}
}

func (p *HeuristicParser) Parse(lang domain.Language, content []byte) (*domain.Tree, error) {
	normalized := stripByLanguage(string(lang), string(content))
	hash := sha256.Sum256([]byte(normalized))
	return &domain.Tree{
		Hash: hex.EncodeToString(hash[:]),
	}, nil
}

func (p *HeuristicParser) StructuralHash(tree *domain.Tree) (string, error) {
	return tree.Hash, nil
}

// stripByLanguage applies language-specific comment/docstring removal followed
// by a universal blank-line normalization pass.
func stripByLanguage(lang, content string) string {
	switch strings.ToLower(lang) {
	case "go", "javascript", "typescript":
		content = stripBlockComments(content)
		content = stripLineComments(content, "//")
	case "python":
		content = stripTripleQuoteDocstrings(content)
		content = stripLineComments(content, "#")
	}
	// Final pass: remove blank lines for all languages (including unknown/empty).
	return stripBlankLines(content)
}

// stripBlockComments removes /* ... */ block comments (including multi-line).
func stripBlockComments(s string) string {
	return blockCommentRe.ReplaceAllString(s, "")
}

// stripLineComments removes lines whose first non-whitespace characters match
// the given prefix (e.g. "//" or "#"). Lines that become empty after stripping
// are handled by the subsequent stripBlankLines pass.
func stripLineComments(s, prefix string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// stripTripleQuoteDocstrings removes """...""" and ”'...”' docstrings.
func stripTripleQuoteDocstrings(s string) string {
	s = tripleDoubleQuoteRe.ReplaceAllString(s, "")
	s = tripleSingleQuoteRe.ReplaceAllString(s, "")
	return s
}

// stripBlankLines removes all blank (empty or whitespace-only) lines.
func stripBlankLines(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}
