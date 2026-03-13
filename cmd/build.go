package cmd

import "github.com/spf13/cobra"

var (
	buildNoCache bool
)

var buildCmd = &cobra.Command{
	Use:               "build [services...]",
	Short:             "Build containers",
	ValidArgsFunction: completeServiceNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		return comp.Build(buildNoCache, args...)
	},
}

func init() {
	buildCmd.Flags().BoolVar(&buildNoCache, "no-cache", false, "build without cache")
	rootCmd.AddCommand(buildCmd)
}
