package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jlavera/mf-cli/internal/config"
)

func TestResolveRoutesProxyPort(t *testing.T) {
	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yml")
	compose := `services:
  web:
    image: python
    ports:
      - "8000:8000"
  temporal_server:
    image: temporalio/temporal
    ports:
      - "7233:7233"
      - "8233:8233"
`
	if err := os.WriteFile(composePath, []byte(compose), 0644); err != nil {
		t.Fatal(err)
	}

	prev := cfg
	t.Cleanup(func() { cfg = prev })
	cfg = &config.Config{
		Project:     "topline",
		ComposeFile: composePath,
		DNS:         config.DNSConfig{Enabled: true, TLD: "mf"},
		Services: []config.Service{
			{Name: "web", Type: "python"},
			{Name: "temporal_server", Type: "temporal", ProxyPort: 8233},
		},
	}

	_, routes, err := resolveRoutes()
	if err != nil {
		t.Fatalf("resolveRoutes: %v", err)
	}

	got := map[string]string{}
	for _, r := range routes {
		got[r.Hostname] = r.Target
	}
	want := map[string]string{
		"web.topline.mf":             "http://localhost:8000",
		"temporal_server.topline.mf": "http://localhost:8233",
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for h, target := range want {
		if got[h] != target {
			t.Errorf("%s → %q, want %q", h, got[h], target)
		}
	}
}
