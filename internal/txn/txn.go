package txn

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

// Immediate executes fn inside a transaction. Since the DB connection pool is restricted
// to 1 connection, db.BeginTx effectively guarantees serialized immediate execution.
// It rolls back if fn returns an error or panics.
func Immediate(ctx context.Context, db *sql.DB, fn domain.TxFunc) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure rollback is called in case of panic or an error being returned
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p) // re-throw panic after rollback
		} else if err != nil {
			_ = tx.Rollback() // err is non-nil; don't change it
		} else {
			// fn returned successfully, so commit
			err = tx.Commit()
		}
	}()

	err = fn(tx)
	return err
}

// Read executes fn inside a read transaction. For SQLite with a single open conn,
// this behaves the same as Immediate, but logically represents read-only intent.
func Read(ctx context.Context, db *sql.DB, fn domain.TxFunc) error {
	opts := &sql.TxOptions{ReadOnly: true}
	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to begin read transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		_ = tx.Rollback() // always rollback read transactions safely
	}()

	return fn(tx)
}
