package compose

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
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
	ServiceType string // python, nodejs, postgres, mysql, redis, celery_worker, flower, proxy, …
	DBName      string
	DBUser      string
	BuildCtx    string // build context path if applicable
	Ports       []string
}

// ---------------------------------------------------------------------------
// Image matcher registry — add new matchers here to extend detection.
// ---------------------------------------------------------------------------

// ImageMatcher defines how to recognize a service from its Docker image name.
type ImageMatcher struct {
	Patterns    []string          // image name prefixes to match
	ServiceType string            // detected type (e.g. "postgres", "redis")
	EnvMappings map[string]string // env var → DetectedService field
}

// DefaultMatchers is the registry of image-based service detectors.
// Add new entries here — no other code changes needed.
var DefaultMatchers = []ImageMatcher{
	{
		Patterns:    []string{"postgres", "bitnami/postgresql"},
		ServiceType: "postgres",
		EnvMappings: map[string]string{"POSTGRES_DB": "db_name", "POSTGRES_USER": "db_user"},
	},
	{
		Patterns:    []string{"mysql", "mariadb", "bitnami/mysql", "bitnami/mariadb"},
		ServiceType: "mysql",
		EnvMappings: map[string]string{"MYSQL_DATABASE": "db_name", "MYSQL_USER": "db_user"},
	},
	{
		Patterns:    []string{"mongo", "bitnami/mongodb"},
		ServiceType: "mongo",
	},
	{
		Patterns:    []string{"redis", "bitnami/redis", "valkey"},
		ServiceType: "redis",
	},
	{
		Patterns:    []string{"rabbitmq", "bitnami/rabbitmq"},
		ServiceType: "rabbitmq",
	},
	{
		Patterns:    []string{"elasticsearch", "opensearch", "bitnami/elasticsearch"},
		ServiceType: "elasticsearch",
	},
	{
		Patterns:    []string{"nginx", "traefik", "caddy", "envoyproxy"},
		ServiceType: "proxy",
	},
	{
		Patterns:    []string{"mailhog", "mailpit", "axllent/mailpit"},
		ServiceType: "mail",
	},
	{
		Patterns:    []string{"minio", "localstack"},
		ServiceType: "storage",
	},
	{
		Patterns:    []string{"mher/flower"},
		ServiceType: "flower",
	},
	{
		Patterns:    []string{"memcached", "bitnami/memcached"},
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

// ClassifyServices analyzes each service in the compose file and assigns types.
// baseDir is the directory containing the compose file, used to resolve
// Dockerfile paths for build-context services.
func ClassifyServices(cf *ComposeFile, baseDir string) []DetectedService {
	var results []DetectedService

	for name, svc := range cf.Services {
		ds := DetectedService{
			Name:  name,
			Ports: extractPorts(svc.Ports),
		}

		ds.BuildCtx = extractBuildContext(svc.Build)

		// Step 1: Try image-based matching using the registry
		imageName := normalizeImageName(svc.Image)
		if imageName != "" {
			if matched := matchImage(imageName, extractEnvMap(svc.Environment), &ds); matched {
				results = append(results, ds)
				continue
			}
		}

		// Step 2: Command-based detection (celery worker/beat/flower)
		cmdStr := extractCommandString(svc.Command, svc.Entrypoint)
		if cmdStr != "" {
			if celeryType := matchCommand(cmdStr); celeryType != "" {
				ds.ServiceType = celeryType
				results = append(results, ds)
				continue
			}
		}

		// Step 3: Service-name heuristic for workers
		nameLower := strings.ToLower(name)
		if strings.Contains(nameLower, "celery") || strings.Contains(nameLower, "worker") {
			if strings.Contains(nameLower, "flower") {
				ds.ServiceType = "flower"
			} else if strings.Contains(nameLower, "beat") {
				ds.ServiceType = "celery_beat"
			} else {
				ds.ServiceType = "celery_worker"
			}
			results = append(results, ds)
			continue
		}
		if strings.Contains(nameLower, "flower") {
			ds.ServiceType = "flower"
			results = append(results, ds)
			continue
		}

		// Step 4: Build services — detect tech from command, image, or Dockerfile
		if ds.BuildCtx != "" {
			ds.ServiceType = detectTechFromCommand(cmdStr)
			if ds.ServiceType == "" {
				ds.ServiceType = detectTechFromImage(imageName)
			}
			if ds.ServiceType == "" && baseDir != "" {
				dfName := extractDockerfilePath(svc.Build)
				dfPath := filepath.Join(baseDir, ds.BuildCtx, dfName)
				ds.ServiceType = readDockerfileBaseImage(dfPath)
			}
			results = append(results, ds)
			continue
		}

		// Step 5: Image-only services that didn't match the registry
		if imageName != "" {
			ds.ServiceType = detectTechFromImage(imageName)
			results = append(results, ds)
			continue
		}

		// Step 6: Unknown
		results = append(results, ds)
	}

	return results
}

// matchImage tries to match the image against the DefaultMatchers registry.
func matchImage(imageName string, envVars map[string]string, ds *DetectedService) bool {
	for _, m := range DefaultMatchers {
		for _, pattern := range m.Patterns {
			if imageMatchesPattern(imageName, pattern) {
				ds.ServiceType = m.ServiceType

				for envKey, field := range m.EnvMappings {
					if val, ok := envVars[envKey]; ok {
						switch field {
						case "db_name":
							ds.DBName = stripEnvInterpolation(val)
						case "db_user":
							ds.DBUser = stripEnvInterpolation(val)
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

// detectTechFromCommand infers the technology from a command/entrypoint string.
func detectTechFromCommand(cmdStr string) string {
	if cmdStr == "" {
		return ""
	}
	lower := strings.ToLower(cmdStr)
	for _, kw := range []string{"python", "manage.py", "gunicorn", "uvicorn", "django", "flask", "celery", "pytest", "pip"} {
		if strings.Contains(lower, kw) {
			return "python"
		}
	}
	for _, kw := range []string{"node", "npm", "yarn", "pnpm", "next", "vite", "nuxt", "nest", "tsx", "ts-node"} {
		if strings.Contains(lower, kw) {
			return "nodejs"
		}
	}
	for _, kw := range []string{"ruby", "rails", "bundle", "rake"} {
		if strings.Contains(lower, kw) {
			return "ruby"
		}
	}
	for _, kw := range []string{"java", "gradle", "mvn", "spring"} {
		if strings.Contains(lower, kw) {
			return "java"
		}
	}
	return ""
}

// detectTechFromImage infers the technology from a Docker image name.
func detectTechFromImage(imageName string) string {
	if imageName == "" {
		return ""
	}
	lower := strings.ToLower(imageName)
	for _, kw := range []string{"python", "django", "flask"} {
		if strings.Contains(lower, kw) {
			return "python"
		}
	}
	for _, kw := range []string{"node", "deno", "bun"} {
		if strings.Contains(lower, kw) {
			return "nodejs"
		}
	}
	for _, kw := range []string{"ruby", "rails"} {
		if strings.Contains(lower, kw) {
			return "ruby"
		}
	}
	for _, kw := range []string{"golang", "go:"} {
		if strings.Contains(lower, kw) {
			return "go"
		}
	}
	return ""
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

// extractDockerfilePath returns the dockerfile name from the build field.
// Defaults to "Dockerfile" when a build directive exists.
func extractDockerfilePath(build interface{}) string {
	if build == nil {
		return ""
	}
	if m, ok := build.(map[string]interface{}); ok {
		if df, ok := m["dockerfile"]; ok {
			if s, ok := df.(string); ok {
				return s
			}
		}
	}
	return "Dockerfile"
}

// readDockerfileBaseImage reads a Dockerfile and returns the base image from
// the first FROM instruction that matches a known technology. For multi-stage
// builds (e.g. FROM node:20 AS builder / FROM nginx:alpine) the first
// technology-matching stage wins, which is typically the one that reveals the
// project's language.
func readDockerfileBaseImage(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		upper := strings.ToUpper(line)
		if !strings.HasPrefix(upper, "FROM ") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		image := parts[1]
		if tech := detectTechFromImage(image); tech != "" {
			return tech
		}
	}
	return ""
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

// stripEnvInterpolation extracts the default value from bash-style variable
// interpolation. Examples:
//   - "${DB_NAME:-mydb}"  → "mydb"
//   - "${DB_NAME-mydb}"   → "mydb"
//   - "${DB_NAME}"        → ""
//   - "mydb"              → "mydb"
func stripEnvInterpolation(val string) string {
	s := strings.TrimSpace(val)
	if !strings.HasPrefix(s, "${") || !strings.HasSuffix(s, "}") {
		return s
	}
	inner := s[2 : len(s)-1] // strip ${ and }
	if idx := strings.Index(inner, ":-"); idx >= 0 {
		return inner[idx+2:]
	}
	if idx := strings.Index(inner, "-"); idx >= 0 {
		return inner[idx+1:]
	}
	return ""
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
