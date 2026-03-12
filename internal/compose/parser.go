package compose

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ComposeFile represents the relevant parts of a docker-compose file.
type ComposeFile struct {
	Services map[string]ComposeService `yaml:"services"`
}

// ComposeService represents a single service in a compose file.
type ComposeService struct {
	Image       string      `yaml:"image"`
	Build       interface{} `yaml:"build"`       // string or map
	Ports       interface{} `yaml:"ports"`        // []string or []map
	Environment interface{} `yaml:"environment"`  // map or []string
	Command     interface{} `yaml:"command"`      // string or []string
	Entrypoint  interface{} `yaml:"entrypoint"`   // string or []string
	DependsOn   interface{} `yaml:"depends_on"`   // []string or map
}

// DetectedService holds the classification result for a compose service.
type DetectedService struct {
	Name        string
	Role        string // backend, db, redis, celery_worker, flower, frontend, proxy, mail, etc.
	ServiceType string // postgres, mysql, redis, etc.
	DBName      string // database name (if role=db)
	DBUser      string // database user (if role=db)
	BuildCtx    string // build context path if applicable
	Ports       []string
}

// ---------------------------------------------------------------------------
// Image matcher registry — add new matchers here to extend detection.
// ---------------------------------------------------------------------------

// ImageMatcher defines how to recognize a service from its Docker image name.
type ImageMatcher struct {
	Patterns    []string          // image name prefixes to match
	Role        string            // detected role
	ServiceType string            // sub-type (e.g. "postgres")
	EnvMappings map[string]string // env var → DetectedService field
}

// DefaultMatchers is the registry of image-based service detectors.
// Add new entries here — no other code changes needed.
var DefaultMatchers = []ImageMatcher{
	{
		Patterns:    []string{"postgres", "bitnami/postgresql"},
		Role:        "db",
		ServiceType: "postgres",
		EnvMappings: map[string]string{"POSTGRES_DB": "db_name", "POSTGRES_USER": "db_user"},
	},
	{
		Patterns:    []string{"mysql", "mariadb", "bitnami/mysql", "bitnami/mariadb"},
		Role:        "db",
		ServiceType: "mysql",
		EnvMappings: map[string]string{"MYSQL_DATABASE": "db_name", "MYSQL_USER": "db_user"},
	},
	{
		Patterns:    []string{"mongo", "bitnami/mongodb"},
		Role:        "db",
		ServiceType: "mongo",
	},
	{
		Patterns:    []string{"redis", "bitnami/redis", "valkey"},
		Role:        "redis",
		ServiceType: "redis",
	},
	{
		Patterns:    []string{"rabbitmq", "bitnami/rabbitmq"},
		Role:        "rabbitmq",
		ServiceType: "rabbitmq",
	},
	{
		Patterns:    []string{"elasticsearch", "opensearch", "bitnami/elasticsearch"},
		Role:        "search",
		ServiceType: "elasticsearch",
	},
	{
		Patterns:    []string{"nginx", "traefik", "caddy", "envoyproxy"},
		Role:        "proxy",
		ServiceType: "proxy",
	},
	{
		Patterns:    []string{"mailhog", "mailpit", "axllent/mailpit"},
		Role:        "mail",
		ServiceType: "mail",
	},
	{
		Patterns:    []string{"minio", "localstack"},
		Role:        "storage",
		ServiceType: "storage",
	},
	{
		Patterns:    []string{"mher/flower"},
		Role:        "flower",
		ServiceType: "flower",
	},
	{
		Patterns:    []string{"memcached", "bitnami/memcached"},
		Role:        "cache",
		ServiceType: "memcached",
	},
}

// ---------------------------------------------------------------------------
// Compose file discovery
// ---------------------------------------------------------------------------

// DefaultComposeFileNames lists compose file names to look for, in priority order.
var DefaultComposeFileNames = []string{
	"docker-compose.yml",
	"docker-compose.yaml",
	"compose.yml",
	"compose.yaml",
}

// FindComposeFile searches for a compose file in the given directory.
// Returns the full path and file name, or an error if none found.
func FindComposeFile(dir string) (string, error) {
	for _, name := range DefaultComposeFileNames {
		path := dir + "/" + name
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no docker-compose file found in %s (tried: %s)",
		dir, strings.Join(DefaultComposeFileNames, ", "))
}

