package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jlavera/mf-cli/internal/config"
)

// withConfig writes compose to a temp file and installs a config for the test.
func withConfig(t *testing.T, compose string, services []config.Service) {
	t.Helper()
	composePath := filepath.Join(t.TempDir(), "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(compose), 0644); err != nil {
		t.Fatal(err)
	}
	prev := cfg
	t.Cleanup(func() { cfg = prev })
	cfg = &config.Config{
		Project:     "topline",
		ComposeFile: composePath,
		DNS:         config.DNSConfig{Enabled: true, TLD: "mf"},
		Services:    services,
	}
}

func routeMap(t *testing.T) map[string]string {
	t.Helper()
	_, routes, err := resolveRoutes()
	if err != nil {
		t.Fatalf("resolveRoutes: %v", err)
	}
	got := map[string]string{}
	for _, r := range routes {
		got[r.Hostname] = r.Target
	}
	return got
}

func assertRoutes(t *testing.T, got, want map[string]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for h, target := range want {
		if got[h] != target {
			t.Errorf("%s → %q, want %q", h, got[h], target)
		}
	}
}

func warningFor(t *testing.T, svc string) string {
	t.Helper()
	_, services, err := serviceRoutes()
	if err != nil {
		t.Fatalf("serviceRoutes: %v", err)
	}
	return services[svc].Warning
}

func TestResolveRoutesProxyPort(t *testing.T) {
	withConfig(t, `services:
  web:
    image: python
    ports:
      - "8000:8000"
  temporal_server:
    image: temporalio/temporal
    ports:
      - "7233:7233"
      - "8233:8233"
`, []config.Service{
		{Name: "web", Type: "python"},
		{Name: "temporal_server", Type: "temporal", ProxyPort: 8233},
	})

	assertRoutes(t, routeMap(t), map[string]string{
		"web.topline.mf":             "http://localhost:8000",
		"temporal_server.topline.mf": "http://localhost:8233",
	})

	w := warningFor(t, "temporal_server")
	if !strings.Contains(w, "only 8233 is routed") {
		t.Errorf("warning should name proxy_port 8233, got %q", w)
	}
}

func TestResolveRoutesNamedPorts(t *testing.T) {
	withConfig(t, `services:
  temporal_server:
    image: temporalio/temporal
    ports:
      - name: grpc
        target: 7233
        published: "7233"
        app_protocol: grpc
      - name: ui
        target: 8233
        published: "8233"
  admin:
    image: node
    ports:
      - name: app
        target: 3000
        published: "3000"
      - name: docs
        target: 3001
        published: "3001"
`, []config.Service{
		{Name: "temporal_server", Type: "temporal"},
		{Name: "admin", Type: "nodejs", Hostname: "backoffice.my-app.mf"},
	})

	assertRoutes(t, routeMap(t), map[string]string{
		// Default skips the non-HTTP grpc port; grpc gets no named route.
		"temporal_server.topline.mf":    "http://localhost:8233",
		"ui.temporal_server.topline.mf": "http://localhost:8233",
		// Named ports build on a hostname override.
		"backoffice.my-app.mf":      "http://localhost:3000",
		"app.backoffice.my-app.mf":  "http://localhost:3000",
		"docs.backoffice.my-app.mf": "http://localhost:3001",
	})

	if w := warningFor(t, "temporal_server"); w != "" {
		t.Errorf("all ports named → no warning, got %q", w)
	}
}

func TestServiceRoutesWarnings(t *testing.T) {
	withConfig(t, `services:
  web:
    image: python
    ports:
      - "8000:8000"
  multi:
    image: temporalio/temporal
    ports:
      - "7233:7233"
      - "8233:8233"
  mixed:
    image: node
    ports:
      - name: ui
        target: 3000
        published: "3000"
      - "3001:3001"
`, []config.Service{
		{Name: "web", Type: "python"},
		{Name: "multi", Type: "temporal"},
		{Name: "mixed", Type: "nodejs"},
	})

	if w := warningFor(t, "web"); w != "" {
		t.Errorf("single port → no warning, got %q", w)
	}
	w := warningFor(t, "multi")
	if !strings.Contains(w, "multi publishes 2 ports") || !strings.Contains(w, "only 7233 is routed (https://multi.topline.mf)") {
		t.Errorf("unexpected warning for multi: %q", w)
	}
	if w := warningFor(t, "mixed"); !strings.Contains(w, "only 3000 is routed") {
		t.Errorf("partially named ports should warn, got %q", w)
	}
}
