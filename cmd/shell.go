package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var shellCmd = &cobra.Command{
	Use:   "shell [service]",
	Short: "Open a shell in a container (default: backend service)",
	Long: `Opens an interactive bash shell in the specified service container.
If no service is specified, uses the backend service from config.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		service := cfg.Services.Backend
		if len(args) > 0 {
			service = args[0]
		}
		if service == "" {
			return fmt.Errorf("no backend service configured — specify a service name or set services.backend in mf.yaml")
		}
		return comp.Exec(service, "bash")
	},
}

var psqlCmd = &cobra.Command{
	Use:   "psql",
	Short: "Open a database shell",
	RunE: func(cmd *cobra.Command, args []string) error {
		service := cfg.Services.DB
		if service == "" {
			return fmt.Errorf("no database service configured — set services.db in mf.yaml")
		}

		switch cfg.Database.Type {
		case "postgres":
			shellArgs := []string{"psql"}
			if cfg.Database.User != "" {
				shellArgs = append(shellArgs, "-U", cfg.Database.User)
			}
			if cfg.Database.Name != "" {
				shellArgs = append(shellArgs, "-d", cfg.Database.Name)
			}
			return comp.Exec(service, shellArgs...)
		case "mysql":
			shellArgs := []string{"mysql"}
			if cfg.Database.User != "" {
				shellArgs = append(shellArgs, "-u", cfg.Database.User)
			}
			if cfg.Database.Name != "" {
				shellArgs = append(shellArgs, cfg.Database.Name)
			}
			return comp.Exec(service, shellArgs...)
		case "mongo":
			return comp.Exec(service, "mongosh")
		default:
			return fmt.Errorf("unsupported database type %q — set database.type in mf.yaml", cfg.Database.Type)
		}
	},
}

var redisCliCmd = &cobra.Command{
	Use:   "redis-cli",
	Short: "Open redis-cli in the Redis container",
	RunE: func(cmd *cobra.Command, args []string) error {
		service := cfg.Services.Redis
		if service == "" {
			return fmt.Errorf("no redis service configured — set services.redis in mf.yaml")
		}
		return comp.Exec(service, "redis-cli")
	},
}

func init() {
	rootCmd.AddCommand(shellCmd)
	rootCmd.AddCommand(psqlCmd)
	rootCmd.AddCommand(redisCliCmd)
}
