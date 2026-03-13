package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jlavera/mf-cli/internal/runner"
	"github.com/spf13/cobra"
)

var (
	e2eFile    string
	e2eProject string
)

var e2eCmd = &cobra.Command{
	Use:   "e2e",
	Short: "Manage end-to-end tests",
}

var e2eInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install E2E dependencies and browsers",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, pm, err := e2eConfig()
		if err != nil {
			return err
		}

		fmt.Println("Installing dependencies...")
		if err := runner.RunInDir(dir, pm, "install"); err != nil {
			return err
		}

		if cfg.E2E.Framework == "playwright" {
			browser := cfg.E2E.Browser
			if browser == "" {
				browser = "chromium"
			}
			fmt.Printf("Installing Playwright %s...\n", browser)
			return runner.RunInDir(dir, "npx", "playwright", "install", browser)
		}

		return nil
	},
}

var e2eRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run E2E tests",
	Long: `Run end-to-end tests.

Examples:
  mf e2e run                              # run all tests
  mf e2e run -f tests/smoke.spec.ts       # run specific file
  mf e2e run --project approval           # run specific project`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _, err := e2eConfig()
		if err != nil {
			return err
		}

		testArgs := []string{"playwright", "test"}
		if e2eFile != "" {
			testArgs = append(testArgs, e2eFile)
		}
		if e2eProject != "" {
			testArgs = append(testArgs, "--project", e2eProject+"*")
		}

		return runner.RunInDir(dir, "npx", testArgs...)
	},
}

var e2eUICmd = &cobra.Command{
	Use:   "ui",
	Short: "Run E2E tests with interactive UI mode",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, pm, err := e2eConfig()
		if err != nil {
			return err
		}
		return runner.RunInDir(dir, pm, "run", "test:ui")
	},
}

var e2eHeadedCmd = &cobra.Command{
	Use:   "headed",
	Short: "Run E2E tests with visible browser",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, pm, err := e2eConfig()
		if err != nil {
			return err
		}
		return runner.RunInDir(dir, pm, "run", "test:headed")
	},
}

var e2eDebugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Run E2E tests in debug mode",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, pm, err := e2eConfig()
		if err != nil {
			return err
		}
		return runner.RunInDir(dir, pm, "run", "test:debug")
	},
}

var e2eReportCmd = &cobra.Command{
	Use:   "report",
	Short: "View the E2E test report",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _, err := e2eConfig()
		if err != nil {
			return err
		}
		return runner.RunInDir(dir, "npx", "playwright", "show-report")
	},
}

func init() {
	e2eRunCmd.Flags().StringVarP(&e2eFile, "file", "f", "", "specific test file to run")
	e2eRunCmd.Flags().StringVar(&e2eProject, "project", "", "playwright project to run (e.g. approval, rejection)")

	// Register completions for e2e flags
	e2eRunCmd.RegisterFlagCompletionFunc("file", completeE2EFiles)
	e2eRunCmd.RegisterFlagCompletionFunc("project", completeE2EProjects)

	e2eCmd.AddCommand(e2eInstallCmd)
	e2eCmd.AddCommand(e2eRunCmd)
	e2eCmd.AddCommand(e2eUICmd)
	e2eCmd.AddCommand(e2eHeadedCmd)
	e2eCmd.AddCommand(e2eDebugCmd)
	e2eCmd.AddCommand(e2eReportCmd)
	rootCmd.AddCommand(e2eCmd)
}

// e2eConfig returns the absolute e2e directory and package manager.
func e2eConfig() (string, string, error) {
	if cfg.E2E.Path == "" {
		return "", "", fmt.Errorf("no e2e path configured — set e2e.path in mf.yaml")
	}

	dir := cfg.E2E.Path
	if !filepath.IsAbs(dir) {
		cwd, _ := os.Getwd()
		dir = filepath.Join(cwd, dir)
	}

	pm := cfg.Frontend.PackageManager
	if pm == "" {
		pm = "npm"
	}

	return dir, pm, nil
}
