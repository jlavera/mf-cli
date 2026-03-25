package cmd

import "github.com/spf13/cobra"

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop and remove containers",
	RunE: func(cmd *cobra.Command, args []string) error {
		return comp.Down()
	},
}

func init() {
	downCmd.GroupID = "general"
	rootCmd.AddCommand(downCmd)
}
