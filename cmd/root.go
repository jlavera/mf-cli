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

// configRequired lists commands where mf.yaml must exist.
// All other commands will try to load config but won't fail if missing.
var configRequired = map[string]bool{
	"up": true, "stop": true, "build": true, "down": true,
	"logs": true, "restart": true, "clean": true, "rebuild": true,
	"shell": true, "psql": true, "redis-cli": true,
	"test": true, "format": true, "format-all": true, "lint": true,
	"sort-imports": true, "pre-commit": true,
	"start": true, "check": true, // celery/debug subcommands
}

var rootCmd = &cobra.Command{
	Use:   "mf",
	Short: "mf — a CLI for docker-compose based projects",
	Long: `mf is a project CLI that wraps docker-compose with sensible
defaults and organized subcommands. Configure it with an mf.yaml
file in your project root (run 'mf init' to generate one).`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config for commands that explicitly opt out
		for c := cmd; c != nil; c = c.Parent() {
			if _, ok := c.Annotations["skipConfig"]; ok {
				return nil
			}
		}

		// Try to load config
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			// Only fail if this command actually needs config
			if configRequired[cmd.Name()] {
				return err
			}
			// Otherwise silently skip (completion, help, etc.)
			return nil
		}
		comp = compose.New(cfg)
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", config.DefaultConfigFile, "config file path")
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
