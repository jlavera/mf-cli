package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the mf.yaml configuration file.
type Config struct {
	Project     string        `yaml:"project"`
	ComposeFile string        `yaml:"compose_file"`
	EnvFile     string        `yaml:"env_file"`
	Services    []Service     `yaml:"services"`
	DNS         DNSConfig     `yaml:"dns,omitempty"`
	E2E         E2EConfig     `yaml:"e2e,omitempty"`
	Scripts     ScriptsConfig `yaml:"scripts,omitempty"`
	Test        TestConfig    `yaml:"test,omitempty"`
}

// Service represents a single service detected from docker-compose.
type Service struct {
	Name           string `yaml:"name"`
	Type           string `yaml:"type"`                      // python, nodejs, postgres, mysql, mongo, redis, celery_worker, celery_beat, flower, proxy, mail, storage, …
	Hostname       string `yaml:"hostname,omitempty"`         // DNS hostname override; "false" to opt out
	DBName         string `yaml:"db_name,omitempty"`
	DBUser         string `yaml:"db_user,omitempty"`
	Path           string `yaml:"path,omitempty"`             // local project directory (e.g. ./frontend)
	PackageManager string `yaml:"package_manager,omitempty"` // npm, yarn, pnpm
}

// appTypes are technology types that represent application services.
var appTypes = map[string]bool{
	"python": true, "nodejs": true, "ruby": true, "java": true, "go": true,
}

// dbTypes are types that represent database services.
var dbTypes = map[string]bool{
	"postgres": true, "mysql": true, "mongo": true,
}

// Backend returns the name of the first app-type service (python, nodejs, etc.).
func (c *Config) Backend() string {
	for _, s := range c.Services {
		if appTypes[s.Type] {
			return s.Name
		}
	}
	return ""
}

// Databases returns all database services (postgres, mysql, mongo).
func (c *Config) Databases() []Service {
	var dbs []Service
	for _, s := range c.Services {
		if dbTypes[s.Type] {
			dbs = append(dbs, s)
		}
	}
	return dbs
}

// Redis returns the first redis service name, or "".
func (c *Config) Redis() string {
	for _, s := range c.Services {
		if s.Type == "redis" {
			return s.Name
		}
	}
	return ""
}

// Workers returns names of celery_worker / celery_beat services.
func (c *Config) Workers() []string {
	var names []string
	for _, s := range c.Services {
		if s.Type == "celery_worker" || s.Type == "celery_beat" {
			names = append(names, s.Name)
		}
	}
	return names
}

// Flower returns the first flower service name, or "".
func (c *Config) Flower() string {
	for _, s := range c.Services {
		if s.Type == "flower" {
			return s.Name
		}
	}
	return ""
}

// NodeJSProjects returns all services with type "nodejs".
func (c *Config) NodeJSProjects() []Service {
	var projs []Service
	for _, s := range c.Services {
		if s.Type == "nodejs" {
			projs = append(projs, s)
		}
	}
	return projs
}

// FindService returns a pointer to the service with the given name, or nil.
func (c *Config) FindService(name string) *Service {
	for i := range c.Services {
		if c.Services[i].Name == name {
			return &c.Services[i]
		}
	}
	return nil
}

// DNSConfig holds local DNS resolution settings.
type DNSConfig struct {
	Enabled bool   `yaml:"enabled,omitempty"`
	TLD     string `yaml:"tld,omitempty"`     // TLD suffix (default: "mf")
	Address string `yaml:"address,omitempty"` // IP returned by DNS (default: "127.0.0.1")
}

const DefaultDNSPort = 5354

// ResolveHostnames derives hostnames for services that are routable (i.e. have
// at least one published port). A service is skipped when it has no ports or
// when its hostname is explicitly set to "false". An explicit hostname value
// overrides the auto-derived <service>.<project>.<tld> name.
//
// Requiring a published port keeps `mf status` and the routes written by
// `mf up` consistent: a hostname is only reported when it can actually be
// proxied to a backend.
//
// projectName is the compose project name (from compose file name: or directory).
// ports maps service name → list of port mappings from the compose file.
func (c *Config) ResolveHostnames(projectName string, ports map[string][]string) map[string]string {
	result := make(map[string]string)
	if !c.DNS.Enabled {
		return result
	}
	for i := range c.Services {
		svc := &c.Services[i]
		if svc.Hostname == "false" {
			continue
		}
		if len(ports[svc.Name]) == 0 {
			continue
		}
		if svc.Hostname != "" {
			result[svc.Name] = svc.Hostname
			continue
		}
		result[svc.Name] = svc.Name + "." + projectName + "." + c.DNS.TLD
	}
	return result
}

// E2EConfig holds end-to-end testing settings.
type E2EConfig struct {
	Path      string `yaml:"path,omitempty"`
	Framework string `yaml:"framework,omitempty"` // playwright, cypress
	Browser   string `yaml:"browser,omitempty"`
}

