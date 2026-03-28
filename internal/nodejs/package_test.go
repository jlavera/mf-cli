package nodejs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writePackageJSON(t *testing.T, dir string, scripts map[string]string) {
	t.Helper()
	data, _ := json.Marshal(map[string]any{"scripts": scripts})
	if err := os.WriteFile(filepath.Join(dir, "package.json"), data, 0644); err != nil {
		t.Fatalf("writePackageJSON: %v", err)
	}
}

func TestReadScripts(t *testing.T) {
	dir := t.TempDir()
	writePackageJSON(t, dir, map[string]string{
		"dev":   "vite",
		"build": "vite build",
		"test":  "vitest",
	})

	scripts, err := ReadScripts(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scripts) != 3 {
		t.Fatalf("expected 3 scripts, got %d", len(scripts))
	}
	if scripts["dev"] != "vite" {
		t.Errorf("expected dev=vite, got %q", scripts["dev"])
	}
}

func TestReadScripts_NoScripts(t *testing.T) {
	dir := t.TempDir()
	data, _ := json.Marshal(map[string]any{"name": "myapp"})
	os.WriteFile(filepath.Join(dir, "package.json"), data, 0644)

	scripts, err := ReadScripts(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scripts) != 0 {
		t.Errorf("expected empty map, got %v", scripts)
	}
}

func TestReadScripts_MissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := ReadScripts(dir)
	if err == nil {
		t.Error("expected error for missing package.json, got nil")
	}
}

func TestReadScripts_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte("not json"), 0644)
	_, err := ReadScripts(dir)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}
