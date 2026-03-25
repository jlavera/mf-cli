package cmd

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

const repo = "jlavera/mf-cli"

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update mf to the latest release",
	Long: `Downloads and installs the latest release from GitHub.
Requires a GITHUB_TOKEN environment variable for private repo access.`,
	Annotations: map[string]string{"skipConfig": "true"},
	RunE:        runUpdate,
}

func init() {
	updateCmd.GroupID = "general"
	rootCmd.AddCommand(updateCmd)
}

func tokenPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "mf", "token"), nil
}

// getToken returns the GitHub token from GITHUB_TOKEN env var or ~/.config/mf/token.
func getToken() (string, error) {
	if t := os.Getenv("GITHUB_TOKEN"); t != "" {
		return t, nil
	}
	path, err := tokenPath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf(
			"no GitHub token found — set GITHUB_TOKEN or run the install script first\n"+
				"Create a token at: https://github.com/settings/tokens (repo scope)",
		)
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", fmt.Errorf("token file %s is empty — re-run the install script", path)
	}
	return token, nil
}

func runUpdate(cmd *cobra.Command, args []string) error {
	token, err := getToken()
	if err != nil {
		return err
	}

	fmt.Printf("Current version: %s\n", appVersion)
	fmt.Println("Checking for latest release...")

	latest, err := fetchLatestVersion(token)
	if err != nil {
		return fmt.Errorf("could not fetch latest release: %w", err)
	}

	if latest == appVersion {
		fmt.Printf("Already up to date (%s)\n", appVersion)
		return nil
	}

	fmt.Printf("Updating %s → %s\n", appVersion, latest)

	dest, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine executable path: %w", err)
	}
	dest, err = filepath.EvalSymlinks(dest)
	if err != nil {
		return fmt.Errorf("could not resolve symlink: %w", err)
	}

	if err := downloadRelease(token, latest, dest); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Printf("Updated to %s\n", latest)
	return nil
}

type ghRelease struct {
	TagName string `json:"tag_name"`
}

func fetchLatestVersion(token string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("GitHub API returned %d — check your GITHUB_TOKEN", resp.StatusCode)
	}

	var release ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	if release.TagName == "" {
		return "", fmt.Errorf("no releases found")
	}
	return release.TagName, nil
}

func downloadRelease(token, version, dest string) error {
	goos := runtime.GOOS
	arch := runtime.GOARCH
	tarball := fmt.Sprintf("mf_%s_%s.tar.gz", goos, arch)
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, version, tarball)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed (HTTP %d)", resp.StatusCode)
	}

	// Extract binary from tar.gz into a temp file
	tmpFile, err := os.CreateTemp(filepath.Dir(dest), ".mf-update-*")
	if err != nil {
		return fmt.Errorf("could not create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if err := extractBinary(resp.Body, tmpFile); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	// Replace the current binary
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return err
	}
	return os.Rename(tmpPath, dest)
}

func extractBinary(r io.Reader, w io.Writer) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("could not read gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Name == "mf" || filepath.Base(hdr.Name) == "mf" {
			_, err = io.Copy(w, tr)
			return err
		}
	}
	return fmt.Errorf("binary 'mf' not found in archive")
}
