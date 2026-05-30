package proxy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const launchdLabel = "com.mf-cli.proxy"

// launchDaemonsDir is where system-wide LaunchDaemons live. The proxy runs as a
// root LaunchDaemon so it can bind ports 80/443 directly (no pfctl redirect).
const launchDaemonsDir = "/Library/LaunchDaemons"

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
    <string>/tmp/mf-proxy.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/mf-proxy.log</string>
</dict>
</plist>
`, launchdLabel, mfBinary, httpPort, httpsPort)

	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		return fmt.Errorf("could not write plist %s (try running with sudo): %w", plistPath, err)
	}
	return nil
}

// LoadLaunchd loads (starts) the launchd job. Requires root.
func LoadLaunchd() error {
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
	return exec.Command("pgrep", "-f", "mf proxy start").Run() == nil
}
