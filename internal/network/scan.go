package network

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type HostScan struct {
	IP        string `json:"ip"`
	OpenPorts []int  `json:"openPorts"`
}

type ScanOptions struct {
	CIDR        string
	Ports       []int
	Timeout     time.Duration
	Concurrency int
	MaxHosts    int
}

func ParsePorts(value string) ([]int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return []int{22, 2375, 2376}, nil
	}

	var ports []int
	seen := map[int]bool{}
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		port, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid port %q", part)
		}
		if port < 1 || port > 65535 {
			return nil, fmt.Errorf("port out of range: %d", port)
		}
		if seen[port] {
			continue
		}
		seen[port] = true
		ports = append(ports, port)
	}
	sort.Ints(ports)
	return ports, nil
}

func ScanCIDR(ctx context.Context, opts ScanOptions) ([]HostScan, error) {
	prefix, err := netip.ParsePrefix(opts.CIDR)
	if err != nil {
		return nil, err
	}
	if !prefix.Addr().Is4() {
		return nil, fmt.Errorf("only IPv4 CIDR scanning is supported in this phase")
	}

	ports := append([]int(nil), opts.Ports...)
	sort.Ints(ports)
	if len(ports) == 0 {
		ports = []int{22, 2375, 2376}
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 300 * time.Millisecond
	}
	concurrency := opts.Concurrency
	if concurrency <= 0 {
		concurrency = 64
	}
	maxHosts := opts.MaxHosts
	if maxHosts <= 0 {
		maxHosts = 1024
	}

	ips := hosts(prefix)
	if len(ips) > maxHosts {
		return nil, fmt.Errorf("CIDR %s has %d hosts; use /24 or narrower, or raise max host limit explicitly", opts.CIDR, len(ips))
	}
	jobs := make(chan netip.Addr)
	results := make(chan HostScan, len(ips))

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ip := range jobs {
				openPorts := scanHost(ctx, ip.String(), ports, timeout)
				if len(openPorts) > 0 {
					results <- HostScan{IP: ip.String(), OpenPorts: openPorts}
				}
			}
		}()
	}

	for _, ip := range ips {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			close(results)
			return nil, ctx.Err()
		case jobs <- ip:
		}
	}
	close(jobs)
	wg.Wait()
	close(results)

	var scans []HostScan
	for result := range results {
		scans = append(scans, result)
	}
	sort.Slice(scans, func(i, j int) bool {
		return scans[i].IP < scans[j].IP
	})
	return scans, nil
}

func LocalIPv4CIDRs() []string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	var cidrs []string
	seen := map[string]bool{}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil || !isPrivateIPv4(ip) {
				continue
			}
			ones, bits := ipNet.Mask.Size()
			if bits != 32 || ones == 32 || ones < 24 {
				continue
			}
			networkIP := ip.Mask(ipNet.Mask)
			cidr := fmt.Sprintf("%s/%d", networkIP.String(), ones)
			if !seen[cidr] {
				seen[cidr] = true
				cidrs = append(cidrs, cidr)
			}
		}
	}
	sort.Strings(cidrs)
	return cidrs
}

func LocalIPv4Addresses() []string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	var addresses []string
	seen := map[string]bool{}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil || !isPrivateIPv4(ip) {
				continue
			}
			address := ip.String()
			if !seen[address] {
				seen[address] = true
				addresses = append(addresses, address)
			}
		}
	}
	sort.Slice(addresses, func(i, j int) bool {
		return ipv4ToUint(addresses[i]) < ipv4ToUint(addresses[j])
	})
	return addresses
}

func PrimaryLocalIPv4() string {
	addresses := LocalIPv4Addresses()
	if len(addresses) == 0 {
		return ""
	}
	return addresses[0]
}

func scanHost(ctx context.Context, ip string, ports []int, timeout time.Duration) []int {
	var open []int
	for _, port := range ports {
		address := net.JoinHostPort(ip, strconv.Itoa(port))
		dialer := net.Dialer{Timeout: timeout}
		conn, err := dialer.DialContext(ctx, "tcp", address)
		if err != nil {
			continue
		}
		_ = conn.Close()
		open = append(open, port)
	}
	return open
}

func hosts(prefix netip.Prefix) []netip.Addr {
	var values []netip.Addr
	prefix = prefix.Masked()
	for ip := prefix.Addr(); prefix.Contains(ip); ip = ip.Next() {
		if ip == prefix.Addr() {
			continue
		}
		values = append(values, ip)
	}
	if len(values) > 1 {
		values = values[:len(values)-1]
	}
	return values
}

func isPrivateIPv4(ip net.IP) bool {
	return ip[0] == 10 ||
		(ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31) ||
		(ip[0] == 192 && ip[1] == 168)
}
