package cmd

import (
	"fmt"

	"github.com/jlavera/mf-cli/internal/compose"
	"github.com/jlavera/mf-cli/internal/proxy"
	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:               "down [services...]",
	Short:             "Stop and remove containers",
	ValidArgsFunction: completeServiceNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := comp.Down(args...); err != nil {
			return err
		}
		if cfg.DNS.Enabled {
			unregisterRoutes()
		}
		return nil
	},
}

func unregisterRoutes() {
	cf, err := compose.ParseComposeFile(cfg.ComposeFile)
	if err != nil {
		return
	}
	projectName := dnsProjectName(cf)
	if err := proxy.RemoveRoutes(projectName); err != nil {
		fmt.Printf("⚠️  Could not remove routes: %v\n", err)
		return
	}
	// The proxy daemon polls the routes directory and drops the routes shortly.
}

func init() {
	downCmd.GroupID = "general"
	rootCmd.AddCommand(downCmd)
}
