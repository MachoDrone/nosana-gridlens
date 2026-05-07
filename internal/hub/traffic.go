package hub

import (
	"sync"
	"time"

	"github.com/MachoDrone/nosana-gridlens/internal/network"
)

type trafficEvent struct {
	start time.Time
	end   time.Time
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
	m.AddSample(at, at.Add(time.Second), usage)
}

func (m *trafficMeter) AddSample(start time.Time, end time.Time, usage network.TrafficUsage) {
	if usage.TotalBytes == 0 && usage.BytesIn+usage.BytesOut > 0 {
		usage.TotalBytes = usage.BytesIn + usage.BytesOut
	}
	if usage.TotalBytes == 0 {
		return
	}
	if !end.After(start) {
		end = start.Add(time.Second)
	}
	duration := end.Sub(start).Seconds()
	if duration <= 0 {
		duration = 1
	}
	if usage.PeakBytesPerSecond == 0 {
		usage.PeakBytesPerSecond = float64(usage.TotalBytes) / duration
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, trafficEvent{start: start, end: end, usage: usage})
	m.trimLocked(end)
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
	total.PeakBytesPerSecond = m.peakLocked(now)
	return total
}

func (m *trafficMeter) trimLocked(now time.Time) {
	cutoff := now.Add(-m.window)
	keep := 0
	for _, event := range m.events {
		if event.end.Before(cutoff) {
			continue
		}
		m.events[keep] = event
		keep++
	}
	m.events = m.events[:keep]
}

func (m *trafficMeter) peakLocked(now time.Time) float64 {
	if len(m.events) == 0 {
		return 0
	}
	cutoff := now.Add(-m.window)
	points := []time.Time{cutoff}
	for _, event := range m.events {
		if event.start.After(cutoff) || event.start.Equal(cutoff) {
			points = append(points, event.start)
		}
	}

	var peak float64
	for _, point := range points {
		var rate float64
		for _, event := range m.events {
			if point.Before(event.start) || !point.Before(event.end) {
				continue
			}
			rate += event.usage.PeakBytesPerSecond
		}
		if rate > peak {
			peak = rate
		}
	}
	return peak
}
