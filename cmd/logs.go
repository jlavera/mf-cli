package cmd

import "github.com/spf13/cobra"

var logsCmd = &cobra.Command{
	Use:               "logs [services...]",
	Short:             "View container logs (follow mode)",
	ValidArgsFunction: completeServiceNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		return comp.Logs(args...)
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)
}
