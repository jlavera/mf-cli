package cmd

import (
	"fmt"
	"os"

	"github.com/jlavera/mf-cli/internal/compose"
	"github.com/jlavera/mf-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	cfgFile    string
	cfg        *config.Config
	comp       *compose.Compose
	appVersion = "dev"
)

// SetVersion is called from main to inject the build-time version.
func SetVersion(v string) {
	appVersion = v
	rootCmd.Version = v
}

var rootCmd = &cobra.Command{
	Use:          "mf",
	Short:        "mf — a CLI for docker-compose based projects",
	Long: `mf is a project CLI that wraps docker-compose with sensible
defaults and organized subcommands. Configure it with an mf.yaml
file in your project root (run 'mf init' to generate one).`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", config.DefaultConfigFile, "config file path")

	rootCmd.AddGroup(
		&cobra.Group{ID: "general", Title: "General Commands:"},
		&cobra.Group{ID: "stack", Title: "Stack Commands:"},
	)

	cobra.OnInitialize(loadConfig)
}

// commandsSkipConfig lists command names that should never trigger config loading.
// This includes Cobra built-in commands and our own init command.
var commandsSkipConfig = map[string]bool{
	"init":             true,
	"help":             true,
	"completion":       true,
	"__complete":       true,
	"__completeNoDesc": true,
}

// loadConfig loads the mf.yaml config file. Skipped for commands that
// don't need it (init, help, completion, etc.).
func loadConfig() {
	if shouldSkipConfig() {
		// For completion commands, still try to load config silently
		// so that dynamic completions (service names) work when possible.
		if isCompletionCommand() {
			loadConfigSilent()
		}
		return
	}

	var err error
	cfg, err = config.Load(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	comp = compose.New(cfg)
}

// shouldSkipConfig checks if the current command should skip mandatory config loading.
func shouldSkipConfig() bool {
	if len(os.Args) < 2 {
		return true // root command with no args — just shows help
	}

	cmd, _, _ := rootCmd.Find(os.Args[1:])
	if cmd != nil {
		// Check annotation
		if _, ok := cmd.Annotations["skipConfig"]; ok {
			return true
		}
		// Check built-in skip list (walk up to check parent commands too)
		for c := cmd; c != nil; c = c.Parent() {
			if commandsSkipConfig[c.Name()] {
				return true
			}
		}
	}

	return false
}

// isCompletionCommand returns true if the current invocation is a completion request.
func isCompletionCommand() bool {
	if len(os.Args) < 2 {
		return false
	}
	return os.Args[1] == "__complete" || os.Args[1] == "__completeNoDesc"
}

// loadConfigSilent tries to load config without failing. Used during completion
// so that service name suggestions work when config exists, but TAB doesn't
// break when it doesn't.
func loadConfigSilent() {
	var err error
	cfg, err = config.Load(cfgFile)
	if err != nil {
		return // silently skip — completions will degrade gracefully
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
