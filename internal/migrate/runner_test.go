package migrate

import (
	"os"
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/db"
)

func TestUpAndDown(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "prowiki-migrate-test-*.sqlite")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	cfg := db.Config{Path: tmpPath}
	database, err := db.Open(cfg)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	if err := Up(database); err != nil {
		t.Fatalf("Up migration failed: %v", err)
	}

	// Verify a table exists
	var name string
	err = database.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='projects'").Scan(&name)
	if err != nil || name != "projects" {
		t.Fatalf("expected projects table to exist, got err: %v, name: %s", err, name)
	}

	if err := Down(database); err != nil {
		t.Fatalf("Down migration failed: %v", err)
	}

	// Verify the table no longer exists
	err = database.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='projects'").Scan(&name)
	if err == nil {
		t.Fatalf("expected projects table to be dropped, but it exists")
	}
}
