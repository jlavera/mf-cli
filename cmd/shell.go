package cmd

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/jlavera/mf-cli/internal/config"
	"github.com/spf13/cobra"
)

var shellCmd = &cobra.Command{
	Use:   "shell [service]",
	Short: "Open a shell in a container (default: backend service)",
	Long: `Opens an interactive bash shell in the specified service container.
If no service is specified, uses the backend service from config.`,
	ValidArgsFunction: completeSingleServiceName,
	RunE: func(cmd *cobra.Command, args []string) error {
		service := cfg.Backend()
		if len(args) > 0 {
			service = args[0]
		}
		if service == "" {
			return fmt.Errorf("no backend service configured — specify a service name or add a service with type: python in mf.yaml")
		}
		return execShell(service)
	},
}

var psqlCmd = &cobra.Command{
	Use:               "psql [service]",
	Short:             "Open a database shell",
	ValidArgsFunction: completeDatabaseServiceNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		dbs := cfg.Databases()
		if len(dbs) == 0 {
			return fmt.Errorf("no databases configured — add a service with type: postgres in mf.yaml")
		}

		var db *config.Service
		if len(args) > 0 {
			for i, d := range dbs {
				if d.Name == args[0] {
					db = &dbs[i]
					break
				}
			}
			if db == nil {
				return fmt.Errorf("database service %q not found in mf.yaml", args[0])
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
			return comp.Exec(db.Name, shellArgs...)
		case "mysql":
			shellArgs := []string{"mysql"}
			if db.DBUser != "" {
				shellArgs = append(shellArgs, "-u", db.DBUser)
			}
			if db.DBName != "" {
				shellArgs = append(shellArgs, db.DBName)
			}
			return comp.Exec(db.Name, shellArgs...)
		case "mongo":
			return comp.Exec(db.Name, "mongosh")
		default:
			return fmt.Errorf("unsupported database type %q for service %q", db.Type, db.Name)
		}
	},
}

var redisCliCmd = &cobra.Command{
	Use:               "redis-cli [service]",
	Short:             "Open redis-cli in a Redis container",
	ValidArgsFunction: completeSingleServiceName,
	RunE: func(cmd *cobra.Command, args []string) error {
		service := cfg.Redis()
		if len(args) > 0 {
			service = args[0]
		}
		if service == "" {
			return fmt.Errorf("no redis service configured — specify a service name or add a service with type: redis in mf.yaml")
		}
		return comp.Exec(service, "redis-cli")
	},
}

// execShell tries bash first; falls back to sh if bash is not available in the container.
func execShell(service string) error {
	err := comp.Exec(service, "bash")
	if err == nil {
		return nil
	}
	if isBashNotFound(err) {
		return comp.Exec(service, "sh")
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
