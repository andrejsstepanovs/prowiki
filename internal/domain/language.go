package domain

import (
	"path/filepath"
	"strings"
)

// LanguageFromPath returns the programming Language for the given file path
// based on its extension. Returns an empty Language for unrecognised extensions.
func LanguageFromPath(path string) Language {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return Language("go")
	case ".py":
		return Language("python")
	case ".js":
		return Language("javascript")
	case ".ts":
		return Language("typescript")
	default:
		return Language("")
	}
}
