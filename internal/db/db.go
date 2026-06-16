package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // register modernc pure-Go sqlite driver
)

// Config represents database connection configuration
type Config struct {
	Path string
}

// Open opens a database connection and applies required PRAGMAs for performance and concurrency.
func Open(cfg Config) (*sql.DB, error) {
	// Open the database using modernc.org/sqlite driver
	db, err := sql.Open("sqlite", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Apply connection-level PRAGMAs
	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA foreign_keys=ON;",
		"PRAGMA busy_timeout=5000;",
	}

	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to execute pragma %s: %w", p, err)
		}
	}

	// Connection pool tuning: single writer expectation
	// Because the standard `database/sql` pool cannot distinguish between read and write
	// transactions, concurrent writes can lead to "database is locked" errors in SQLite,
	// even with WAL and a busy_timeout. By restricting the pool to a single open connection,
	// we enforce strict serialization of all access (read and write) through this pool,
	// which is the safest approach for an embedded SQLite DB in Go to completely avoid busy locks.
	db.SetMaxOpenConns(1)

	return db, nil
}

// Pinger validates that the database connection is alive.
func Pinger(ctx context.Context, db *sql.DB) error {
	return db.PingContext(ctx)
}
