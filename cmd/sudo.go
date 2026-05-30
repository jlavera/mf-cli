package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// elevate ensures the current command runs as root. When already running as
// root it returns (false, nil) and the caller proceeds normally. When not root,
// it prints the given human-readable reason explaining why administrator access
// is needed, then re-runs the exact same `mf` invocation under sudo, streaming
// its I/O. In that case it returns handled=true and the caller must stop (return
// the error as-is).
//
// reason should be a short explanation of *what* requires root and *why*, shown
// to the user before the sudo password prompt.
func elevate(reason string) (handled bool, err error) {
	if os.Geteuid() == 0 {
		return false, nil
	}

	fmt.Println("🔒 Administrator access required.")
	if reason != "" {
		fmt.Println(reason)
	}
	fmt.Println("Re-running with sudo — you may be prompted for your password.")
	fmt.Println()

	mfBin, err := os.Executable()
	if err != nil {
		return true, fmt.Errorf("could not determine mf binary path: %w", err)
	}

	// os.Args[1:] preserves the subcommand and all flags exactly as typed.
	sudoArgs := append([]string{mfBin}, os.Args[1:]...)
	c := exec.Command("sudo", sudoArgs...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if runErr := c.Run(); runErr != nil {
		// Mirror the child's exit code without printing a redundant error,
		// since sudo / the child already reported the failure.
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		return true, runErr
	}
	return true, nil
}
