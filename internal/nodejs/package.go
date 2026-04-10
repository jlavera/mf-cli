package nodejs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type packageJSON struct {
	Scripts map[string]string `json:"scripts"`
}

// ReadScripts parses the package.json in dir and returns its scripts map.
// Returns an empty map (not an error) if package.json has no scripts field.
func ReadScripts(dir string) (map[string]string, error) {
	path := filepath.Join(dir, "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", path, err)
	}

	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("could not parse %s: %w", path, err)
	}

	if pkg.Scripts == nil {
		return map[string]string{}, nil
	}
	return pkg.Scripts, nil
}

// DetectPackageManager infers the package manager from lockfiles in dir.
// Defaults to "npm" if none are found.
func DetectPackageManager(dir string) string {
	if _, err := os.Stat(filepath.Join(dir, "pnpm-lock.yaml")); err == nil {
		return "pnpm"
	}
	if _, err := os.Stat(filepath.Join(dir, "yarn.lock")); err == nil {
		return "yarn"
	}
	return "npm"
}

// HasPackageJSON reports whether dir contains a package.json file.
func HasPackageJSON(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "package.json"))
	return err == nil
}
