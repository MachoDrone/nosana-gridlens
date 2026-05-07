package network

import (
	"context"
	"reflect"
	"testing"
)

func TestParsePorts(t *testing.T) {
	ports, err := ParsePorts("2376,22,2375,22")
	if err != nil {
		t.Fatalf("ParsePorts returned error: %v", err)
	}
	want := []int{22, 2375, 2376}
	if !reflect.DeepEqual(ports, want) {
		t.Fatalf("expected %v, got %v", want, ports)
	}
}

func TestParsePortsRejectsInvalidPort(t *testing.T) {
	if _, err := ParsePorts("0"); err == nil {
		t.Fatalf("expected error for port 0")
	}
	if _, err := ParsePorts("nope"); err == nil {
		t.Fatalf("expected error for invalid port")
	}
}

func TestScanCIDRRejectsBroadNetwork(t *testing.T) {
	_, err := ScanCIDR(context.Background(), ScanOptions{CIDR: "192.168.0.0/16", Ports: []int{22}, MaxHosts: 10})
	if err == nil {
		t.Fatalf("expected broad CIDR to be rejected")
	}
}
