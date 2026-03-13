package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestMain builds the mf binary once before all tests.
var binaryPath string

func TestMain(m *testing.M) {
	// Build the binary into a temp location
	tmp, err := os.MkdirTemp("", "mf-test-*")
	if err != nil {
		panic(err)
	}
	binaryPath = filepath.Join(tmp, "mf")

	cmd := exec.Command("go", "build", "-o", binaryPath, "..")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("failed to build mf binary: " + err.Error())
	}

	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

// runMF executes the mf binary with the given args and returns stdout, stderr, error.
func runMF(dir string, args ...string) (string, string, error) {
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func TestRootHelp(t *testing.T) {
	stdout, _, err := runMF("", "--help")
	if err != nil {
		t.Fatalf("mf --help failed: %v", err)
	}

	expectedCommands := []string{
		"up", "stop", "build", "down", "logs", "restart", "clean", "rebuild",
		"shell", "psql", "redis-cli",
		"celery", "flower",
		"test",
		"frontend", "e2e",
		"format", "lint", "pre-commit",
		"debug",
		"init",
		"completion",
	}

	for _, cmd := range expectedCommands {
		if !strings.Contains(stdout, cmd) {
			t.Errorf("help output missing command %q", cmd)
		}
	}
}

func TestInitHelp(t *testing.T) {
	stdout, _, err := runMF("", "init", "--help")
	if err != nil {
		t.Fatalf("mf init --help failed: %v", err)
	}

	if !strings.Contains(stdout, "--file") {
		t.Error("init help missing --file flag")
	}
	if !strings.Contains(stdout, "--force") {
		t.Error("init help missing --force flag")
	}
	if !strings.Contains(stdout, "docker-compose") {
		t.Error("init help missing docker-compose description")
	}
}

func TestInitGeneratesConfig(t *testing.T) {
	// Create a temp dir with a docker-compose.yml
	dir := t.TempDir()
	composeContent := `services:
  web:
    build: .
    ports:
      - "8000:8000"
  db:
    image: postgres:15
    environment:
      POSTGRES_DB: testdb
      POSTGRES_USER: testuser
  redis:
    image: redis:7-alpine
`
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(composeContent), 0644)

	stdout, stderr, err := runMF(dir, "init")
	if err != nil {
		t.Fatalf("mf init failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	// Check summary output
	if !strings.Contains(stdout, "found 3 services") {
		t.Errorf("expected '3 services' in output, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Generated mf.yaml") {
		t.Errorf("expected 'Generated mf.yaml' in output, got:\n%s", stdout)
	}

	// Check config file was created
	configPath := filepath.Join(dir, "mf.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("mf.yaml not created: %v", err)
	}

	config := string(data)
	if !strings.Contains(config, "backend: web") {
		t.Error("config missing 'backend: web'")
	}
	if !strings.Contains(config, "type: postgres") {
		t.Error("config missing 'type: postgres'")
	}
	if !strings.Contains(config, "name: testdb") {
		t.Error("config missing 'name: testdb'")
	}
	if !strings.Contains(config, "user: testuser") {
		t.Error("config missing 'user: testuser'")
	}
}

func TestInitWithFileFlag(t *testing.T) {
	dir := t.TempDir()
	composeContent := `services:
  app:
    build: .
    ports:
      - "3000:3000"
`
	customPath := filepath.Join(dir, "custom-compose.yml")
	os.WriteFile(customPath, []byte(composeContent), 0644)

	stdout, _, err := runMF(dir, "init", "--file", customPath)
	if err != nil {
		t.Fatalf("mf init --file failed: %v", err)
	}

	if !strings.Contains(stdout, "found 1 services") {
		t.Errorf("expected '1 services' in output, got:\n%s", stdout)
	}
}

func TestInitNoComposeFile(t *testing.T) {
	dir := t.TempDir()
	_, stderr, err := runMF(dir, "init")
	if err == nil {
		t.Fatal("expected error when no compose file exists")
	}

	if !strings.Contains(stderr, "no docker-compose file found") {
		t.Errorf("expected 'no docker-compose file found' error, got:\n%s", stderr)
	}
}

func TestInitForceOverwrite(t *testing.T) {
	dir := t.TempDir()
	composeContent := `services:
  web:
    build: .
    ports:
      - "8000:8000"
`
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(composeContent), 0644)

	// First init
	_, _, err := runMF(dir, "init")
	if err != nil {
		t.Fatalf("first init failed: %v", err)
	}

	// Second init with --force
	stdout, _, err := runMF(dir, "init", "--force")
	if err != nil {
		t.Fatalf("init --force failed: %v", err)
	}
	if !strings.Contains(stdout, "Generated mf.yaml") {
		t.Error("force overwrite didn't generate config")
	}
}

func TestSubcommandHelp(t *testing.T) {
	subcommands := []struct {
		args     []string
		contains []string
	}{
		{[]string{"celery", "--help"}, []string{"start", "stop", "restart", "logs"}},
		{[]string{"frontend", "--help"}, []string{"install", "dev", "build", "lint", "type-check"}},
		{[]string{"e2e", "--help"}, []string{"install", "run", "ui", "headed", "debug", "report"}},
		{[]string{"debug", "--help"}, []string{"check", "clean"}},
		{[]string{"test", "--help"}, []string{"--file", "--method", "--debug"}},
		{[]string{"build", "--help"}, []string{"--no-cache"}},
		{[]string{"format", "--help"}, []string{"--check"}},
		{[]string{"pre-commit", "--help"}, []string{"--all", "--local"}},
	}

	for _, tc := range subcommands {
		t.Run(strings.Join(tc.args, "_"), func(t *testing.T) {
			stdout, _, err := runMF("", tc.args...)
			if err != nil {
				t.Fatalf("%v failed: %v", tc.args, err)
			}
			for _, s := range tc.contains {
				if !strings.Contains(stdout, s) {
					t.Errorf("help for %v missing %q", tc.args, s)
				}
			}
		})
	}
}

func TestNoConfigError(t *testing.T) {
	dir := t.TempDir()
	// Commands that require config should fail gracefully
	_, stderr, err := runMF(dir, "up")
	if err == nil {
		t.Fatal("expected error when no mf.yaml exists")
	}
	if !strings.Contains(stderr, "mf.yaml") {
		t.Errorf("expected mf.yaml error message, got:\n%s", stderr)
	}
}

func TestServiceNameCompletion(t *testing.T) {
	dir := t.TempDir()

	// Create compose file + mf.yaml
	composeContent := `services:
  web:
    build: .
    ports:
      - "8000:8000"
  db:
    image: postgres:15
  redis:
    image: redis:7
`
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(composeContent), 0644)

	// Run init to create mf.yaml
	_, _, err := runMF(dir, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Test service completion for 'up'
	stdout, _, err := runMF(dir, "__complete", "up", "")
	if err != nil {
		t.Fatalf("completion failed: %v", err)
	}

	for _, svc := range []string{"web", "db", "redis"} {
		if !strings.Contains(stdout, svc) {
			t.Errorf("completion for 'up' missing service %q\nstdout: %s", svc, stdout)
		}
	}

	// Verify ShellCompDirectiveNoFileComp is set (directive :4)
	if !strings.Contains(stdout, ":4") {
		t.Error("expected ShellCompDirectiveNoFileComp (:4) in completion output")
	}
}

func TestServiceCompletionFiltersUsedArgs(t *testing.T) {
	dir := t.TempDir()

	composeContent := `services:
  web:
    build: .
  db:
    image: postgres:15
  redis:
    image: redis:7
`
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(composeContent), 0644)
	runMF(dir, "init")

	// Complete 'up web <TAB>' — web should be filtered out
	stdout, _, _ := runMF(dir, "__complete", "up", "web", "")
	if strings.Contains(stdout, "\nweb\n") {
		t.Error("completion should filter out already-used service 'web'")
	}
	// db and redis should still be present
	if !strings.Contains(stdout, "db") || !strings.Contains(stdout, "redis") {
		t.Errorf("expected db and redis in completion, got:\n%s", stdout)
	}
}

func TestShellCompletionSingleArg(t *testing.T) {
	dir := t.TempDir()

	composeContent := `services:
  web:
    build: .
  db:
    image: postgres:15
`
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(composeContent), 0644)
	runMF(dir, "init")

	// Complete 'shell web <TAB>' — should return nothing (only 1 arg)
	stdout, _, _ := runMF(dir, "__complete", "shell", "web", "")
	if strings.Contains(stdout, "db") {
		t.Error("shell completion should not suggest more services after first arg")
	}
}

