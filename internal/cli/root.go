package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/MachoDrone/nosana-gridlens/internal/app"
	"github.com/MachoDrone/nosana-gridlens/internal/deps"
	"github.com/MachoDrone/nosana-gridlens/internal/execx"
	"github.com/MachoDrone/nosana-gridlens/internal/wireguard"
)

const (
	hubPort     = 8787
	udpPortFrom = 51871
	udpPortTo   = 51999
)

type App struct {
	out    io.Writer
	err    io.Writer
	runner execx.Runner
	now    func() time.Time
}

func New(out io.Writer, err io.Writer) *App {
	return &App{
		out:    out,
		err:    err,
		runner: execx.OSRunner{},
		now:    time.Now,
	}
}

func NewWithRunner(out io.Writer, err io.Writer, runner execx.Runner) *App {
	cli := New(out, err)
	cli.runner = runner
	return cli
}

func (a *App) Run(ctx context.Context, args []string) int {
	if len(args) == 0 {
		a.printHelp()
		return 0
	}

	switch args[0] {
	case "version":
		fmt.Fprintf(a.out, "%s %s\n", app.Name, app.Version)
		return 0
	case "deps":
		return a.runDeps(ctx, args[1:])
	case "pc":
		return a.runPC(ctx, args[1:])
	case "nosana":
		return a.runNosana(ctx, args[1:])
	case "setup":
		return a.runSetup(ctx, args[1:])
	case "doctor":
		return a.runDoctor(ctx, args[1:])
	case "help", "-h", "--help":
		a.printHelp()
		return 0
	default:
		fmt.Fprintf(a.err, "unknown command: %s\n\n", args[0])
		a.printHelp()
		return 2
	}
}

func (a *App) printHelp() {
	fmt.Fprintln(a.out, "GridLens")
	fmt.Fprintln(a.out)
	fmt.Fprintln(a.out, "Usage:")
	fmt.Fprintln(a.out, "  gridlens version")
	fmt.Fprintln(a.out, "  gridlens deps check [--json]")
	fmt.Fprintln(a.out, "  gridlens pc scan [--cidr CIDR] [--json]")
	fmt.Fprintln(a.out, "  gridlens pc add NAME --address IP [--ssh user@host] [--container NAME] [--pattern GLOB]")
	fmt.Fprintln(a.out, "  gridlens pc list [--json]")
	fmt.Fprintln(a.out, "  gridlens nosana detect [--json]")
	fmt.Fprintln(a.out, "  gridlens setup wireguard --dry-run [--json]")
	fmt.Fprintln(a.out, "  gridlens doctor wireguard [--json]")
}

func (a *App) runDeps(ctx context.Context, args []string) int {
	if len(args) == 0 || args[0] != "check" {
		fmt.Fprintln(a.err, "usage: gridlens deps check [--json]")
		return 2
	}

	jsonOutput := hasFlag(args[1:], "--json")
	discovery := a.discover(ctx)
	if jsonOutput {
		return writeJSON(a.out, discovery)
	}

	fmt.Fprintln(a.out, "Dependency check:")
	for _, status := range discovery.Dependencies {
		if status.Found {
			fmt.Fprintf(a.out, "  %s: found %s\n", status.Command, status.Path)
			continue
		}
		fmt.Fprintf(a.out, "  %s: missing", status.Command)
		if status.PackageHint != "" {
			fmt.Fprintf(a.out, " (package: %s)", status.PackageHint)
		}
		fmt.Fprintln(a.out)
	}
	return 0
}

