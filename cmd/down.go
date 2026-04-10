package cmd

import "github.com/spf13/cobra"

var downCmd = &cobra.Command{
	Use:               "down [services...]",
	Short:             "Stop and remove containers",
	ValidArgsFunction: completeServiceNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		return comp.Down(args...)
	},
}

func init() {
	downCmd.GroupID = "general"
	rootCmd.AddCommand(downCmd)
}
