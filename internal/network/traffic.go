package network

import "net/netip"

type TrafficUsage struct {
	BytesIn            int64   `json:"bytesIn"`
	BytesOut           int64   `json:"bytesOut"`
	TotalBytes         int64   `json:"totalBytes"`
	BytesPerSecond     float64 `json:"bytesPerSecond,omitempty"`
	PeakBytesPerSecond float64 `json:"peakBytesPerSecond,omitempty"`
	WindowSeconds      int     `json:"windowSeconds,omitempty"`
	SSHSessions        int     `json:"sshSessions,omitempty"`
	PortChecks         int     `json:"portChecks,omitempty"`
	Estimated          bool    `json:"estimated"`
}

func (u TrafficUsage) Add(v TrafficUsage) TrafficUsage {
	u.BytesIn += v.BytesIn
	u.BytesOut += v.BytesOut
	u.TotalBytes = u.BytesIn + u.BytesOut
	if v.PeakBytesPerSecond > u.PeakBytesPerSecond {
		u.PeakBytesPerSecond = v.PeakBytesPerSecond
	}
	u.SSHSessions += v.SSHSessions
	u.PortChecks += v.PortChecks
	u.Estimated = u.Estimated || v.Estimated
	return u
}

func EstimateScanTraffic(cidr string, ports []int, maxHosts int, results []HostScan) (TrafficUsage, error) {
	prefix, err := netip.ParsePrefix(cidr)
	if err != nil {
		return TrafficUsage{}, err
	}
	if !prefix.Addr().Is4() {
		return TrafficUsage{}, nil
	}

	if len(ports) == 0 {
		ports = []int{22, 2375, 2376}
	}
	if maxHosts <= 0 {
		maxHosts = 1024
	}

	hostCount := len(hosts(prefix))
	if hostCount > maxHosts {
		hostCount = maxHosts
	}
	portChecks := hostCount * len(ports)
	openPorts := 0
	for _, result := range results {
		openPorts += len(result.OpenPorts)
	}

	usage := TrafficUsage{
		BytesOut:   int64(portChecks * 96),
		BytesIn:    int64(openPorts * 96),
		PortChecks: portChecks,
		Estimated:  true,
	}
	usage.TotalBytes = usage.BytesIn + usage.BytesOut
	return usage, nil
}