func (a *App) runSetup(ctx context.Context, args []string) int {
	if len(args) == 0 || args[0] != "wireguard" {
		fmt.Fprintln(a.err, "usage: gridlens setup wireguard --dry-run [--json]")
		return 2
	}

	flags := args[1:]
	if !hasFlag(flags, "--dry-run") {
		fmt.Fprintln(a.err, "gridlens setup wireguard only supports --dry-run in this phase")
		fmt.Fprintln(a.err, "No privileged operation was run.")
		return 2
	}

	plan := BuildSetupWireGuardPlan(ctx, a.runner, a.now())
	if hasFlag(flags, "--json") {
		return writeJSON(a.out, plan)
	}

	fmt.Fprintln(a.out, "GridLens WireGuard setup dry-run")
	fmt.Fprintln(a.out)
	fmt.Fprintln(a.out, "No privileged operation was run.")
	fmt.Fprintln(a.out, "No WireGuard interface, config file, service, firewall rule, or route was modified.")
	fmt.Fprintln(a.out)
	fmt.Fprintf(a.out, "Planned GridLens interface: %s\n", plan.InterfaceName)
	fmt.Fprintf(a.out, "Hub UI port: %d\n", plan.HubPort)
	fmt.Fprintf(a.out, "WireGuard UDP port range: %d-%d\n", plan.UDPPortRange.Start, plan.UDPPortRange.End)
	fmt.Fprintln(a.out)

	printDiscovery(a.out, plan.Discovery)

	if len(plan.PlannedActions) > 0 {
		fmt.Fprintln(a.out)
		fmt.Fprintln(a.out, "Dry-run actions:")
		for _, action := range plan.PlannedActions {
			fmt.Fprintf(a.out, "  - %s\n", action.Description)
			if len(action.Command) > 0 {
				fmt.Fprintf(a.out, "    command: %s\n", strings.Join(action.Command, " "))
			}
		}
	}

	if len(plan.SafetyNotes) > 0 {
		fmt.Fprintln(a.out)
		fmt.Fprintln(a.out, "Safety notes:")
		for _, note := range plan.SafetyNotes {
			fmt.Fprintf(a.out, "  - %s\n", note)
		}
	}

	return 0
}

func (a *App) runDoctor(ctx context.Context, args []string) int {
	if len(args) == 0 || args[0] != "wireguard" {
		fmt.Fprintln(a.err, "usage: gridlens doctor wireguard [--json]")
		return 2
	}

	report := BuildWireGuardDoctorReport(ctx, a.runner, a.now())
	if hasFlag(args[1:], "--json") {
		return writeJSON(a.out, report)
	}

	fmt.Fprintln(a.out, "GridLens WireGuard Doctor")
	fmt.Fprintln(a.out)
	for _, check := range report.Checks {
		fmt.Fprintf(a.out, "%-5s %s: %s\n", strings.ToUpper(check.Status), check.Name, check.Message)
	}
	fmt.Fprintln(a.out)
	fmt.Fprintf(a.out, "Overall status: %s\n", report.Status)
	return 0
}

func (a *App) discover(ctx context.Context) Discovery {
	return Discover(ctx, a.runner)
}

func printDiscovery(out io.Writer, discovery Discovery) {
	osLabel := discovery.OS.PrettyName
	if osLabel == "" {
		osLabel = discovery.OS.ID
	}
	if osLabel == "" {
		osLabel = "unknown"
	}
	fmt.Fprintf(out, "OS: %s\n", osLabel)
	if discovery.PackageManager != nil {
		fmt.Fprintf(out, "Package manager: %s (%s)\n", discovery.PackageManager.Name, discovery.PackageManager.Path)
	} else {
		fmt.Fprintln(out, "Package manager: not detected")
	}

	if len(discovery.ExistingWireGuardInterfaces) == 0 {
		fmt.Fprintln(out, "Existing WireGuard interfaces: none detected")
	} else {
		fmt.Fprintf(out, "Existing WireGuard interfaces: %s\n", strings.Join(discovery.ExistingWireGuardInterfaces, ", "))
	}
	fmt.Fprintf(out, "GridLens interface ownership: %s\n", discovery.GridLensInterface.OwnershipStatus)
}

func hasFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag {
			return true
		}
	}
	return false
}

func writeJSON(out io.Writer, value any) int {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return 1
	}
	return 0
}

type Discovery struct {
	OS                          deps.OSInfo                  `json:"os"`
	PackageManager              *deps.PackageManager         `json:"packageManager,omitempty"`
	Dependencies                []deps.DependencyStatus      `json:"dependencies"`
	ExistingWireGuardInterfaces []string                     `json:"existingWireGuardInterfaces"`
	GridLensInterface           wireguard.InterfaceOwnership `json:"gridLensInterface"`
	WireGuardDiscoveryError     string                       `json:"wireGuardDiscoveryError,omitempty"`
}

func Discover(ctx context.Context, runner execx.Runner) Discovery {
	dependencyStatuses := deps.DetectDependencies(runner)
	interfaces, err := wireguard.ExistingInterfaces(ctx, runner)

	discovery := Discovery{
		OS:                          deps.DetectOS(),
		PackageManager:              deps.DetectPackageManager(runner),
		Dependencies:                dependencyStatuses,
		ExistingWireGuardInterfaces: interfaces,
		GridLensInterface:           wireguard.InspectInterfaceOwnership(wireguard.DefaultInterface, interfaces, wireguard.ConfigPath),
	}
	if err != nil {
		discovery.WireGuardDiscoveryError = err.Error()
	}
	return discovery
}
