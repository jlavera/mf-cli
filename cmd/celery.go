package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var celeryCmd = &cobra.Command{
	Use:   "celery",
	Short: "Manage Celery workers",
}

var celeryStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start Celery workers",
	RunE: func(cmd *cobra.Command, args []string) error {
		workers := cfg.Services.Workers
		if len(workers) == 0 {
			return fmt.Errorf("no worker services configured — set services.workers in mf.yaml")
		}
		return comp.Up(workers...)
	},
}

var celeryStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop Celery workers",
	RunE: func(cmd *cobra.Command, args []string) error {
		workers := cfg.Services.Workers
		if len(workers) == 0 {
			return fmt.Errorf("no worker services configured — set services.workers in mf.yaml")
		}
		return comp.Stop(workers...)
	},
}

var celeryRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart Celery workers",
	RunE: func(cmd *cobra.Command, args []string) error {
		workers := cfg.Services.Workers
		if len(workers) == 0 {
			return fmt.Errorf("no worker services configured — set services.workers in mf.yaml")
		}
		return comp.Restart(workers...)
	},
}

var celeryLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View Celery worker logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		workers := cfg.Services.Workers
		if len(workers) == 0 {
			return fmt.Errorf("no worker services configured — set services.workers in mf.yaml")
		}
		return comp.Logs(workers...)
	},
}

var flowerCmd = &cobra.Command{
	Use:   "flower",
	Short: "Manage Celery Flower",
}

var flowerLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View Flower logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		service := cfg.Services.Flower
		if service == "" {
			return fmt.Errorf("no flower service configured — set services.flower in mf.yaml")
		}
		return comp.Logs(service)
	},
}

func init() {
	celeryCmd.AddCommand(celeryStartCmd)
	celeryCmd.AddCommand(celeryStopCmd)
	celeryCmd.AddCommand(celeryRestartCmd)
	celeryCmd.AddCommand(celeryLogsCmd)
	celeryCmd.GroupID = "stack"
	rootCmd.AddCommand(celeryCmd)

	flowerCmd.AddCommand(flowerLogsCmd)
	flowerCmd.GroupID = "stack"
	rootCmd.AddCommand(flowerCmd)
}
