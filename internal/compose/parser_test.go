package compose

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
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

	detected := ClassifyServices(cf, "testdata")
	if len(detected) != 6 {
		t.Fatalf("expected 6 detected services, got %d", len(detected))
	}

	// Build a map for easier lookup
	byName := make(map[string]DetectedService)
	for _, ds := range detected {
		byName[ds.Name] = ds
	}

	tests := []struct {
		name         string
		expectedType string
		dbName       string
		dbUser       string
	}{
		{"db", "postgres", "myapp", "postgres"},
		{"redis", "redis", "", ""},
		{"celery_worker", "celery_worker", "", ""},
		{"celery_flower", "flower", "", ""},
		{"nginx", "proxy", "", ""},
		{"web", "python", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds, ok := byName[tt.name]
			if !ok {
				t.Fatalf("service %q not found in results", tt.name)
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

	detected := ClassifyServices(cf, "testdata")
	byName := make(map[string]DetectedService)
	for _, ds := range detected {
		byName[ds.Name] = ds
	}

	tests := []struct {
		name         string
		expectedType string
	}{
		{"api", "python"},
		{"frontend", "nodejs"},
		{"db", "postgres"},
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
			if ds.ServiceType != tt.expectedType {
				t.Errorf("service %q: expected type %q, got %q", tt.name, tt.expectedType, ds.ServiceType)
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

	detected := ClassifyServices(cf, "testdata")
	if len(detected) != 1 {
		t.Fatalf("expected 1 service, got %d", len(detected))
	}

	ds := detected[0]
	if ds.Name != "app" {
		t.Errorf("expected name 'app', got %q", ds.Name)
	}
	// Build context with Dockerfile → type detected from FROM instruction
	if ds.ServiceType != "python" {
		t.Errorf("expected type 'python' (from Dockerfile), got %q", ds.ServiceType)
	}
}

func TestDetectTechFromDockerfile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{"python", "FROM python:3.12-slim\nWORKDIR /app\n", []string{"python"}},
		{"node", "FROM node:20-alpine\nWORKDIR /app\n", []string{"nodejs"}},
		{"multistage node+nginx", "FROM node:20 AS builder\nRUN npm ci\nFROM nginx:alpine\nCOPY --from=builder /app/dist /usr/share/nginx/html\n", []string{"nodejs"}},
		{"multistage maven+temurin", "FROM maven:3.9-eclipse-temurin-21 AS build\nRUN mvn package\nFROM eclipse-temurin:21-jre\n", []string{"java"}},
		{"openjdk", "FROM openjdk:17-slim\n", []string{"java"}},
		{"amazoncorretto", "FROM amazoncorretto:21\n", []string{"java"}},
		{"platform flag", "FROM --platform=linux/amd64 node:20\n", []string{"nodejs"}},
		{"stage reference ignored", "FROM node:20 AS java-builder\nFROM java-builder\n", []string{"nodejs"}},
		{"multistage node+python", "FROM node:20 AS assets\nFROM python:3.12\n", []string{"nodejs", "python"}},
		{"golang", "FROM golang:1.22\nWORKDIR /app\n", []string{"go"}},
		{"ruby", "FROM ruby:3.3\nWORKDIR /app\n", []string{"ruby"}},
		{"unknown", "FROM alpine:3.19\nRUN apk add curl\n", nil},
		{"empty", "", nil},
		{"comments only", "# just a comment\n", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := dir + "/Dockerfile"
			os.WriteFile(path, []byte(tt.content), 0644)
			got := detectTech("", "", path, "")
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("detectTech() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectTechFromCommandAndImage(t *testing.T) {
	tests := []struct {
		cmd   string
		image string
		want  []string
	}{
		{"python manage.py runserver 0.0.0.0:8000", "", []string{"python"}},
		{"python3.12 -m uvicorn main:app", "", []string{"python"}},
		{`sh -c "npm install && npm run dev --workspace=admin"`, "", []string{"nodejs"}},
		{"npm run bundle", "", []string{"nodejs"}},
		{"./mvnw spring-boot:run", "", []string{"java"}},
		{"java -jar app.jar", "", []string{"java"}},
		{"bundle exec rails s", "", []string{"ruby"}},
		{"NODE_ENV=dev node server.js", "", []string{"nodejs"}},
		{"", "node:20-alpine", []string{"nodejs"}},
		{"", "eclipse-temurin:21", []string{"java"}},
		{"npm start", "node:20", []string{"nodejs"}},
		{"python app.py", "node:20", []string{"nodejs", "python"}},
		{"", "alpine:3", nil},
	}

	for _, tt := range tests {
		t.Run(tt.cmd+"|"+tt.image, func(t *testing.T) {
			got := detectTech(tt.cmd, tt.image, "", "")
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("detectTech(%q, %q) = %v, want %v", tt.cmd, tt.image, got, tt.want)
			}
		})
	}
}

func TestDetectTechFromManifests(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  []string
	}{
		{"pom.xml", []string{"pom.xml"}, []string{"java"}},
		{"package.json", []string{"package.json"}, []string{"nodejs"}},
		{"both", []string{"pom.xml", "package.json"}, []string{"java", "nodejs"}},
		{"none", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range tt.files {
				os.WriteFile(filepath.Join(dir, f), []byte("{}"), 0644)
			}
			got := detectTech("", "", "", dir)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("detectTech() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClassifyServicesAmbiguous(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "api"), 0755)
	os.WriteFile(filepath.Join(dir, "api", "Dockerfile"), []byte("FROM maven:3.9\n"), 0644)
	os.WriteFile(filepath.Join(dir, "api", "package.json"), []byte("{}"), 0644)

	cf := &ComposeFile{Services: map[string]ComposeService{
		"api": {Build: "./api"},
		"app": {Image: "node:20", Command: "python app.py"},
	}}
	for _, ds := range ClassifyServices(cf, dir) {
		if ds.ServiceType != "" {
			t.Errorf("%s: expected empty type, got %q", ds.Name, ds.ServiceType)
		}
		want := map[string][]string{"api": {"java", "nodejs"}, "app": {"nodejs", "python"}}[ds.Name]
		if !reflect.DeepEqual(ds.Candidates, want) {
			t.Errorf("%s: candidates = %v, want %v", ds.Name, ds.Candidates, want)
		}
	}
}

func TestClassifyServicesJavaBuild(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "api"), 0755)
	os.WriteFile(filepath.Join(dir, "api", "pom.xml"), []byte("<project/>"), 0644)

	cf := &ComposeFile{Services: map[string]ComposeService{"api": {Build: "./api"}}}
	ds := ClassifyServices(cf, dir)[0]
	if ds.ServiceType != "java" || ds.Candidates != nil {
		t.Errorf("expected java with no candidates, got %q %v", ds.ServiceType, ds.Candidates)
	}
}

func TestExtractDockerfilePath(t *testing.T) {
	if got := extractDockerfilePath(nil); got != "" {
		t.Errorf("nil build: expected empty, got %q", got)
	}
	if got := extractDockerfilePath("."); got != "Dockerfile" {
		t.Errorf("string build: expected 'Dockerfile', got %q", got)
	}
	m := map[string]interface{}{"context": ".", "dockerfile": "Dockerfile.dev"}
	if got := extractDockerfilePath(m); got != "Dockerfile.dev" {
		t.Errorf("map build with dockerfile: expected 'Dockerfile.dev', got %q", got)
	}
	m2 := map[string]interface{}{"context": "."}
	if got := extractDockerfilePath(m2); got != "Dockerfile" {
		t.Errorf("map build without dockerfile: expected 'Dockerfile', got %q", got)
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

func TestExtractPortSpecs(t *testing.T) {
	src := `
- "8000:8000"
- "127.0.0.1:5432:5432/tcp"
- 9000
- "${WEB_PORT:-3000}:3000"
- name: ui
  target: 8233
  published: "8233"
- name: grpc
  target: 7233
  published: 7233
  app_protocol: grpc
- target: 9999
`
	var ports interface{}
	if err := yaml.Unmarshal([]byte(src), &ports); err != nil {
		t.Fatal(err)
	}

	got := ExtractPortSpecs(ports)
	want := []struct {
		name, published, target, proto string
		http                           bool
	}{
		{"", "8000", "8000", "", true},
		{"", "5432", "5432", "", true},
		{"", "9000", "9000", "", true},
		{"", "3000", "3000", "", true},
		{"ui", "8233", "8233", "", true},
		{"grpc", "7233", "7233", "grpc", false},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d specs, want %d (target-only entry should be skipped): %+v", len(got), len(want), got)
	}
	for i, w := range want {
		g := got[i]
		if g.Name != w.name || g.Published != w.published || g.Target != w.target || g.AppProtocol != w.proto || g.IsHTTP() != w.http {
			t.Errorf("spec %d = %+v (http=%v), want %+v", i, g, g.IsHTTP(), w)
		}
	}

	if hp := got[3].HostPort(map[string]string{"WEB_PORT": "3100"}); hp != "3100" {
		t.Errorf("HostPort with env = %q, want 3100", hp)
	}
}
