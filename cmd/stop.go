package cmd

import "github.com/spf13/cobra"

var stopCmd = &cobra.Command{
	Use:               "stop [services...]",
	Short:             "Stop containers without removing them",
	ValidArgsFunction: completeServiceNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		return comp.Stop(args...)
	},
}

func init() {
	stopCmd.GroupID = "general"
	rootCmd.AddCommand(stopCmd)
}
