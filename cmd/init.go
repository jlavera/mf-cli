package cmd

import (
	"bufio"
	"bytes"
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
	initCmd.RegisterFlagCompletionFunc("file", completeComposeFiles)
	initCmd.GroupID = "general"
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
		fmt.Printf("\n%s already exists. Overwrite? [y/N] ", outputPath)
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

	// 10. Install shell completions
	setupCompletions()

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
			cfg.Services.Databases = append(cfg.Services.Databases, config.DatabaseService{
				Service: ds.Name,
				Type:    ds.ServiceType,
				DBName:  ds.DBName,
				DBUser:  ds.DBUser,
			})
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

// setupCompletions detects the current shell and installs mf completions.
func setupCompletions() {
	shell := filepath.Base(os.Getenv("SHELL"))

	switch shell {
	case "zsh":
		installZshCompletions()
	case "bash":
		installBashCompletions()
	case "fish":
		installFishCompletions()
	default:
		fmt.Printf("\n💡 To enable tab completions, run: mf completion %s --help\n", shell)
	}
}

func installZshCompletions() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	// Prefer Homebrew's site-functions dir — already in fpath, no .zshrc changes needed.
	dest, noRcNeeded := zshCompletionDest(home)

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		fmt.Printf("\n⚠️  Could not create completions dir: %v\n", err)
		return
	}

	var buf bytes.Buffer
	if err := rootCmd.GenZshCompletion(&buf); err != nil {
		return
	}
	if err := os.WriteFile(dest, buf.Bytes(), 0644); err != nil {
		fmt.Printf("\n⚠️  Could not write completion file: %v\n", err)
		return
	}

	fmt.Printf("\n✅ Installed zsh completions → %s\n", dest)

	if noRcNeeded {
		fmt.Println("   No shell config changes needed. Reload your shell to activate.")
		return
	}

	// ~/.zsh/completions fallback: check if fpath entry is already in ~/.zshrc
	compDir := filepath.Dir(dest)
	zshrc := filepath.Join(home, ".zshrc")
	if !fileContains(zshrc, compDir) {
		fmt.Printf("   Add to ~/.zshrc:\n")
		fmt.Printf("     fpath=(%s $fpath)\n", compDir)
		fmt.Printf("     autoload -U compinit && compinit\n")
		fmt.Printf("   Then: source ~/.zshrc\n")
	} else {
		fmt.Println("   Reload your shell or run: source ~/.zshrc")
	}
}

// zshCompletionDest returns the best destination for the zsh completion file.
// It prefers Homebrew's site-functions (already in fpath, no .zshrc edits needed).
// Returns (path, noRcNeeded).
func zshCompletionDest(home string) (string, bool) {
	brewPrefixes := []string{"/opt/homebrew", "/usr/local"}
	for _, prefix := range brewPrefixes {
		dir := filepath.Join(prefix, "share", "zsh", "site-functions")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return filepath.Join(dir, "_mf"), true
		}
	}
	return filepath.Join(home, ".zsh", "completions", "_mf"), false
}

func installBashCompletions() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	compDir := filepath.Join(home, ".bash_completion.d")
	if err := os.MkdirAll(compDir, 0755); err != nil {
		fmt.Printf("\n⚠️  Could not create %s: %v\n", compDir, err)
		return
	}

	dest := filepath.Join(compDir, "mf")
	var buf bytes.Buffer
	if err := rootCmd.GenBashCompletion(&buf); err != nil {
		return
	}
	if err := os.WriteFile(dest, buf.Bytes(), 0644); err != nil {
		fmt.Printf("\n⚠️  Could not write completion file: %v\n", err)
		return
	}

	fmt.Printf("\n✅ Installed bash completions → %s\n", dest)

	bashrc := filepath.Join(home, ".bashrc")
	sourceLine := fmt.Sprintf("source %s", dest)
	if !fileContains(bashrc, sourceLine) {
		fmt.Printf("   Add to ~/.bashrc:\n")
		fmt.Printf("     %s\n", sourceLine)
		fmt.Printf("   Then: source ~/.bashrc\n")
	} else {
		fmt.Println("   Reload your shell or run: source ~/.bashrc")
	}
}

func installFishCompletions() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	compDir := filepath.Join(home, ".config", "fish", "completions")
	if err := os.MkdirAll(compDir, 0755); err != nil {
		fmt.Printf("\n⚠️  Could not create %s: %v\n", compDir, err)
		return
	}

	dest := filepath.Join(compDir, "mf.fish")
	var buf bytes.Buffer
	if err := rootCmd.GenFishCompletion(&buf, true); err != nil {
		return
	}
	if err := os.WriteFile(dest, buf.Bytes(), 0644); err != nil {
		fmt.Printf("\n⚠️  Could not write completion file: %v\n", err)
		return
	}

	fmt.Printf("\n✅ Installed fish completions → %s\n", dest)
	fmt.Println("   Completions are active in new shell sessions.")
}

// fileContains reports whether the file at path contains the given substring.
func fileContains(path, substr string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), substr)
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
