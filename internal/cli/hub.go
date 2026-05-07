package cli

import (
	"context"
	"flag"
	"fmt"
	"net/http"

	"github.com/MachoDrone/nosana-gridlens/internal/config"
	"github.com/MachoDrone/nosana-gridlens/internal/hub"
)

func (a *App) runHub(ctx context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(a.err, "usage: gridlens hub start [--addr 127.0.0.1:8787]")
		return 2
	}

	switch args[0] {
	case "start":
		return a.runHubStart(ctx, args[1:])
	default:
		fmt.Fprintf(a.err, "unknown hub command: %s\n", args[0])
		return 2
	}
}

func (a *App) runHubStart(_ context.Context, args []string) int {
	fs := flag.NewFlagSet("hub start", flag.ContinueOnError)
	fs.SetOutput(a.err)
	addr := fs.String("addr", "127.0.0.1:8787", "HTTP listen address")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	path, err := configPath()
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}

	server := hub.NewServer(a.runner, a.now, path)
	fmt.Fprintf(a.out, "GridLens Hub listening on http://%s\n", *addr)
	fmt.Fprintln(a.out, "Press Ctrl+C to stop.")
	if err := http.ListenAndServe(*addr, server.Handler()); err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	return 0
}

func configPath() (string, error) {
	return config.Path()
}
