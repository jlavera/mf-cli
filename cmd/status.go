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
		maxState := len("not created")
		for _, svc := range cfg.Services {
			if len(svc.Name) > maxName {
				maxName = len(svc.Name)
			}
			if len(svc.Type) > maxType {
				maxType = len(svc.Type)
			}
			if cs, ok := byService[svc.Name]; ok {
				if l := len(formatState(cs)); l > maxState {
					maxState = l
				}
			}
		}

		for _, svc := range cfg.Services {
			cs, ok := byService[svc.Name]
			state := "not created"
			icon := "❌"
			ports := ""
			if ok {
				state = formatState(cs)
				if isHealthy(cs) {
					icon = "✅"
				}
				ports = formatPorts(cs.Publishers)
			}

			svcType := svc.Type
			if svcType == "" {
				svcType = "-"
			}
			line := fmt.Sprintf("  %s %-*s  %-*s  %-*s", icon, maxName, svc.Name, maxType, svcType, maxState, state)
			if ports != "" {
				line += "  " + ports
			}
			fmt.Println(line)
		}

		return nil
	},
}

func formatPorts(publishers []compose.PortMapping) string {
	seen := make(map[string]bool)
	var parts []string
	for _, p := range publishers {
		if p.PublishedPort == 0 {
			continue
		}
		key := fmt.Sprintf("%d->%d", p.PublishedPort, p.TargetPort)
		if seen[key] {
			continue
		}
		seen[key] = true
		parts = append(parts, fmt.Sprintf("host:%d → container:%d", p.PublishedPort, p.TargetPort))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ", ")
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
