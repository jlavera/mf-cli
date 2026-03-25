package cmd

import "github.com/spf13/cobra"

var upCmd = &cobra.Command{
	Use:               "up [services...]",
	Short:             "Start containers in detached mode",
	ValidArgsFunction: completeServiceNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		return comp.Up(args...)
	},
}

func init() {
	upCmd.GroupID = "general"
	rootCmd.AddCommand(upCmd)
}
