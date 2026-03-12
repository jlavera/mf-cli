package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart [services...]",
	Short: "Restart containers (down + up)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			// Restart specific services
			return comp.Restart(args...)
		}
		// Full restart: down + up
		fmt.Println("Stopping all containers...")
		if err := comp.Down(); err != nil {
			return err
		}
		fmt.Println("Starting all containers...")
		return comp.Up()
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)
}
