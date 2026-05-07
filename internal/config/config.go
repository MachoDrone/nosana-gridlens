package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const EnvPath = "GRIDLENS_CONFIG_PATH"

type Config struct {
	DefaultContainerPatterns []string `json:"defaultContainerPatterns"`
	PCs                      []PC     `json:"pcs"`
}

type PC struct {
	Name              string   `json:"name"`
	Address           string   `json:"address,omitempty"`
	SSHTarget         string   `json:"sshTarget,omitempty"`
	Runtimes          []string `json:"runtimes,omitempty"`
	ContainerNames    []string `json:"containerNames,omitempty"`
	ContainerPatterns []string `json:"containerPatterns,omitempty"`
}

func Default() Config {
	return Config{
		DefaultContainerPatterns: []string{"nosana-node", "*nosana*"},
		PCs:                      []PC{},
	}
}

func Path() (string, error) {
	if path := os.Getenv(EnvPath); path != "" {
		return path, nil
	}

	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gridlens", "config.json"), nil
}

func Load(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return Config{}, err
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	cfg.ApplyDefaults()
	return cfg, nil
}

func Save(path string, cfg Config) error {
	cfg.ApplyDefaults()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (c *Config) ApplyDefaults() {
	if len(c.DefaultContainerPatterns) == 0 {
		c.DefaultContainerPatterns = Default().DefaultContainerPatterns
	}
	if c.PCs == nil {
		c.PCs = []PC{}
	}
}

func (c *Config) AddOrUpdatePC(pc PC) error {
	pc.Name = strings.TrimSpace(pc.Name)
	pc.Address = strings.TrimSpace(pc.Address)
	pc.SSHTarget = strings.TrimSpace(pc.SSHTarget)
	pc.Runtimes = cleanList(pc.Runtimes)
	pc.ContainerNames = cleanList(pc.ContainerNames)
	pc.ContainerPatterns = cleanList(pc.ContainerPatterns)

	if pc.Name == "" {
		return fmt.Errorf("PC name is required")
	}
	if pc.Address == "" && pc.SSHTarget == "" {
		return fmt.Errorf("address or ssh target is required")
	}

	for i := range c.PCs {
		if c.PCs[i].Name == pc.Name {
			c.PCs[i] = pc
			return nil
		}
	}
	c.PCs = append(c.PCs, pc)
	return nil
}

func (c *Config) RemovePC(name string) bool {
	for i := range c.PCs {
		if c.PCs[i].Name == name {
			c.PCs = append(c.PCs[:i], c.PCs[i+1:]...)
			return true
		}
	}
	return false
}

func cleanList(values []string) []string {
	var cleaned []string
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		cleaned = append(cleaned, value)
	}
	return cleaned
}