// ---------------------------------------------------------------------------
// Parsing
// ---------------------------------------------------------------------------

// ParseComposeFile reads and parses a docker-compose YAML file.
func ParseComposeFile(path string) (*ComposeFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read compose file: %w", err)
	}

	cf := &ComposeFile{}
	if err := yaml.Unmarshal(data, cf); err != nil {
		return nil, fmt.Errorf("could not parse compose file: %w", err)
	}

	if len(cf.Services) == 0 {
		return nil, fmt.Errorf("no services found in %s", path)
	}

	return cf, nil
}

// ---------------------------------------------------------------------------
// Classification
// ---------------------------------------------------------------------------

// ClassifyServices analyzes each service in the compose file and assigns roles.
func ClassifyServices(cf *ComposeFile) []DetectedService {
	var results []DetectedService

	for name, svc := range cf.Services {
		ds := DetectedService{
			Name:  name,
			Ports: extractPorts(svc.Ports),
		}

		// Extract build context
		ds.BuildCtx = extractBuildContext(svc.Build)

		// Step 1: Try image-based matching using the registry
		imageName := normalizeImageName(svc.Image)
		if imageName != "" {
			if matched := matchImage(imageName, extractEnvMap(svc.Environment), &ds); matched {
				results = append(results, ds)
				continue
			}
		}

		// Step 2: Command-based detection (for services with build context)
		cmdStr := extractCommandString(svc.Command, svc.Entrypoint)
		if cmdStr != "" {
			if role := matchCommand(cmdStr); role != "" {
				ds.Role = role
				ds.ServiceType = role
				results = append(results, ds)
				continue
			}
		}

		// Step 3: Service-name heuristic for workers
		nameLower := strings.ToLower(name)
		if strings.Contains(nameLower, "celery") || strings.Contains(nameLower, "worker") {
			if strings.Contains(nameLower, "flower") {
				ds.Role = "flower"
				ds.ServiceType = "flower"
			} else if strings.Contains(nameLower, "beat") {
				ds.Role = "celery_beat"
				ds.ServiceType = "celery_beat"
			} else {
				ds.Role = "celery_worker"
				ds.ServiceType = "celery_worker"
			}
			results = append(results, ds)
			continue
		}
		if strings.Contains(nameLower, "flower") {
			ds.Role = "flower"
			ds.ServiceType = "flower"
			results = append(results, ds)
			continue
		}

		// Step 4: Port-based fallback for build services
		if ds.BuildCtx != "" {
			ds.Role = classifyByPorts(ds.Ports)
			ds.ServiceType = ds.Role
			results = append(results, ds)
			continue
		}

		// Step 5: Unknown
		ds.Role = "unknown"
		ds.ServiceType = "unknown"
		results = append(results, ds)
	}

	return results
}

// matchImage tries to match the image against the DefaultMatchers registry.
// Returns true if a match was found.
func matchImage(imageName string, envVars map[string]string, ds *DetectedService) bool {
	for _, m := range DefaultMatchers {
		for _, pattern := range m.Patterns {
			if imageMatchesPattern(imageName, pattern) {
				ds.Role = m.Role
				ds.ServiceType = m.ServiceType

				// Extract env var mappings
				for envKey, field := range m.EnvMappings {
					if val, ok := envVars[envKey]; ok {
						switch field {
						case "db_name":
							ds.DBName = val
						case "db_user":
							ds.DBUser = val
						}
					}
				}
				return true
			}
		}
	}
	return false
}

// imageMatchesPattern checks if an image name matches a pattern.
// Matches if the image name starts with the pattern, optionally followed by : or /.
// e.g. "postgres:15" matches "postgres", "bitnami/postgresql:14" matches "bitnami/postgresql"
func imageMatchesPattern(imageName, pattern string) bool {
	if imageName == pattern {
		return true
	}
	// Check prefix match with separator (: for tag, / for sub-path)
	if strings.HasPrefix(imageName, pattern+":") || strings.HasPrefix(imageName, pattern+"/") {
		return true
	}
	// Also match if the image name after the registry matches
	// e.g. "docker.io/library/postgres:15" should match "postgres"
	parts := strings.Split(imageName, "/")
	lastPart := parts[len(parts)-1]
	// Strip tag
	if idx := strings.Index(lastPart, ":"); idx >= 0 {
		lastPart = lastPart[:idx]
	}
	return lastPart == pattern
}

