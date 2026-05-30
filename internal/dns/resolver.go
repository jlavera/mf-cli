package dns

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ResolverConfig represents the contents of a /etc/resolver/<tld> file.
type ResolverConfig struct {
	Nameserver string
	Port       int
}

// ResolverFilePath returns the path to the macOS resolver file for the given TLD.
func ResolverFilePath(tld string) string {
	return "/etc/resolver/" + tld
}

// ParseResolverFile reads and parses an existing /etc/resolver/<tld> file.
// Returns nil if the file does not exist.
func ParseResolverFile(path string) (*ResolverConfig, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not open resolver file %s: %w", path, err)
	}
	defer f.Close()

	rc := &ResolverConfig{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "nameserver":
			rc.Nameserver = fields[1]
		case "port":
			p, err := strconv.Atoi(fields[1])
			if err == nil {
				rc.Port = p
			}
		}
	}
	return rc, scanner.Err()
}

// CheckConflict compares the desired config against an existing resolver file.
// Returns an error describing the conflict, or nil if compatible.
func CheckConflict(resolverPath, wantAddr string, wantPort int) error {
	existing, err := ParseResolverFile(resolverPath)
	if err != nil {
		return err
	}
	if existing == nil {
		return nil
	}

	if existing.Nameserver != wantAddr || existing.Port != wantPort {
		return fmt.Errorf(
			"DNS conflict — %s already points to %s:%d, but this project expects %s:%d",
			resolverPath, existing.Nameserver, existing.Port, wantAddr, wantPort,
		)
	}
	return nil
}

// WriteResolverFile creates the resolver file at path (e.g. /etc/resolver/<tld>).
// Writing under /etc/resolver requires root privileges; the caller should run
// via sudo.
func WriteResolverFile(path, nameserver string, port int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("could not create %s: %w", filepath.Dir(path), err)
	}
	content := fmt.Sprintf("nameserver %s\nport %d\n", nameserver, port)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("could not write resolver file %s (try running with sudo): %w", path, err)
	}
	return nil
}

// RemoveResolverFile deletes the resolver file.
func RemoveResolverFile(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("could not remove resolver file %s: %w", path, err)
	}
	return nil
}
