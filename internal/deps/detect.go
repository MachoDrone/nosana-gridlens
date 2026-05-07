package deps

import (
	"os"
	"strings"

	"github.com/MachoDrone/nosana-gridlens/internal/execx"
)

type DependencyStatus struct {
	Command     string `json:"command"`
	Found       bool   `json:"found"`
	Path        string `json:"path,omitempty"`
	Required    bool   `json:"required"`
	PackageHint string `json:"packageHint,omitempty"`
}

type PackageManager struct {
	Name                  string   `json:"name"`
	Path                  string   `json:"path,omitempty"`
	InstallCommand        []string `json:"installCommand,omitempty"`
	UpdateMetadataCommand []string `json:"updateMetadataCommand,omitempty"`
	CanCheckUpdates       bool     `json:"canCheckUpdates"`
}

type OSInfo struct {
	ID         string `json:"id,omitempty"`
	Name       string `json:"name,omitempty"`
	VersionID  string `json:"versionId,omitempty"`
	PrettyName string `json:"prettyName,omitempty"`
	Family     string `json:"family,omitempty"`
	Supported  bool   `json:"supported"`
}

type commandRequirement struct {
	command     string
	required    bool
	packageHint string
}

var requiredCommands = []commandRequirement{
	{command: "wg", required: true, packageHint: "wireguard-tools"},
	{command: "wg-quick", required: true, packageHint: "wireguard-tools"},
	{command: "ip", required: true, packageHint: "iproute2"},
	{command: "ss", required: true, packageHint: "iproute2"},
	{command: "systemctl", required: true, packageHint: "systemd"},
}

func DetectDependencies(runner execx.Runner) []DependencyStatus {
	statuses := make([]DependencyStatus, 0, len(requiredCommands))
	for _, requirement := range requiredCommands {
		path, err := runner.LookPath(requirement.command)
		statuses = append(statuses, DependencyStatus{
			Command:     requirement.command,
			Found:       err == nil,
			Path:        path,
			Required:    requirement.required,
			PackageHint: requirement.packageHint,
		})
	}
	return statuses
}

func MissingRequired(statuses []DependencyStatus) []DependencyStatus {
	var missing []DependencyStatus
	for _, status := range statuses {
		if status.Required && !status.Found {
			missing = append(missing, status)
		}
	}
	return missing
}

func DetectPackageManager(runner execx.Runner) *PackageManager {
	candidates := []struct {
		name     string
		packages []string
		update   []string
		check    bool
	}{
		{name: "apt-get", packages: []string{"wireguard-tools", "iproute2"}, update: []string{"sudo", "apt-get", "update"}, check: true},
		{name: "dnf", packages: []string{"wireguard-tools", "iproute"}, check: true},
		{name: "yum", packages: []string{"wireguard-tools", "iproute"}, check: true},
		{name: "pacman", packages: []string{"wireguard-tools", "iproute2"}, check: true},
		{name: "zypper", packages: []string{"wireguard-tools", "iproute2"}, check: true},
		{name: "apk", packages: []string{"wireguard-tools", "iproute2"}, check: true},
	}

	for _, candidate := range candidates {
		path, err := runner.LookPath(candidate.name)
		if err != nil {
			continue
		}

		var install []string
		if candidate.name == "apt-get" {
			install = []string{"sudo", "apt-get", "install", "-y"}
			install = append(install, candidate.packages...)
		}

		return &PackageManager{
			Name:                  candidate.name,
			Path:                  path,
			InstallCommand:        install,
			UpdateMetadataCommand: candidate.update,
			CanCheckUpdates:       candidate.check,
		}
	}

	return nil
}

func DetectOS() OSInfo {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return OSInfo{Supported: false}
	}
	return ParseOSRelease(string(data))
}

func ParseOSRelease(data string) OSInfo {
	values := map[string]string{}
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[key] = strings.Trim(strings.TrimSpace(value), `"`)
	}

	info := OSInfo{
		ID:         values["ID"],
		Name:       values["NAME"],
		VersionID:  values["VERSION_ID"],
		PrettyName: values["PRETTY_NAME"],
	}
	info.Family, info.Supported = supportedFamily(info.ID, values["ID_LIKE"])
	return info
}

func supportedFamily(id string, idLike string) (string, bool) {
	fields := append([]string{id}, strings.Fields(idLike)...)
	for _, field := range fields {
		switch strings.ToLower(field) {
		case "debian", "ubuntu":
			return "debian", true
		}
	}
	return "", false
}
