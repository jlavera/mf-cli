package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/jlavera/mf-cli/internal/compose"
	"github.com/spf13/cobra"
)

// completeDatabaseServiceNames returns configured database service names for `mf psql` completion.
func completeDatabaseServiceNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 || cfg == nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}
	var names []string
	for _, db := range cfg.Services.Databases {
		if strings.HasPrefix(db.Service, toComplete) {
			names = append(names, db.Service)
		}
	}
	if names == nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

// completeServiceNames returns all service names from the docker-compose file
// for shell autocompletion. It reads the compose file directly so completions
// work even before running a command.
func completeServiceNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	services := getComposeServiceNames()
	if services == nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	// Filter out services already provided as args
	used := make(map[string]bool)
	for _, a := range args {
		used[a] = true
	}

	var completions []string
	for _, s := range services {
		if !used[s] && strings.HasPrefix(s, toComplete) {
			completions = append(completions, s)
		}
	}

	if completions == nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeSingleServiceName completes a single service name (for commands
// that take exactly one service argument like `mf shell`).
func completeSingleServiceName(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}
	return completeServiceNames(cmd, args, toComplete)
}

// completeTestFiles returns Python test file paths for `mf test -f` completion.
func completeTestFiles(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Let the shell do file completion, filtered to Python files
	return []string{"py"}, cobra.ShellCompDirectiveFilterFileExt
}

// completeE2EFiles returns test file paths for `mf e2e run -f` completion.
func completeE2EFiles(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Let the shell do file completion, filtered to common test file extensions
	return []string{"ts", "js", "mjs"}, cobra.ShellCompDirectiveFilterFileExt
}

// completeComposeFiles returns YAML file paths for `mf init --file` completion.
func completeComposeFiles(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"yml", "yaml"}, cobra.ShellCompDirectiveFilterFileExt
}

// completeE2EProjects returns known Playwright project name suggestions.
func completeE2EProjects(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{}, cobra.ShellCompDirectiveNoFileComp
}

// getComposeServiceNames reads the compose file and returns all service names.
func getComposeServiceNames() []string {
	// Determine compose file path
	composeFile := "docker-compose.yml"
	if cfg != nil && cfg.ComposeFile != "" {
		composeFile = cfg.ComposeFile
	}

	// Try to find the compose file
	if !filepath.IsAbs(composeFile) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil
		}
		composeFile = filepath.Join(cwd, composeFile)
	}

	cf, err := compose.ParseComposeFile(composeFile)
	if err != nil {
		return nil
	}

	names := make([]string, 0, len(cf.Services))
	for name := range cf.Services {
		names = append(names, name)
	}
	return names
}
