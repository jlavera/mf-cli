package compose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jlavera/mf-cli/internal/config"
)

// setupFakeCompose creates a fake docker-compose binary that records its args
// line-by-line into a file, and returns a Compose instance + that args file path.
func setupFakeCompose(t *testing.T) (*Compose, string) {
	t.Helper()
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args.txt")

	// Write each arg on its own line so we can parse them unambiguously.
	script := "#!/bin/sh\nfor a in \"$@\"; do echo \"$a\"; done > " + argsFile + "\n"
	if err := os.WriteFile(filepath.Join(dir, "docker-compose"), []byte(script), 0755); err != nil {
		t.Fatalf("create fake docker-compose: %v", err)
	}

	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))

	cfg := &config.Config{ComposeFile: "docker-compose.yml"}
	return New(cfg), argsFile
}

func readArgs(t *testing.T, argsFile string) []string {
	t.Helper()
	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read args file: %v", err)
	}
	var args []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line != "" {
			args = append(args, line)
		}
	}
	return args
}

func assertArgs(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("args mismatch\n got:  %v\n want: %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("args[%d] mismatch: got %q, want %q\nfull got:  %v\nfull want: %v", i, got[i], want[i], got, want)
		}
	}
}

// base args that every command will include from baseArgs()
// e.g. ["-f", "docker-compose.yml", <subcommand>, ...]

func TestDown_NoServices(t *testing.T) {
	c, argsFile := setupFakeCompose(t)
	if err := c.Down(); err != nil {
		t.Fatalf("Down() error: %v", err)
	}
	assertArgs(t, readArgs(t, argsFile), []string{"-f", "docker-compose.yml", "down"})
}

func TestDown_SingleService(t *testing.T) {
	c, argsFile := setupFakeCompose(t)
	if err := c.Down("web"); err != nil {
		t.Fatalf("Down(web) error: %v", err)
	}
	assertArgs(t, readArgs(t, argsFile), []string{"-f", "docker-compose.yml", "down", "web"})
}

func TestDown_MultipleServices(t *testing.T) {
	c, argsFile := setupFakeCompose(t)
	if err := c.Down("web", "db"); err != nil {
		t.Fatalf("Down(web,db) error: %v", err)
	}
	assertArgs(t, readArgs(t, argsFile), []string{"-f", "docker-compose.yml", "down", "web", "db"})
}

func TestStop_NoServices(t *testing.T) {
	c, argsFile := setupFakeCompose(t)
	if err := c.Stop(); err != nil {
		t.Fatalf("Stop() error: %v", err)
	}
	assertArgs(t, readArgs(t, argsFile), []string{"-f", "docker-compose.yml", "stop"})
}

func TestStop_SingleService(t *testing.T) {
	c, argsFile := setupFakeCompose(t)
	if err := c.Stop("web"); err != nil {
		t.Fatalf("Stop(web) error: %v", err)
	}
	assertArgs(t, readArgs(t, argsFile), []string{"-f", "docker-compose.yml", "stop", "web"})
}

func TestStop_MultipleServices(t *testing.T) {
	c, argsFile := setupFakeCompose(t)
	if err := c.Stop("web", "redis"); err != nil {
		t.Fatalf("Stop(web,redis) error: %v", err)
	}
	assertArgs(t, readArgs(t, argsFile), []string{"-f", "docker-compose.yml", "stop", "web", "redis"})
}

func TestUp_NoServices(t *testing.T) {
	c, argsFile := setupFakeCompose(t)
	if err := c.Up(); err != nil {
		t.Fatalf("Up() error: %v", err)
	}
	assertArgs(t, readArgs(t, argsFile), []string{"-f", "docker-compose.yml", "up", "-d"})
}

func TestUp_SingleService(t *testing.T) {
	c, argsFile := setupFakeCompose(t)
	if err := c.Up("web"); err != nil {
		t.Fatalf("Up(web) error: %v", err)
	}
	assertArgs(t, readArgs(t, argsFile), []string{"-f", "docker-compose.yml", "up", "-d", "web"})
}

func TestUp_MultipleServices(t *testing.T) {
	c, argsFile := setupFakeCompose(t)
	if err := c.Up("web", "db"); err != nil {
		t.Fatalf("Up(web,db) error: %v", err)
	}
	assertArgs(t, readArgs(t, argsFile), []string{"-f", "docker-compose.yml", "up", "-d", "web", "db"})
}

