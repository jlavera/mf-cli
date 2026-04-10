package cmd

import (
	"fmt"

	"github.com/jlavera/mf-cli/internal/runner"
	"github.com/spf13/cobra"
)

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Debug port utilities",
}

var debugCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check if debug ports are in use",
	RunE: func(cmd *cobra.Command, args []string) error {
		port := cfg.Test.DebugPort
		if port == 0 {
			port = 5679
		}
		fmt.Printf("\nChecking debug ports (%d):\n\n", port)
		// lsof may return non-zero if no matches — that's OK
		_ = runner.Run("bash", "-c",
			fmt.Sprintf("lsof -i :%d | grep LISTEN || echo 'No debug ports in use'", port))
		return nil
	},
}

var debugCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Kill processes using debug ports",
	RunE: func(cmd *cobra.Command, args []string) error {
		port := cfg.Test.DebugPort
		if port == 0 {
			port = 5679
		}
		fmt.Println("Cleaning debug ports...")
		_ = runner.Run("bash", "-c",
			fmt.Sprintf("lsof -i :%d -t | xargs kill -9 2>/dev/null || echo 'No processes to kill'", port))
		fmt.Println("Debug ports cleaned")
		return nil
	},
}

func init() {
	debugCmd.AddCommand(debugCheckCmd)
	debugCmd.AddCommand(debugCleanCmd)
	debugCmd.GroupID = "general"
	rootCmd.AddCommand(debugCmd)
}
