package nosana

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/MachoDrone/nosana-gridlens/internal/config"
	"github.com/MachoDrone/nosana-gridlens/internal/execx"
)

type Report struct {
	GeneratedAt time.Time      `json:"generatedAt"`
	ConfigPath  string         `json:"configPath,omitempty"`
	Targets     []TargetReport `json:"targets"`
	Summary     Summary        `json:"summary"`
}

type Summary struct {
	TargetsScanned    int `json:"targetsScanned"`
	RuntimesAvailable int `json:"runtimesAvailable"`
	ContainersSeen    int `json:"containersSeen"`
	NosanaMatches     int `json:"nosanaMatches"`
}

type TargetReport struct {
	Name       string          `json:"name"`
	Scope      string          `json:"scope"`
	Address    string          `json:"address,omitempty"`
	SSHTarget  string          `json:"sshTarget,omitempty"`
	Runtimes   []RuntimeReport `json:"runtimes"`
	Skipped    bool            `json:"skipped"`
	SkipReason string          `json:"skipReason,omitempty"`
}

type RuntimeReport struct {
	Type       string      `json:"type"`
	Available  bool        `json:"available"`
	Command    []string    `json:"command,omitempty"`
	Error      string      `json:"error,omitempty"`
	Containers []Container `json:"containers"`
}

type Options struct {
	ConfigPath    string
	IncludeNested bool
	Now           time.Time
}

func Detect(ctx context.Context, runner execx.Runner, cfg config.Config, opts Options) Report {
	cfg.ApplyDefaults()
	report := Report{
		GeneratedAt: opts.Now.UTC(),
		ConfigPath:  opts.ConfigPath,
	}

	localMatcher := Matcher{Patterns: cfg.DefaultContainerPatterns}
	local := detectLocal(ctx, runner, localMatcher, opts.IncludeNested)
	report.Targets = append(report.Targets, local)

	for _, pc := range cfg.PCs {
		report.Targets = append(report.Targets, detectPC(ctx, runner, pc, cfg.DefaultContainerPatterns, opts.IncludeNested))
	}

	report.Summary = summarize(report.Targets)
	return report
}

func detectLocal(ctx context.Context, runner execx.Runner, matcher Matcher, includeNested bool) TargetReport {
	target := TargetReport{Name: "local", Scope: "local"}
	target.Runtimes = append(target.Runtimes, detectLocalRuntime(ctx, runner, "docker", matcher, includeNested))
	target.Runtimes = append(target.Runtimes, detectLocalRuntime(ctx, runner, "podman", matcher, false))
	return target
}

func detectPC(ctx context.Context, runner execx.Runner, pc config.PC, defaults []string, includeNested bool) TargetReport {
	target := TargetReport{
		Name:      pc.Name,
		Scope:     "configured",
		Address:   pc.Address,
		SSHTarget: pc.SSHTarget,
	}
	if pc.SSHTarget == "" {
		target.Skipped = true
		target.SkipReason = "no ssh target configured; add --ssh user@host for remote read-only container discovery"
		return target
	}

	if _, err := runner.LookPath("ssh"); err != nil {
		target.Skipped = true
		target.SkipReason = "ssh command not found locally"
		return target
	}

	matcher := Matcher{
		ExactNames: pc.ContainerNames,
		Patterns:   append(append([]string{}, defaults...), pc.ContainerPatterns...),
	}
	for _, runtimeName := range runtimesForPC(pc) {
		target.Runtimes = append(target.Runtimes, detectRemoteRuntime(ctx, runner, pc.SSHTarget, runtimeName, matcher, includeNested))
	}
	return target
}

func runtimesForPC(pc config.PC) []string {
	if len(pc.Runtimes) == 0 {
		return []string{"docker", "podman"}
	}
	var runtimes []string
	for _, runtimeName := range pc.Runtimes {
		runtimeName = strings.ToLower(strings.TrimSpace(runtimeName))
		switch runtimeName {
		case "", "auto":
			runtimes = append(runtimes, "docker", "podman")
		case "docker", "podman":
			runtimes = append(runtimes, runtimeName)
		}
	}
	if len(runtimes) == 0 {
		return []string{"docker", "podman"}
	}
	return runtimes
}

func detectLocalRuntime(ctx context.Context, runner execx.Runner, runtimeName string, matcher Matcher, includeNested bool) RuntimeReport {
	if _, err := runner.LookPath(runtimeName); err != nil {
		return RuntimeReport{Type: runtimeName, Available: false, Error: runtimeName + " command not found"}
	}

	var result execx.Result
	var command []string
	switch runtimeName {
	case "docker":
		command = []string{"docker", "ps", "--format", "{{json .}}"}
		result = runner.Run(ctx, "docker", "ps", "--format", "{{json .}}")
	case "podman":
		command = []string{"podman", "ps", "--format", "json"}
		result = runner.Run(ctx, "podman", "ps", "--format", "json")
	default:
		return RuntimeReport{Type: runtimeName, Available: false, Error: "unsupported runtime"}
	}

	report := RuntimeReport{Type: runtimeName, Available: result.OK(), Command: command}
	if !result.OK() {
		report.Error = resultError(result)
		return report
	}

	containers, err := parseRuntimeOutput(runtimeName, "local", result.Stdout, matcher)
	if err != nil {
		report.Available = false
		report.Error = err.Error()
		return report
	}

	if runtimeName == "docker" && includeNested {
		addNestedPodman(ctx, runner, "local", containers, matcher)
		demoteRuntimeWrappers(containers)
	}
	report.Containers = containers
	return report
}

