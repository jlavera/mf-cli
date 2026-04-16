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
		"up", "stop", "build", "down", "logs", "bounce", "clean", "rebuild",
		"shell", "psql", "redis-cli",
		"celery", "flower",
		"run",
		"test",
		"e2e",
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
	if !strings.Contains(config, "name: web") {
		t.Error("config missing 'name: web'")
	}
	if !strings.Contains(config, "type: postgres") {
		t.Error("config missing 'type: postgres'")
	}
	if !strings.Contains(config, "db_name: testdb") {
		t.Error("config missing 'db_name: testdb'")
	}
	if !strings.Contains(config, "db_user: testuser") {
		t.Error("config missing 'db_user: testuser'")
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

// TestServiceFilterHelp verifies that commands supporting service filtering
// advertise [services...] in their usage line.
func TestServiceFilterHelp(t *testing.T) {
	cmds := []string{"up", "down", "stop", "logs", "build", "bounce", "rebuild"}
	for _, name := range cmds {
		t.Run(name, func(t *testing.T) {
			stdout, _, err := runMF("", name, "--help")
			if err != nil {
				t.Fatalf("mf %s --help failed: %v", name, err)
			}
			if !strings.Contains(stdout, "[services...]") {
				t.Errorf("mf %s --help missing [services...] in usage", name)
			}
		})
	}
}

// TestServiceArgAccepted verifies that passing a service name arg to each
// service-filter command is accepted by the CLI (fails with missing mf.yaml,
// not with an "unknown argument" or parse error).
func TestServiceArgAccepted(t *testing.T) {
	dir := t.TempDir()
	cmds := [][]string{
		{"up", "web"},
		{"down", "web"},
		{"stop", "web"},
		{"logs", "web"},
		{"build", "web"},
		{"bounce", "web"},
		{"rebuild", "web"},
	}
	for _, args := range cmds {
		t.Run(args[0], func(t *testing.T) {
			_, stderr, err := runMF(dir, args...)
			if err == nil {
				t.Fatalf("mf %v expected error (no mf.yaml), got none", args)
			}
			if !strings.Contains(stderr, "mf.yaml") {
				t.Errorf("mf %v: expected mf.yaml error, got: %s", args, stderr)
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

// TestShellPassthroughHelp verifies that commands with DisableFlagParsing
// still handle --help correctly (even without an mf.yaml) and advertise the
// extra-args passthrough in their usage.
func TestShellPassthroughHelp(t *testing.T) {
	dir := t.TempDir() // no mf.yaml on purpose

	cases := []string{"psql", "shell", "redis-cli"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			stdout, stderr, err := runMF(dir, name, "--help")
			if err != nil {
				t.Fatalf("mf %s --help failed: %v\nstderr: %s", name, err, stderr)
			}
			if !strings.Contains(stdout, "-- extra-args") {
				t.Errorf("mf %s --help should document the `--` passthrough, got:\n%s", name, stdout)
			}
		})
	}
}

// TestPsqlUnknownServiceError verifies that a non-flag first arg that doesn't
// match any configured DB service produces a helpful error, while a flag-like
// first arg is accepted (passthrough to psql).
func TestPsqlUnknownServiceError(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "mf.yaml"), []byte(`project: test
compose_file: docker-compose.yml
services:
  - name: db
    type: postgres
    db_name: appdb
    db_user: appuser
`), 0644)

	// Non-flag, non-service first arg should error clearly.
	_, stderr, err := runMF(dir, "psql", "nosuch")
	if err == nil {
		t.Fatal("expected error for unknown service")
	}
	if !strings.Contains(stderr, "unknown service") {
		t.Errorf("expected 'unknown service' error, got: %s", stderr)
	}

	// Flag-like first arg should NOT trip the service check (it reaches the
	// docker exec stage which fails in this test env — that's fine, we just
	// want to assert it got past arg parsing).
	_, stderr, err = runMF(dir, "psql", "-c", `\dt`)
	if err == nil {
		return // unlikely but ok
	}
	if strings.Contains(stderr, "unknown service") {
		t.Errorf("flag-like first arg should not be treated as a service name, got: %s", stderr)
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

func TestInitDetectsNodeJSProjects(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(`services:
  web:
    build: .
    ports:
      - "8000:8000"
`), 0644)

	frontendDir := filepath.Join(dir, "frontend")
	os.MkdirAll(frontendDir, 0755)
	os.WriteFile(filepath.Join(frontendDir, "package.json"), []byte(`{"name":"fe","scripts":{"dev":"vite","build":"vite build"}}`), 0644)

	_, stderr, err := runMF(dir, "init")
	if err != nil {
		t.Fatalf("mf init failed: %v\nstderr: %s", err, stderr)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "mf.yaml"))
	config := string(data)
	if !strings.Contains(config, "name: frontend") {
		t.Error("mf.yaml missing frontend service entry")
	}
	if !strings.Contains(config, "path: ./frontend") {
		t.Error("mf.yaml missing path for frontend service")
	}
}

func TestRunHelp(t *testing.T) {
	stdout, _, err := runMF("", "run", "--help")
	if err != nil {
		t.Fatalf("mf run --help failed: %v", err)
	}
	for _, s := range []string{"service", "script", "install"} {
		if !strings.Contains(stdout, s) {
			t.Errorf("run --help missing %q", s)
		}
	}
}

func TestRunProjectCompletion(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "mf.yaml"), []byte(`project: test
compose_file: docker-compose.yml
services:
  - name: web
    type: python
  - name: frontend
    type: nodejs
    path: ./frontend
    package_manager: npm
  - name: api
    type: nodejs
    path: ./api
    package_manager: npm
`), 0644)

	stdout, _, err := runMF(dir, "__complete", "run", "")
	if err != nil {
		t.Fatalf("completion failed: %v", err)
	}
	if !strings.Contains(stdout, "frontend") {
		t.Errorf("expected 'frontend' in project completion, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "api") {
		t.Errorf("expected 'api' in project completion, got:\n%s", stdout)
	}
}

func TestRunScriptCompletion(t *testing.T) {
	dir := t.TempDir()

	apiDir := filepath.Join(dir, "api")
	os.MkdirAll(apiDir, 0755)
	os.WriteFile(filepath.Join(apiDir, "package.json"), []byte(`{
		"scripts": {"test": "jest", "test:watch": "jest --watch", "build": "tsc"}
	}`), 0644)

	os.WriteFile(filepath.Join(dir, "mf.yaml"), []byte(`project: test
compose_file: docker-compose.yml
services:
  - name: web
    type: python
  - name: api
    type: nodejs
    path: ./api
    package_manager: npm
`), 0644)

	stdout, _, err := runMF(dir, "__complete", "run", "api", "")
	if err != nil {
		t.Fatalf("completion failed: %v", err)
	}
	for _, s := range []string{"test", "test:watch", "build", "install"} {
		if !strings.Contains(stdout, s) {
			t.Errorf("expected %q in script completion, got:\n%s", s, stdout)
		}
	}
}

func TestNodeJSMultipleProjects(t *testing.T) {
	dir := t.TempDir()

	for _, name := range []string{"frontend", "api"} {
		d := filepath.Join(dir, name)
		os.MkdirAll(d, 0755)
		os.WriteFile(filepath.Join(d, "package.json"), []byte(`{"scripts":{"dev":"node .","build":"tsc"}}`), 0644)
	}

	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(`services:
  web:
    build: .
    ports:
      - "8000:8000"
`), 0644)

	_, stderr, err := runMF(dir, "init")
	if err != nil {
		t.Fatalf("mf init failed: %v\nstderr: %s", err, stderr)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "mf.yaml"))
	config := string(data)
	if !strings.Contains(config, "name: frontend") {
		t.Error("mf.yaml missing nodejs entry 'frontend'")
	}
	if !strings.Contains(config, "name: api") {
		t.Error("mf.yaml missing nodejs entry 'api'")
	}
}