func TestCompletionGenerationDoesNotHang(t *testing.T) {
	// Test that 'mf completion bash/zsh/fish' works in directories
	// both with and without mf.yaml — should never hang or error.
	shells := []string{"bash", "zsh", "fish"}

	// Without config
	dirNoConfig := t.TempDir()
	for _, shell := range shells {
		t.Run("no_config_"+shell, func(t *testing.T) {
			stdout, stderr, err := runMF(dirNoConfig, "completion", shell)
			if err != nil {
				t.Fatalf("completion %s failed: %v\nstderr: %s", shell, err, stderr)
			}
			if len(stdout) < 100 {
				t.Errorf("completion %s output too short (%d bytes)", shell, len(stdout))
			}
		})
	}

	// With config
	dirWithConfig := t.TempDir()
	os.WriteFile(filepath.Join(dirWithConfig, "docker-compose.yml"), []byte(`services:
  web:
    build: .
`), 0644)
	runMF(dirWithConfig, "init")

	for _, shell := range shells {
		t.Run("with_config_"+shell, func(t *testing.T) {
			stdout, stderr, err := runMF(dirWithConfig, "completion", shell)
			if err != nil {
				t.Fatalf("completion %s failed: %v\nstderr: %s", shell, err, stderr)
			}
			if len(stdout) < 100 {
				t.Errorf("completion %s output too short (%d bytes)", shell, len(stdout))
			}
		})
	}
}

