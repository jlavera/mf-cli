package compose

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ComposeFile represents the relevant parts of a docker-compose file.
type ComposeFile struct {
	Name     string                       `yaml:"name"`
	Services map[string]ComposeService `yaml:"services"`
}

// ProjectName returns the compose project name. Falls back to the parent
// directory name of composePath if the file has no explicit name: field.
func (cf *ComposeFile) ProjectName(composePath string) string {
	if cf.Name != "" {
		return cf.Name
	}
	return filepath.Base(filepath.Dir(composePath))
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
	BuildCtx    string   // build context path if applicable
	Ports       []string
	Candidates  []string // set (with ServiceType empty) when several types matched
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

// TechMatcher defines how to recognize an application technology. Every
// matcher is evaluated against every signal; a service gets a type only when
// exactly one technology matches.
type TechMatcher struct {
	ServiceType   string
	Executables   []string // executable names in command/entrypoint (versions stripped)
	ImageKeywords []string // tokens in the image name or Dockerfile FROM images
	Manifests     []string // files at the root of the build context
}

// TechMatchers is the registry of technology detectors. Add new entries here.
var TechMatchers = []TechMatcher{
	{
		ServiceType:   "python",
		Executables:   []string{"python", "pip", "manage.py", "gunicorn", "uvicorn", "daphne", "hypercorn", "celery", "pytest", "poetry", "pipenv", "flask", "django-admin"},
		ImageKeywords: []string{"python", "django", "flask"},
	},
	{
		ServiceType:   "nodejs",
		Executables:   []string{"node", "npm", "npx", "yarn", "pnpm", "bun", "deno", "next", "vite", "nuxt", "nest", "tsx", "ts-node", "nodemon", "pm2"},
		ImageKeywords: []string{"node", "nodejs", "deno", "bun"},
		Manifests:     []string{"package.json"},
	},
	{
		ServiceType:   "ruby",
		Executables:   []string{"ruby", "rails", "bundle", "rake", "puma", "rackup", "sidekiq"},
		ImageKeywords: []string{"ruby", "rails"},
	},
	{
		ServiceType:   "java",
		Executables:   []string{"java", "mvn", "mvnw", "gradle", "gradlew"},
		ImageKeywords: []string{"java", "maven", "openjdk", "jdk", "jre", "temurin", "corretto", "amazoncorretto", "zulu", "semeru", "sapmachine"},
		Manifests:     []string{"pom.xml"},
	},
	{
		ServiceType:   "go",
		Executables:   []string{"go"},
		ImageKeywords: []string{"golang"},
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

		// Step 4: Tech detection — every TechMatcher runs against the command,
		// image, Dockerfile FROM lines and build-context manifests.
		var dfPath, ctxDir string
		if ds.BuildCtx != "" && baseDir != "" {
			ctxDir = filepath.Join(baseDir, ds.BuildCtx)
			dfPath = filepath.Join(ctxDir, extractDockerfilePath(svc.Build))
		}
		setTech(&ds, detectTech(cmdStr, imageName, dfPath, ctxDir))
		results = append(results, ds)
	}

	return results
}

// setTech assigns the service type when exactly one candidate matched, and
// records the candidates (leaving the type empty) when several did.
func setTech(ds *DetectedService, candidates []string) {
	switch len(candidates) {
	case 0:
	case 1:
		ds.ServiceType = candidates[0]
	default:
		ds.Candidates = candidates
	}
}

// matchImage runs every DefaultMatchers entry against the image. It returns
// true when at least one matched; if matchers of different service types
// match, the service is marked ambiguous instead of picking one.
func matchImage(imageName string, envVars map[string]string, ds *DetectedService) bool {
	var matched []ImageMatcher
	types := map[string]bool{}
	for _, m := range DefaultMatchers {
		for _, pattern := range m.Patterns {
			if imageMatchesPattern(imageName, pattern) {
				matched = append(matched, m)
				types[m.ServiceType] = true
				break
			}
		}
	}
	if len(matched) == 0 {
		return false
	}
	if len(types) > 1 {
		setTech(ds, sortedKeys(types))
		return true
	}

	ds.ServiceType = matched[0].ServiceType
	for _, m := range matched {
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
	}
	return true
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

// detectTech runs every TechMatcher against all available signals and returns
// the sorted, deduplicated set of matching technologies. Empty arguments are
// skipped. More than one result means the service is ambiguous.
func detectTech(cmdStr, imageName, dockerfilePath, ctxDir string) []string {
	images := dockerfileBaseImages(dockerfilePath)
	if imageName != "" {
		images = append(images, imageName)
	}
	executables := commandExecutables(cmdStr)

	found := map[string]bool{}
	for _, m := range TechMatchers {
		if m.matchesCommand(executables) || m.matchesImages(images) || m.matchesManifests(ctxDir) {
			found[m.ServiceType] = true
		}
	}
	return sortedKeys(found)
}

func (m TechMatcher) matchesCommand(executables []string) bool {
	for _, exe := range executables {
		for _, kw := range m.Executables {
			if exe == normalizeToken(kw) {
				return true
			}
		}
	}
	return false
}

func (m TechMatcher) matchesImages(images []string) bool {
	for _, img := range images {
		for _, tok := range tokenize(img) {
			for _, kw := range m.ImageKeywords {
				if tok == kw {
					return true
				}
			}
		}
	}
	return false
}

func (m TechMatcher) matchesManifests(ctxDir string) bool {
	if ctxDir == "" {
		return false
	}
	for _, name := range m.Manifests {
		if _, err := os.Stat(filepath.Join(ctxDir, name)); err == nil {
			return true
		}
	}
	return false
}

// commandWrappers are leading words that don't reveal the technology; the
// executable after them is used instead (e.g. `sh -c "npm run dev"`).
var commandWrappers = map[string]bool{
	"sh": true, "bash": true, "-c": true, "exec": true, "env": true, "sudo": true,
}

// commandExecutables returns the normalized executable name of every
// sub-command in cmdStr (split on &&, ||, ;, |). Only executables are
// matched, so arguments like `npm run bundle` don't look like Ruby.
func commandExecutables(cmdStr string) []string {
	s := strings.NewReplacer(`"`, " ", `'`, " ", "&&", ";", "||", ";", "|", ";").Replace(cmdStr)
	var exes []string
	for _, segment := range strings.Split(s, ";") {
		for _, field := range strings.Fields(segment) {
			if commandWrappers[field] || strings.Contains(field, "=") {
				continue
			}
			exes = append(exes, normalizeToken(filepath.Base(strings.ToLower(field))))
			break
		}
	}
	return exes
}

// tokenize lowercases s, splits it on non-alphanumeric characters and
// normalizes each token (e.g. "maven:3.9-eclipse-temurin-21" → maven, eclipse, temurin).
func tokenize(s string) []string {
	fields := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})
	var tokens []string
	for _, f := range fields {
		if t := normalizeToken(f); t != "" {
			tokens = append(tokens, t)
		}
	}
	return tokens
}

