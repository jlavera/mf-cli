package cmd

import "github.com/spf13/cobra"

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Stop containers and remove volumes (complete reset)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return comp.DownVolumes()
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}
