package cli

import (
	"fmt"
	"path/filepath"

	"github.com/andrejsstepanovs/prowiki/internal/di"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the web dashboard and API",
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

		fmt.Printf("Starting ProWiki server in %s...\n", absDir)
		return c.Server.Start(ctx)
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
}
