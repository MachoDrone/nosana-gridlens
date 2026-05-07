package nosana

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Container struct {
	Runtime     string      `json:"runtime"`
	Source      string      `json:"source"`
	ID          string      `json:"id,omitempty"`
	Name        string      `json:"name"`
	Image       string      `json:"image,omitempty"`
	Status      string      `json:"status,omitempty"`
	Ports       string      `json:"ports,omitempty"`
	Matched     bool        `json:"matched"`
	MatchReason string      `json:"matchReason,omitempty"`
	Nested      []Container `json:"nested,omitempty"`
}

func ParseDockerPSJSONLines(source string, output string, matcher Matcher) ([]Container, error) {
	var containers []Container
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var raw map[string]any
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			return nil, fmt.Errorf("parse docker ps line: %w", err)
		}

		container := Container{
			Runtime: "docker",
			Source:  source,
			ID:      stringField(raw, "ID", "Id"),
			Name:    stringField(raw, "Names", "Name"),
			Image:   stringField(raw, "Image"),
			Status:  stringField(raw, "Status", "State"),
			Ports:   stringField(raw, "Ports"),
		}
		container.Matched, container.MatchReason = matcher.Match(container.Name)
		containers = append(containers, container)
	}
	return containers, nil
}

func ParsePodmanPSJSON(source string, output string, matcher Matcher) ([]Container, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return []Container{}, nil
	}

	var rawContainers []map[string]any
	if err := json.Unmarshal([]byte(output), &rawContainers); err != nil {
		return nil, fmt.Errorf("parse podman ps json: %w", err)
	}

	containers := make([]Container, 0, len(rawContainers))
	for _, raw := range rawContainers {
		name := firstStringListValue(raw, "Names", "names", "NamesHistory")
		if name == "" {
			name = stringField(raw, "Name", "name")
		}

		container := Container{
			Runtime: "podman",
			Source:  source,
			ID:      stringField(raw, "Id", "ID", "id"),
			Name:    name,
			Image:   stringField(raw, "Image", "ImageName", "image", "imageName"),
			Status:  stringField(raw, "Status", "status", "State", "state"),
			Ports:   portsField(raw),
		}
		container.Matched, container.MatchReason = matcher.Match(container.Name)
		containers = append(containers, container)
	}
	return containers, nil
}

func stringField(raw map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			return typed
		case fmt.Stringer:
			return typed.String()
		default:
			return fmt.Sprint(typed)
		}
	}
	return ""
}

func firstStringListValue(raw map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case []any:
			if len(typed) == 0 {
				continue
			}
			return fmt.Sprint(typed[0])
		case []string:
			if len(typed) == 0 {
				continue
			}
			return typed[0]
		case string:
			return typed
		}
	}
	return ""
}

func portsField(raw map[string]any) string {
	if value := stringField(raw, "Ports", "ports"); value != "" {
		return value
	}
	if value, ok := raw["Ports"]; ok {
		return fmt.Sprint(value)
	}
	if value, ok := raw["ports"]; ok {
		return fmt.Sprint(value)
	}
	return ""
}
