package hub

import (
	"sync"
	"time"

	"github.com/MachoDrone/nosana-gridlens/internal/network"
)

type trafficEvent struct {
	at    time.Time
	usage network.TrafficUsage
}

type trafficMeter struct {
	mu     sync.Mutex
	window time.Duration
	events []trafficEvent
}

func newTrafficMeter(window time.Duration) *trafficMeter {
	if window <= 0 {
		window = time.Minute
	}
	return &trafficMeter{window: window}
}

func (m *trafficMeter) Add(at time.Time, usage network.TrafficUsage) {
	if usage.TotalBytes == 0 && usage.BytesIn+usage.BytesOut > 0 {
		usage.TotalBytes = usage.BytesIn + usage.BytesOut
	}
	if usage.TotalBytes == 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, trafficEvent{at: at, usage: usage})
	m.trimLocked(at)
}

func (m *trafficMeter) Snapshot(now time.Time) network.TrafficUsage {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.trimLocked(now)

	var total network.TrafficUsage
	for _, event := range m.events {
		total = total.Add(event.usage)
	}
	total.WindowSeconds = int(m.window.Seconds())
	if total.WindowSeconds > 0 {
		total.BytesPerSecond = float64(total.TotalBytes) / m.window.Seconds()
	}
	return total
}

func (m *trafficMeter) trimLocked(now time.Time) {
	cutoff := now.Add(-m.window)
	keep := 0
	for _, event := range m.events {
		if event.at.Before(cutoff) {
			continue
		}
		m.events[keep] = event
		keep++
	}
	m.events = m.events[:keep]
}