func detectRemoteRuntime(ctx context.Context, runner execx.Runner, sshTarget string, runtimeName string, matcher Matcher, includeNested bool) RuntimeReport {
	remoteCommand := remoteRuntimeCommand(runtimeName)
	if remoteCommand == "" {
		return RuntimeReport{Type: runtimeName, Available: false, Error: "unsupported runtime"}
	}

	command := []string{"ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=3", sshTarget, remoteCommand}
	result := runner.Run(ctx, command[0], command[1:]...)
	report := RuntimeReport{Type: runtimeName, Available: result.OK(), Command: command}
	if !result.OK() {
		report.Error = resultError(result)
		return report
	}

	containers, err := parseRuntimeOutput(runtimeName, sshTarget, result.Stdout, matcher)
	if err != nil {
		report.Available = false
		report.Error = err.Error()
		return report
	}

	if runtimeName == "docker" && includeNested {
		addRemoteNestedPodman(ctx, runner, sshTarget, containers, matcher)
		demoteRuntimeWrappers(containers)
	}
	report.Containers = containers
	return report
}

func remoteRuntimeCommand(runtimeName string) string {
	switch runtimeName {
	case "docker":
		return "docker ps --format '{{json .}}'"
	case "podman":
		return "podman ps --format json"
	default:
		return ""
	}
}

func parseRuntimeOutput(runtimeName string, source string, output string, matcher Matcher) ([]Container, error) {
	switch runtimeName {
	case "docker":
		return ParseDockerPSJSONLines(source, output, matcher)
	case "podman":
		return ParsePodmanPSJSON(source, output, matcher)
	default:
		return nil, fmt.Errorf("unsupported runtime: %s", runtimeName)
	}
}

func addNestedPodman(ctx context.Context, runner execx.Runner, source string, containers []Container, matcher Matcher) {
	for i := range containers {
		if containers[i].ID == "" {
			continue
		}
		result := runner.Run(ctx, "docker", "exec", containers[i].ID, "sh", "-lc", "command -v podman >/dev/null 2>&1 && podman ps --format json || true")
		if !result.OK() || strings.TrimSpace(result.Stdout) == "" {
			continue
		}
		nested, err := ParsePodmanPSJSON(source+" nested in "+containers[i].Name, result.Stdout, matcher)
		if err == nil {
			containers[i].Nested = nested
		}
	}
}

func addRemoteNestedPodman(ctx context.Context, runner execx.Runner, sshTarget string, containers []Container, matcher Matcher) {
	for i := range containers {
		if containers[i].ID == "" {
			continue
		}
		remoteCommand := fmt.Sprintf("docker exec %s sh -lc 'command -v podman >/dev/null 2>&1 && podman ps --format json || true'", containers[i].ID)
		args := []string{"-o", "BatchMode=yes", "-o", "ConnectTimeout=3", sshTarget, remoteCommand}
		result := runner.Run(ctx, "ssh", args...)
		if !result.OK() || strings.TrimSpace(result.Stdout) == "" {
			continue
		}
		nested, err := ParsePodmanPSJSON(sshTarget+" nested in "+containers[i].Name, result.Stdout, matcher)
		if err == nil {
			containers[i].Nested = nested
		}
	}
}

func demoteRuntimeWrappers(containers []Container) {
	for i := range containers {
		if !containers[i].Matched || !looksLikeRuntimeWrapper(containers[i]) || !hasNestedMatch(containers[i]) {
			continue
		}
		containers[i].Matched = false
		containers[i].MatchReason = "runtime wrapper; nested Nosana containers counted instead"
	}
}

func looksLikeRuntimeWrapper(container Container) bool {
	name := strings.ToLower(container.Name)
	image := strings.ToLower(container.Image)
	return strings.HasPrefix(name, "podman") || strings.Contains(image, "nosana/podman")
}

func hasNestedMatch(container Container) bool {
	for _, nested := range container.Nested {
		if nested.Matched || hasNestedMatch(nested) {
			return true
		}
	}
	return false
}

func summarize(targets []TargetReport) Summary {
	var summary Summary
	summary.TargetsScanned = len(targets)
	for _, target := range targets {
		for _, runtimeReport := range target.Runtimes {
			if runtimeReport.Available {
				summary.RuntimesAvailable++
			}
			for _, container := range runtimeReport.Containers {
				addContainerSummary(&summary, container)
			}
		}
	}
	return summary
}

func addContainerSummary(summary *Summary, container Container) {
	summary.ContainersSeen++
	if container.Matched {
		summary.NosanaMatches++
	}
	for _, nested := range container.Nested {
		addContainerSummary(summary, nested)
	}
}

func resultError(result execx.Result) string {
	if result.Stderr != "" {
		return result.Stderr
	}
	if result.Err != nil {
		return result.Err.Error()
	}
	return fmt.Sprintf("exit code %d", result.ExitCode)
}
