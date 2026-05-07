package deps

import (
	"testing"

	"github.com/MachoDrone/nosana-gridlens/internal/execx"
)

func TestDetectDependenciesUsesRunnerLookPath(t *testing.T) {
	runner := execx.NewFakeRunner()
	runner.SetPath("wg", "/usr/bin/wg")
	runner.SetPath("ip", "/usr/sbin/ip")

	statuses := DetectDependencies(runner)

	found := map[string]DependencyStatus{}
	for _, status := range statuses {
		found[status.Command] = status
	}

	if !found["wg"].Found || found["wg"].Path != "/usr/bin/wg" {
		t.Fatalf("expected wg to be found, got %+v", found["wg"])
	}
	if found["wg-quick"].Found {
		t.Fatalf("expected wg-quick to be missing")
	}
	if found["wg-quick"].PackageHint != "wireguard-tools" {
		t.Fatalf("expected wireguard-tools hint, got %q", found["wg-quick"].PackageHint)
	}
}

func TestParseOSReleaseSupportsUbuntuLikeDebian(t *testing.T) {
	info := ParseOSRelease(`ID=ubuntu
NAME="Ubuntu"
VERSION_ID="24.04"
PRETTY_NAME="Ubuntu 24.04 LTS"
`)

	if !info.Supported {
		t.Fatalf("expected Ubuntu to be supported")
	}
	if info.Family != "debian" {
		t.Fatalf("expected debian family, got %q", info.Family)
	}
}

func TestDetectPackageManagerPrefersAptGet(t *testing.T) {
	runner := execx.NewFakeRunner()
	runner.SetPath("apt-get", "/usr/bin/apt-get")
	runner.SetPath("dnf", "/usr/bin/dnf")

	pm := DetectPackageManager(runner)
	if pm == nil {
		t.Fatal("expected package manager")
	}
	if pm.Name != "apt-get" {
		t.Fatalf("expected apt-get, got %s", pm.Name)
	}
}
