package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jlavera/mf-cli/internal/runner"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update mf to the latest version (git pull + make install)",
	Annotations: map[string]string{
		"skipConfig": "true",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		repoDir, err := findRepoDir()
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Updating from %s\n", repoDir)

		branch, err := currentBranch(repoDir)
		if err != nil {
			return fmt.Errorf("cannot determine current branch: %w", err)
		}

		if err := runner.RunInDir(repoDir, "git", "pull", "origin", branch); err != nil {
			return fmt.Errorf("git pull failed: %w", err)
		}

		if err := runner.RunInDir(repoDir, "make", "install"); err != nil {
			return fmt.Errorf("make install failed: %w", err)
		}

		fmt.Fprintln(os.Stderr, "Updated successfully!")
		return nil
	},
}

// findRepoDir resolves the mf binary's real path and walks up to find the git repo root.
func findRepoDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cannot determine executable path: %w", err)
	}

	real, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("cannot resolve symlink: %w", err)
	}

	dir := filepath.Dir(real)
	repo, err := gitRepoRoot(dir)
	if err == nil {
		return repo, nil
	}

	return "", fmt.Errorf(
		"cannot find mf source repository from %s — was it installed from a git clone?", dir,
	)
}

func currentBranch(repoDir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	branch := string(out)
	if len(branch) > 0 && branch[len(branch)-1] == '\n' {
		branch = branch[:len(branch)-1]
	}
	return branch, nil
}

func gitRepoRoot(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	root := string(out)
	if len(root) > 0 && root[len(root)-1] == '\n' {
		root = root[:len(root)-1]
	}
	return root, nil
}

func init() {
	updateCmd.GroupID = "general"
	rootCmd.AddCommand(updateCmd)
}
