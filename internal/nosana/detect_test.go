package nosana

import (
	"context"
	"testing"
	"time"

	"github.com/MachoDrone/nosana-gridlens/internal/config"
	"github.com/MachoDrone/nosana-gridlens/internal/execx"
)

func TestMatcherSupportsExactAndGlobNames(t *testing.T) {
	matcher := Matcher{
		ExactNames: []string{"host-a"},
		Patterns:   []string{"*nosana*"},
	}

	if matched, _ := matcher.Match("host-a"); !matched {
		t.Fatalf("expected exact name match")
	}
	if matched, _ := matcher.Match("gpu-nosana-1"); !matched {
		t.Fatalf("expected glob match")
	}
	if matched, _ := matcher.Match("other"); matched {
		t.Fatalf("did not expect unrelated match")
	}
}

func TestDetectLocalDockerAndNestedPodman(t *testing.T) {
	runner := execx.NewFakeRunner()
	runner.SetPath("docker", "/usr/bin/docker")
	runner.SetResult("docker", []string{"ps", "--format", "{{json .}}"}, execx.Result{
		ExitCode: 0,
		Stdout:   `{"ID":"abc123","Names":"docker-shell","Image":"ubuntu","Status":"Up 2 hours"}`,
	})
	runner.SetResult("docker", []string{"exec", "abc123", "sh", "-lc", "command -v podman >/dev/null 2>&1 && podman ps --format json || true"}, execx.Result{
		ExitCode: 0,
		Stdout:   `[{"Id":"nested1","Names":["nosana-node-1"],"Image":"nosana/node","Status":"running"}]`,
	})

	report := Detect(context.Background(), runner, config.Default(), Options{
		IncludeNested: true,
		Now:           time.Unix(0, 0),
	})

	if report.Summary.ContainersSeen != 2 {
		t.Fatalf("expected docker parent and nested podman container, got %+v", report.Summary)
	}
	if report.Summary.NosanaMatches != 1 {
		t.Fatalf("expected one Nosana match, got %+v", report.Summary)
	}
}

func TestConfiguredPCWithSSHUsesManualContainerNames(t *testing.T) {
	runner := execx.NewFakeRunner()
	runner.SetPath("ssh", "/usr/bin/ssh")
	cfg := config.Default()
	if err := cfg.AddOrUpdatePC(config.PC{
		Name:           "nodebox",
		Address:        "192.168.0.167",
		SSHTarget:      "grid@192.168.0.167",
		Runtimes:       []string{"docker"},
		ContainerNames: []string{"custom-host-a"},
	}); err != nil {
		t.Fatalf("AddOrUpdatePC returned error: %v", err)
	}
	runner.SetResult("ssh", []string{"-o", "BatchMode=yes", "-o", "ConnectTimeout=3", "grid@192.168.0.167", "docker ps --format '{{json .}}'"}, execx.Result{
		ExitCode: 0,
		Stdout:   `{"ID":"def456","Names":"custom-host-a","Image":"nosana/node","Status":"Up"}`,
	})

	report := Detect(context.Background(), runner, cfg, Options{Now: time.Unix(0, 0)})

	if report.Summary.NosanaMatches != 1 {
		t.Fatalf("expected manual container name match, got %+v", report.Summary)
	}
}

func TestConfiguredPCWithoutSSHIsSkipped(t *testing.T) {
	runner := execx.NewFakeRunner()
	cfg := config.Default()
	if err := cfg.AddOrUpdatePC(config.PC{Name: "nodebox", Address: "192.168.0.167"}); err != nil {
		t.Fatalf("AddOrUpdatePC returned error: %v", err)
	}

	report := Detect(context.Background(), runner, cfg, Options{Now: time.Unix(0, 0)})
	if len(report.Targets) != 2 {
		t.Fatalf("expected local plus configured target, got %+v", report.Targets)
	}
	if !report.Targets[1].Skipped {
		t.Fatalf("expected configured PC without ssh target to be skipped: %+v", report.Targets[1])
	}
}
