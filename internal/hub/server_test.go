package hub

import (
	"bytes"
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

func TestBulkPCsStoresIPsAndDoesNotStorePassword(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	server := NewServer(execx.NewFakeRunner(), func() time.Time { return time.Unix(0, 0) }, path)

	body := bytes.NewBufferString(`{
		"addresses": "192.168.0.101, 192.168.0.110-111",
		"username": "grid",
		"password": "secret",
		"containerNames": ["custom-a"],
		"containerPatterns": ["nosana-*"],
		"runtimes": ["docker", "podman"]
	}`)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/config/pcs/bulk", body)
	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(loaded.PCs) != 3 {
		t.Fatalf("expected three PCs, got %+v", loaded.PCs)
	}
	if loaded.PCs[0].SSHTarget != "grid@192.168.0.101" {
		t.Fatalf("unexpected SSH target: %+v", loaded.PCs[0])
	}

	data := response.Body.String()
	if bytes.Contains([]byte(data), []byte("secret")) {
		t.Fatalf("password leaked in response: %s", data)
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
