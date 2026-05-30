package config

import "testing"

func TestResolveHostnames(t *testing.T) {
	cfg := &Config{
		DNS: DNSConfig{Enabled: true, TLD: "mf", Address: "127.0.0.1"},
		Services: []Service{
			{Name: "web", Type: "nodejs"},
			{Name: "api", Type: "python"},
			{Name: "admin", Type: "nodejs", Hostname: "backoffice.my-app.mf"},
			{Name: "worker", Type: "celery_worker"},
			{Name: "hidden", Type: "nodejs", Hostname: "false"},
		},
	}

	ports := map[string][]string{
		"web":    {"3000:3000"},
		"api":    {"3001:3001"},
		"admin":  {"3002:3002"},
		"worker": {}, // no published ports
		"hidden": {"3003:3003"},
	}

	got := cfg.ResolveHostnames("my-app", ports)

	want := map[string]string{
		"web":   "web.my-app.mf",
		"api":   "api.my-app.mf",
		"admin": "backoffice.my-app.mf", // explicit override
	}

	if len(got) != len(want) {
		t.Fatalf("got %d hostnames, want %d: %v", len(got), len(want), got)
	}
	for svc, hostname := range want {
		if got[svc] != hostname {
			t.Errorf("%s = %q, want %q", svc, got[svc], hostname)
		}
	}

	// worker has no ports → skipped.
	if _, ok := got["worker"]; ok {
		t.Error("worker should be skipped (no published ports)")
	}
	// hidden opted out via hostname: false.
	if _, ok := got["hidden"]; ok {
		t.Error("hidden should be skipped (hostname: false)")
	}
}

func TestResolveHostnamesDisabled(t *testing.T) {
	cfg := &Config{
		DNS:      DNSConfig{Enabled: false},
		Services: []Service{{Name: "web", Type: "nodejs"}},
	}
	got := cfg.ResolveHostnames("my-app", map[string][]string{"web": {"3000:3000"}})
	if len(got) != 0 {
		t.Errorf("expected no hostnames when DNS disabled, got %v", got)
	}
}
