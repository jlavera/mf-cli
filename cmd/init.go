package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jlavera/mf-cli/internal/compose"
	"github.com/jlavera/mf-cli/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scan a docker-compose file and generate mf.yaml",
	Long: `Scans your docker-compose file, detects services and their roles,
and generates an mf.yaml configuration file for the project.

By default, looks for docker-compose.yml/yaml or compose.yml/yaml
in the current directory. Use --file to specify a different path.`,
	Annotations: map[string]string{"skipConfig": "true"},
	RunE:        runInit,
}

var (
	initFile  string
	initForce bool
)

func init() {
	initCmd.Flags().StringVarP(&initFile, "file", "f", "", "path to docker-compose file")
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite existing mf.yaml without prompting")

	// Register file completion for the --file flag (yml/yaml files)
	initCmd.RegisterFlagCompletionFunc("file", completeComposeFiles)

	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	// 1. Find compose file
	composePath, err := resolveComposeFile(initFile)
	if err != nil {
		return err
	}
	fmt.Printf("📄 Found compose file: %s\n", composePath)

	// 2. Check for existing mf.yaml
	outputPath := config.DefaultConfigFile
	if config.Exists(outputPath) && !initForce {
		fmt.Printf("\n⚠️  %s already exists. Overwrite? [y/N] ", outputPath)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// 3. Parse compose file
	cf, err := compose.ParseComposeFile(composePath)
	if err != nil {
		return err
	}

	// 4. Classify services
	detected := compose.ClassifyServices(cf)

	// 5. Derive project name from directory
	cwd, _ := os.Getwd()
	projectName := filepath.Base(cwd)

	// 6. Build config from detected services
	newCfg := buildConfig(projectName, filepath.Base(composePath), detected)

	// 7. Detect frontend/e2e paths from filesystem
	detectProjectPaths(cwd, &newCfg)

	// 8. Write config
	if err := config.Write(outputPath, &newCfg); err != nil {
		return err
	}

	// 9. Print summary
	printSummary(composePath, detected, outputPath)

	return nil
}

// resolveComposeFile finds the compose file either from the flag or by scanning CWD.
func resolveComposeFile(flagPath string) (string, error) {
	if flagPath != "" {
		if _, err := os.Stat(flagPath); err != nil {
			return "", fmt.Errorf("compose file not found: %s", flagPath)
		}
		return flagPath, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("could not determine current directory: %w", err)
	}
	return compose.FindComposeFile(cwd)
}

// buildConfig creates a Config from detected services.
func buildConfig(projectName, composeFileName string, detected []DetectedService) config.Config {
	cfg := config.Config{
		Project:     projectName,
		ComposeFile: composeFileName,
	}

	for _, ds := range detected {
		switch ds.Role {
		case "backend", "app":
			if cfg.Services.Backend == "" {
				cfg.Services.Backend = ds.Name
			}
		case "db":
			cfg.Services.DB = ds.Name
			cfg.Database.Type = ds.ServiceType
			cfg.Database.Name = ds.DBName
			cfg.Database.User = ds.DBUser
		case "redis":
			cfg.Services.Redis = ds.Name
		case "celery_worker", "celery_beat":
			cfg.Services.Workers = append(cfg.Services.Workers, ds.Name)
		case "flower":
			cfg.Services.Flower = ds.Name
		case "frontend":
			if cfg.Services.Backend == "" {
				// If no backend yet, this might be the main service
				// Keep it noted but don't assign to backend
			}
			if ds.BuildCtx != "" && ds.BuildCtx != "." {
				cfg.Frontend.Path = "./" + strings.TrimPrefix(ds.BuildCtx, "./")
			}
		}
	}

	return cfg
}

// detectProjectPaths scans the filesystem for frontend and e2e directories.
func detectProjectPaths(cwd string, cfg *config.Config) {
	// Detect frontend path if not already set from compose build context
	if cfg.Frontend.Path == "" {
		frontendDirs := []string{"frontend", "client", "web", "app"}
		for _, dir := range frontendDirs {
			path := filepath.Join(cwd, dir)
			if hasPackageJSON(path) {
				cfg.Frontend.Path = "./" + dir
				break
			}
		}
	}

	// Detect frontend package manager
	if cfg.Frontend.Path != "" {
		absPath := cfg.Frontend.Path
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(cwd, absPath)
		}
		cfg.Frontend.PackageManager = detectPackageManager(absPath)
	}

	// Detect e2e path
	e2eDirs := []string{"e2e", "tests/e2e", "e2e-tests", "test/e2e"}
	for _, dir := range e2eDirs {
		path := filepath.Join(cwd, dir)
		if hasPlaywrightConfig(path) {
			cfg.E2E.Path = "./" + dir
			cfg.E2E.Framework = "playwright"
			cfg.E2E.Browser = "chromium"
			break
		}
		if hasCypressConfig(path) {
			cfg.E2E.Path = "./" + dir
			cfg.E2E.Framework = "cypress"
			cfg.E2E.Browser = "chromium"
			break
		}
	}
}

func hasPackageJSON(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "package.json"))
	return err == nil
}

func detectPackageManager(dir string) string {
	if _, err := os.Stat(filepath.Join(dir, "pnpm-lock.yaml")); err == nil {
		return "pnpm"
	}
	if _, err := os.Stat(filepath.Join(dir, "yarn.lock")); err == nil {
		return "yarn"
	}
	return "npm"
}

func hasPlaywrightConfig(dir string) bool {
	configs := []string{"playwright.config.ts", "playwright.config.js", "playwright.config.mjs"}
	for _, c := range configs {
		if _, err := os.Stat(filepath.Join(dir, c)); err == nil {
			return true
		}
	}
	return false
}

func hasCypressConfig(dir string) bool {
	configs := []string{"cypress.config.ts", "cypress.config.js", "cypress.json"}
	for _, c := range configs {
		if _, err := os.Stat(filepath.Join(dir, c)); err == nil {
			return true
		}
	}
	return false
}

// printSummary outputs what was detected.
func printSummary(composePath string, detected []DetectedService, outputPath string) {
	fmt.Printf("\n✅ Scanned %s — found %d services\n\n", filepath.Base(composePath), len(detected))

	maxName := 0
	for _, ds := range detected {
		if len(ds.Name) > maxName {
			maxName = len(ds.Name)
		}
	}

	for _, ds := range detected {
		extra := ""
		switch ds.Role {
		case "db":
			parts := []string{ds.ServiceType}
			if ds.DBName != "" {
				parts = append(parts, fmt.Sprintf("db: %s", ds.DBName))
			}
			if ds.DBUser != "" {
				parts = append(parts, fmt.Sprintf("user: %s", ds.DBUser))
			}
			extra = fmt.Sprintf(" (%s)", strings.Join(parts, ", "))
		case "backend":
			extra = " (main service)"
		case "app":
			extra = " (app service)"
		}

		fmt.Printf("  %-*s → %s%s\n", maxName, ds.Name, ds.Role, extra)
	}

	fmt.Printf("\n✅ Generated %s\n", outputPath)
}

// DetectedService is re-exported for use in this package.
type DetectedService = compose.DetectedService