// ScriptsConfig holds paths to project scripts.
type ScriptsConfig struct {
	PreCommit      string `yaml:"pre_commit,omitempty"`
	PreCommitLocal string `yaml:"pre_commit_local,omitempty"`
	Format         string `yaml:"format,omitempty"`
	Lint           string `yaml:"lint,omitempty"`
	Ruff           string `yaml:"ruff,omitempty"`
}

// TestConfig holds test runner settings.
type TestConfig struct {
	Runner    string            `yaml:"runner,omitempty"`
	Env       map[string]string `yaml:"env,omitempty"`
	DebugPort int               `yaml:"debug_port,omitempty"`
}

// DefaultConfigFile is the default config file name.
const DefaultConfigFile = "mf.yaml"

// Load reads and parses an mf.yaml file, applying defaults for missing values.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read config file %s: %w", path, err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("could not parse config file %s: %w", path, err)
	}

	applyDefaults(cfg)
	return cfg, nil
}

// header is prepended to every generated mf.yaml as a quick command reference.
const header = `# mf - docker-compose project manager
#
# General Commands:
#   mf up [services...]       Start containers (docker-compose up -d)
#   mf stop [services...]     Stop containers
#   mf down                   Stop and remove containers
#   mf clean                  Stop and remove containers + volumes
#   mf build [services...]    Build images (--no-cache)
#   mf rebuild [services...]  Build --no-cache then up
#   mf bounce [services...]   Restart (with args: docker-compose restart; no args: down + up)
#   mf logs [services...]     Follow container logs
#   mf shell [service]        Open bash/sh shell in a container (default: backend)
#   mf psql [service]         Open database shell (postgres/mysql/mongo)
#   mf redis-cli [service]    Open redis-cli in a Redis container
#   mf test [apps...]         Run backend tests (-f file, -m marker, --debug)
#   mf format [--check]       Format backend code
#   mf lint                   Lint backend code
#   mf sort-imports           Sort imports (ruff)
#   mf format-all             format + lint + sort-imports
#   mf pre-commit [--all] [--local]   Run pre-commit hooks
#   mf debug check|clean      Inspect/kill debug port (default 5679)
#   mf update                    Update mf to the latest version
#   mf init [-f file] [-e env-file] [--force]   (Re)generate this file from a compose file
#   mf dns install|uninstall|start|stop|status   Local DNS for .mf domains
#   mf proxy install|uninstall|start|stop|status   Reverse proxy with HTTPS
#
# Stack Commands:
#   mf celery start|stop|restart|logs   Manage Celery workers
#   mf flower logs                      Follow Flower logs
#   mf run <service> <script> [args...]   Run a package.json script (services with path: set)
#   mf e2e install|run|ui|headed|debug|report

`

// Write serializes and writes a Config to the given path.
func Write(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("could not serialize config: %w", err)
	}

	output := append([]byte(header), data...)
	if err := os.WriteFile(path, output, 0644); err != nil {
		return fmt.Errorf("could not write config file %s: %w", path, err)
	}
	return nil
}

// applyDefaults fills in sensible defaults for any missing config values.
func applyDefaults(cfg *Config) {
	if cfg.ComposeFile == "" {
		cfg.ComposeFile = "docker-compose.yml"
	}
	if cfg.EnvFile == "" {
		cfg.EnvFile = ".env"
	}
	// If no backend exists, add a default "web" python entry.
	if cfg.Backend() == "" {
		cfg.Services = append(cfg.Services, Service{Name: "web", Type: "python"})
	}
	for i := range cfg.Services {
		if cfg.Services[i].Path != "" && cfg.Services[i].PackageManager == "" {
			cfg.Services[i].PackageManager = "npm"
		}
	}
	if cfg.Test.Runner == "" {
		cfg.Test.Runner = "pytest"
	}
	if cfg.Test.DebugPort == 0 {
		cfg.Test.DebugPort = 5679
	}
	if cfg.DNS.Enabled {
		if cfg.DNS.TLD == "" {
			cfg.DNS.TLD = "mf"
		}
		if cfg.DNS.Address == "" {
			cfg.DNS.Address = "127.0.0.1"
		}
	}
	if cfg.E2E.Browser == "" && cfg.E2E.Path != "" {
		cfg.E2E.Browser = "chromium"
	}
	if cfg.E2E.Framework == "" && cfg.E2E.Path != "" {
		cfg.E2E.Framework = "playwright"
	}
}

// Exists checks if a config file exists at the given path.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// FindConfigFile looks for mf.yaml in the given directory.
func FindConfigFile(dir string) (string, error) {
	path := filepath.Join(dir, DefaultConfigFile)
	if Exists(path) {
		return path, nil
	}
	return "", fmt.Errorf("no %s found in %s — run 'mf init' to create one", DefaultConfigFile, dir)
}
