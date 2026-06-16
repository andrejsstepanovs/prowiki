package logger_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/andrejsstepanovs/prowiki/internal/logger"
	"pgregory.net/rapid"
)

// ---------------------------------------------------------------------------
// Unit tests
// ---------------------------------------------------------------------------

// TestNew_InfoLevel verifies that a logger created at InfoLevel suppresses
// Debug messages and passes Info/Warn/Error ones through.
func TestNew_InfoLevel(t *testing.T) {
	var buf bytes.Buffer
	l := logger.New(slog.LevelInfo, &buf)

	l.Debug("should be hidden")
	if buf.Len() != 0 {
		t.Fatalf("expected debug line to be suppressed, got: %s", buf.String())
	}

	l.Info("visible")
	if buf.Len() == 0 {
		t.Fatal("expected info line to appear")
	}
}

// TestNew_DebugLevel verifies that Debug messages are emitted when level is Debug.
func TestNew_DebugLevel(t *testing.T) {
	var buf bytes.Buffer
	l := logger.New(slog.LevelDebug, &buf)

	l.Debug("debug-msg")
	if !strings.Contains(buf.String(), "debug-msg") {
		t.Fatalf("expected debug-msg in output, got: %s", buf.String())
	}
}

// TestJSONOutput verifies every emitted line is valid JSON with level/ts/msg.
func TestJSONOutput(t *testing.T) {
	var buf bytes.Buffer
	l := logger.New(slog.LevelDebug, &buf)

	l.Info("hello", "key", "value")

	line := strings.TrimSpace(buf.String())
	if line == "" {
		t.Fatal("no output produced")
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, line)
	}

	for _, field := range []string{"level", "time", "msg"} {
		if _, ok := m[field]; !ok {
			t.Errorf("required field %q missing from JSON output: %s", field, line)
		}
	}
}

// TestTimeFormat verifies the timestamp uses RFC3339Milli (millisecond precision).
func TestTimeFormat(t *testing.T) {
	var buf bytes.Buffer
	l := logger.New(slog.LevelInfo, &buf)
	l.Info("ts-test")

	var m map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &m); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	tsRaw, ok := m["time"].(string)
	if !ok {
		t.Fatalf("time field is not a string: %v", m["time"])
	}

	_, err := time.Parse(logger.RFC3339Milli, tsRaw)
	if err != nil {
		t.Errorf("time %q does not match RFC3339Milli: %v", tsRaw, err)
	}
}

// TestLevelMethods verifies Warn, Error, and Debug emit the correct "level" field.
func TestLevelMethods(t *testing.T) {
	cases := []struct {
		fn   func(l *logger.Logger)
		want string
	}{
		{func(l *logger.Logger) { l.Warn("w") }, "WARN"},
		{func(l *logger.Logger) { l.Error("e") }, "ERROR"},
		{func(l *logger.Logger) { l.Debug("d") }, "DEBUG"},
	}

	for _, tc := range cases {
		var buf bytes.Buffer
		l := logger.New(slog.LevelDebug, &buf)
		tc.fn(l)

		var m map[string]any
		if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &m); err != nil {
			t.Fatalf("output is not valid JSON for level %s: %v", tc.want, err)
		}
		got, _ := m["level"].(string)
		if got != tc.want {
			t.Errorf("expected level=%q, got %q", tc.want, got)
		}
	}
}

// ---------------------------------------------------------------------------
// PROWIKI_LOG_LEVEL env-var tests
// ---------------------------------------------------------------------------

// TestNewFromEnv_DefaultsToInfo verifies unset env var produces info level.
func TestNewFromEnv_DefaultsToInfo(t *testing.T) {
	t.Setenv("PROWIKI_LOG_LEVEL", "")

	var buf bytes.Buffer
	l := logger.NewFromEnv(&buf)

	l.Debug("hidden")
	if buf.Len() != 0 {
		t.Fatalf("expected debug suppressed at default info level, got: %s", buf.String())
	}

	l.Info("visible")
	if buf.Len() == 0 {
		t.Fatal("expected info line at default info level")
	}
}

// TestNewFromEnv_UnrecognizedDefaultsToInfo verifies unrecognized value falls back to info.
func TestNewFromEnv_UnrecognizedDefaultsToInfo(t *testing.T) {
	t.Setenv("PROWIKI_LOG_LEVEL", "verbose")

	level := logger.LevelFromEnv()
	if level != slog.LevelInfo {
		t.Errorf("expected LevelInfo for unrecognized value, got %v", level)
	}
}

// TestNewFromEnv_Levels verifies each recognized value maps to the correct level.
func TestNewFromEnv_Levels(t *testing.T) {
	cases := []struct {
		env  string
		want slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"WARN", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
	}

	for _, tc := range cases {
		t.Run(tc.env, func(t *testing.T) {
			t.Setenv("PROWIKI_LOG_LEVEL", tc.env)
			got := logger.LevelFromEnv()
			if got != tc.want {
				t.Errorf("PROWIKI_LOG_LEVEL=%q: expected %v, got %v", tc.env, tc.want, got)
			}
		})
	}
}

// TestNewFromEnv_WriteToAnyWriter verifies the logger writes to a custom writer.
func TestNewFromEnv_WriteToAnyWriter(t *testing.T) {
	t.Setenv("PROWIKI_LOG_LEVEL", "debug")

	var buf bytes.Buffer
	l := logger.NewFromEnv(&buf)
	l.Info("test-message")

	if !strings.Contains(buf.String(), "test-message") {
		t.Fatalf("expected output to contain 'test-message', got: %s", buf.String())
	}
}

// TestNewFromEnv_Stdout verifies NewFromEnv accepts os.Stdout (smoke test).
func TestNewFromEnv_Stdout(t *testing.T) {
	_ = logger.NewFromEnv(os.Stdout)
}

// TestExtraFields verifies additional key/value pairs appear in the JSON output.
func TestExtraFields(t *testing.T) {
	var buf bytes.Buffer
	l := logger.New(slog.LevelInfo, &buf)
	l.Info("with-fields", "job_id", 42, "stage", "level_1_overview")

	var m map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &m); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if v, ok := m["job_id"]; !ok || v != float64(42) {
		t.Errorf("expected job_id=42, got %v", m["job_id"])
	}
	if v, ok := m["stage"]; !ok || v != "level_1_overview" {
		t.Errorf("expected stage=level_1_overview, got %v", m["stage"])
	}
}

// Feature: prowiki-gap-analysis, Property 22: Structured log lines are valid JSON with required fields
// Validates: Requirements 10.1
func TestJSONOutput_Property(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		var buf bytes.Buffer
		l := logger.New(slog.LevelDebug, &buf)

		msg := rapid.String().Draw(rt, "msg")
		levelChoice := rapid.SampledFrom([]int{0, 1, 2, 3}).Draw(rt, "levelChoice")

		switch levelChoice {
		case 0:
			l.Debug(msg)
		case 1:
			l.Info(msg)
		case 2:
			l.Warn(msg)
		case 3:
			l.Error(msg)
		}

		line := strings.TrimSpace(buf.String())
		if line == "" {
			rt.Fatalf("no output produced")
		}

		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			rt.Fatalf("output is not valid JSON: %v\noutput: %s", err, line)
		}

		for _, field := range []string{"level", "time", "msg"} {
			if _, ok := m[field]; !ok {
				rt.Fatalf("required field %q missing from JSON output: %s", field, line)
			}
		}

		if gotMsg, ok := m["msg"].(string); !ok || gotMsg != msg {
			rt.Fatalf("expected msg %q, got %v", msg, m["msg"])
		}
	})
}

