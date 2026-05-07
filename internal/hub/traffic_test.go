package hub

import (
	"testing"
	"time"

	"github.com/MachoDrone/nosana-gridlens/internal/network"
)

func TestTrafficMeterSnapshotsRollingWindow(t *testing.T) {
	meter := newTrafficMeter(10 * time.Second)
	now := time.Unix(100, 0)

	meter.Add(now.Add(-11*time.Second), network.TrafficUsage{BytesIn: 100, TotalBytes: 100, Estimated: true})
	meter.Add(now.Add(-5*time.Second), network.TrafficUsage{BytesOut: 200, TotalBytes: 200, SSHSessions: 1, Estimated: true})
	meter.Add(now, network.TrafficUsage{BytesIn: 300, TotalBytes: 300, PortChecks: 2, Estimated: true})

	snapshot := meter.Snapshot(now)
	if snapshot.TotalBytes != 500 {
		t.Fatalf("expected only recent traffic in snapshot, got %+v", snapshot)
	}
	if snapshot.BytesPerSecond != 50 {
		t.Fatalf("expected 50 B/s, got %+v", snapshot)
	}
	if snapshot.SSHSessions != 1 || snapshot.PortChecks != 2 {
		t.Fatalf("expected aggregated event counts, got %+v", snapshot)
	}
}
