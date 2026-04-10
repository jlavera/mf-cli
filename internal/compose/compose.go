package compose

import (
	"encoding/json"
	"strings"

	"github.com/jlavera/mf-cli/internal/config"
	"github.com/jlavera/mf-cli/internal/runner"
)

// Compose wraps docker-compose commands using the project config.
type Compose struct {
	cfg *config.Config
}

// New creates a new Compose runner from config.
func New(cfg *config.Config) *Compose {
	return &Compose{cfg: cfg}
}

// baseArgs returns the base docker-compose args (file flag).
func (c *Compose) baseArgs() []string {
	return []string{"-f", c.cfg.ComposeFile}
}

// run builds the full arg list and executes docker-compose.
func (c *Compose) run(args ...string) error {
	fullArgs := append(c.baseArgs(), args...)
	return runner.Run("docker-compose", fullArgs...)
}

// Up starts services in detached mode.
func (c *Compose) Up(services ...string) error {
	args := append([]string{"up", "-d"}, services...)
	return c.run(args...)
}

// Stop stops services without removing them.
func (c *Compose) Stop(services ...string) error {
	args := append([]string{"stop"}, services...)
	return c.run(args...)
}

// Build builds services. If noCache is true, builds without cache.
func (c *Compose) Build(noCache bool, services ...string) error {
	args := []string{"build"}
	if noCache {
		args = append(args, "--no-cache")
	}
	args = append(args, services...)
	return c.run(args...)
}

// Down stops and removes containers.
func (c *Compose) Down(services ...string) error {
	args := append([]string{"down"}, services...)
	return c.run(args...)
}

// DownVolumes stops and removes containers and volumes.
func (c *Compose) DownVolumes() error {
	return c.run("down", "-v")
}

// Logs follows logs for services.
func (c *Compose) Logs(services ...string) error {
	args := append([]string{"logs", "-f"}, services...)
	return c.run(args...)
}

// Restart restarts services.
func (c *Compose) Restart(services ...string) error {
	args := append([]string{"restart"}, services...)
	return c.run(args...)
}

// Exec executes a command in a running service container with interactive TTY.
func (c *Compose) Exec(service string, command ...string) error {
	args := append([]string{"exec", service}, command...)
	return c.run(args...)
}

// ExecWithEnv executes a command with additional environment variables.
func (c *Compose) ExecWithEnv(service string, env map[string]string, command ...string) error {
	args := []string{"exec"}
	for k, v := range env {
		args = append(args, "-e", k+"="+v)
	}
	args = append(args, service)
	args = append(args, command...)
	return c.run(args...)
}

// PortMapping holds a single published port from docker-compose ps.
type PortMapping struct {
	URL           string `json:"URL"`
	TargetPort    int    `json:"TargetPort"`
	PublishedPort int    `json:"PublishedPort"`
	Protocol      string `json:"Protocol"`
}

// ContainerStatus holds the state of a single container from docker-compose ps.
type ContainerStatus struct {
	Service    string        `json:"Service"`
	State      string        `json:"State"`
	Health     string        `json:"Health"`
	Status     string        `json:"Status"`
	Publishers []PortMapping `json:"Publishers"`
}

// Ps returns the status of all containers by parsing docker-compose ps --format json.
func (c *Compose) Ps() ([]ContainerStatus, error) {
	fullArgs := append(c.baseArgs(), "ps", "-a", "--format", "json")
	out, err := runner.Output("docker-compose", fullArgs...)
	if err != nil {
		return nil, err
	}

	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}

	var statuses []ContainerStatus

	// docker-compose v2 outputs one JSON object per line
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var cs ContainerStatus
		if err := json.Unmarshal([]byte(line), &cs); err != nil {
			continue
		}
		statuses = append(statuses, cs)
	}

	return statuses, nil
}
