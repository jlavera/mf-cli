package cmd

import (
	"fmt"

	"github.com/jlavera/mf-cli/internal/config"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <service> <task> [args...]",
	Short: "Run a package.json script or Maven phase/goal inside a service container",
	Long: `Run a task inside a service's container via docker-compose exec.

For nodejs services, the task is a script from the service's package.json.
Use 'install' as the script name to run the package manager's install command.

For java services, the task is a Maven lifecycle phase or plugin goal, run
with mvn (or ./mvnw when the project has a Maven wrapper). Extra arguments
are passed through, so several phases/goals can be chained.

Examples:
  mf run frontend dev
  mf run api test
  mf run api test:watch
  mf run frontend build -- --mode production
  mf run backend spring-boot:run
  mf run backend clean install -DskipTests`,
	GroupID: "stack",
	// Flag parsing is disabled so task flags like -DskipTests or --watch are
	// forwarded instead of being rejected by Cobra.
	DisableFlagParsing: true,
	ValidArgsFunction:  completeRunArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if hasHelpFlag(args) {
			return cmd.Help()
		}
		args = dropFirstDoubleDash(args)
		if len(args) < 2 {
			return fmt.Errorf("requires a service and a task: mf run <service> <task> [args...]")
		}
		svcName := args[0]
		task := args[1]
		rest := args[2:]

		svc := cfg.FindService(svcName)
		if svc == nil {
			return fmt.Errorf("service %q not found in mf.yaml", svcName)
		}

		pm := svc.PackageManager
		if pm == "" {
			pm = config.DefaultPackageManager(svc.Type)
		}

		switch svc.Type {
		case "java":
			return comp.Exec(svcName, append([]string{pm, task}, rest...)...)
		case "nodejs":
			if task == "install" {
				return comp.Exec(svcName, pm, "install")
			}
			return comp.Exec(svcName, append([]string{pm, "run", task}, rest...)...)
		default:
			return fmt.Errorf("service %q has no runnable type (got %q); set type: nodejs or java in mf.yaml", svcName, svc.Type)
		}
	},
}

// dropFirstDoubleDash removes the first "--" separator, matching what Cobra
// did before flag parsing was disabled (`mf run web build -- --mode prod`).
func dropFirstDoubleDash(args []string) []string {
	for i, a := range args {
		if a == "--" {
			return append(append([]string{}, args[:i]...), args[i+1:]...)
		}
	}
	return args
}

func init() {
	runCmd.GroupID = "stack"
	rootCmd.AddCommand(runCmd)
}