// matchCommand detects service roles from command/entrypoint strings.
func matchCommand(cmdStr string) string {
	lower := strings.ToLower(cmdStr)

	if strings.Contains(lower, "celery") {
		if strings.Contains(lower, "flower") {
			return "flower"
		}
		if strings.Contains(lower, "beat") {
			return "celery_beat"
		}
		if strings.Contains(lower, "worker") {
			return "celery_worker"
		}
		// Generic celery — assume worker
		return "celery_worker"
	}

	return ""
}

// classifyByPorts guesses a service role based on exposed ports.
func classifyByPorts(ports []string) string {
	backendPorts := map[string]bool{"8000": true, "8080": true, "5000": true, "4000": true, "3000": true}
	frontendPorts := map[string]bool{"5173": true, "4200": true, "3001": true}

	for _, p := range ports {
		hostPort := extractHostPort(p)
		if frontendPorts[hostPort] {
			return "frontend"
		}
	}
	for _, p := range ports {
		hostPort := extractHostPort(p)
		if backendPorts[hostPort] {
			return "backend"
		}
	}
	return "app"
}

// ---------------------------------------------------------------------------
// Helper functions for extracting values from compose YAML interfaces
// ---------------------------------------------------------------------------

// normalizeImageName extracts the image name, handling empty strings.
func normalizeImageName(image string) string {
	return strings.TrimSpace(image)
}

// extractBuildContext extracts the build context path from the build field.
func extractBuildContext(build interface{}) string {
	if build == nil {
		return ""
	}
	switch v := build.(type) {
	case string:
		return v
	case map[string]interface{}:
		if ctx, ok := v["context"]; ok {
			if s, ok := ctx.(string); ok {
				return s
			}
		}
	}
	return "."
}

// extractEnvMap converts the environment field to a map.
// Handles both map format (KEY: value) and list format (KEY=value).
func extractEnvMap(env interface{}) map[string]string {
	result := make(map[string]string)
	if env == nil {
		return result
	}

	switch v := env.(type) {
	case map[string]interface{}:
		for key, val := range v {
			if s, ok := val.(string); ok {
				result[key] = s
			} else if val != nil {
				result[key] = fmt.Sprintf("%v", val)
			}
		}
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				parts := strings.SplitN(s, "=", 2)
				if len(parts) == 2 {
					result[parts[0]] = parts[1]
				}
			}
		}
	}
	return result
}

// extractPorts normalizes the ports field to a string slice.
func extractPorts(ports interface{}) []string {
	if ports == nil {
		return nil
	}

	var result []string
	switch v := ports.(type) {
	case []interface{}:
		for _, item := range v {
			switch p := item.(type) {
			case string:
				result = append(result, p)
			case int:
				result = append(result, fmt.Sprintf("%d", p))
			case map[string]interface{}:
				// Long syntax: {target: 80, published: 8080}
				if published, ok := p["published"]; ok {
					if target, ok := p["target"]; ok {
						result = append(result, fmt.Sprintf("%v:%v", published, target))
					}
				}
			}
		}
	}
	return result
}

// extractHostPort extracts the host port from a port mapping string like "8000:8000".
func extractHostPort(portMapping string) string {
	parts := strings.Split(portMapping, ":")
	if len(parts) >= 2 {
		return parts[0]
	}
	return portMapping
}

// extractCommandString combines command and entrypoint into a single string for matching.
func extractCommandString(command, entrypoint interface{}) string {
	var parts []string

	if s := interfaceToString(entrypoint); s != "" {
		parts = append(parts, s)
	}
	if s := interfaceToString(command); s != "" {
		parts = append(parts, s)
	}

	return strings.Join(parts, " ")
}

// interfaceToString converts a string or []string interface{} to a single string.
func interfaceToString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []interface{}:
		var parts []string
		for _, item := range val {
			if s, ok := item.(string); ok {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, " ")
	}
	return ""
}
