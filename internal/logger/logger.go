// Package logger provides a structured JSON logger wrapping log/slog.
// Log level is controlled by the PROWIKI_LOG_LEVEL environment variable
// (debug | info | warn | error); defaults to info when unset or unrecognized.
package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

// RFC3339Milli is the time format used in every log line: RFC 3339 with
// millisecond precision, e.g. "2006-01-02T15:04:05.000Z07:00".
const RFC3339Milli = "2006-01-02T15:04:05.000Z07:00"

// Logger wraps slog.Logger and emits one JSON object per log line.
type Logger struct {
	slog *slog.Logger
}

// New constructs a Logger that writes JSON to w at the given level.
// The JSON handler replaces the default time key with an RFC3339Milli
// timestamp so log aggregators can parse it without additional config.
func New(level slog.Level, w io.Writer) *Logger {
	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if len(groups) == 0 && a.Key == slog.TimeKey {
				a.Value = slog.StringValue(a.Value.Time().Format(RFC3339Milli))
			}
			return a
		},
	}
	handler := slog.NewJSONHandler(w, opts)
	return &Logger{slog: slog.New(handler)}
}

// NewFromEnv constructs a Logger whose level is read from the
// PROWIKI_LOG_LEVEL environment variable. Unrecognized or unset values
// default to slog.LevelInfo.
func NewFromEnv(w io.Writer) *Logger {
	return New(levelFromEnv(), w)
}

// levelFromEnv parses PROWIKI_LOG_LEVEL and returns the corresponding
// slog.Level, defaulting to slog.LevelInfo.
func levelFromEnv() slog.Level {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("PROWIKI_LOG_LEVEL"))) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "info":
		return slog.LevelInfo
	default:
		return slog.LevelInfo
	}
}

// Info logs a message at INFO level. Extra key-value pairs can be appended
// as alternating string keys and arbitrary values (same signature as slog).
func (l *Logger) Info(msg string, fields ...any) {
	l.slog.Info(msg, fields...)
}

// Warn logs a message at WARN level.
func (l *Logger) Warn(msg string, fields ...any) {
	l.slog.Warn(msg, fields...)
}

// Error logs a message at ERROR level.
func (l *Logger) Error(msg string, fields ...any) {
	l.slog.Error(msg, fields...)
}

// Debug logs a message at DEBUG level.
func (l *Logger) Debug(msg string, fields ...any) {
	l.slog.Debug(msg, fields...)
}

// LevelFromEnv is exported for tests and DI that need to inspect the
// resolved level without constructing a full Logger.
func LevelFromEnv() slog.Level {
	return levelFromEnv()
}

// ensure time.Time is imported (used at compile time inside ReplaceAttr)
var _ = time.RFC3339
