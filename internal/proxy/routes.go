package proxy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Route maps a hostname to a backend target URL.
type Route struct {
	Hostname string `json:"hostname"`
	Target   string `json:"target"`
}

// RoutesDir returns the directory where per-project route files are stored.
// It lives under AppSupportDir so the root proxy daemon and the user-run
// `mf up`/`down` agree on a single location.
func RoutesDir() string {
	return filepath.Join(AppSupportDir, "routes")
}

// EnsureRoutesDir creates the routes directory and makes it world-writable so
// that `mf up` (running as the user) can drop route files that the root proxy
// daemon reads. Intended to be called from `mf proxy install` (running as root).
func EnsureRoutesDir() error {
	// The parent (AppSupportDir) may have been created 0700 by GenerateCA's
	// MkdirAll. Make it world-traversable so non-root users can reach the
	// routes subdir below, otherwise writing route files is denied even though
	// routes/ itself is 0777.
	if err := os.MkdirAll(AppSupportDir, 0755); err != nil {
		return fmt.Errorf("could not create %s: %w", AppSupportDir, err)
	}
	if err := os.Chmod(AppSupportDir, 0755); err != nil {
		return fmt.Errorf("could not set permissions on %s: %w", AppSupportDir, err)
	}

	dir := RoutesDir()
	if err := os.MkdirAll(dir, 0777); err != nil {
		return fmt.Errorf("could not create routes directory: %w", err)
	}
	// MkdirAll is subject to umask, so chmod explicitly to guarantee 0777.
	return os.Chmod(dir, 0777)
}

// WriteRoutes writes a project's route file to the shared routes directory.
func WriteRoutes(projectName string, routes []Route) error {
	dir := RoutesDir()
	if err := os.MkdirAll(dir, 0777); err != nil {
		return fmt.Errorf("could not create routes directory: %w", err)
	}

	data, err := json.MarshalIndent(routes, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(dir, projectName+".json")
	return os.WriteFile(path, data, 0644)
}

// RemoveRoutes deletes a project's route file.
func RemoveRoutes(projectName string) error {
	path := filepath.Join(RoutesDir(), projectName+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// LoadAllRoutes reads all route files from the routes directory and returns
// a merged hostname → target map.
func LoadAllRoutes() (map[string]string, error) {
	dir := RoutesDir()

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return make(map[string]string), nil
	}
	if err != nil {
		return nil, err
	}

	routes := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var fileRoutes []Route
		if err := json.Unmarshal(data, &fileRoutes); err != nil {
			continue
		}
		for _, r := range fileRoutes {
			routes[r.Hostname] = r.Target
		}
	}
	return routes, nil
}
