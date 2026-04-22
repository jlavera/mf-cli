package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rebuildClean bool

var rebuildCmd = &cobra.Command{
	Use:               "rebuild [services...]",
	Short:             "Full rebuild: down + build --no-cache + up",
	ValidArgsFunction: completeServiceNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		if rebuildClean {
			if len(args) > 0 {
				return fmt.Errorf("--clean cannot be combined with specific services: it removes all project volumes (a project-wide reset), which would wipe data for services you didn't ask to rebuild. Run `mf rebuild --clean` (no services) for a full reset, or omit --clean to rebuild only %v", args)
			}
			fmt.Println("Cleaning containers and volumes...")
			if err := comp.DownVolumes(); err != nil {
				return err
			}
		} else {
			fmt.Println("Stopping containers...")
			if len(args) > 0 {
				if err := comp.Stop(args...); err != nil {
					return err
				}
			} else {
				if err := comp.Down(); err != nil {
					return err
				}
			}
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
	rebuildCmd.GroupID = "general"
	rebuildCmd.Flags().BoolVarP(&rebuildClean, "clean", "c", false, "Remove volumes before rebuilding (full reset)")
	rootCmd.AddCommand(rebuildCmd)
}
