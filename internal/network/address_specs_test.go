package network

import (
	"reflect"
	"testing"
)

func TestParseAddressSpecsSupportsSinglesRangesAndCIDR(t *testing.T) {
	got, err := ParseAddressSpecs("192.168.0.10, 192.168.0.12-14\n192.168.0.16/30", 32)
	if err != nil {
		t.Fatalf("ParseAddressSpecs returned error: %v", err)
	}

	want := []string{
		"192.168.0.10",
		"192.168.0.12",
		"192.168.0.13",
		"192.168.0.14",
		"192.168.0.17",
		"192.168.0.18",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestParseAddressSpecsSupportsFullIPRange(t *testing.T) {
	got, err := ParseAddressSpecs("192.168.0.250-192.168.1.1", 8)
	if err != nil {
		t.Fatalf("ParseAddressSpecs returned error: %v", err)
	}
	want := []string{"192.168.0.250", "192.168.0.251", "192.168.0.252", "192.168.0.253", "192.168.0.254", "192.168.0.255", "192.168.1.0", "192.168.1.1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestParseAddressSpecsRejectsBroadInputs(t *testing.T) {
	if _, err := ParseAddressSpecs("192.168.0.0/24", 16); err == nil {
		t.Fatalf("expected broad CIDR to be rejected")
	}
	if _, err := ParseAddressSpecs("192.168.0.1-99", 16); err == nil {
		t.Fatalf("expected broad range to be rejected")
	}
}
