package cmd

import (
	"fmt"
	"net"
	"os"
	"os/exec"

	"github.com/jlavera/mf-cli/internal/config"
	localdns "github.com/jlavera/mf-cli/internal/dns"
	"github.com/spf13/cobra"
)

var dnsCmd = &cobra.Command{
	Use:   "dns",
	Short: "Manage local DNS resolution for .mf domains",
	Long: `Manage the local DNS resolver that points *.<tld> (default *.mf) at 127.0.0.1.

It works by writing /etc/resolver/<tld> so macOS forwards those lookups to a
small DNS daemon that mf installs as a root LaunchDaemon. Combined with
'mf proxy', this lets you reach services by name (e.g. https://web.my-project.mf).

Run 'sudo mf dns install' once per machine, then enable DNS per project with a
'dns:' block in mf.yaml. macOS only.`,
}

var dnsInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Create resolver file and start DNS daemon (may require sudo)",
	Annotations: map[string]string{"skipConfig": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		tld := dnsInstallTLD
		addr := dnsInstallAddr
		port := dnsInstallPort

		resolverPath := localdns.ResolverFilePath(tld)

		// Check for conflicts before elevating so we fail fast with a clear
		// message (the resolver file is world-readable, no root needed to read).
		if err := localdns.CheckConflict(resolverPath, addr, port); err != nil {
			return err
		}

		if handled, err := elevate(fmt.Sprintf(
			"Local DNS setup needs administrator access to:\n"+
				"  • write %s (tells macOS to route *.%s lookups to mf)\n"+
				"  • install a system DNS service in /Library/LaunchDaemons",
			resolverPath, tld)); handled {
			return err
		}

		mfBin, err := os.Executable()
		if err != nil {
			return fmt.Errorf("could not determine mf binary path: %w", err)
		}

		fmt.Printf("📄 Writing resolver file: %s\n", resolverPath)
		if err := localdns.WriteResolverFile(resolverPath, addr, port); err != nil {
			return fmt.Errorf("failed to write resolver file (try: sudo mf dns install): %w", err)
		}

		fmt.Println("⚙️  Installing launchd daemon...")
		if err := localdns.InstallLaunchd(mfBin, addr, port); err != nil {
			return err
		}
		if err := localdns.LoadLaunchd(); err != nil {
			return fmt.Errorf("failed to load launchd job: %w", err)
		}

		fmt.Printf("\n✅ DNS installed — *.%s resolves to %s\n", tld, addr)
		return nil
	},
}

var dnsUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Stop DNS daemon and remove resolver file",
	Annotations: map[string]string{"skipConfig": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		tld := dnsUninstallTLD

		resolverPath := localdns.ResolverFilePath(tld)

		if handled, err := elevate(fmt.Sprintf(
			"Uninstalling local DNS needs administrator access to delete %s\n"+
				"and the system DNS service in /Library/LaunchDaemons.",
			resolverPath)); handled {
			return err
		}

		fmt.Printf("🗑  Removing resolver file: %s\n", resolverPath)
		if err := localdns.RemoveResolverFile(resolverPath); err != nil {
			return err
		}

		fmt.Println("⚙️  Removing launchd daemon...")
		if err := localdns.RemoveLaunchd(); err != nil {
			return err
		}

		fmt.Println("✅ DNS removed")
		return nil
	},
}

var (
	dnsStartAddr string
	dnsStartPort int
)

var dnsStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Run the DNS server in the foreground",
	Annotations: map[string]string{"skipConfig": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		listenAddr := fmt.Sprintf("%s:%d", dnsStartAddr, dnsStartPort)
		replyIP := net.ParseIP(dnsStartAddr)
		if replyIP == nil {
			return fmt.Errorf("invalid address: %s", dnsStartAddr)
		}

		srv := localdns.NewServer(listenAddr, replyIP)
		return srv.Start()
	},
}

var dnsStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the DNS daemon without uninstalling",
	Annotations: map[string]string{"skipConfig": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if handled, err := elevate(
			"Stopping the system DNS service needs administrator access\n" +
				"(it runs as a root LaunchDaemon)."); handled {
			return err
		}
		if err := localdns.UnloadLaunchd(); err != nil {
			return err
		}
		fmt.Println("✅ DNS daemon stopped (will restart on next boot; use 'mf dns uninstall' to remove)")
		return nil
	},
}

var dnsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show DNS daemon status and resolver configuration",
	Annotations: map[string]string{"skipConfig": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if localdns.IsRunning() {
			fmt.Println("✅ DNS daemon is running")
		} else {
			fmt.Println("❌ DNS daemon is not running")
		}

		tld := dnsStatusTLD
		resolverPath := localdns.ResolverFilePath(tld)
		rc, err := localdns.ParseResolverFile(resolverPath)
		if err != nil {
			return err
		}
		if rc == nil {
			fmt.Printf("\n📄 No resolver file at %s\n", resolverPath)
		} else {
			fmt.Printf("\n📄 Resolver file: %s\n", resolverPath)
			fmt.Printf("   nameserver %s\n", rc.Nameserver)
			fmt.Printf("   port %d\n", rc.Port)
		}

		// Quick verification
		fmt.Printf("\n🔍 Testing resolution of test.%s...\n", tld)
		out, err := exec.Command("dscacheutil", "-q", "host", "-a", "name", "test."+tld).Output()
		if err == nil && len(out) > 0 {
			fmt.Printf("   %s", out)
		} else {
			fmt.Println("   Could not resolve (DNS may not be working)")
		}

		return nil
	},
}

var (
	dnsInstallTLD   string
	dnsInstallAddr  string
	dnsInstallPort  int
	dnsUninstallTLD string
	dnsStatusTLD    string
)

func init() {
	dnsInstallCmd.Flags().StringVar(&dnsInstallTLD, "tld", "mf", "TLD suffix")
	dnsInstallCmd.Flags().StringVar(&dnsInstallAddr, "addr", "127.0.0.1", "DNS address")
	dnsInstallCmd.Flags().IntVar(&dnsInstallPort, "port", config.DefaultDNSPort, "DNS port")

	dnsUninstallCmd.Flags().StringVar(&dnsUninstallTLD, "tld", "mf", "TLD suffix to remove")

	dnsStartCmd.Flags().StringVar(&dnsStartAddr, "addr", "127.0.0.1", "listen address")
	dnsStartCmd.Flags().IntVar(&dnsStartPort, "port", config.DefaultDNSPort, "listen port")

	dnsStatusCmd.Flags().StringVar(&dnsStatusTLD, "tld", "mf", "TLD suffix to check")

	dnsCmd.AddCommand(dnsInstallCmd, dnsUninstallCmd, dnsStartCmd, dnsStopCmd, dnsStatusCmd)
	dnsCmd.GroupID = "general"
	rootCmd.AddCommand(dnsCmd)
}
