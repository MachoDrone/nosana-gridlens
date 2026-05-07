package network

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"
	"sort"
	"strconv"
	"strings"
)

func ParseAddressSpecs(input string, maxHosts int) ([]string, error) {
	if maxHosts <= 0 {
		maxHosts = 1024
	}

	parts := strings.FieldsFunc(input, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\t' || r == ' '
	})

	seen := map[string]bool{}
	var addresses []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		expanded, err := expandAddressSpec(part, maxHosts)
		if err != nil {
			return nil, err
		}
		for _, address := range expanded {
			if !seen[address] {
				seen[address] = true
				addresses = append(addresses, address)
			}
		}
	}

	sort.Slice(addresses, func(i, j int) bool {
		return ipv4ToUint(addresses[i]) < ipv4ToUint(addresses[j])
	})
	return addresses, nil
}

func expandAddressSpec(spec string, maxHosts int) ([]string, error) {
	if strings.Contains(spec, "/") {
		prefix, err := netip.ParsePrefix(spec)
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR %q: %w", spec, err)
		}
		if !prefix.Addr().Is4() {
			return nil, fmt.Errorf("only IPv4 CIDR is supported: %s", spec)
		}

		expanded := hosts(prefix)
		if len(expanded) > maxHosts {
			return nil, fmt.Errorf("CIDR %s expands to %d hosts; maximum is %d", spec, len(expanded), maxHosts)
		}

		addresses := make([]string, 0, len(expanded))
		for _, addr := range expanded {
			addresses = append(addresses, addr.String())
		}
		return addresses, nil
	}

	if strings.Contains(spec, "-") {
		return expandRangeSpec(spec, maxHosts)
	}

	ip := net.ParseIP(spec).To4()
	if ip == nil {
		return nil, fmt.Errorf("invalid IPv4 address %q", spec)
	}
	return []string{ip.String()}, nil
}

func expandRangeSpec(spec string, maxHosts int) ([]string, error) {
	startText, endText, ok := strings.Cut(spec, "-")
	if !ok {
		return nil, fmt.Errorf("invalid range %q", spec)
	}

	startIP := net.ParseIP(strings.TrimSpace(startText)).To4()
	if startIP == nil {
		return nil, fmt.Errorf("invalid range start %q", startText)
	}

	endText = strings.TrimSpace(endText)
	var endIP net.IP
	if strings.Contains(endText, ".") {
		endIP = net.ParseIP(endText).To4()
		if endIP == nil {
			return nil, fmt.Errorf("invalid range end %q", endText)
		}
	} else {
		lastOctet, err := strconv.Atoi(endText)
		if err != nil || lastOctet < 0 || lastOctet > 255 {
			return nil, fmt.Errorf("invalid range end %q", endText)
		}
		endIP = net.IPv4(startIP[0], startIP[1], startIP[2], byte(lastOctet)).To4()
	}

	start := binary.BigEndian.Uint32(startIP)
	end := binary.BigEndian.Uint32(endIP)
	if end < start {
		return nil, fmt.Errorf("range end must be greater than or equal to start: %s", spec)
	}
	count := int(end-start) + 1
	if count > maxHosts {
		return nil, fmt.Errorf("range %s expands to %d hosts; maximum is %d", spec, count, maxHosts)
	}

	addresses := make([]string, 0, count)
	for value := start; value <= end; value++ {
		var raw [4]byte
		binary.BigEndian.PutUint32(raw[:], value)
		addresses = append(addresses, net.IP(raw[:]).String())
		if value == end {
			break
		}
	}
	return addresses, nil
}

func ipv4ToUint(address string) uint32 {
	ip := net.ParseIP(address).To4()
	if ip == nil {
		return 0
	}
	return binary.BigEndian.Uint32(ip)
}
