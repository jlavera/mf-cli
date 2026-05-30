package dns

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseResolverFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mf")
	content := "# managed by mf\nnameserver 127.0.0.1\nport 5354\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rc, err := ParseResolverFile(path)
	if err != nil {
		t.Fatalf("ParseResolverFile: %v", err)
	}
	if rc == nil {
		t.Fatal("expected config, got nil")
	}
	if rc.Nameserver != "127.0.0.1" {
		t.Errorf("nameserver = %q, want 127.0.0.1", rc.Nameserver)
	}
	if rc.Port != 5354 {
		t.Errorf("port = %d, want 5354", rc.Port)
	}
}

func TestParseResolverFileMissing(t *testing.T) {
	rc, err := ParseResolverFile(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("expected nil error for missing file, got %v", err)
	}
	if rc != nil {
		t.Errorf("expected nil config for missing file, got %+v", rc)
	}
}

func TestCheckConflict(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mf")
	if err := WriteResolverFile(path, "127.0.0.1", 5354); err != nil {
		t.Fatal(err)
	}

	// Matching config → no conflict.
	if err := CheckConflict(path, "127.0.0.1", 5354); err != nil {
		t.Errorf("expected no conflict for matching config, got %v", err)
	}

	// Different address → conflict.
	if err := CheckConflict(path, "127.0.0.2", 5354); err == nil {
		t.Error("expected conflict for differing address, got nil")
	}

	// Different port → conflict.
	if err := CheckConflict(path, "127.0.0.1", 9999); err == nil {
		t.Error("expected conflict for differing port, got nil")
	}

	// No existing file → no conflict.
	if err := CheckConflict(filepath.Join(dir, "absent"), "127.0.0.1", 5354); err != nil {
		t.Errorf("expected no conflict when file absent, got %v", err)
	}
}
