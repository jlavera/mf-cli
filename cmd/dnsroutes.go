package cmd

import (
	"os"
	"sort"
	"strings"

	"github.com/jlavera/mf-cli/internal/compose"
	"github.com/jlavera/mf-cli/internal/proxy"
)

// dnsProjectName returns the project identifier used as the middle DNS segment
// (<service>.<project>.<tld>). It prefers the mf.yaml `project:` value and only
// falls back to the compose project name when that is empty.
func dnsProjectName(cf *compose.ComposeFile) string {
	if cfg.Project != "" {
		return cfg.Project
	}
	return cf.ProjectName(cfg.ComposeFile)
}

// dnsEnv builds the variable map used to expand ${VAR:-default} interpolation in
// compose port mappings: the project's env_file overlaid with the current process
// environment, which takes precedence (matching docker-compose).
func dnsEnv() map[string]string {
	env := compose.LoadEnvFile(cfg.EnvFile)
	for _, kv := range os.Environ() {
		if i := strings.IndexByte(kv, '='); i > 0 {
			env[kv[:i]] = kv[i+1:]
		}
	}
	return env
}

// resolveRoutes parses the compose file and returns the project name plus the
// proxy routes for every routable service (one with a published port), sorted by
// hostname for stable output.
func resolveRoutes() (string, []proxy.Route, error) {
	cf, err := compose.ParseComposeFile(cfg.ComposeFile)
	if err != nil {
		return "", nil, err
	}

	projectName := dnsProjectName(cf)
	env := dnsEnv()

	ports := make(map[string][]string)
	for name, svc := range cf.Services {
		ports[name] = compose.ExtractPortsPublic(svc.Ports)
	}

	hostnames := cfg.ResolveHostnames(projectName, ports)

	var routes []proxy.Route
	for svcName, hostname := range hostnames {
		svcPorts := ports[svcName]
		if len(svcPorts) == 0 {
			continue
		}
		routes = append(routes, proxy.Route{
			Hostname: hostname,
			Target:   "http://localhost:" + compose.HostPort(svcPorts[0], env),
		})
	}

	sort.Slice(routes, func(i, j int) bool { return routes[i].Hostname < routes[j].Hostname })
	return projectName, routes, nil
}
