package hub

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/MachoDrone/nosana-gridlens/internal/app"
	"github.com/MachoDrone/nosana-gridlens/internal/config"
	"github.com/MachoDrone/nosana-gridlens/internal/execx"
	"github.com/MachoDrone/nosana-gridlens/internal/network"
	"github.com/MachoDrone/nosana-gridlens/internal/nosana"
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	runner     execx.Runner
	now        func() time.Time
	configPath string
}

func NewServer(runner execx.Runner, now func() time.Time, configPath string) *Server {
	return &Server{
		runner:     runner,
		now:        now,
		configPath: configPath,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/healthz", s.handleHealthz)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/nosana", s.handleNosana)
	mux.HandleFunc("/api/pc/scan", s.handlePCScan)

	staticRoot, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticRoot)))
	return mux
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"app":       app.Name,
		"version":   app.Version,
		"timestamp": s.now().UTC(),
	})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	path := s.configPath
	if path == "" {
		var err error
		path, err = config.Path()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}
	cfg, err := config.Load(path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"configPath": path,
		"config":     cfg,
	})
}

func (s *Server) handleNosana(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	path := s.configPath
	if path == "" {
		var err error
		path, err = config.Path()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}
	cfg, err := config.Load(path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	report := nosana.Detect(ctx, s.runner, cfg, nosana.Options{
		ConfigPath:    path,
		IncludeNested: true,
		Now:           s.now(),
	})
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handlePCScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	ports, err := network.ParsePorts(r.URL.Query().Get("ports"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	cidrs := []string{}
	if cidr := strings.TrimSpace(r.URL.Query().Get("cidr")); cidr != "" {
		cidrs = append(cidrs, cidr)
	} else {
		cidrs = network.LocalIPv4CIDRs()
	}
	if len(cidrs) == 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("no local IPv4 CIDR detected"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	var results []network.HostScan
	for _, cidr := range cidrs {
		scanResults, err := network.ScanCIDR(ctx, network.ScanOptions{
			CIDR:        cidr,
			Ports:       ports,
			Timeout:     300 * time.Millisecond,
			Concurrency: 64,
			MaxHosts:    1024,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		results = append(results, scanResults...)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"generatedAt": s.now().UTC(),
		"cidrs":       cidrs,
		"ports":       ports,
		"results":     results,
	})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(value)
}
