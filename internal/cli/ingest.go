package cli

import (
	"fmt"
	"path/filepath"

	"github.com/andrejsstepanovs/prowiki/internal/di"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var ingestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Run the ingestion scanner",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		dir := viper.GetString("dir")
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return err
		}

		c, err := di.NewContainer(ctx, absDir)
		if err != nil {
			return err
		}
		defer c.DB.Close()

		fmt.Printf("Starting ingestion in %s...\n", absDir)
		if err := c.IngestionService.Run(ctx); err != nil {
			return fmt.Errorf("ingestion failed: %w", err)
		}
		fmt.Println("Ingestion complete.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(ingestCmd)
}
