package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndDefaults(t *testing.T) {
	// Write a minimal config file
	dir := t.TempDir()
	path := filepath.Join(dir, "mf.yaml")
	content := `project: test-project
services:
  backend: web
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Check explicit values
	if cfg.Project != "test-project" {
		t.Errorf("expected project 'test-project', got %q", cfg.Project)
	}
	if cfg.Services.Backend != "web" {
		t.Errorf("expected backend 'web', got %q", cfg.Services.Backend)
	}

	// Check defaults
	if cfg.ComposeFile != "docker-compose.yml" {
		t.Errorf("expected default compose_file 'docker-compose.yml', got %q", cfg.ComposeFile)
	}
	if cfg.Test.Runner != "pytest" {
		t.Errorf("expected default test runner 'pytest', got %q", cfg.Test.Runner)
	}
	if cfg.Test.DebugPort != 5679 {
		t.Errorf("expected default debug port 5679, got %d", cfg.Test.DebugPort)
	}
}

func TestLoadFullConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mf.yaml")
	content := `project: my-project
compose_file: compose.yml
services:
  backend: api
  databases:
    - service: postgres
      type: postgres
      db_name: mydb
      db_user: admin
  redis: cache
  workers:
    - worker1
    - worker2
  flower: monitor
frontend:
  path: ./client
  package_manager: yarn
e2e:
  path: ./e2e
  framework: playwright
  browser: firefox
scripts:
  pre_commit: ./hooks/pre-commit.sh
  format: ./tools/format.sh
  lint: ./tools/lint.sh
  ruff: ./tools/ruff.sh
test:
  runner: pytest
  env:
    ENV: test
    DEBUG: "1"
  debug_port: 9999
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.ComposeFile != "compose.yml" {
		t.Errorf("compose_file: got %q", cfg.ComposeFile)
	}
	if cfg.Services.Backend != "api" {
		t.Errorf("backend: got %q", cfg.Services.Backend)
	}
	if len(cfg.Services.Workers) != 2 {
		t.Errorf("workers: expected 2, got %d", len(cfg.Services.Workers))
	}
	if len(cfg.Services.Databases) != 1 {
		t.Errorf("databases: expected 1, got %d", len(cfg.Services.Databases))
	}
	if cfg.Services.Databases[0].Type != "postgres" {
		t.Errorf("db type: got %q", cfg.Services.Databases[0].Type)
	}
	if cfg.Services.Databases[0].DBName != "mydb" {
		t.Errorf("db name: got %q", cfg.Services.Databases[0].DBName)
	}
	if cfg.Frontend.Path != "./client" {
		t.Errorf("frontend path: got %q", cfg.Frontend.Path)
	}
	if cfg.Frontend.PackageManager != "yarn" {
		t.Errorf("package_manager: got %q", cfg.Frontend.PackageManager)
	}
	if cfg.E2E.Browser != "firefox" {
		t.Errorf("e2e browser: got %q", cfg.E2E.Browser)
	}
	if cfg.Test.DebugPort != 9999 {
		t.Errorf("debug port: got %d", cfg.Test.DebugPort)
	}
	if cfg.Test.Env["DEBUG"] != "1" {
		t.Errorf("test env DEBUG: got %q", cfg.Test.Env["DEBUG"])
	}
}

func TestLoadMissing(t *testing.T) {
	_, err := Load("/nonexistent/mf.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mf.yaml")
	if err := os.WriteFile(path, []byte(":::invalid"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestWriteAndReadback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mf.yaml")

	cfg := &Config{
		Project:     "roundtrip",
		ComposeFile: "docker-compose.yml",
		Services: ServicesConfig{
			Backend: "web",
			Databases: []DatabaseService{
				{Service: "db", Type: "postgres", DBName: "testdb", DBUser: "testuser"},
			},
		},
	}

	if err := Write(path, cfg); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load after write failed: %v", err)
	}

	if loaded.Project != "roundtrip" {
		t.Errorf("roundtrip project: got %q", loaded.Project)
	}
	if len(loaded.Services.Databases) != 1 || loaded.Services.Databases[0].DBName != "testdb" {
		t.Errorf("roundtrip db name: got %+v", loaded.Services.Databases)
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mf.yaml")

	if Exists(path) {
		t.Error("expected Exists=false for non-existent file")
	}

	os.WriteFile(path, []byte("project: x"), 0644)
	if !Exists(path) {
		t.Error("expected Exists=true after creating file")
	}
}
