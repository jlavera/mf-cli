package cmd

import (
	"fmt"

	"github.com/jlavera/mf-cli/internal/runner"
	"github.com/spf13/cobra"
)

var (
	preCommitAll   bool
	preCommitLocal bool
)

var preCommitCmd = &cobra.Command{
	Use:   "pre-commit",
	Short: "Run pre-commit hooks",
	RunE: func(cmd *cobra.Command, args []string) error {
		if preCommitLocal {
			script := cfg.Scripts.PreCommitLocal
			if script == "" {
				return fmt.Errorf("no local pre-commit script configured — set scripts.pre_commit_local in mf.yaml")
			}
			return runner.Run(script)
		}

		script := cfg.Scripts.PreCommit
		if script == "" {
			return fmt.Errorf("no pre-commit script configured — set scripts.pre_commit in mf.yaml")
		}

		if preCommitAll {
			return runner.Run(script, "--all-files")
		}
		return runner.Run(script)
	},
}

func init() {
	preCommitCmd.Flags().BoolVar(&preCommitAll, "all", false, "run on all files")
	preCommitCmd.Flags().BoolVar(&preCommitLocal, "local", false, "run local pre-commit script (in Docker)")
	rootCmd.AddCommand(preCommitCmd)
}
