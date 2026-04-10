package cmd

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/jlavera/mf-cli/internal/config"
	"gopkg.in/yaml.v3"
)

// TestInitFixtures runs mf init against each fixture in testdata/ and
// compares the generated mf.yaml against the fixture's expected.yaml.
//
// Fixture layout:
//
//	cmd/testdata/<name>/docker-compose.yml   — compose input
//	cmd/testdata/<name>/expected.yaml        — expected mf.yaml output (without header)
//	cmd/testdata/<name>/**                   — any supporting files (package.json, etc.)
func TestInitFixtures(t *testing.T) {
	fixtures := []string{
		"postgres-only",
		"python-project",
		"nodejs-project",
		"react-project",
		"full-stack",
		"monorepo",
	}

	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			dir := setupFixtureDir(t, fixture, "testproject")

			stdout, stderr, err := runMF(dir, "init", "--force")
			if err != nil {
				t.Fatalf("mf init failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
			}

			var got config.Config
			genData, err := os.ReadFile(filepath.Join(dir, "mf.yaml"))
			if err != nil {
				t.Fatalf("mf.yaml not created: %v", err)
			}
			if err := yaml.Unmarshal(genData, &got); err != nil {
				t.Fatalf("parse mf.yaml: %v", err)
			}

			var want config.Config
			expData, err := os.ReadFile(filepath.Join("testdata", fixture, "expected.yaml"))
			if err != nil {
				t.Fatalf("expected.yaml not found: %v", err)
			}
			if err := yaml.Unmarshal(expData, &want); err != nil {
				t.Fatalf("parse expected.yaml: %v", err)
			}

			if got.Project != want.Project {
				t.Errorf("project: got %q, want %q", got.Project, want.Project)
			}
			if got.ComposeFile != want.ComposeFile {
				t.Errorf("compose_file: got %q, want %q", got.ComposeFile, want.ComposeFile)
			}
			assertServices(t, got.Services, want.Services)
		})
	}
}

// setupFixtureDir copies a testdata fixture into a fresh temp dir under a
// known project name so mf derives a predictable project name from it.
func setupFixtureDir(t *testing.T, fixtureName, projectName string) string {
	t.Helper()
	parent := t.TempDir()
	projectDir := filepath.Join(parent, projectName)
	src := filepath.Join("testdata", fixtureName)
	if err := copyDirRecursive(src, projectDir); err != nil {
		t.Fatalf("copy fixture %s: %v", fixtureName, err)
	}
	return projectDir
}

func copyDirRecursive(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

// assertServices compares two []Service slices. Services are sorted by name
// before comparison since map iteration in compose classification is
// non-deterministic.
func assertServices(t *testing.T, got, want []config.Service) {
	t.Helper()

	sortServices := func(s []config.Service) {
		sort.Slice(s, func(i, j int) bool { return s[i].Name < s[j].Name })
	}
	sortServices(got)
	sortServices(want)

	if len(got) != len(want) {
		t.Errorf("services: got %d entries, want %d\n  got:  %+v\n  want: %+v", len(got), len(want), got, want)
		return
	}
	for i, w := range want {
		g := got[i]
		if g.Name != w.Name {
			t.Errorf("services[%d].name: got %q, want %q", i, g.Name, w.Name)
		}
		if g.Type != w.Type {
			t.Errorf("services[%d] (%s) .type: got %q, want %q", i, g.Name, g.Type, w.Type)
		}
		if g.DBName != w.DBName {
			t.Errorf("services[%d] (%s) .db_name: got %q, want %q", i, g.Name, g.DBName, w.DBName)
		}
		if g.DBUser != w.DBUser {
			t.Errorf("services[%d] (%s) .db_user: got %q, want %q", i, g.Name, g.DBUser, w.DBUser)
		}
		if g.Path != w.Path {
			t.Errorf("services[%d] (%s) .path: got %q, want %q", i, g.Name, g.Path, w.Path)
		}
		if g.PackageManager != w.PackageManager {
			t.Errorf("services[%d] (%s) .package_manager: got %q, want %q", i, g.Name, g.PackageManager, w.PackageManager)
		}
	}
}
