package migrate

import (
	"database/sql"
	"fmt"

	"github.com/andrejsstepanovs/prowiki/migrations"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// Up applies all pending migrations using the embedded filesystem.
func Up(db *sql.DB) error {
	m, err := getMigrateInstance(db)
	if err != nil {
		return err
	}
	// Do not close the migrate instance here as it closes the underlying DB connection.
	// Only close source driver if needed, but migrate v4 handles it.
	
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration up failed: %w", err)
	}
	return nil
}

// Down rolls back all migrations.
func Down(db *sql.DB) error {
	m, err := getMigrateInstance(db)
	if err != nil {
		return err
	}
	
	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration down failed: %w", err)
	}
	return nil
}

func getMigrateInstance(db *sql.DB) (*migrate.Migrate, error) {
	// The embed.FS root is the package directory, so we look in "." (the root of the embed.FS for that package)
	sourceDriver, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("failed to create source driver: %w", err)
	}

	dbDriver, err := sqlite.WithInstance(db, &sqlite.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create db driver: %w", err)
	}

	m, err := migrate.NewWithInstance(
		"iofs",
		sourceDriver,
		"sqlite",
		dbDriver,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return m, nil
}