// normalizeToken strips trailing version digits and dots ("python3.12" → "python").
func normalizeToken(s string) string {
	return strings.TrimRight(s, "0123456789.")
}

func sortedKeys(m map[string]bool) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
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

// dockerfileBaseImages returns the base image of every FROM instruction in
// the Dockerfile at path (all stages of a multi-stage build). Returns nil if
// path is empty or unreadable.
func dockerfileBaseImages(path string) []string {
	if path == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var images []string
	stages := map[string]bool{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) < 2 || !strings.EqualFold(parts[0], "FROM") {
			continue
		}
		for i, p := range parts[1:] {
			if strings.HasPrefix(p, "--") {
				continue
			}
			if !stages[strings.ToLower(p)] {
				images = append(images, p)
			}
			if rest := parts[i+2:]; len(rest) >= 2 && strings.EqualFold(rest[0], "AS") {
				stages[strings.ToLower(rest[1])] = true
			}
			break
		}
	}
	return images
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

// ExtractPortsPublic is the exported version of extractPorts for use by other packages.
func ExtractPortsPublic(ports interface{}) []string {
	return extractPorts(ports)
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

// PortSpec is a published port from a compose service's ports list. Name and
// AppProtocol are only available with the long syntax.
type PortSpec struct {
	Name        string
	Published   string // raw host port; may contain ${VAR} interpolation
	Target      string
	AppProtocol string

	mapping string // raw short-syntax mapping, resolved lazily by HostPort
}

// HostPort returns the host-side port, expanding ${VAR:-default} with env.
func (p PortSpec) HostPort(env map[string]string) string {
	if p.mapping != "" {
		return HostPort(p.mapping, env)
	}
	return ExpandEnv(p.Published, env)
}

// IsHTTP reports whether the port can be reverse-proxied over HTTP. Ports with
// no app_protocol are assumed to be HTTP.
func (p PortSpec) IsHTTP() bool {
	switch strings.ToLower(p.AppProtocol) {
	case "", "http", "https":
		return true
	}
	return false
}

// ExtractPortSpecs parses a service's ports field (short and long syntax).
// Long-syntax entries without a published port are skipped since they aren't
// reachable on a fixed host port.
func ExtractPortSpecs(ports interface{}) []PortSpec {
	items, ok := ports.([]interface{})
	if !ok {
		return nil
	}

	var result []PortSpec
	for _, item := range items {
		switch p := item.(type) {
		case string:
			result = append(result, shortPortSpec(p))
		case int:
			result = append(result, shortPortSpec(fmt.Sprintf("%d", p)))
		case map[string]interface{}:
			published, ok := p["published"]
			if !ok || published == nil {
				continue
			}
			spec := PortSpec{Published: fmt.Sprintf("%v", published)}
			if v, ok := p["target"]; ok && v != nil {
				spec.Target = fmt.Sprintf("%v", v)
			}
			if v, ok := p["name"].(string); ok {
				spec.Name = v
			}
			if v, ok := p["app_protocol"].(string); ok {
				spec.AppProtocol = v
			}
			result = append(result, spec)
		}
	}
	return result
}

func shortPortSpec(mapping string) PortSpec {
	p := strings.SplitN(ExpandEnv(mapping, nil), "/", 2)[0]
	parts := strings.Split(p, ":")
	return PortSpec{
		Published: HostPort(mapping, nil),
		Target:    parts[len(parts)-1],
		mapping:   mapping,
	}
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
