package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/jlavera/mf-cli/internal/proxy"
	"github.com/spf13/cobra"
)

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Manage the local reverse proxy for .mf domains",
	Long: `Manage the reverse proxy that routes *.<tld> hostnames to the right container.

It listens on ports 80/443 as a root LaunchDaemon and forwards each request to
the matching service's published port, terminating HTTPS with a locally-trusted
certificate authority. Routes are registered automatically by 'mf up' and removed
by 'mf down' (the proxy polls for changes, no restart needed).

Run 'sudo mf proxy install' once per machine (alongside 'mf dns install'). macOS only.`,
}

var proxyInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Generate local CA, trust it, and start the proxy daemon (may require sudo)",
	Annotations: map[string]string{"skipConfig": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if handled, err := elevate(fmt.Sprintf(
			"Local proxy setup needs administrator access to:\n"+
				"  • add a local HTTPS certificate authority to your System keychain\n"+
				"  • bind ports %d/%d via a system service in /Library/LaunchDaemons",
			proxyHTTPPort, proxyHTTPSPort)); handled {
			return err
		}

		if proxy.CAExists() {
			fmt.Println("🔐 Using existing local CA")
		} else {
			fmt.Println("🔐 Generating local CA...")
			if _, err := proxy.GenerateCA(); err != nil {
				return fmt.Errorf("failed to generate CA (try: sudo mf proxy install): %w", err)
			}

			fmt.Println("🔐 Trusting CA in macOS keychain...")
			trustCmd := exec.Command("security", "add-trusted-cert", "-d", "-r", "trustRoot",
				"-k", "/Library/Keychains/System.keychain", proxy.RootCertPath())
			trustCmd.Stdout = os.Stdout
			trustCmd.Stderr = os.Stderr
			if err := trustCmd.Run(); err != nil {
				return fmt.Errorf("failed to trust CA (try: sudo mf proxy install): %w", err)
			}
		}

		fmt.Println("📁 Preparing routes directory...")
		if err := proxy.EnsureRoutesDir(); err != nil {
			return fmt.Errorf("failed to create routes directory (try sudo): %w", err)
		}

		mfBin, err := os.Executable()
		if err != nil {
			return fmt.Errorf("could not determine mf binary path: %w", err)
		}

		fmt.Println("⚙️  Installing launchd daemon...")
		if err := proxy.InstallLaunchd(mfBin, proxyHTTPPort, proxyHTTPSPort); err != nil {
			return err
		}
		if err := proxy.LoadLaunchd(); err != nil {
			return fmt.Errorf("failed to load launchd job: %w", err)
		}

		fmt.Printf("\n✅ Proxy installed — HTTPS reverse proxy active on :%d/:%d\n", proxyHTTPPort, proxyHTTPSPort)
		return nil
	},
}

var proxyUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Stop proxy daemon, untrust CA, and clean up",
	Annotations: map[string]string{"skipConfig": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if handled, err := elevate(
			"Uninstalling the local proxy needs administrator access to untrust the\n" +
				"certificate authority and remove the system service in /Library/LaunchDaemons."); handled {
			return err
		}

		fmt.Println("⚙️  Removing launchd daemon...")
		if err := proxy.RemoveLaunchd(); err != nil {
			return err
		}

		if proxy.CAExists() {
			fmt.Println("🔐 Removing CA from keychain...")
			exec.Command("security", "remove-trusted-cert", "-d", proxy.RootCertPath()).Run()
		}

		fmt.Println("🗑  Removing CA and certificates...")
		if err := proxy.RemoveCA(); err != nil {
			return err
		}

		fmt.Println("✅ Proxy removed")
		return nil
	},
}

var (
	proxyHTTPPort  int
	proxyHTTPSPort int
)

var proxyStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Run the reverse proxy in the foreground",
	Annotations: map[string]string{"skipConfig": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		ca, err := proxy.LoadCA()
		if err != nil {
			return err
		}

		httpAddr := fmt.Sprintf(":%d", proxyHTTPPort)
		httpsAddr := fmt.Sprintf(":%d", proxyHTTPSPort)
		srv := proxy.NewServer(httpAddr, httpsAddr, ca)
		return srv.Start()
	},
}

var proxyStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the proxy daemon without uninstalling",
	Annotations: map[string]string{"skipConfig": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if handled, err := elevate(
			"Stopping the system proxy service needs administrator access\n" +
				"(it runs as a root LaunchDaemon)."); handled {
			return err
		}
		if err := proxy.UnloadLaunchd(); err != nil {
			return err
		}
		fmt.Println("✅ Proxy daemon stopped (will restart on next boot; use 'mf proxy uninstall' to remove)")
		return nil
	},
}

var proxyStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show proxy status and active routes",
	Annotations: map[string]string{"skipConfig": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if proxy.IsRunning() {
			fmt.Println("✅ Proxy daemon is running")
		} else {
			fmt.Println("❌ Proxy daemon is not running")
		}

		routes, err := proxy.LoadAllRoutes()
		if err != nil {
			return err
		}
		if len(routes) == 0 {
			fmt.Println("\nNo active routes")
		} else {
			fmt.Printf("\nActive routes (%d):\n", len(routes))
			for hostname, target := range routes {
				fmt.Printf("  %s → %s\n", hostname, target)
			}
		}
		return nil
	},
}

func init() {
	proxyInstallCmd.Flags().IntVar(&proxyHTTPPort, "http-port", 80, "HTTP listen port")
	proxyInstallCmd.Flags().IntVar(&proxyHTTPSPort, "https-port", 443, "HTTPS listen port")

	proxyStartCmd.Flags().IntVar(&proxyHTTPPort, "http-port", 80, "HTTP listen port")
	proxyStartCmd.Flags().IntVar(&proxyHTTPSPort, "https-port", 443, "HTTPS listen port")

	proxyCmd.AddCommand(proxyInstallCmd, proxyUninstallCmd, proxyStartCmd, proxyStopCmd, proxyStatusCmd)
	proxyCmd.GroupID = "general"
	rootCmd.AddCommand(proxyCmd)
}
