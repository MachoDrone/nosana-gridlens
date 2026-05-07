package wireguard

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"

	"github.com/MachoDrone/nosana-gridlens/internal/execx"
)

const (
	DefaultInterface = "glwg0"
	ConfigPath       = "/etc/wireguard/glwg0.conf"
	OwnerMarker      = "Owner marker: gridlens"
)

type InterfaceOwnership struct {
	Name            string `json:"name"`
	Exists          bool   `json:"exists"`
	Owned           bool   `json:"owned"`
	OwnershipStatus string `json:"ownershipStatus"`
	ConfigPath      string `json:"configPath"`
	Error           string `json:"error,omitempty"`
}

func ExistingInterfaces(ctx context.Context, runner execx.Runner) ([]string, error) {
	if _, err := runner.LookPath("wg"); err != nil {
		return []string{}, nil
	}

	result := runner.Run(ctx, "wg", "show", "interfaces")
	if !result.OK() {
		if result.Err != nil {
			return nil, result.Err
		}
		return nil, fmt.Errorf("wg show interfaces failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}

	interfaces := strings.Fields(result.Stdout)
	sort.Strings(interfaces)
	return interfaces, nil
}

func InspectInterfaceOwnership(name string, existing []string, configPath string) InterfaceOwnership {
	status := InterfaceOwnership{
		Name:            name,
		ConfigPath:      configPath,
		OwnershipStatus: "absent",
	}

	status.Exists = contains(existing, name) || interfaceExists(name)

	data, err := os.ReadFile(configPath)
	if err == nil {
		if strings.Contains(string(data), OwnerMarker) || strings.Contains(string(data), "Managed by GridLens") {
			status.Owned = true
			status.OwnershipStatus = "gridlens-owned"
			return status
		}
		if status.Exists {
			status.OwnershipStatus = "exists-without-gridlens-marker"
			return status
		}
		status.OwnershipStatus = "config-without-gridlens-marker"
		return status
	}

	if errors.Is(err, os.ErrNotExist) {
		if status.Exists {
			status.OwnershipStatus = "exists-without-gridlens-config"
		}
		return status
	}

	status.Error = err.Error()
	if status.Exists {
		status.OwnershipStatus = "unknown-read-error"
	} else {
		status.OwnershipStatus = "absent-config-unreadable"
	}
	return status
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func interfaceExists(name string) bool {
	_, err := net.InterfaceByName(name)
	return err == nil
}
