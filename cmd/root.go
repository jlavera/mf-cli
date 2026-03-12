package cmd

import (
	"fmt"
	"os"

	"github.com/jlavera/mf-cli/internal/compose"
	"github.com/jlavera/mf-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     *config.Config
	comp    *compose.Compose
)

var rootCmd = &cobra.Command{
	Use:   "mf",
	Short: "mf — a CLI for docker-compose based projects",
	Long: `mf is a project CLI that wraps docker-compose with sensible
defaults and organized subcommands. Configure it with an mf.yaml
file in your project root (run 'mf init' to generate one).`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", config.DefaultConfigFile, "config file path")

	// Load config before any command that needs it.
	// Commands that don't need config (like init) set their own annotation
	// to skip this.
	cobra.OnInitialize(loadConfig)
}

// loadConfig loads the mf.yaml config file. Skipped for commands that
// annotate themselves with "skipConfig".
func loadConfig() {
	// Find the active command to check for skipConfig annotation
	cmd, _, _ := rootCmd.Find(os.Args[1:])
	if cmd != nil {
		if _, ok := cmd.Annotations["skipConfig"]; ok {
			return
		}
	}

	var err error
	cfg, err = config.Load(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	comp = compose.New(cfg)
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
