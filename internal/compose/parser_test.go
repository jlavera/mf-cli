package compose

import (
	"os"
	"testing"
)

func TestParseComposeFile(t *testing.T) {
	cf, err := ParseComposeFile("testdata/docker-compose.yml")
	if err != nil {
		t.Fatalf("failed to parse compose file: %v", err)
	}

	if len(cf.Services) != 6 {
		t.Errorf("expected 6 services, got %d", len(cf.Services))
	}

	// Check that expected services exist
	expectedServices := []string{"web", "db", "redis", "celery_worker", "celery_flower", "nginx"}
	for _, name := range expectedServices {
		if _, ok := cf.Services[name]; !ok {
			t.Errorf("expected service %q not found", name)
		}
	}
}

func TestClassifyServices(t *testing.T) {
	cf, err := ParseComposeFile("testdata/docker-compose.yml")
	if err != nil {
		t.Fatalf("failed to parse compose file: %v", err)
	}

	detected := ClassifyServices(cf)
	if len(detected) != 6 {
		t.Fatalf("expected 6 detected services, got %d", len(detected))
	}

	// Build a map for easier lookup
	byName := make(map[string]DetectedService)
	for _, ds := range detected {
		byName[ds.Name] = ds
	}

	// Check classifications
	tests := []struct {
		name         string
		expectedRole string
		expectedType string
		dbName       string
		dbUser       string
	}{
		{"db", "db", "postgres", "topline", "postgres"},
		{"redis", "redis", "redis", "", ""},
		{"celery_worker", "celery_worker", "celery_worker", "", ""},
		{"celery_flower", "flower", "flower", "", ""},
		{"nginx", "proxy", "proxy", "", ""},
		{"web", "backend", "backend", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds, ok := byName[tt.name]
			if !ok {
				t.Fatalf("service %q not found in results", tt.name)
			}
			if ds.Role != tt.expectedRole {
				t.Errorf("service %q: expected role %q, got %q", tt.name, tt.expectedRole, ds.Role)
			}
			if ds.ServiceType != tt.expectedType {
				t.Errorf("service %q: expected type %q, got %q", tt.name, tt.expectedType, ds.ServiceType)
			}
			if tt.dbName != "" && ds.DBName != tt.dbName {
				t.Errorf("service %q: expected db_name %q, got %q", tt.name, tt.dbName, ds.DBName)
			}
			if tt.dbUser != "" && ds.DBUser != tt.dbUser {
				t.Errorf("service %q: expected db_user %q, got %q", tt.name, tt.dbUser, ds.DBUser)
			}
		})
	}
}

