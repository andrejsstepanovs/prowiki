package txn

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/db"
)

func TestImmediate(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "prowiki-txn-test-*.sqlite")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	database, err := db.Open(db.Config{Path: tmpPath})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	_, err = database.Exec("CREATE TABLE test (val INTEGER)")
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	ctx := context.Background()

	// Test Commit
	err = Immediate(ctx, database, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO test (val) VALUES (1)")
		return err
	})
	if err != nil {
		t.Fatalf("expected nil error on commit, got %v", err)
	}

	var count int
	database.QueryRow("SELECT COUNT(*) FROM test").Scan(&count)
	if count != 1 {
		t.Fatalf("expected 1 row, got %d", count)
	}

	// Test Rollback on error
	expectedErr := errors.New("boom")
	err = Immediate(ctx, database, func(tx *sql.Tx) error {
		tx.ExecContext(ctx, "INSERT INTO test (val) VALUES (2)")
		return expectedErr
	})
	if err != expectedErr {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}

	database.QueryRow("SELECT COUNT(*) FROM test").Scan(&count)
	if count != 1 {
		t.Fatalf("expected 1 row after rollback, got %d", count)
	}

	// Test Rollback on panic
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic to be propagated")
			}
		}()
		_ = Immediate(ctx, database, func(tx *sql.Tx) error {
			tx.ExecContext(ctx, "INSERT INTO test (val) VALUES (3)")
			panic("boom")
		})
	}()

	database.QueryRow("SELECT COUNT(*) FROM test").Scan(&count)
	if count != 1 {
		t.Fatalf("expected 1 row after panic rollback, got %d", count)
	}
}
