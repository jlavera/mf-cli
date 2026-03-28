package cmd

import (
	"fmt"
	"strings"

	"github.com/jlavera/mf-cli/internal/compose"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of each service",
	RunE: func(cmd *cobra.Command, args []string) error {
		statuses, err := comp.Ps()
		if err != nil {
			return fmt.Errorf("could not get container status: %w", err)
		}

		byService := make(map[string]compose.ContainerStatus)
		for _, cs := range statuses {
			byService[cs.Service] = cs
		}

		maxName := 0
		maxType := 0
		for _, svc := range cfg.Services {
			if len(svc.Name) > maxName {
				maxName = len(svc.Name)
			}
			if len(svc.Type) > maxType {
				maxType = len(svc.Type)
			}
		}

		for _, svc := range cfg.Services {
			cs, ok := byService[svc.Name]
			state := "not created"
			icon := "❌"
			if ok {
				state = formatState(cs)
				if isHealthy(cs) {
					icon = "✅"
				}
			}

			svcType := svc.Type
			if svcType == "" {
				svcType = "-"
			}
			fmt.Printf("  %s %-*s  %-*s  %s\n", icon, maxName, svc.Name, maxType, svcType, state)
		}

		return nil
	},
}

func formatState(cs compose.ContainerStatus) string {
	state := strings.ToLower(cs.State)
	health := strings.ToLower(cs.Health)

	if health != "" && health != "none" {
		return state + " (" + health + ")"
	}
	return state
}

func isHealthy(cs compose.ContainerStatus) bool {
	return strings.ToLower(cs.State) == "running"
}

func init() {
	statusCmd.GroupID = "general"
	rootCmd.AddCommand(statusCmd)
}
