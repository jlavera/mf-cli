package proxy

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const launchdLabel = "com.mf-cli.proxy"

// launchDaemonsDir is where system-wide LaunchDaemons live. The proxy runs as a
// root LaunchDaemon so it can bind ports 80/443 directly (no pfctl redirect).
const launchDaemonsDir = "/Library/LaunchDaemons"

// LogPath is where launchd sends the daemon's stdout/stderr.
const LogPath = "/tmp/mf-proxy.log"

// PlistPath returns the path to the launchd plist for the proxy daemon.
func PlistPath() string {
	return filepath.Join(launchDaemonsDir, launchdLabel+".plist")
}

// InstallLaunchd writes the LaunchDaemon plist that auto-starts `mf proxy start`.
// Must run as root (the plist lives under /Library/LaunchDaemons and the daemon
// binds privileged ports).
func InstallLaunchd(mfBinary string, httpPort, httpsPort int) error {
	plistPath := PlistPath()

	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>proxy</string>
        <string>start</string>
        <string>--http-port</string>
        <string>%d</string>
        <string>--https-port</string>
        <string>%d</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%s</string>
    <key>StandardErrorPath</key>
    <string>%s</string>
</dict>
</plist>
`, launchdLabel, mfBinary, httpPort, httpsPort, LogPath, LogPath)

	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		return fmt.Errorf("could not write plist %s (try running with sudo): %w", plistPath, err)
	}
	return nil
}

// LoadLaunchd (re)loads the launchd job. Requires root. Any previously loaded
// instance is unloaded first: `launchctl load` on an already-loaded job is a
// no-op, so a reinstall would otherwise keep running the old binary/flags.
func LoadLaunchd() error {
	_ = UnloadLaunchd()
	out, err := exec.Command("launchctl", "load", PlistPath()).CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl load failed: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// UnloadLaunchd unloads (stops) the launchd job. Requires root.
func UnloadLaunchd() error {
	out, err := exec.Command("launchctl", "unload", PlistPath()).CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "Could not find specified service") ||
			strings.Contains(string(out), "no such file") {
			return nil
		}
		return fmt.Errorf("launchctl unload failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// RemoveLaunchd stops the daemon and removes the plist.
func RemoveLaunchd() error {
	_ = UnloadLaunchd()
	if err := os.Remove(PlistPath()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("could not remove plist %s (try running with sudo): %w", PlistPath(), err)
	}
	return nil
}

// IsRunning reports whether the proxy process is currently running. Uses pgrep
// (works without root) because a root LaunchDaemon won't appear in a non-root
// user's `launchctl list`.
func IsRunning() bool {
	out, err := exec.Command("pgrep", "-f", "mf proxy start").Output()
	if err != nil {
		return false
	}
	self := strconv.Itoa(os.Getpid())
	for _, pid := range strings.Fields(string(out)) {
		if pid != self {
			return true
		}
	}
	return false
}

// WaitListening polls until something accepts TCP connections on
// 127.0.0.1:port or the timeout elapses.
func WaitListening(port int, timeout time.Duration) bool {
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	deadline := time.Now().Add(timeout)
	for {
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// TailLog returns up to the last n lines of the daemon log.
func TailLog(n int) string {
	data, err := os.ReadFile(LogPath)
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
