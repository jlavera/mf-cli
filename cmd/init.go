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
	"github.com/jlavera/mf-cli/internal/nodejs"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scan a docker-compose file and generate mf.yaml",
	Long: `Scans your docker-compose file, detects services and generates,
an mf.yaml configuration file for the project.

By default, looks for docker-compose.yml/yaml or compose.yml/yaml
in the current directory. Use --file to specify a different path.`,
	Annotations: map[string]string{"skipConfig": "true"},
	RunE:        runInit,
}

var (
	initFile    string
	initEnvFile string
	initForce   bool
)

func init() {
	initCmd.Flags().StringVarP(&initFile, "file", "f", "", "path to docker-compose file")
	initCmd.Flags().StringVarP(&initEnvFile, "env-file", "e", ".env", "path to env file passed to docker-compose --env-file")
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
	detected := compose.ClassifyServices(cf, filepath.Dir(composePath))

	// 5. Derive project name from directory
	cwd, _ := os.Getwd()
	projectName := filepath.Base(cwd)

	// 6. Build config from detected services
	newCfg := buildConfig(projectName, filepath.Base(composePath), detected)
	newCfg.EnvFile = initEnvFile

	// 7. Detect frontend/e2e paths from filesystem
	detectProjectPaths(cwd, &newCfg)

	// 8. Write config
	if err := config.Write(outputPath, &newCfg); err != nil {
		return err
	}

	// 9. Print summary
	printSummary(composePath, detected, newCfg, outputPath)

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
		svc := config.Service{
			Name: ds.Name,
			Type: ds.ServiceType,
		}

		svc.DBName = ds.DBName
		svc.DBUser = ds.DBUser

		// If the service has a non-root build context, record it as a local project path.
		if ds.BuildCtx != "" && ds.BuildCtx != "." {
			svc.Path = "./" + strings.TrimPrefix(ds.BuildCtx, "./")
		}

		cfg.Services = append(cfg.Services, svc)
	}

	return cfg
}

// detectProjectPaths scans the filesystem for Node.js and e2e directories.
// For each nodejs service, it tries to locate a package.json in common
// locations (./<name>, ./apps/<name>, ./packages/<name>) and fills in
// path/package_manager. Also scans well-known top-level directories for
// additional Node.js projects not yet in the service list.
func detectProjectPaths(cwd string, cfg *config.Config) {
	// First pass: for each existing nodejs service, try to find its package.json
	for i := range cfg.Services {
		svc := &cfg.Services[i]
		if svc.Path != "" {
			continue
		}
		if svc.Type != "nodejs" {
			continue
		}
		candidates := []string{
			svc.Name,
			"apps/" + svc.Name,
			"packages/" + svc.Name,
		}
		for _, dir := range candidates {
			absPath := filepath.Join(cwd, dir)
			if nodejs.HasPackageJSON(absPath) {
				svc.Path = "./" + dir
				svc.PackageManager = nodejs.DetectPackageManager(absPath)
				break
			}
		}
	}

	// Second pass: scan well-known top-level directories for projects not yet listed
	nodeDirs := []string{"frontend", "client", "web", "app", "api", "server", "backend"}
	for _, dir := range nodeDirs {
		absPath := filepath.Join(cwd, dir)
		if !nodejs.HasPackageJSON(absPath) {
			continue
		}
		pm := nodejs.DetectPackageManager(absPath)
		if svc := cfg.FindService(dir); svc != nil {
			if svc.Path == "" {
				svc.Path = "./" + dir
			}
			if svc.PackageManager == "" {
				svc.PackageManager = pm
			}
			if svc.Type == "" {
				svc.Type = "nodejs"
			}
		} else {
			cfg.Services = append(cfg.Services, config.Service{
				Name:           dir,
				Type:           "nodejs",
				Path:           "./" + dir,
				PackageManager: pm,
			})
		}
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
func printSummary(composePath string, detected []DetectedService, newCfg config.Config, outputPath string) {
	fmt.Printf("\n✅ Scanned %s — found %d services\n\n", filepath.Base(composePath), len(detected))

	maxName := 0
	for _, ds := range detected {
		if len(ds.Name) > maxName {
			maxName = len(ds.Name)
		}
	}

	for _, ds := range detected {
		extra := ""
		switch ds.ServiceType {
		case "postgres", "mysql", "mongo":
			parts := []string{}
			if ds.DBName != "" {
				parts = append(parts, fmt.Sprintf("db: %s", ds.DBName))
			}
			if ds.DBUser != "" {
				parts = append(parts, fmt.Sprintf("user: %s", ds.DBUser))
			}
			if len(parts) > 0 {
				extra = fmt.Sprintf(" (%s)", strings.Join(parts, ", "))
			}
		}

		svcType := ds.ServiceType
		if svcType == "" {
			svcType = "unknown"
		}
		fmt.Printf("  %-*s → %s%s\n", maxName, ds.Name, svcType, extra)
	}

	fmt.Printf("\n✅ Generated %s\n", outputPath)
}

// DetectedService is re-exported for use in this package.
type DetectedService = compose.DetectedService
