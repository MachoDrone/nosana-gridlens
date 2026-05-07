package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/MachoDrone/nosana-gridlens/internal/deps"
	"github.com/MachoDrone/nosana-gridlens/internal/execx"
)

type DoctorCheck struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type WireGuardDoctorReport struct {
	GeneratedAt             time.Time     `json:"generatedAt"`
	Component               string        `json:"component"`
	Status                  string        `json:"status"`
	PrivilegedOperationsRun bool          `json:"privilegedOperationsRun"`
	Discovery               Discovery     `json:"discovery"`
	Checks                  []DoctorCheck `json:"checks"`
}

func BuildWireGuardDoctorReport(ctx context.Context, runner execx.Runner, now time.Time) WireGuardDoctorReport {
	discovery := Discover(ctx, runner)
	checks := []DoctorCheck{
		checkOS(discovery),
		checkPackageManager(discovery),
	}

	for _, status := range discovery.Dependencies {
		checks = append(checks, checkDependency(status))
	}

	checks = append(checks,
		checkExistingInterfaces(discovery),
		checkGridLensInterfaceName(discovery),
		checkGridLensOwnership(discovery),
		DoctorCheck{
			ID:      "privileged_operations",
			Name:    "Privileged operations",
			Status:  "pass",
			Message: "none run",
		},
	)

	return WireGuardDoctorReport{
		GeneratedAt:             now.UTC(),
		Component:               "wireguard",
		Status:                  aggregateStatus(checks),
		PrivilegedOperationsRun: false,
		Discovery:               discovery,
		Checks:                  checks,
	}
}

func checkOS(discovery Discovery) DoctorCheck {
	if discovery.OS.Supported {
		label := discovery.OS.PrettyName
		if label == "" {
			label = discovery.OS.ID
		}
		return DoctorCheck{
			ID:      "os_supported",
			Name:    "OS supported",
			Status:  "pass",
			Message: label,
		}
	}
	return DoctorCheck{
		ID:      "os_supported",
		Name:    "OS supported",
		Status:  "warn",
		Message: "first supported platform is Ubuntu/Debian with systemd",
	}
}

func checkPackageManager(discovery Discovery) DoctorCheck {
	if discovery.PackageManager == nil {
		return DoctorCheck{
			ID:      "package_manager",
			Name:    "Package manager",
			Status:  "warn",
			Message: "not detected",
		}
	}
	return DoctorCheck{
		ID:      "package_manager",
		Name:    "Package manager",
		Status:  "pass",
		Message: discovery.PackageManager.Name,
	}
}

func checkDependency(status deps.DependencyStatus) DoctorCheck {
	if status.Found {
		return DoctorCheck{
			ID:      "dependency_" + status.Command,
			Name:    "Command " + status.Command,
			Status:  "pass",
			Message: status.Path,
		}
	}

	message := "missing"
	if status.PackageHint != "" {
		message = fmt.Sprintf("missing; package hint: %s", status.PackageHint)
	}

	return DoctorCheck{
		ID:      "dependency_" + status.Command,
		Name:    "Command " + status.Command,
		Status:  "fail",
		Message: message,
	}
}

func checkExistingInterfaces(discovery Discovery) DoctorCheck {
	if discovery.WireGuardDiscoveryError != "" {
		return DoctorCheck{
			ID:      "existing_wireguard_interfaces",
			Name:    "Existing WireGuard interfaces",
			Status:  "warn",
			Message: discovery.WireGuardDiscoveryError,
		}
	}
	if len(discovery.ExistingWireGuardInterfaces) == 0 {
		return DoctorCheck{
			ID:      "existing_wireguard_interfaces",
			Name:    "Existing WireGuard interfaces",
			Status:  "pass",
			Message: "none detected",
		}
	}
	return DoctorCheck{
		ID:      "existing_wireguard_interfaces",
		Name:    "Existing WireGuard interfaces",
		Status:  "warn",
		Message: strings.Join(discovery.ExistingWireGuardInterfaces, ", ") + " detected read-only",
	}
}

func checkGridLensInterfaceName(discovery Discovery) DoctorCheck {
	return DoctorCheck{
		ID:      "gridlens_interface_name",
		Name:    "GridLens interface name",
		Status:  "pass",
		Message: discovery.GridLensInterface.Name,
	}
}

func checkGridLensOwnership(discovery Discovery) DoctorCheck {
	status := discovery.GridLensInterface
	switch status.OwnershipStatus {
	case "absent":
		return DoctorCheck{
			ID:      "gridlens_ownership",
			Name:    "GridLens ownership",
			Status:  "pass",
			Message: "glwg0 is absent and available for future GridLens setup",
		}
	case "gridlens-owned":
		return DoctorCheck{
			ID:      "gridlens_ownership",
			Name:    "GridLens ownership",
			Status:  "pass",
			Message: "glwg0 has a GridLens ownership marker",
		}
	default:
		return DoctorCheck{
			ID:      "gridlens_ownership",
			Name:    "GridLens ownership",
			Status:  "fail",
			Message: "glwg0 exists or has config without a GridLens ownership marker",
		}
	}
}

func aggregateStatus(checks []DoctorCheck) string {
	status := "pass"
	for _, check := range checks {
		switch check.Status {
		case "fail":
			return "fail"
		case "warn":
			status = "warn"
		}
	}
	return status
}
