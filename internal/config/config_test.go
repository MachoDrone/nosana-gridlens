package config

import (
	"path/filepath"
	"testing"
)

func TestLoadMissingUsesDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(cfg.DefaultContainerPatterns) == 0 {
		t.Fatalf("expected default container patterns")
	}
	if cfg.PCs == nil {
		t.Fatalf("expected non-nil PCs")
	}
}

func TestAddOrUpdatePCStoresManualContainerNames(t *testing.T) {
	cfg := Default()
	err := cfg.AddOrUpdatePC(PC{
		Name:              "nodebox",
		Address:           "192.168.0.167",
		SSHTarget:         "grid@192.168.0.167",
		Runtimes:          []string{"docker", "podman", "docker"},
		ContainerNames:    []string{"nosana-a", "nosana-b", "nosana-a"},
		ContainerPatterns: []string{"gpu-*"},
	})
	if err != nil {
		t.Fatalf("AddOrUpdatePC returned error: %v", err)
	}

	if len(cfg.PCs) != 1 {
		t.Fatalf("expected one PC, got %+v", cfg.PCs)
	}
	pc := cfg.PCs[0]
	if len(pc.ContainerNames) != 2 {
		t.Fatalf("expected deduplicated container names, got %+v", pc.ContainerNames)
	}
	if len(pc.Runtimes) != 2 {
		t.Fatalf("expected deduplicated runtimes, got %+v", pc.Runtimes)
	}
}

func TestSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := Default()
	if err := cfg.AddOrUpdatePC(PC{Name: "nodebox", Address: "192.168.0.167"}); err != nil {
		t.Fatalf("AddOrUpdatePC returned error: %v", err)
	}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(loaded.PCs) != 1 || loaded.PCs[0].Name != "nodebox" {
		t.Fatalf("unexpected loaded config: %+v", loaded)
	}
}
