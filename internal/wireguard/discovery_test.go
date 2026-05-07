package wireguard

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/MachoDrone/nosana-gridlens/internal/execx"
)

func TestExistingInterfacesUsesReadOnlyWGShowInterfaces(t *testing.T) {
	runner := execx.NewFakeRunner()
	runner.SetPath("wg", "/usr/bin/wg")
	runner.SetResult("wg", []string{"show", "interfaces"}, execx.Result{Stdout: "wg0 corpvpn glwg0", ExitCode: 0})

	interfaces, err := ExistingInterfaces(context.Background(), runner)
	if err != nil {
		t.Fatalf("ExistingInterfaces returned error: %v", err)
	}

	want := []string{"corpvpn", "glwg0", "wg0"}
	if len(interfaces) != len(want) {
		t.Fatalf("expected %v, got %v", want, interfaces)
	}
	for i := range want {
		if interfaces[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, interfaces)
		}
	}

	if len(runner.Commands) != 1 || runner.Commands[0].String() != "wg show interfaces" {
		t.Fatalf("expected only read-only wg show interfaces, got %+v", runner.Commands)
	}
}

func TestInspectInterfaceOwnershipRefusesUnmarkedConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "glwg0.conf")
	writeFile(t, path, "[Interface]\nAddress = 10.94.23.1/24\n")

	status := InspectInterfaceOwnership("glwg0", []string{"glwg0"}, path)

	if status.Owned {
		t.Fatalf("unmarked config must not be treated as owned")
	}
	if status.OwnershipStatus != "exists-without-gridlens-marker" {
		t.Fatalf("unexpected ownership status: %+v", status)
	}
}

func TestInspectInterfaceOwnershipAcceptsGridLensMarker(t *testing.T) {
	path := filepath.Join(t.TempDir(), "glwg0.conf")
	writeFile(t, path, "# Owner marker: gridlens\n[Interface]\nAddress = 10.94.23.1/24\n")

	status := InspectInterfaceOwnership("glwg0", []string{"glwg0"}, path)

	if !status.Owned {
		t.Fatalf("marked config should be treated as GridLens-owned: %+v", status)
	}
	if status.OwnershipStatus != "gridlens-owned" {
		t.Fatalf("unexpected ownership status: %+v", status)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
}
