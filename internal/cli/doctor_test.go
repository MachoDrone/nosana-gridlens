package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/MachoDrone/nosana-gridlens/internal/execx"
)

func TestDoctorWireGuardJSON(t *testing.T) {
	runner := runnerWithAllDependencies()
	runner.SetResult("wg", []string{"show", "interfaces"}, execx.Result{
		Stdout:   "wg0 corpvpn",
		ExitCode: 0,
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewWithRunner(&stdout, &stderr, runner)

	code := app.Run(context.Background(), []string{"doctor", "wireguard", "--json"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, stderr.String())
	}

	var report WireGuardDoctorReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("doctor output is not JSON: %v\n%s", err, stdout.String())
	}

	if report.Component != "wireguard" {
		t.Fatalf("expected wireguard component, got %s", report.Component)
	}
	if report.PrivilegedOperationsRun {
		t.Fatalf("doctor must not run privileged operations")
	}

	foundExistingWGWarning := false
	for _, check := range report.Checks {
		if check.ID == "existing_wireguard_interfaces" && check.Status == "warn" {
			foundExistingWGWarning = true
		}
	}
	if !foundExistingWGWarning {
		t.Fatalf("expected existing WireGuard warning in %+v", report.Checks)
	}
}

func TestDoctorFailsUnownedGridLensInterface(t *testing.T) {
	runner := runnerWithAllDependencies()
	runner.SetResult("wg", []string{"show", "interfaces"}, execx.Result{
		Stdout:   "glwg0",
		ExitCode: 0,
	})

	report := BuildWireGuardDoctorReport(context.Background(), runner, fixedNow())
	if report.Status != "fail" {
		t.Fatalf("expected fail for unowned glwg0, got %s", report.Status)
	}
}

func fixedNow() time.Time {
	return time.Unix(0, 0)
}
