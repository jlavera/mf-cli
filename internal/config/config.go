package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the mf.yaml configuration file.
type Config struct {
	Project     string         `yaml:"project"`
	ComposeFile string         `yaml:"compose_file"`
	Services    ServicesConfig `yaml:"services"`
	Frontend    FrontendConfig `yaml:"frontend,omitempty"`
	E2E         E2EConfig      `yaml:"e2e,omitempty"`
	Scripts     ScriptsConfig  `yaml:"scripts,omitempty"`
	Test        TestConfig     `yaml:"test,omitempty"`
}

// ServicesConfig maps roles to docker-compose service names.
type ServicesConfig struct {
	Backend   string            `yaml:"backend"`
	Databases []DatabaseService `yaml:"databases,omitempty"`
	Redis     string            `yaml:"redis,omitempty"`
	Workers   []string          `yaml:"workers,omitempty"`
	Flower    string            `yaml:"flower,omitempty"`
}

// DatabaseService holds a database service name and its connection details.
type DatabaseService struct {
	Service string `yaml:"service"`
	Type    string `yaml:"type,omitempty"` // postgres, mysql, mongo
	DBName  string `yaml:"db_name,omitempty"`
	DBUser  string `yaml:"db_user,omitempty"`
}

// FrontendConfig holds frontend project settings.
type FrontendConfig struct {
	Path           string `yaml:"path,omitempty"`
	PackageManager string `yaml:"package_manager,omitempty"` // npm, yarn, pnpm
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
#   mf update                    Update mf to the latest release (requires GITHUB_TOKEN)
#   mf init [-f file] [--force]       (Re)generate this file from a compose file
#
# Stack Commands:
#   mf celery start|stop|restart|logs   Manage Celery workers
#   mf flower logs                      Follow Flower logs
#   mf frontend install|dev|build|preview|lint|type-check|check-all|restart
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
	if cfg.Services.Backend == "" {
		cfg.Services.Backend = "web"
	}
	if cfg.Frontend.PackageManager == "" && cfg.Frontend.Path != "" {
		cfg.Frontend.PackageManager = "npm"
	}
	if cfg.Test.Runner == "" {
		cfg.Test.Runner = "pytest"
	}
	if cfg.Test.DebugPort == 0 {
		cfg.Test.DebugPort = 5679
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
