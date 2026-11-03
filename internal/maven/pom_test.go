package maven

import (
	"os"
	"path/filepath"
	"testing"
)

const samplePom = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
  <modelVersion>4.0.0</modelVersion>
  <artifactId>api</artifactId>
  <build>
    <plugins>
      <plugin>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-maven-plugin</artifactId>
      </plugin>
      <plugin>
        <artifactId>maven-surefire-plugin</artifactId>
      </plugin>
    </plugins>
    <pluginManagement>
      <plugins>
        <plugin>
          <artifactId>flyway-maven-plugin</artifactId>
        </plugin>
      </plugins>
    </pluginManagement>
  </build>
  <profiles>
    <profile>
      <id>lint</id>
      <build>
        <plugins>
          <plugin>
            <artifactId>spotless-maven-plugin</artifactId>
          </plugin>
        </plugins>
      </build>
    </profile>
  </profiles>
</project>`

func writePom(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(content), 0644); err != nil {
		t.Fatalf("writePom: %v", err)
	}
}

func TestReadTasks(t *testing.T) {
	dir := t.TempDir()
	writePom(t, dir, samplePom)

	tasks, err := ReadTasks(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	set := map[string]bool{}
	for _, task := range tasks {
		if set[task] {
			t.Errorf("duplicate task %q", task)
		}
		set[task] = true
	}
	want := append([]string{}, LifecyclePhases...)
	want = append(want, "spring-boot:run", "flyway:migrate", "spotless:apply")
	for _, w := range want {
		if !set[w] {
			t.Errorf("missing task %q in %v", w, tasks)
		}
	}
	if set["quarkus:dev"] {
		t.Errorf("unexpected goal from undeclared plugin: %v", tasks)
	}
}

func TestReadTasksMinimalPom(t *testing.T) {
	dir := t.TempDir()
	writePom(t, dir, `<project><artifactId>x</artifactId></project>`)

	tasks, err := ReadTasks(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != len(LifecyclePhases) {
		t.Errorf("expected only lifecycle phases, got %v", tasks)
	}
}

func TestReadTasksErrors(t *testing.T) {
	if _, err := ReadTasks(t.TempDir()); err == nil {
		t.Error("expected error for missing pom.xml")
	}

	dir := t.TempDir()
	writePom(t, dir, "<project><build>")
	if _, err := ReadTasks(dir); err == nil {
		t.Error("expected error for malformed pom.xml")
	}
}

func TestDetectBuildTool(t *testing.T) {
	dir := t.TempDir()
	if got := DetectBuildTool(dir); got != "mvn" {
		t.Errorf("got %q, want mvn", got)
	}
	os.WriteFile(filepath.Join(dir, "mvnw"), []byte("#!/bin/sh\n"), 0755)
	if got := DetectBuildTool(dir); got != "./mvnw" {
		t.Errorf("got %q, want ./mvnw", got)
	}
}

func TestHasPomXML(t *testing.T) {
	dir := t.TempDir()
	if HasPomXML(dir) {
		t.Error("expected false for empty dir")
	}
	writePom(t, dir, samplePom)
	if !HasPomXML(dir) {
		t.Error("expected true after writing pom.xml")
	}
}
