package cmd

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/jlavera/mf-cli/internal/config"
	"github.com/spf13/cobra"
)

// splitServiceAndPassthrough separates the optional service-name argument from
// extra args that should be forwarded to the underlying tool (psql, redis-cli,
// bash, ...). The first non-flag arg is treated as a service name iff
// isService returns true for it; otherwise it (and all following args) are
// forwarded. A literal "--" explicitly terminates mf's own parsing.
func splitServiceAndPassthrough(args []string, isService func(string) bool) (service string, extra []string, err error) {
	if len(args) == 0 {
		return "", nil, nil
	}
	first := args[0]
	rest := args[1:]
	switch {
	case first == "--":
		return "", rest, nil
	case !strings.HasPrefix(first, "-"):
		if !isService(first) {
			return "", nil, fmt.Errorf("unknown service %q (use `--` to pass args to the underlying tool)", first)
		}
		service = first
		if len(rest) > 0 && rest[0] == "--" {
			rest = rest[1:]
		}
		return service, rest, nil
	default:
		return "", args, nil
	}
}

var shellCmd = &cobra.Command{
	Use:   "shell [service] [-- extra-args...]",
	Short: "Open a shell in a container (default: backend service)",
	Long: `Opens an interactive bash shell in the specified service container.
If no service is specified, uses the backend service from config.

Extra args are forwarded to the underlying shell, e.g.:
  mf shell -c "ls /app"
  mf shell web -c "env"`,
	DisableFlagParsing: true,
	ValidArgsFunction:  completeSingleServiceName,
	RunE: func(cmd *cobra.Command, args []string) error {
		if hasHelpFlag(args) {
			return cmd.Help()
		}
		service, extra, err := splitServiceAndPassthrough(args, func(name string) bool {
			return cfg.FindService(name) != nil
		})
		if err != nil {
			return err
		}
		if service == "" {
			service = cfg.Backend()
		}
		if service == "" {
			return fmt.Errorf("no backend service configured — specify a service name or add a service with type: python in mf.yaml")
		}
		return execShell(service, extra)
	},
}

var psqlCmd = &cobra.Command{
	Use:   "psql [service] [-- extra-args...]",
	Short: "Open a database shell",
	Long: `Opens a database shell in the configured database service.

Extra args are forwarded to the underlying client, e.g.:
  mf psql -c "\dt"
  mf psql mydb -c "SELECT 1"`,
	DisableFlagParsing: true,
	ValidArgsFunction:  completeDatabaseServiceNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		if hasHelpFlag(args) {
			return cmd.Help()
		}
		dbs := cfg.Databases()
		if len(dbs) == 0 {
			return fmt.Errorf("no databases configured — add a service with type: postgres in mf.yaml")
		}

		isDB := func(name string) bool {
			for _, d := range dbs {
				if d.Name == name {
					return true
				}
			}
			return false
		}
		serviceName, extra, err := splitServiceAndPassthrough(args, isDB)
		if err != nil {
			return err
		}

		var db *config.Service
		if serviceName != "" {
			for i, d := range dbs {
				if d.Name == serviceName {
					db = &dbs[i]
					break
				}
			}
		} else if len(dbs) == 1 {
			db = &dbs[0]
		} else {
			names := make([]string, len(dbs))
			for i, d := range dbs {
				names[i] = d.Name
			}
			return fmt.Errorf("multiple databases configured — specify one: mf psql <%s>", strings.Join(names, "|"))
		}

		switch db.Type {
		case "postgres":
			shellArgs := []string{"psql"}
			if db.DBUser != "" {
				shellArgs = append(shellArgs, "-U", db.DBUser)
			}
			if db.DBName != "" {
				shellArgs = append(shellArgs, "-d", db.DBName)
			}
			shellArgs = append(shellArgs, extra...)
			return comp.Exec(db.Name, shellArgs...)
		case "mysql":
			shellArgs := []string{"mysql"}
			if db.DBUser != "" {
				shellArgs = append(shellArgs, "-u", db.DBUser)
			}
			if db.DBName != "" {
				shellArgs = append(shellArgs, db.DBName)
			}
			shellArgs = append(shellArgs, extra...)
			return comp.Exec(db.Name, shellArgs...)
		case "mongo":
			shellArgs := append([]string{"mongosh"}, extra...)
			return comp.Exec(db.Name, shellArgs...)
		default:
			return fmt.Errorf("unsupported database type %q for service %q", db.Type, db.Name)
		}
	},
}

var redisCliCmd = &cobra.Command{
	Use:   "redis-cli [service] [-- extra-args...]",
	Short: "Open redis-cli in a Redis container",
	Long: `Opens redis-cli in the configured Redis service.

Extra args are forwarded to redis-cli, e.g.:
  mf redis-cli -n 1
  mf redis-cli PING`,
	DisableFlagParsing: true,
	ValidArgsFunction:  completeSingleServiceName,
	RunE: func(cmd *cobra.Command, args []string) error {
		if hasHelpFlag(args) {
			return cmd.Help()
		}
		service, extra, err := splitServiceAndPassthrough(args, func(name string) bool {
			return cfg.FindService(name) != nil
		})
		if err != nil {
			return err
		}
		if service == "" {
			service = cfg.Redis()
		}
		if service == "" {
			return fmt.Errorf("no redis service configured — specify a service name or add a service with type: redis in mf.yaml")
		}
		cliArgs := append([]string{"redis-cli"}, extra...)
		return comp.Exec(service, cliArgs...)
	},
}

// hasHelpFlag reports whether args contain a help flag. Needed because
// DisableFlagParsing prevents Cobra from handling --help automatically.
func hasHelpFlag(args []string) bool {
	for _, a := range args {
		if a == "-h" || a == "--help" {
			return true
		}
		if a == "--" {
			return false
		}
	}
	return false
}

// execShell tries bash first; falls back to sh if bash is not available in the container.
func execShell(service string, extra []string) error {
	bashArgs := append([]string{"bash"}, extra...)
	err := comp.Exec(service, bashArgs...)
	if err == nil {
		return nil
	}
	if isBashNotFound(err) {
		shArgs := append([]string{"sh"}, extra...)
		return comp.Exec(service, shArgs...)
	}
	return err
}

// isBashNotFound reports whether an exec error was caused by bash not being
// installed in the container (exit 126 from OCI runtime, 127 from shell).
func isBashNotFound(err error) bool {
	var exitErr *exec.ExitError
	if err == nil || !errors.As(err, &exitErr) {
		return false
	}
	code := exitErr.ExitCode()
	return code == 126 || code == 127
}

func init() {
	shellCmd.GroupID = "general"
	psqlCmd.GroupID = "general"
	redisCliCmd.GroupID = "general"
	rootCmd.AddCommand(shellCmd)
	rootCmd.AddCommand(psqlCmd)
	rootCmd.AddCommand(redisCliCmd)
}
