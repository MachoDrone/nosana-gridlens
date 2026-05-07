package cli

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/MachoDrone/nosana-gridlens/internal/config"
	"github.com/MachoDrone/nosana-gridlens/internal/network"
)

type PCScanReport struct {
	GeneratedAt time.Time          `json:"generatedAt"`
	CIDRs       []string           `json:"cidrs"`
	Ports       []int              `json:"ports"`
	Results     []network.HostScan `json:"results"`
}

func (a *App) runPC(ctx context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(a.err, "usage: gridlens pc <scan|add|list|remove>")
		return 2
	}

	switch args[0] {
	case "scan":
		return a.runPCScan(ctx, args[1:])
	case "add":
		return a.runPCAdd(args[1:])
	case "list":
		return a.runPCList(args[1:])
	case "remove":
		return a.runPCRemove(args[1:])
	default:
		fmt.Fprintf(a.err, "unknown pc command: %s\n", args[0])
		return 2
	}
}

func (a *App) runPCScan(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("pc scan", flag.ContinueOnError)
	fs.SetOutput(a.err)
	cidr := fs.String("cidr", "", "CIDR to scan; defaults to local non-loopback IPv4 CIDRs")
	portsValue := fs.String("ports", "22,2375,2376", "comma-separated TCP ports to probe")
	timeout := fs.Duration("timeout", 300*time.Millisecond, "TCP connect timeout")
	concurrency := fs.Int("concurrency", 64, "parallel host probes")
	maxHosts := fs.Int("max-hosts", 1024, "maximum hosts to scan")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	ports, err := network.ParsePorts(*portsValue)
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 2
	}

	cidrs := []string{}
	if strings.TrimSpace(*cidr) != "" {
		cidrs = append(cidrs, strings.TrimSpace(*cidr))
	} else {
		cidrs = network.LocalIPv4CIDRs()
	}
	if len(cidrs) == 0 {
		fmt.Fprintln(a.err, "no local IPv4 CIDR detected; pass --cidr")
		return 2
	}

	report := PCScanReport{
		GeneratedAt: a.now().UTC(),
		CIDRs:       cidrs,
		Ports:       ports,
	}
	for _, scanCIDR := range cidrs {
		results, err := network.ScanCIDR(ctx, network.ScanOptions{
			CIDR:        scanCIDR,
			Ports:       ports,
			Timeout:     *timeout,
			Concurrency: *concurrency,
			MaxHosts:    *maxHosts,
		})
		if err != nil {
			fmt.Fprintf(a.err, "scan %s: %v\n", scanCIDR, err)
			return 1
		}
		report.Results = append(report.Results, results...)
	}

	if *jsonOutput {
		return writeJSON(a.out, report)
	}

	fmt.Fprintf(a.out, "GridLens PC scan\n\n")
	fmt.Fprintf(a.out, "CIDRs: %s\n", strings.Join(report.CIDRs, ", "))
	fmt.Fprintf(a.out, "Ports: %v\n", report.Ports)
	if len(report.Results) == 0 {
		fmt.Fprintln(a.out, "No hosts with selected open ports found.")
		return 0
	}
	fmt.Fprintln(a.out, "Candidates:")
	for _, result := range report.Results {
		fmt.Fprintf(a.out, "  %s open ports: %v\n", result.IP, result.OpenPorts)
	}
	return 0
}

func (a *App) runPCAdd(args []string) int {
	name := ""
	parseArgs := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		name = args[0]
		parseArgs = args[1:]
	}

	fs := flag.NewFlagSet("pc add", flag.ContinueOnError)
	fs.SetOutput(a.err)
	address := fs.String("address", "", "PC address")
	sshTarget := fs.String("ssh", "", "SSH target such as user@192.168.0.10")
	var runtimes repeatedStringFlag
	var containers repeatedStringFlag
	var patterns repeatedStringFlag
	fs.Var(&runtimes, "runtime", "runtime to inspect; repeat for docker and podman")
	fs.Var(&containers, "container", "exact container name to treat as a Nosana host; repeatable")
	fs.Var(&patterns, "pattern", "glob container-name pattern to treat as a Nosana host; repeatable")
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	if name == "" && fs.NArg() == 1 {
		name = fs.Arg(0)
	}
	if name == "" || fs.NArg() > 1 {
		fmt.Fprintln(a.err, "usage: gridlens pc add NAME --address IP [--ssh user@host] [--container NAME] [--pattern GLOB]")
		return 2
	}

	path, cfg, ok := a.loadConfig()
	if !ok {
		return 1
	}
	pc := config.PC{
		Name:              name,
		Address:           *address,
		SSHTarget:         *sshTarget,
		Runtimes:          runtimes,
		ContainerNames:    containers,
		ContainerPatterns: patterns,
	}
	if err := cfg.AddOrUpdatePC(pc); err != nil {
		fmt.Fprintln(a.err, err)
		return 2
	}
	if err := config.Save(path, cfg); err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}

	fmt.Fprintf(a.out, "Saved PC %q to %s\n", pc.Name, path)
	return 0
}

func (a *App) runPCList(args []string) int {
	jsonOutput := hasFlag(args, "--json")
	path, cfg, ok := a.loadConfig()
	if !ok {
		return 1
	}
	if jsonOutput {
		return writeJSON(a.out, struct {
			ConfigPath string        `json:"configPath"`
			Config     config.Config `json:"config"`
		}{ConfigPath: path, Config: cfg})
	}

	fmt.Fprintf(a.out, "Config: %s\n", path)
	if len(cfg.PCs) == 0 {
		fmt.Fprintln(a.out, "No PCs configured.")
		return 0
	}
	for _, pc := range cfg.PCs {
		fmt.Fprintf(a.out, "- %s", pc.Name)
		if pc.Address != "" {
			fmt.Fprintf(a.out, " address=%s", pc.Address)
		}
		if pc.SSHTarget != "" {
			fmt.Fprintf(a.out, " ssh=%s", pc.SSHTarget)
		}
		if len(pc.ContainerNames) > 0 {
			fmt.Fprintf(a.out, " containers=%s", strings.Join(pc.ContainerNames, ","))
		}
		if len(pc.ContainerPatterns) > 0 {
			fmt.Fprintf(a.out, " patterns=%s", strings.Join(pc.ContainerPatterns, ","))
		}
		fmt.Fprintln(a.out)
	}
	return 0
}

func (a *App) runPCRemove(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(a.err, "usage: gridlens pc remove NAME")
		return 2
	}
	path, cfg, ok := a.loadConfig()
	if !ok {
		return 1
	}
	if !cfg.RemovePC(args[0]) {
		fmt.Fprintf(a.err, "PC %q is not configured\n", args[0])
		return 1
	}
	if err := config.Save(path, cfg); err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	fmt.Fprintf(a.out, "Removed PC %q from %s\n", args[0], path)
	return 0
}

func (a *App) loadConfig() (string, config.Config, bool) {
	path, err := config.Path()
	if err != nil {
		fmt.Fprintln(a.err, err)
		return "", config.Config{}, false
	}
	cfg, err := config.Load(path)
	if err != nil {
		fmt.Fprintln(a.err, err)
		return "", config.Config{}, false
	}
	return path, cfg, true
}
