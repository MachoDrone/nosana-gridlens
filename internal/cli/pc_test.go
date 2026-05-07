package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/MachoDrone/nosana-gridlens/internal/config"
	"github.com/MachoDrone/nosana-gridlens/internal/execx"
)

func TestPCAddSupportsNameBeforeFlags(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv(config.EnvPath, path)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewWithRunner(&stdout, &stderr, execx.NewFakeRunner())

	code := app.Run(context.Background(), []string{
		"pc", "add", "gpu-box",
		"--address", "192.168.0.167",
		"--ssh", "grid@192.168.0.167",
		"--container", "custom-a",
		"--pattern", "nosana-*",
	})
	if code != 0 {
		t.Fatalf("pc add returned %d: %s", code, stderr.String())
	}

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(loaded.PCs) != 1 {
		t.Fatalf("expected one PC, got %+v", loaded.PCs)
	}
	if loaded.PCs[0].Name != "gpu-box" || loaded.PCs[0].ContainerNames[0] != "custom-a" {
		t.Fatalf("unexpected saved PC: %+v", loaded.PCs[0])
	}
}

func TestPCListJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv(config.EnvPath, path)
	cfg := config.Default()
	if err := cfg.AddOrUpdatePC(config.PC{Name: "gpu-box", Address: "192.168.0.167"}); err != nil {
		t.Fatalf("AddOrUpdatePC returned error: %v", err)
	}
	if err := config.Save(path, cfg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewWithRunner(&stdout, &stderr, execx.NewFakeRunner())
	code := app.Run(context.Background(), []string{"pc", "list", "--json"})
	if code != 0 {
		t.Fatalf("pc list returned %d: %s", code, stderr.String())
	}

	var output struct {
		ConfigPath string        `json:"configPath"`
		Config     config.Config `json:"config"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if output.ConfigPath != path || len(output.Config.PCs) != 1 {
		t.Fatalf("unexpected pc list output: %+v", output)
	}
}