func TestBuild_NoCache_NoServices(t *testing.T) {
	c, argsFile := setupFakeCompose(t)
	if err := c.Build(true); err != nil {
		t.Fatalf("Build(true) error: %v", err)
	}
	assertArgs(t, readArgs(t, argsFile), []string{"-f", "docker-compose.yml", "build", "--no-cache"})
}

func TestBuild_NoCache_SingleService(t *testing.T) {
	c, argsFile := setupFakeCompose(t)
	if err := c.Build(true, "web"); err != nil {
		t.Fatalf("Build(true,web) error: %v", err)
	}
	assertArgs(t, readArgs(t, argsFile), []string{"-f", "docker-compose.yml", "build", "--no-cache", "web"})
}

func TestBuild_WithCache_MultipleServices(t *testing.T) {
	c, argsFile := setupFakeCompose(t)
	if err := c.Build(false, "web", "db"); err != nil {
		t.Fatalf("Build(false,web,db) error: %v", err)
	}
	assertArgs(t, readArgs(t, argsFile), []string{"-f", "docker-compose.yml", "build", "web", "db"})
}

func TestLogs_NoServices(t *testing.T) {
	c, argsFile := setupFakeCompose(t)
	if err := c.Logs(); err != nil {
		t.Fatalf("Logs() error: %v", err)
	}
	assertArgs(t, readArgs(t, argsFile), []string{"-f", "docker-compose.yml", "logs", "-f"})
}

func TestLogs_SingleService(t *testing.T) {
	c, argsFile := setupFakeCompose(t)
	if err := c.Logs("web"); err != nil {
		t.Fatalf("Logs(web) error: %v", err)
	}
	assertArgs(t, readArgs(t, argsFile), []string{"-f", "docker-compose.yml", "logs", "-f", "web"})
}

func TestLogs_MultipleServices(t *testing.T) {
	c, argsFile := setupFakeCompose(t)
	if err := c.Logs("web", "worker"); err != nil {
		t.Fatalf("Logs(web,worker) error: %v", err)
	}
	assertArgs(t, readArgs(t, argsFile), []string{"-f", "docker-compose.yml", "logs", "-f", "web", "worker"})
}

func TestRestart_NoServices(t *testing.T) {
	c, argsFile := setupFakeCompose(t)
	if err := c.Restart(); err != nil {
		t.Fatalf("Restart() error: %v", err)
	}
	assertArgs(t, readArgs(t, argsFile), []string{"-f", "docker-compose.yml", "restart"})
}

func TestRestart_SingleService(t *testing.T) {
	c, argsFile := setupFakeCompose(t)
	if err := c.Restart("web"); err != nil {
		t.Fatalf("Restart(web) error: %v", err)
	}
	assertArgs(t, readArgs(t, argsFile), []string{"-f", "docker-compose.yml", "restart", "web"})
}

func TestDownVolumes(t *testing.T) {
	c, argsFile := setupFakeCompose(t)
	if err := c.DownVolumes(); err != nil {
		t.Fatalf("DownVolumes() error: %v", err)
	}
	assertArgs(t, readArgs(t, argsFile), []string{"-f", "docker-compose.yml", "down", "-v"})
}

// TestEnvFile_Propagated verifies that when EnvFile is set on the config,
// docker-compose is invoked with --env-file <path> right after -f.
func TestEnvFile_Propagated(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args.txt")
	script := "#!/bin/sh\nfor a in \"$@\"; do echo \"$a\"; done > " + argsFile + "\n"
	if err := os.WriteFile(filepath.Join(dir, "docker-compose"), []byte(script), 0755); err != nil {
		t.Fatalf("create fake docker-compose: %v", err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))

	cfg := &config.Config{ComposeFile: "docker-compose.yml", EnvFile: ".env.dev"}
	c := New(cfg)
	if err := c.Up("web"); err != nil {
		t.Fatalf("Up(web) error: %v", err)
	}
	assertArgs(t, readArgs(t, argsFile), []string{
		"-f", "docker-compose.yml",
		"--env-file", ".env.dev",
		"up", "-d", "web",
	})
}
