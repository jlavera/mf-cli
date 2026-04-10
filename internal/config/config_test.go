package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mf.yaml")
	content := `project: test-project
services:
  - name: web
    type: python
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Project != "test-project" {
		t.Errorf("expected project 'test-project', got %q", cfg.Project)
	}
	if cfg.Backend() != "web" {
		t.Errorf("expected backend 'web', got %q", cfg.Backend())
	}
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

func TestLoadMinimalAddsDefaultBackend(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mf.yaml")
	content := `project: bare
services: []
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Backend() != "web" {
		t.Errorf("expected default backend 'web', got %q", cfg.Backend())
	}
}

func TestLoadFullConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mf.yaml")
	content := `project: my-project
compose_file: compose.yml
services:
  - name: api
    type: python
  - name: postgres
    type: postgres
    db_name: mydb
    db_user: admin
  - name: cache
    type: redis
  - name: worker1
    type: celery_worker
  - name: worker2
    type: celery_beat
  - name: monitor
    type: flower
  - name: frontend
    type: nodejs
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
	if cfg.Backend() != "api" {
		t.Errorf("backend: got %q", cfg.Backend())
	}
	workers := cfg.Workers()
	if len(workers) != 2 {
		t.Errorf("workers: expected 2, got %d", len(workers))
	}
	dbs := cfg.Databases()
	if len(dbs) != 1 {
		t.Errorf("databases: expected 1, got %d", len(dbs))
	}
	if dbs[0].Type != "postgres" {
		t.Errorf("db type: got %q", dbs[0].Type)
	}
	if dbs[0].DBName != "mydb" {
		t.Errorf("db name: got %q", dbs[0].DBName)
	}
	if cfg.Redis() != "cache" {
		t.Errorf("redis: got %q", cfg.Redis())
	}
	if cfg.Flower() != "monitor" {
		t.Errorf("flower: got %q", cfg.Flower())
	}

	projs := cfg.NodeJSProjects()
	if len(projs) != 1 {
		t.Fatalf("nodejs projects: expected 1, got %d", len(projs))
	}
	if projs[0].Path != "./client" {
		t.Errorf("nodejs path: got %q", projs[0].Path)
	}
	if projs[0].PackageManager != "yarn" {
		t.Errorf("nodejs package_manager: got %q", projs[0].PackageManager)
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
		Services: []Service{
			{Name: "web", Type: "python"},
			{Name: "db", Type: "postgres", DBName: "testdb", DBUser: "testuser"},
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
	dbs := loaded.Databases()
	if len(dbs) != 1 || dbs[0].DBName != "testdb" {
		t.Errorf("roundtrip db name: got %+v", dbs)
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

func TestPackageManagerDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mf.yaml")
	content := `project: test
services:
  - name: frontend
    type: nodejs
    path: ./frontend
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	projs := cfg.NodeJSProjects()
	if len(projs) != 1 {
		t.Fatalf("expected 1 nodejs project, got %d", len(projs))
	}
	if projs[0].PackageManager != "npm" {
		t.Errorf("expected default package_manager 'npm', got %q", projs[0].PackageManager)
	}
}
