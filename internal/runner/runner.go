package runner

import (
	"fmt"
	"os"
	"os/exec"
)

// Run executes a command with inherited stdin/stdout/stderr for interactive use.
func Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %s %v: %w", name, args, err)
	}
	return nil
}

// RunInDir executes a command in a specific directory with inherited stdio.
func RunInDir(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed in %s: %s %v: %w", dir, name, args, err)
	}
	return nil
}

// Interactive executes a command replacing the current process (for truly interactive
// commands like shell/psql where we need full TTY control).
// Falls back to Run if exec is not available.
func Interactive(name string, args ...string) error {
	path, err := exec.LookPath(name)
	if err != nil {
		return fmt.Errorf("command not found: %s: %w", name, err)
	}

	cmd := exec.Command(path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
