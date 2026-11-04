package maven

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// LifecyclePhases are the standard Maven lifecycle phases, always offered as tasks.
var LifecyclePhases = []string{
	"clean", "validate", "compile", "test", "package", "verify", "install", "deploy",
}

// PluginGoals maps a plugin artifactId to the goals offered as tasks when the
// plugin is declared in pom.xml. Add entries here to extend task discovery.
var PluginGoals = map[string][]string{
	"spring-boot-maven-plugin": {"spring-boot:run", "spring-boot:build-image"},
	"quarkus-maven-plugin":     {"quarkus:dev", "quarkus:build"},
	"jetty-maven-plugin":       {"jetty:run"},
	"tomcat7-maven-plugin":     {"tomcat7:run"},
	"flyway-maven-plugin":      {"flyway:migrate", "flyway:info", "flyway:clean", "flyway:validate"},
	"liquibase-maven-plugin":   {"liquibase:update", "liquibase:rollback", "liquibase:status"},
	"exec-maven-plugin":        {"exec:java", "exec:exec"},
	"spotless-maven-plugin":    {"spotless:apply", "spotless:check"},
	"maven-checkstyle-plugin":  {"checkstyle:check"},
	"jib-maven-plugin":         {"jib:build", "jib:dockerBuild"},
	"micronaut-maven-plugin":   {"mn:run"},
}

type plugin struct {
	ArtifactID string `xml:"artifactId"`
}

type build struct {
	Plugins        []plugin `xml:"plugins>plugin"`
	ManagedPlugins []plugin `xml:"pluginManagement>plugins>plugin"`
}

type pom struct {
	Build    build `xml:"build"`
	Profiles []struct {
		Build build `xml:"build"`
	} `xml:"profiles>profile"`
}

// HasPomXML reports whether dir contains a pom.xml file.
func HasPomXML(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "pom.xml"))
	return err == nil
}

// DetectBuildTool returns "./mvnw" when the Maven wrapper is present in dir,
// otherwise "mvn".
func DetectBuildTool(dir string) string {
	if _, err := os.Stat(filepath.Join(dir, "mvnw")); err == nil {
		return "./mvnw"
	}
	return "mvn"
}

// ReadTasks parses the pom.xml in dir and returns the lifecycle phases plus
// goals of known plugins declared in it, deduped and sorted.
func ReadTasks(dir string) ([]string, error) {
	path := filepath.Join(dir, "pom.xml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", path, err)
	}

	var p pom
	if err := xml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("could not parse %s: %w", path, err)
	}

	seen := map[string]bool{}
	var tasks []string
	add := func(t string) {
		if !seen[t] {
			seen[t] = true
			tasks = append(tasks, t)
		}
	}
	for _, phase := range LifecyclePhases {
		add(phase)
	}

	builds := []build{p.Build}
	for _, prof := range p.Profiles {
		builds = append(builds, prof.Build)
	}
	for _, b := range builds {
		for _, pl := range append(b.Plugins, b.ManagedPlugins...) {
			for _, goal := range PluginGoals[pl.ArtifactID] {
				add(goal)
			}
		}
	}

	sort.Strings(tasks)
	return tasks, nil
}
