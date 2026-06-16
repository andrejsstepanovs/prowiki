package llm

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseStrict parses JSON content into T, falling back to basic repairs if needed.
func ParseStrict[T any](raw string) (T, error) {
	var out T
	
	// Attempt 1: Direct unmarshal
	err := json.Unmarshal([]byte(raw), &out)
	if err == nil {
		return out, nil
	}

	// Attempt 2: Repair markdown fences and prose
	repaired := repairMarkdownFences(raw)
	err2 := json.Unmarshal([]byte(repaired), &out)
	if err2 == nil {
		return out, nil
	}

	return out, fmt.Errorf("failed to parse structured output: direct_err=%v, repaired_err=%v", err, err2)
}

func repairMarkdownFences(raw string) string {
	s := strings.TrimSpace(raw)
	
	// Find the first { or [ to strip leading prose
	firstBrace := strings.Index(s, "{")
	firstBracket := strings.Index(s, "[")
	startIdx := -1
	
	if firstBrace != -1 && firstBracket != -1 {
		if firstBrace < firstBracket {
			startIdx = firstBrace
		} else {
			startIdx = firstBracket
		}
	} else if firstBrace != -1 {
		startIdx = firstBrace
	} else if firstBracket != -1 {
		startIdx = firstBracket
	}
	
	if startIdx != -1 {
		s = s[startIdx:]
	}

	// Find the last } or ] to strip trailing prose
	lastBrace := strings.LastIndex(s, "}")
	lastBracket := strings.LastIndex(s, "]")
	endIdx := -1
	
	if lastBrace != -1 && lastBracket != -1 {
		if lastBrace > lastBracket {
			endIdx = lastBrace
		} else {
			endIdx = lastBracket
		}
	} else if lastBrace != -1 {
		endIdx = lastBrace
	} else if lastBracket != -1 {
		endIdx = lastBracket
	}
	
	if endIdx != -1 && endIdx < len(s) {
		s = s[:endIdx+1]
	}
	
	return strings.TrimSpace(s)
}
