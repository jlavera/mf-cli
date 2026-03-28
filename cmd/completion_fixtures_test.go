package cmd

import (
	"path/filepath"
	"strings"
	"testing"
)

// completionCheck describes one __complete invocation and what to expect.
type completionCheck struct {
	desc    string
	args    []string // everything after "__complete"
	include []string // substrings that must appear in stdout
	exclude []string // substrings that must NOT appear in stdout
}

func TestCompletionFixtures(t *testing.T) {
	fixtures := []struct {
		fixture string
		checks  []completionCheck
	}{
		{
			fixture: "postgres-only",
			checks: []completionCheck{
				{
					desc:    "up lists all services",
					args:    []string{"up", ""},
					include: []string{"db"},
				},
				{
					desc:    "up filters already-used service",
					args:    []string{"up", "db", ""},
					exclude: []string{"db"},
				},
				{
					desc:    "shell lists services",
					args:    []string{"shell", ""},
					include: []string{"db"},
				},
				{
					desc:    "shell allows only one arg",
					args:    []string{"shell", "db", ""},
					exclude: []string{"db"},
				},
				{
					desc:    "psql lists database services",
					args:    []string{"psql", ""},
					include: []string{"db"},
				},
				{
					desc:    "run has no projects",
					args:    []string{"run", ""},
					exclude: []string{"db"},
				},
			},
		},
		{
			fixture: "python-project",
			checks: []completionCheck{
				{
					desc:    "up lists all services",
					args:    []string{"up", ""},
					include: []string{"web", "db"},
				},
				{
					desc:    "up filters already-used service",
					args:    []string{"up", "web", ""},
					include: []string{"db"},
					exclude: []string{"web"},
				},
				{
					desc:    "shell lists services",
					args:    []string{"shell", ""},
					include: []string{"web", "db"},
				},
				{
					desc:    "shell allows only one arg",
					args:    []string{"shell", "web", ""},
					exclude: []string{"web", "db"},
				},
				{
					desc:    "psql lists only database services",
					args:    []string{"psql", ""},
					include: []string{"db"},
					exclude: []string{"web"},
				},
				{
					desc:    "run has no projects",
					args:    []string{"run", ""},
					exclude: []string{"web", "db"},
				},
			},
		},
		{
			fixture: "nodejs-project",
			checks: []completionCheck{
				{
					desc:    "up lists all services",
					args:    []string{"up", ""},
					include: []string{"api", "db"},
				},
				{
					desc:    "psql lists only database services",
					args:    []string{"psql", ""},
					include: []string{"db"},
					exclude: []string{"api"},
				},
				{
					desc:    "run lists nodejs projects",
					args:    []string{"run", ""},
					include: []string{"api"},
				},
				{
					desc:    "run api lists package.json scripts",
					args:    []string{"run", "api", ""},
					include: []string{"start", "dev", "build", "test", "install"},
				},
			},
		},
		{
			fixture: "react-project",
			checks: []completionCheck{
				{
					desc:    "up lists all services",
					args:    []string{"up", ""},
					include: []string{"frontend"},
				},
				{
					desc:    "run lists nodejs projects",
					args:    []string{"run", ""},
					include: []string{"frontend"},
				},
				{
					desc:    "run frontend lists package.json scripts",
					args:    []string{"run", "frontend", ""},
					include: []string{"dev", "build", "preview", "test", "install"},
				},
			},
		},
		{
			fixture: "full-stack",
			checks: []completionCheck{
				{
					desc:    "up lists all compose services",
					args:    []string{"up", ""},
					include: []string{"web", "db", "api", "frontend"},
				},
				{
					desc:    "up filters already-used service",
					args:    []string{"up", "web", ""},
					include: []string{"db", "api", "frontend"},
					exclude: []string{"web"},
				},
				{
					desc:    "shell lists all services",
					args:    []string{"shell", ""},
					include: []string{"web", "db", "api", "frontend"},
				},
				{
					desc:    "shell allows only one arg",
					args:    []string{"shell", "web", ""},
					exclude: []string{"web", "db", "api", "frontend"},
				},
				{
					desc:    "psql lists only database services",
					args:    []string{"psql", ""},
					include: []string{"db"},
					exclude: []string{"web", "api", "frontend"},
				},
				{
					desc:    "run lists nodejs projects",
					args:    []string{"run", ""},
					include: []string{"frontend", "api"},
					exclude: []string{"web", "db"},
				},
				{
					desc:    "run frontend lists package.json scripts",
					args:    []string{"run", "frontend", ""},
					include: []string{"dev", "build", "preview", "test", "install"},
				},
				{
					desc:    "run api lists package.json scripts",
					args:    []string{"run", "api", ""},
					include: []string{"start", "dev", "build", "test", "install"},
				},
			},
		},
		{
		// All 6 compose services should appear in service completions.
			// psql scopes to postgres only. run is empty (no nodejs entries).
			fixture: "monorepo",
			checks: []completionCheck{
				{
					desc:    "up lists all services",
					args:    []string{"up", ""},
					include: []string{"admin", "api", "langgraph", "mcp-gcs-server", "web", "postgres"},
				},
				{
					desc:    "up filters already-used service",
					args:    []string{"up", "api", ""},
					include: []string{"admin", "langgraph", "mcp-gcs-server", "web", "postgres"},
					exclude: []string{"api"},
				},
				{
					desc:    "shell lists all services",
					args:    []string{"shell", ""},
					include: []string{"admin", "api", "langgraph", "mcp-gcs-server", "web", "postgres"},
				},
				{
					desc:    "psql lists only database services",
					args:    []string{"psql", ""},
					include: []string{"postgres"},
					exclude: []string{"admin", "api", "langgraph", "mcp-gcs-server", "web"},
				},
				{
					desc:    "run lists all nodejs services",
					args:    []string{"run", ""},
					include: []string{"admin", "api", "langgraph", "mcp-gcs-server", "web"},
					exclude: []string{"postgres"},
				},
				{
					desc:    "run api lists package.json scripts",
					args:    []string{"run", "api", ""},
					include: []string{"start:dev", "build", "test", "install"},
				},
			},
		},
	}

	for _, fx := range fixtures {
		fx := fx
		t.Run(fx.fixture, func(t *testing.T) {
			dir := setupFixtureDir(t, fx.fixture, "testproject")

			// mf.yaml must exist before completions that read from cfg (psql, run).
			if _, _, err := runMF(dir, "init", "--force"); err != nil {
				t.Fatalf("mf init failed: %v", err)
			}

			for _, chk := range fx.checks {
				chk := chk
				t.Run(chk.desc, func(t *testing.T) {
					args := append([]string{"__complete"}, chk.args...)
					stdout, _, _ := runMF(dir, args...)

					for _, want := range chk.include {
						if !strings.Contains(stdout, want) {
							t.Errorf("expected %q in completion output\nargs: %v\nstdout:\n%s",
								want, chk.args, stdout)
						}
					}
					for _, unwanted := range chk.exclude {
						// Check for whole-word match to avoid false positives
						// (e.g. "web" inside "webserver"). Completions are newline-separated.
						for _, line := range strings.Split(stdout, "\n") {
							if strings.TrimSpace(line) == unwanted {
								t.Errorf("unexpected %q in completion output\nargs: %v\nstdout:\n%s",
									unwanted, chk.args, stdout)
							}
						}
					}
				})
			}
		})
	}
}

// TestCompletionDirective verifies the ShellCompDirectiveNoFileComp flag (:4)
// is set for service completions so the shell doesn't fall back to file completion.
func TestCompletionDirective(t *testing.T) {
	dir := setupFixtureDir(t, "python-project", "testproject")
	if _, _, err := runMF(dir, "init", "--force"); err != nil {
		t.Fatalf("mf init failed: %v", err)
	}

	for _, cmd := range []string{"up", "stop", "build", "logs", "bounce", "rebuild"} {
		stdout, _, _ := runMF(filepath.Join(dir), "__complete", cmd, "")
		if !strings.Contains(stdout, ":4") {
			t.Errorf("%q completion missing ShellCompDirectiveNoFileComp (:4)\nstdout: %s", cmd, stdout)
		}
	}
}
