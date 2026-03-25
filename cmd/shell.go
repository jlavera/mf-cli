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
		service := cfg.Services.Backend
		if len(args) > 0 {
			service = args[0]
		}
		if service == "" {
			return fmt.Errorf("no backend service configured — specify a service name or set services.backend in mf.yaml")
		}
		return execShell(service)
	},
}

var psqlCmd = &cobra.Command{
	Use:               "psql [service]",
	Short:             "Open a database shell",
	ValidArgsFunction: completeDatabaseServiceNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(cfg.Services.Databases) == 0 {
			return fmt.Errorf("no databases configured — set services.databases in mf.yaml")
		}

		var db *config.DatabaseService
		if len(args) > 0 {
			for i, d := range cfg.Services.Databases {
				if d.Service == args[0] {
					db = &cfg.Services.Databases[i]
					break
				}
			}
			if db == nil {
				return fmt.Errorf("database service %q not found in mf.yaml", args[0])
			}
		} else if len(cfg.Services.Databases) == 1 {
			db = &cfg.Services.Databases[0]
		} else {
			names := make([]string, len(cfg.Services.Databases))
			for i, d := range cfg.Services.Databases {
				names[i] = d.Service
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
			return comp.Exec(db.Service, shellArgs...)
		case "mysql":
			shellArgs := []string{"mysql"}
			if db.DBUser != "" {
				shellArgs = append(shellArgs, "-u", db.DBUser)
			}
			if db.DBName != "" {
				shellArgs = append(shellArgs, db.DBName)
			}
			return comp.Exec(db.Service, shellArgs...)
		case "mongo":
			return comp.Exec(db.Service, "mongosh")
		default:
			return fmt.Errorf("unsupported database type %q for service %q", db.Type, db.Service)
		}
	},
}

var redisCliCmd = &cobra.Command{
	Use:               "redis-cli [service]",
	Short:             "Open redis-cli in a Redis container",
	ValidArgsFunction: completeSingleServiceName,
	RunE: func(cmd *cobra.Command, args []string) error {
		service := cfg.Services.Redis
		if len(args) > 0 {
			service = args[0]
		}
		if service == "" {
			return fmt.Errorf("no redis service configured — specify a service name or set services.redis in mf.yaml")
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
// installed in the container (exit code 127 = command not found).
func isBashNotFound(err error) bool {
	var exitErr *exec.ExitError
	return err != nil && errors.As(err, &exitErr) && exitErr.ExitCode() == 127
}

func init() {
	shellCmd.GroupID = "general"
	psqlCmd.GroupID = "general"
	redisCliCmd.GroupID = "general"
	rootCmd.AddCommand(shellCmd)
	rootCmd.AddCommand(psqlCmd)
	rootCmd.AddCommand(redisCliCmd)
}
