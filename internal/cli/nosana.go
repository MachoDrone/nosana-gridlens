package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/MachoDrone/nosana-gridlens/internal/nosana"
)

func (a *App) runNosana(ctx context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(a.err, "usage: gridlens nosana detect [--json]")
		return 2
	}

	switch args[0] {
	case "detect":
		return a.runNosanaDetect(ctx, args[1:])
	default:
		fmt.Fprintf(a.err, "unknown nosana command: %s\n", args[0])
		return 2
	}
}

func (a *App) runNosanaDetect(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("nosana detect", flag.ContinueOnError)
	fs.SetOutput(a.err)
	jsonOutput := fs.Bool("json", false, "write JSON output")
	includeNested := fs.Bool("nested", true, "inspect Podman inside Docker containers with read-only podman ps")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	path, cfg, ok := a.loadConfig()
	if !ok {
		return 1
	}

	report := nosana.Detect(ctx, a.runner, cfg, nosana.Options{
		ConfigPath:           path,
		IncludeNested:        *includeNested,
		Now:                  a.now(),
		MaxConcurrentTargets: 32,
		MaxConcurrentNested:  8,
	})
	if *jsonOutput {
		return writeJSON(a.out, report)
	}

	fmt.Fprintln(a.out, "GridLens Nosana discovery")
	fmt.Fprintln(a.out)
	fmt.Fprintf(a.out, "Config: %s\n", report.ConfigPath)
	fmt.Fprintf(a.out, "Targets scanned: %d\n", report.Summary.TargetsScanned)
	fmt.Fprintf(a.out, "Runtimes available: %d\n", report.Summary.RuntimesAvailable)
	fmt.Fprintf(a.out, "Containers seen: %d\n", report.Summary.ContainersSeen)
	fmt.Fprintf(a.out, "Nosana hosts: %d\n", report.Summary.NosanaHosts)
	fmt.Fprintln(a.out)

	for _, target := range report.Targets {
		fmt.Fprintf(a.out, "%s (%s)\n", target.Name, target.Scope)
		if target.Skipped {
			fmt.Fprintf(a.out, "  skipped: %s\n", target.SkipReason)
			continue
		}
		for _, runtimeReport := range target.Runtimes {
			if !runtimeReport.Available {
				fmt.Fprintf(a.out, "  %s: unavailable (%s)\n", runtimeReport.Type, runtimeReport.Error)
				continue
			}
			fmt.Fprintf(a.out, "  %s: %d containers\n", runtimeReport.Type, len(runtimeReport.Containers))
			for _, container := range runtimeReport.Containers {
				marker := ""
				if container.Matched {
					marker = " [nosana]"
				}
				fmt.Fprintf(a.out, "    - %s%s image=%s status=%s\n", container.Name, marker, container.Image, container.Status)
				for _, nested := range container.Nested {
					nestedMarker := ""
					if nested.Matched {
						nestedMarker = " [nosana]"
					}
					fmt.Fprintf(a.out, "      nested podman: %s%s image=%s status=%s\n", nested.Name, nestedMarker, nested.Image, nested.Status)
				}
			}
		}
	}
	return 0
}
