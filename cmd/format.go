package cmd

import (
	"fmt"

	"github.com/jlavera/mf-cli/internal/runner"
	"github.com/spf13/cobra"
)

var formatCheck bool

var formatCmd = &cobra.Command{
	Use:   "format",
	Short: "Format code",
	Long: `Run the format script configured in mf.yaml (scripts.format).
Use --check to verify formatting without making changes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.Scripts.Format != "" {
			formatArgs := []string{"."}
			if formatCheck {
				formatArgs = []string{"--check", "."}
			}
			return runner.Run(cfg.Scripts.Format, formatArgs...)
		}

		// Fallback: ruff format in the backend container
		service := cfg.Services.Backend
		if service == "" {
			return fmt.Errorf("no format script or backend service configured")
		}

		ruffArgs := []string{"ruff", "format"}
		if formatCheck {
			ruffArgs = append(ruffArgs, "--check")
		}
		ruffArgs = append(ruffArgs, ".")
		return comp.Exec(service, ruffArgs...)
	},
}

func init() {
	formatCmd.Flags().BoolVar(&formatCheck, "check", false, "check formatting without making changes")
	formatCmd.GroupID = "general"
	rootCmd.AddCommand(formatCmd)
}