func TestImageMatchesPattern(t *testing.T) {
	tests := []struct {
		image   string
		pattern string
		want    bool
	}{
		{"postgres:15", "postgres", true},
		{"postgres", "postgres", true},
		{"bitnami/postgresql:14", "bitnami/postgresql", true},
		{"docker.io/library/postgres:15", "postgres", true},
		{"redis:7-alpine", "redis", true},
		{"redis", "redis", true},
		{"myregistry.com/redis:latest", "redis", true},
		{"nginx:latest", "nginx", true},
		{"my-postgres-custom", "postgres", false},
		{"postgresql:15", "postgres", false},
		{"redisinsight", "redis", false},
	}

	for _, tt := range tests {
		t.Run(tt.image+"_"+tt.pattern, func(t *testing.T) {
			got := imageMatchesPattern(tt.image, tt.pattern)
			if got != tt.want {
				t.Errorf("imageMatchesPattern(%q, %q) = %v, want %v", tt.image, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestExtractEnvMap(t *testing.T) {
	// Test map format
	mapEnv := map[string]interface{}{
		"POSTGRES_DB":   "mydb",
		"POSTGRES_USER": "admin",
	}
	result := extractEnvMap(mapEnv)
	if result["POSTGRES_DB"] != "mydb" {
		t.Errorf("expected POSTGRES_DB=mydb, got %q", result["POSTGRES_DB"])
	}

	// Test list format
	listEnv := []interface{}{
		"POSTGRES_DB=mydb",
		"POSTGRES_USER=admin",
	}
	result = extractEnvMap(listEnv)
	if result["POSTGRES_DB"] != "mydb" {
		t.Errorf("expected POSTGRES_DB=mydb from list format, got %q", result["POSTGRES_DB"])
	}

	// Test nil
	result = extractEnvMap(nil)
	if len(result) != 0 {
		t.Errorf("expected empty map for nil, got %v", result)
	}
}

func TestExtractBuildContext(t *testing.T) {
	// String build
	if got := extractBuildContext("."); got != "." {
		t.Errorf("expected '.', got %q", got)
	}

	// Map build
	mapBuild := map[string]interface{}{
		"context":    "./frontend",
		"dockerfile": "Dockerfile",
	}
	if got := extractBuildContext(mapBuild); got != "./frontend" {
		t.Errorf("expected './frontend', got %q", got)
	}

	// Nil build
	if got := extractBuildContext(nil); got != "" {
		t.Errorf("expected empty for nil, got %q", got)
	}
}

func TestMatchCommand(t *testing.T) {
	tests := []struct {
		cmd  string
		want string
	}{
		{"celery -A config worker -l info", "celery_worker"},
		{"celery -A config flower", "flower"},
		{"celery -A config beat -l info", "celery_beat"},
		{"python manage.py runserver", ""},
		{"npm run dev", ""},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			got := matchCommand(tt.cmd)
			if got != tt.want {
				t.Errorf("matchCommand(%q) = %q, want %q", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestClassifyServicesComplex(t *testing.T) {
	cf, err := ParseComposeFile("testdata/compose-complex.yml")
	if err != nil {
		t.Fatalf("failed to parse complex compose: %v", err)
	}

	detected := ClassifyServices(cf)
	byName := make(map[string]DetectedService)
	for _, ds := range detected {
		byName[ds.Name] = ds
	}

	tests := []struct {
		name         string
		expectedRole string
	}{
		{"api", "backend"},
		{"frontend", "frontend"},
		{"db", "db"},
		{"redis", "redis"},
		{"worker", "celery_worker"},
		{"beat", "celery_beat"},
		{"flower", "flower"},
		{"mailpit", "mail"},
		{"minio", "storage"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds, ok := byName[tt.name]
			if !ok {
				t.Fatalf("service %q not found", tt.name)
			}
			if ds.Role != tt.expectedRole {
				t.Errorf("service %q: expected role %q, got %q", tt.name, tt.expectedRole, ds.Role)
			}
		})
	}

	// Check DB details from bitnami/postgresql with list-format env
	db := byName["db"]
	if db.ServiceType != "postgres" {
		t.Errorf("db type: expected postgres, got %q", db.ServiceType)
	}
	if db.DBName != "myapp" {
		t.Errorf("db name: expected myapp, got %q", db.DBName)
	}
	if db.DBUser != "admin" {
		t.Errorf("db user: expected admin, got %q", db.DBUser)
	}

	// Check frontend build context
	fe := byName["frontend"]
	if fe.BuildCtx != "./frontend" {
		t.Errorf("frontend build context: expected './frontend', got %q", fe.BuildCtx)
	}
}

func TestClassifyServicesMinimal(t *testing.T) {
	cf, err := ParseComposeFile("testdata/compose-minimal.yml")
	if err != nil {
		t.Fatalf("failed to parse minimal compose: %v", err)
	}

	detected := ClassifyServices(cf)
	if len(detected) != 1 {
		t.Fatalf("expected 1 service, got %d", len(detected))
	}

	ds := detected[0]
	if ds.Name != "app" {
		t.Errorf("expected name 'app', got %q", ds.Name)
	}
	// Port 3000 with build context → backend
	if ds.Role != "backend" {
		t.Errorf("expected role 'backend', got %q", ds.Role)
	}
}

func TestClassifyServicesWithEnvVars(t *testing.T) {
	cf, err := ParseComposeFile("testdata/compose-envvars.yml")
	if err != nil {
		t.Fatalf("failed to parse envvars compose: %v", err)
	}

	detected := ClassifyServices(cf)
	if len(detected) != 4 {
		t.Fatalf("expected 4 services, got %d", len(detected))
	}

	byName := make(map[string]DetectedService)
	for _, ds := range detected {
		byName[ds.Name] = ds
	}

	// api has build + port 3000 → backend
	api := byName["api"]
	if api.Role != "backend" {
		t.Errorf("api: expected role 'backend', got %q", api.Role)
	}

	// postgres has image postgres:17-alpine → db
	pg := byName["postgres"]
	if pg.Role != "db" {
		t.Errorf("postgres: expected role 'db', got %q", pg.Role)
	}
	if pg.ServiceType != "postgres" {
		t.Errorf("postgres: expected type 'postgres', got %q", pg.ServiceType)
	}
	// env vars have ${VAR:-default} syntax — should be stored as-is
	if pg.DBName == "" {
		t.Error("postgres: expected db_name to be set (even with env var syntax)")
	}

	// web has port 3001 → frontend
	web := byName["web"]
	if web.Role != "frontend" {
		t.Errorf("web: expected role 'frontend', got %q", web.Role)
	}

	// admin has port 3003 → app (not a standard frontend port)
	admin := byName["admin"]
	if admin.Role != "app" {
		t.Errorf("admin: expected role 'app', got %q", admin.Role)
	}
}

func TestFindComposeFile(t *testing.T) {
	dir := t.TempDir()

	// No compose file
	_, err := FindComposeFile(dir)
	if err == nil {
		t.Error("expected error for empty dir")
	}

	// Create docker-compose.yml
	os.WriteFile(dir+"/docker-compose.yml", []byte("services:\n  app:\n    image: alpine"), 0644)
	path, err := FindComposeFile(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != dir+"/docker-compose.yml" {
		t.Errorf("expected docker-compose.yml, got %q", path)
	}
}