func TestCompletionWithEnvVarCompose(t *testing.T) {
	// Test that completion works with compose files using ${VAR:-default} syntax
	dir := t.TempDir()
	composeContent := `services:
  api:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "3000:3000"
    depends_on:
      postgres:
        condition: service_healthy
  postgres:
    image: postgres:17-alpine
    environment:
      POSTGRES_USER: ${DB_USER:-postgres}
      POSTGRES_PASSWORD: ${DB_PASSWORD:-postgres}
      POSTGRES_DB: ${DB_NAME:-mydb}
`
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(composeContent), 0644)
	runMF(dir, "init")

	// __complete should return service names
	stdout, _, err := runMF(dir, "__complete", "up", "")
	if err != nil {
		t.Fatalf("completion failed: %v", err)
	}
	if !strings.Contains(stdout, "api") || !strings.Contains(stdout, "postgres") {
		t.Errorf("expected api and postgres in completions, got:\n%s", stdout)
	}

	// completion zsh should work
	stdout, _, err = runMF(dir, "completion", "zsh")
	if err != nil {
		t.Fatalf("completion zsh failed: %v", err)
	}
	if len(stdout) < 100 {
		t.Error("completion zsh output too short")
	}
}

func TestFileCompletionDirectives(t *testing.T) {
	dir := t.TempDir()

	composeContent := `services:
  web:
    build: .
`
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(composeContent), 0644)
	runMF(dir, "init")

	// Test -f completion filters to .py files
	stdout, _, _ := runMF(dir, "__complete", "test", "--file", "")
	if !strings.Contains(stdout, "py") {
		t.Error("test --file completion should suggest .py extension filter")
	}
	// Directive :8 = ShellCompDirectiveFilterFileExt
	if !strings.Contains(stdout, ":8") {
		t.Error("expected ShellCompDirectiveFilterFileExt (:8)")
	}

	// Init --file completion filters to yml/yaml
	stdout, _, _ = runMF(dir, "__complete", "init", "--file", "")
	if !strings.Contains(stdout, "yml") || !strings.Contains(stdout, "yaml") {
		t.Error("init --file completion should suggest yml/yaml extension filter")
	}
}
