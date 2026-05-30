package cmd

import (
	"fmt"

	"github.com/jlavera/mf-cli/internal/proxy"
	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:               "up [services...]",
	Short:             "Start containers in detached mode",
	ValidArgsFunction: completeServiceNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := comp.Up(args...); err != nil {
			return err
		}
		if cfg.DNS.Enabled {
			registerRoutes()
		}
		return nil
	},
}

func registerRoutes() {
	projectName, routes, err := resolveRoutes()
	if err != nil {
		fmt.Printf("⚠️  Could not parse compose file for DNS routes: %v\n", err)
		return
	}
	if len(routes) == 0 {
		return
	}

	if err := proxy.WriteRoutes(projectName, routes); err != nil {
		fmt.Printf("⚠️  Could not write DNS routes: %v\n", err)
		fmt.Println("   Run 'sudo mf proxy install' to set up the shared routes directory.")
		return
	}
	// The proxy daemon polls the routes directory, so no explicit reload signal
	// is needed (and would be denied across the user/root privilege boundary).

	fmt.Println("\n🌐 Local DNS routes:")
	for _, r := range routes {
		fmt.Printf("   https://%s → %s\n", r.Hostname, r.Target)
	}
}

func init() {
	upCmd.GroupID = "general"
	rootCmd.AddCommand(upCmd)
}
