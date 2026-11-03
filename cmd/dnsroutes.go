package cmd

import (
	"fmt"
	"os"
	"sort"
	"strconv"
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

// serviceRouting describes how one routable service is exposed through the proxy.
type serviceRouting struct {
	Hostname string        // the service's base hostname
	Default  *proxy.Route  // route for Hostname; nil when no HTTP port is available
	Named    []proxy.Route // <port-name>.<Hostname> routes, one per named HTTP port
	Warning  string        // set when some published ports can't get their own subdomain
}

// Routes returns the default route followed by the named ones.
func (s serviceRouting) Routes() []proxy.Route {
	var routes []proxy.Route
	if s.Default != nil {
		routes = append(routes, *s.Default)
	}
	return append(routes, s.Named...)
}

// serviceRoutes parses the compose file and returns the project name plus the
// routing for every service that gets a hostname, keyed by service name.
func serviceRoutes() (string, map[string]serviceRouting, error) {
	cf, err := compose.ParseComposeFile(cfg.ComposeFile)
	if err != nil {
		return "", nil, err
	}

	projectName := dnsProjectName(cf)
	env := dnsEnv()

	specs := make(map[string][]compose.PortSpec)
	ports := make(map[string][]string)
	for name, svc := range cf.Services {
		specs[name] = compose.ExtractPortSpecs(svc.Ports)
		for _, p := range specs[name] {
			ports[name] = append(ports[name], p.Published)
		}
	}

	result := make(map[string]serviceRouting)
	for svcName, hostname := range cfg.ResolveHostnames(projectName, ports) {
		var proxyPort int
		if svc := cfg.FindService(svcName); svc != nil {
			proxyPort = svc.ProxyPort
		}
		result[svcName] = buildServiceRouting(svcName, hostname, specs[svcName], proxyPort, env)
	}
	return projectName, result, nil
}

func buildServiceRouting(svcName, hostname string, specs []compose.PortSpec, proxyPort int, env map[string]string) serviceRouting {
	r := serviceRouting{Hostname: hostname}

	defaultPort := ""
	if proxyPort != 0 {
		defaultPort = strconv.Itoa(proxyPort)
	} else {
		for _, p := range specs {
			if p.IsHTTP() {
				defaultPort = p.HostPort(env)
				break
			}
		}
	}
	if defaultPort != "" {
		r.Default = &proxy.Route{Hostname: hostname, Target: "http://localhost:" + defaultPort}
	}

	unnamed := 0
	for _, p := range specs {
		if p.Name == "" {
			unnamed++
			continue
		}
		if !p.IsHTTP() {
			continue
		}
		r.Named = append(r.Named, proxy.Route{
			Hostname: p.Name + "." + hostname,
			Target:   "http://localhost:" + p.HostPort(env),
		})
	}

	if len(specs) > 1 && unnamed > 0 {
		routed := "no port is routed"
		if defaultPort != "" {
			routed = fmt.Sprintf("only %s is routed (https://%s)", defaultPort, hostname)
		}
		r.Warning = fmt.Sprintf("%s publishes %d ports but not all are named — %s. "+
			"Use the long port syntax with \"name:\" to give each port its own subdomain.",
			svcName, len(specs), routed)
	}
	return r
}

// resolveRoutes returns the project name plus every proxy route (default and
// named) across all routable services, sorted by hostname for stable output.
func resolveRoutes() (string, []proxy.Route, error) {
	projectName, services, err := serviceRoutes()
	if err != nil {
		return "", nil, err
	}

	var routes []proxy.Route
	for _, s := range services {
		routes = append(routes, s.Routes()...)
	}

	sort.Slice(routes, func(i, j int) bool { return routes[i].Hostname < routes[j].Hostname })
	return projectName, routes, nil
}
