package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rebuildCmd = &cobra.Command{
	Use:               "rebuild [services...]",
	Short:             "Full rebuild: down + build --no-cache + up",
	ValidArgsFunction: completeServiceNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Stopping containers...")
		if err := comp.Down(); err != nil {
			return err
		}
		fmt.Println("Building (no cache)...")
		if err := comp.Build(true, args...); err != nil {
			return err
		}
		fmt.Println("Starting containers...")
		return comp.Up(args...)
	},
}

func init() {
	rootCmd.AddCommand(rebuildCmd)
}
