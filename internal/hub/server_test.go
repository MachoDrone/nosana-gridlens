package hub

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/MachoDrone/nosana-gridlens/internal/config"
	"github.com/MachoDrone/nosana-gridlens/internal/execx"
)

func TestHealthz(t *testing.T) {
	server := NewServer(execx.NewFakeRunner(), func() time.Time { return time.Unix(0, 0) }, "")

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["ok"] != true {
		t.Fatalf("unexpected health body: %+v", body)
	}
}

func TestNosanaAPIUsesRealDiscoveryResultShape(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := config.Default()
	if err := cfg.AddOrUpdatePC(config.PC{Name: "nodebox", Address: "192.168.0.10"}); err != nil {
		t.Fatalf("AddOrUpdatePC returned error: %v", err)
	}
	if err := config.Save(path, cfg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	server := NewServer(execx.NewFakeRunner(), func() time.Time { return time.Unix(0, 0) }, path)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/nosana", nil)
	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}

	var body struct {
		Summary struct {
			TargetsScanned int `json:"targetsScanned"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body.Summary.TargetsScanned != 2 {
		t.Fatalf("expected local plus configured target, got %+v", body.Summary)
	}
}
