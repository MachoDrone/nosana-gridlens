package cli

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/MachoDrone/nosana-gridlens/internal/execx"
)

func TestSetupWireGuardDryRunDoesNotTargetExistingInterfaces(t *testing.T) {
	runner := runnerWithAllDependencies()
	runner.SetResult("wg", []string{"show", "interfaces"}, execx.Result{
		Stdout:   "wg0 corpvpn",
		ExitCode: 0,
	})

	plan := BuildSetupWireGuardPlan(context.Background(), runner, time.Unix(0, 0))

	if !plan.DryRun {
		t.Fatalf("expected dry-run plan")
	}
	if plan.PrivilegedOperationsRun {
		t.Fatalf("dry-run must not run privileged operations")
	}
	if plan.InterfaceName != "glwg0" {
		t.Fatalf("expected glwg0, got %s", plan.InterfaceName)
	}

	for _, command := range runner.Commands {
		text := command.String()
		if strings.Contains(text, "wg0") || strings.Contains(text, "corpvpn") {
			t.Fatalf("read-only discovery must not target existing interface in command: %s", text)
		}
	}

	if len(runner.Commands) != 1 || runner.Commands[0].String() != "wg show interfaces" {
		t.Fatalf("expected only wg show interfaces, got %+v", runner.Commands)
	}
}

func TestSetupWireGuardDryRunPlansPromptForMissingDependencies(t *testing.T) {
	runner := execx.NewFakeRunner()
	runner.SetPath("apt-get", "/usr/bin/apt-get")

	plan := BuildSetupWireGuardPlan(context.Background(), runner, time.Unix(0, 0))

	found := false
	for _, action := range plan.PlannedActions {
		if action.ID == "prompt-install-dependencies" {
			found = true
			if !action.Privileged {
				t.Fatalf("dependency install prompt should be marked privileged")
			}
			if strings.Join(action.Command, " ") != "sudo apt-get install -y wireguard-tools iproute2" {
				t.Fatalf("unexpected install command: %v", action.Command)
			}
		}
	}
	if !found {
		t.Fatalf("expected missing dependencies to produce prompt action")
	}
	if plan.PrivilegedOperationsRun {
		t.Fatalf("planning a prompt is not running a privileged operation")
	}
}

func runnerWithAllDependencies() *execx.FakeRunner {
	runner := execx.NewFakeRunner()
	for _, command := range []string{"wg", "wg-quick", "ip", "ss", "systemctl", "apt-get"} {
		runner.SetPath(command, "/usr/bin/"+command)
	}
	return runner
}
