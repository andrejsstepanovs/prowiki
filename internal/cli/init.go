package cli

import (
	"fmt"
	"path/filepath"

	"github.com/andrejsstepanovs/prowiki/internal/db"
	"github.com/andrejsstepanovs/prowiki/internal/migrate"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the ProWiki database",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := viper.GetString("dir")
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return err
		}

		dbPath := filepath.Join(absDir, ".prowiki.db")
		database, err := db.Open(db.Config{Path: dbPath})
		if err != nil {
			return fmt.Errorf("failed to create db: %w", err)
		}
		defer database.Close()

		fmt.Println("Running migrations...")
		if err := migrate.Up(database); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
		fmt.Printf("ProWiki initialized successfully at %s\n", dbPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
