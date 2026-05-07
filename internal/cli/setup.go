package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/MachoDrone/nosana-gridlens/internal/deps"
	"github.com/MachoDrone/nosana-gridlens/internal/execx"
	"github.com/MachoDrone/nosana-gridlens/internal/wireguard"
)

type PortRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type PlannedAction struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Privileged  bool     `json:"privileged"`
	Command     []string `json:"command,omitempty"`
}

type SetupWireGuardPlan struct {
	GeneratedAt             time.Time       `json:"generatedAt"`
	DryRun                  bool            `json:"dryRun"`
	PrivilegedOperationsRun bool            `json:"privilegedOperationsRun"`
	InterfaceName           string          `json:"interfaceName"`
	HubPort                 int             `json:"hubPort"`
	UDPPortRange            PortRange       `json:"udpPortRange"`
	Discovery               Discovery       `json:"discovery"`
	PlannedActions          []PlannedAction `json:"plannedActions"`
	SafetyNotes             []string        `json:"safetyNotes"`
}

func BuildSetupWireGuardPlan(ctx context.Context, runner execx.Runner, now time.Time) SetupWireGuardPlan {
	discovery := Discover(ctx, runner)
	plan := SetupWireGuardPlan{
		GeneratedAt:             now.UTC(),
		DryRun:                  true,
		PrivilegedOperationsRun: false,
		InterfaceName:           wireguard.DefaultInterface,
		HubPort:                 hubPort,
		UDPPortRange:            PortRange{Start: udpPortFrom, End: udpPortTo},
		Discovery:               discovery,
		SafetyNotes: []string{
			"dry-run mode performs read-only discovery only",
			"GridLens will not use wg0",
			"GridLens may manage only resources with a GridLens ownership marker",
		},
	}

	if len(discovery.ExistingWireGuardInterfaces) > 0 {
		plan.SafetyNotes = append(plan.SafetyNotes, fmt.Sprintf(
			"existing WireGuard interfaces detected read-only: %v",
			discovery.ExistingWireGuardInterfaces,
		))
	}

	missing := deps.MissingRequired(discovery.Dependencies)
	if len(missing) > 0 {
		plan.PlannedActions = append(plan.PlannedActions, PlannedAction{
			ID:          "prompt-install-dependencies",
			Description: "would prompt before installing missing targeted dependencies",
			Privileged:  true,
			Command:     installCommand(discovery.PackageManager),
		})
	}

	switch discovery.GridLensInterface.OwnershipStatus {
	case "gridlens-owned":
		plan.PlannedActions = append(plan.PlannedActions, PlannedAction{
			ID:          "inspect-gridlens-interface",
			Description: "would inspect GridLens-owned glwg0 before deciding any future change",
			Privileged:  false,
		})
	case "absent":
		plan.PlannedActions = append(plan.PlannedActions, PlannedAction{
			ID:          "create-gridlens-interface-later",
			Description: "would create only glwg0 in a later non-dry-run phase after explicit confirmation",
			Privileged:  true,
		})
	default:
		plan.PlannedActions = append(plan.PlannedActions, PlannedAction{
			ID:          "refuse-unowned-gridlens-interface",
			Description: "would refuse to modify glwg0 because GridLens ownership is not proven",
			Privileged:  false,
		})
	}

	return plan
}

func installCommand(pm *deps.PackageManager) []string {
	if pm == nil {
		return nil
	}
	return append([]string(nil), pm.InstallCommand...)
}
