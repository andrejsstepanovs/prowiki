package cli

import (
	"fmt"
	"path/filepath"

	"github.com/andrejsstepanovs/prowiki/internal/di"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Start the background queue worker",
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

		fmt.Printf("Starting ProWiki daemon in %s...\n", absDir)
		c.Daemon.Start(ctx)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(daemonCmd)
}
