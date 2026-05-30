package dns

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const launchdLabel = "com.mf-cli.dns"

// launchDaemonsDir is where system-wide LaunchDaemons live. The DNS service is
// installed as a root LaunchDaemon (not a per-user LaunchAgent) so it binds the
// DNS port reliably and is loaded consistently when install runs under sudo.
const launchDaemonsDir = "/Library/LaunchDaemons"

// PlistPath returns the path to the launchd plist for the DNS daemon.
func PlistPath() string {
	return filepath.Join(launchDaemonsDir, launchdLabel+".plist")
}

// InstallLaunchd writes the LaunchDaemon plist that auto-starts `mf dns start`.
// Must run as root (the plist lives under /Library/LaunchDaemons).
func InstallLaunchd(mfBinary, listenAddr string, port int) error {
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
        <string>dns</string>
        <string>start</string>
        <string>--addr</string>
        <string>%s</string>
        <string>--port</string>
        <string>%d</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/mf-dns.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/mf-dns.log</string>
</dict>
</plist>
`, launchdLabel, mfBinary, listenAddr, port)

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

// IsRunning reports whether the DNS server process is currently running.
// Uses pgrep (works without root) because a root LaunchDaemon won't appear in
// a non-root user's `launchctl list`.
func IsRunning() bool {
	return exec.Command("pgrep", "-f", "mf dns start").Run() == nil
}
