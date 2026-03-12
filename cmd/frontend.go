package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jlavera/mf-cli/internal/runner"
	"github.com/spf13/cobra"
)

var (
	frontendBuildProd bool
)

var frontendCmd = &cobra.Command{
	Use:   "frontend",
	Short: "Manage frontend project",
}

var frontendInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install frontend dependencies",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, pm, err := frontendConfig()
		if err != nil {
			return err
		}
		return runner.RunInDir(dir, pm, "install")
	},
}

var frontendDevCmd = &cobra.Command{
	Use:   "dev",
	Short: "Start frontend development server",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, pm, err := frontendConfig()
		if err != nil {
			return err
		}
		return runner.RunInDir(dir, pm, "run", "dev")
	},
}

var frontendBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build frontend for production",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, pm, err := frontendConfig()
		if err != nil {
			return err
		}
		script := "build"
		if frontendBuildProd {
			script = "build:prod"
		}
		return runner.RunInDir(dir, pm, "run", script)
	},
}

var frontendPreviewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Preview production build locally",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, pm, err := frontendConfig()
		if err != nil {
			return err
		}
		return runner.RunInDir(dir, pm, "run", "preview")
	},
}

var frontendLintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Run ESLint on frontend code",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, pm, err := frontendConfig()
		if err != nil {
			return err
		}
		return runner.RunInDir(dir, pm, "run", "lint")
	},
}

var frontendTypeCheckCmd = &cobra.Command{
	Use:   "type-check",
	Short: "Run TypeScript type checking",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, pm, err := frontendConfig()
		if err != nil {
			return err
		}
		return runner.RunInDir(dir, pm, "run", "type-check")
	},
}

var frontendCheckAllCmd = &cobra.Command{
	Use:   "check-all",
	Short: "Run all frontend checks (type-check, lint, build)",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, pm, err := frontendConfig()
		if err != nil {
			return err
		}
		return runner.RunInDir(dir, pm, "run", "check-all")
	},
}

var frontendRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart frontend dev server (clear Vite cache)",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, pm, err := frontendConfig()
		if err != nil {
			return err
		}

		fmt.Println("Clearing Vite cache...")
		os.RemoveAll(filepath.Join(dir, "dist"))
		os.RemoveAll(filepath.Join(dir, "node_modules", ".vite"))

		fmt.Println("Starting frontend dev server...")
		return runner.RunInDir(dir, pm, "run", "dev")
	},
}

func init() {
	frontendBuildCmd.Flags().BoolVar(&frontendBuildProd, "prod", false, "build with production optimizations")

	frontendCmd.AddCommand(frontendInstallCmd)
	frontendCmd.AddCommand(frontendDevCmd)
	frontendCmd.AddCommand(frontendBuildCmd)
	frontendCmd.AddCommand(frontendPreviewCmd)
	frontendCmd.AddCommand(frontendLintCmd)
	frontendCmd.AddCommand(frontendTypeCheckCmd)
	frontendCmd.AddCommand(frontendCheckAllCmd)
	frontendCmd.AddCommand(frontendRestartCmd)
	rootCmd.AddCommand(frontendCmd)
}

// frontendConfig returns the absolute frontend directory and package manager.
func frontendConfig() (string, string, error) {
	if cfg.Frontend.Path == "" {
		return "", "", fmt.Errorf("no frontend path configured — set frontend.path in mf.yaml")
	}

	dir := cfg.Frontend.Path
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
