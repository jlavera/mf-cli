package compose

import (
	"bufio"
	"os"
	"strings"
)

// LoadEnvFile parses a simple .env file into a map of KEY → value. Lines that
// are blank or start with '#' are ignored, an optional leading "export " is
// stripped, and surrounding single/double quotes are removed from values. A
// missing or unreadable file yields an empty map (env files are optional).
func LoadEnvFile(path string) map[string]string {
	env := make(map[string]string)
	if path == "" {
		return env
	}
	f, err := os.Open(path)
	if err != nil {
		return env
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		env[key] = val
	}
	return env
}

// ExpandEnv expands ${VAR}, ${VAR:-default}, ${VAR-default}, and $VAR references
// in s using the provided env map (the same interpolation docker-compose applies
// to compose files). An unset variable with no default expands to "".
func ExpandEnv(s string, env map[string]string) string {
	return os.Expand(s, func(name string) string {
		return expandVar(name, env)
	})
}

// expandVar resolves a single ${...} body, honoring the ":-"/"-" default forms.
func expandVar(name string, env map[string]string) string {
	key := name
	def := ""
	if i := strings.Index(name, ":-"); i >= 0 {
		key, def = name[:i], name[i+2:]
	} else if i := strings.Index(name, ":?"); i >= 0 {
		key = name[:i]
	} else if i := strings.Index(name, ":+"); i >= 0 {
		key = name[:i]
	} else if i := strings.Index(name, "-"); i >= 0 {
		key, def = name[:i], name[i+1:]
	}
	if v, ok := env[key]; ok && v != "" {
		return v
	}
	return def
}

// HostPort returns the host-side port from a docker-compose port mapping,
// expanding any ${VAR:-default} interpolation with env first. Supported forms:
// "3000", "3000:3000", "127.0.0.1:3000:3000", each optionally suffixed with a
// protocol like "/tcp".
func HostPort(mapping string, env map[string]string) string {
	p := ExpandEnv(mapping, env)
	if i := strings.IndexByte(p, '/'); i >= 0 {
		p = p[:i]
	}
	parts := strings.Split(p, ":")
	switch len(parts) {
	case 3:
		return parts[1] // ip:host:container → host
	case 2:
		return parts[0] // host:container → host
	default:
		return parts[0] // container-only → best effort
	}
}
