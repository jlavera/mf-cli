package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <service> <script> [args...]",
	Short: "Run a package.json script inside a service container",
	Long: `Run any script from a service's package.json inside its container
via docker-compose exec.

Use 'install' as the script name to run the package manager's install command.

Examples:
  mf run frontend dev
  mf run api test
  mf run api test:watch
  mf run frontend build -- --mode production`,
	GroupID:           "stack",
	Args:              cobra.MinimumNArgs(2),
	ValidArgsFunction: completeRunArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		svcName := args[0]
		script := args[1]
		rest := args[2:]

		svc := cfg.FindService(svcName)
		if svc == nil {
			return fmt.Errorf("service %q not found in mf.yaml", svcName)
		}

		pm := svc.PackageManager
		if pm == "" {
			pm = "npm"
		}

		if script == "install" {
			return comp.Exec(svcName, pm, "install")
		}

		execArgs := append([]string{pm, "run", script}, rest...)
		return comp.Exec(svcName, execArgs...)
	},
}

func init() {
	runCmd.GroupID = "stack"
	rootCmd.AddCommand(runCmd)
}
