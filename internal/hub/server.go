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
	runner       execx.Runner
	now          func() time.Time
	configPath   string
	trafficMeter *trafficMeter
}

func NewServer(runner execx.Runner, now func() time.Time, configPath string) *Server {
	return &Server{
		runner:       runner,
		now:          now,
		configPath:   configPath,
		trafficMeter: newTrafficMeter(time.Minute),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/healthz", s.handleHealthz)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/config/pcs/bulk", s.handleBulkPCs)
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

type bulkPCRequest struct {
	Addresses         string   `json:"addresses"`
	Username          string   `json:"username"`
	Password          string   `json:"password"`
	Runtimes          []string `json:"runtimes"`
	ContainerNames    []string `json:"containerNames"`
	ContainerPatterns []string `json:"containerPatterns"`
	MaxHosts          int      `json:"maxHosts"`
}

func (s *Server) handleBulkPCs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	var request bulkPCRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	addresses, err := network.ParseAddressSpecs(request.Addresses, request.MaxHosts)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(addresses) == 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("enter at least one IPv4 address, range, or CIDR"))
		return
	}

	path := s.configPath
	if path == "" {
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

	existing := map[string]string{}
	for _, pc := range cfg.PCs {
		if pc.Address != "" {
			existing[pc.Address] = pc.Name
		}
	}

	added := 0
	updated := 0
	username := strings.TrimSpace(request.Username)
	for _, address := range addresses {
		name := "pc-" + strings.ReplaceAll(address, ".", "-")
		if existingName := existing[address]; existingName != "" {
			name = existingName
		}
		pc := config.PC{
			Name:              name,
			Address:           address,
			Runtimes:          request.Runtimes,
			ContainerNames:    request.ContainerNames,
			ContainerPatterns: request.ContainerPatterns,
		}
		if username != "" {
			pc.SSHTarget = username + "@" + address
		}
		if existing[address] != "" {
			updated++
		} else {
			added++
		}
		if err := cfg.AddOrUpdatePC(pc); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}

	if err := config.Save(path, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	warnings := []string{}
	if strings.TrimSpace(request.Password) != "" {
		warnings = append(warnings, "password was accepted for this request but was not stored; use SSH keys now or future GridLens agent enrollment for persistent collection")
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"configPath": path,
		"added":      added,
		"updated":    updated,
		"warnings":   warnings,
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

	startedAt := s.now().UTC()
	report := nosana.Detect(ctx, s.runner, cfg, nosana.Options{
		ConfigPath:           path,
		IncludeNested:        true,
		Now:                  startedAt,
		MaxConcurrentTargets: 32,
		MaxConcurrentNested:  8,
	})
	finishedAt := s.now().UTC()
	s.trafficMeter.AddSample(startedAt, finishedAt, report.CollectionTraffic)
	report.Summary.GridLensTraffic = s.trafficMeter.Snapshot(finishedAt)
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

	startedAt := s.now().UTC()
	var results []network.HostScan
	var traffic network.TrafficUsage
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
		scanTraffic, err := network.EstimateScanTraffic(cidr, ports, 1024, scanResults)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		traffic = traffic.Add(scanTraffic)
		results = append(results, scanResults...)
	}
	now := s.now().UTC()
	s.trafficMeter.AddSample(startedAt, now, traffic)

	writeJSON(w, http.StatusOK, map[string]any{
		"generatedAt": now,
		"cidrs":       cidrs,
		"ports":       ports,
		"results":     results,
		"traffic":     traffic,
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
