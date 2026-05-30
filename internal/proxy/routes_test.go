package proxy

import (
	"testing"
)

func TestWriteAndLoadRoutes(t *testing.T) {
	AppSupportDir = t.TempDir()

	projA := []Route{
		{Hostname: "web.a.mf", Target: "http://localhost:3000"},
		{Hostname: "api.a.mf", Target: "http://localhost:3001"},
	}
	projB := []Route{
		{Hostname: "web.b.mf", Target: "http://localhost:4000"},
	}

	if err := WriteRoutes("proj-a", projA); err != nil {
		t.Fatalf("WriteRoutes(proj-a): %v", err)
	}
	if err := WriteRoutes("proj-b", projB); err != nil {
		t.Fatalf("WriteRoutes(proj-b): %v", err)
	}

	routes, err := LoadAllRoutes()
	if err != nil {
		t.Fatalf("LoadAllRoutes: %v", err)
	}
	if len(routes) != 3 {
		t.Fatalf("expected 3 merged routes, got %d (%v)", len(routes), routes)
	}
	if routes["web.a.mf"] != "http://localhost:3000" {
		t.Errorf("web.a.mf = %q", routes["web.a.mf"])
	}
	if routes["web.b.mf"] != "http://localhost:4000" {
		t.Errorf("web.b.mf = %q", routes["web.b.mf"])
	}

	// Removing a project drops only its routes.
	if err := RemoveRoutes("proj-a"); err != nil {
		t.Fatalf("RemoveRoutes(proj-a): %v", err)
	}
	routes, err = LoadAllRoutes()
	if err != nil {
		t.Fatalf("LoadAllRoutes after remove: %v", err)
	}
	if len(routes) != 1 {
		t.Fatalf("expected 1 route after removing proj-a, got %d", len(routes))
	}
	if _, ok := routes["web.b.mf"]; !ok {
		t.Error("expected proj-b route to remain")
	}
}

func TestLoadAllRoutesEmpty(t *testing.T) {
	AppSupportDir = t.TempDir()
	routes, err := LoadAllRoutes()
	if err != nil {
		t.Fatalf("LoadAllRoutes on empty dir: %v", err)
	}
	if len(routes) != 0 {
		t.Errorf("expected 0 routes, got %d", len(routes))
	}
}

func TestRoutesSignature(t *testing.T) {
	a := map[string]string{"x.mf": "http://localhost:1", "y.mf": "http://localhost:2"}
	b := map[string]string{"y.mf": "http://localhost:2", "x.mf": "http://localhost:1"}
	if routesSignature(a) != routesSignature(b) {
		t.Error("signature should be order-independent")
	}

	c := map[string]string{"x.mf": "http://localhost:9"}
	if routesSignature(a) == routesSignature(c) {
		t.Error("different route sets should have different signatures")
	}
}
