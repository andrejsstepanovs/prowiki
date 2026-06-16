package db

import (
	"context"
	"os"
	"testing"
)

func TestOpenAndPRAGMAs(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "prowiki-db-test-*.sqlite")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	cfg := Config{Path: tmpPath}
	db, err := Open(cfg)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	if err := Pinger(context.Background(), db); err != nil {
		t.Fatalf("failed to ping db: %v", err)
	}

	// Verify PRAGMAs
	checks := map[string]string{
		"journal_mode": "wal",
		"synchronous":  "1", // 1 corresponds to NORMAL
		"foreign_keys": "1", // 1 corresponds to ON
		"busy_timeout": "5000",
	}

	for pragma, expected := range checks {
		var val string
		err := db.QueryRow("PRAGMA " + pragma).Scan(&val)
		if err != nil {
			t.Errorf("failed to query pragma %s: %v", pragma, err)
			continue
		}
		if val != expected {
			t.Errorf("expected PRAGMA %s to be %s, got %s", pragma, expected, val)
		}
	}
}
