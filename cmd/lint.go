package cmd

import (
	"fmt"

	"github.com/jlavera/mf-cli/internal/runner"
	"github.com/spf13/cobra"
)

var lintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Run linting checks",
	Long: `Run the lint script configured in mf.yaml (scripts.lint).
Falls back to running ruff via docker-compose exec if no script is set.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.Scripts.Lint != "" {
			return runner.Run(cfg.Scripts.Lint, args...)
		}
		if cfg.Scripts.Ruff != "" {
			return runner.Run(cfg.Scripts.Ruff, ".")
		}
		// Fallback: run ruff in the backend container
		service := cfg.Services.Backend
		if service == "" {
			return fmt.Errorf("no lint script or backend service configured")
		}
		return comp.Exec(service, "ruff", "check", ".")
	},
}

var sortImportsCmd = &cobra.Command{
	Use:   "sort-imports",
	Short: "Sort Python imports using Ruff",
	RunE: func(cmd *cobra.Command, args []string) error {
		service := cfg.Services.Backend
		if service == "" {
			return fmt.Errorf("no backend service configured — set services.backend in mf.yaml")
		}
		return comp.Exec(service, "ruff", "check", "--select", "I", "--fix", ".")
	},
}

var formatAllCmd = &cobra.Command{
	Use:   "format-all",
	Short: "Run all formatting and linting (format + lint + sort-imports)",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Running format...")
		if err := formatCmd.RunE(formatCmd, nil); err != nil {
			return err
		}
		fmt.Println("\nRunning lint...")
		if err := lintCmd.RunE(lintCmd, nil); err != nil {
			return err
		}
		fmt.Println("\nSorting imports...")
		return sortImportsCmd.RunE(sortImportsCmd, nil)
	},
}

func init() {
	rootCmd.AddCommand(lintCmd)
	rootCmd.AddCommand(sortImportsCmd)
	rootCmd.AddCommand(formatAllCmd)
}
