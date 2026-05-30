package compose

import "testing"

func TestHostPort(t *testing.T) {
	env := map[string]string{
		"ADMIN_HOST_PORT": "9999",
	}
	cases := []struct {
		in   string
		want string
	}{
		{"3000", "3000"},
		{"3000:3000", "3000"},
		{"8080:80", "8080"},
		{"127.0.0.1:3000:3000", "3000"},
		{"127.0.0.1:8080:80", "8080"},
		{"3000:3000/tcp", "3000"},
		{"127.0.0.1:8080:80/tcp", "8080"},
		// env interpolation with default (var unset → default)
		{"${API_HOST_PORT:-3100}:3000", "3100"},
		{"${API_HOST_PORT-3100}:3000", "3100"},
		// env interpolation where the var IS set → value wins
		{"${ADMIN_HOST_PORT:-3103}:3103", "9999"},
	}
	for _, c := range cases {
		if got := HostPort(c.in, env); got != c.want {
			t.Errorf("HostPort(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestExpandEnv(t *testing.T) {
	env := map[string]string{"FOO": "bar", "EMPTY": ""}
	cases := []struct {
		in   string
		want string
	}{
		{"${FOO}", "bar"},
		{"$FOO", "bar"},
		{"${MISSING:-def}", "def"},
		{"${MISSING-def}", "def"},
		{"${FOO:-def}", "bar"},
		{"${EMPTY:-fallback}", "fallback"},
		{"${MISSING}", ""},
		{"prefix-${FOO}-suffix", "prefix-bar-suffix"},
	}
	for _, c := range cases {
		if got := ExpandEnv(c.in, env); got != c.want {
			t.Errorf("ExpandEnv(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
