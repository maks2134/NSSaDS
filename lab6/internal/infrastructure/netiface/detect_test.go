package netiface

import (
	"net"
	"testing"
)

func TestBroadcast(t *testing.T) {
	ip := net.IPv4(192, 168, 1, 10)
	mask := net.CIDRMask(24, 32)
	got := Broadcast(ip, mask)
	if got.String() != "192.168.1.255" {
		t.Fatalf("got %s, want 192.168.1.255", got)
	}
}

func TestBroadcastClassB(t *testing.T) {
	ip := net.IPv4(172, 16, 5, 1)
	mask := net.CIDRMask(16, 32)
	got := Broadcast(ip, mask)
	if got.String() != "172.16.255.255" {
		t.Fatalf("got %s, want 172.16.255.255", got)
	}
}

func TestIsLinkLocal(t *testing.T) {
	if !isLinkLocal(net.IPv4(169, 254, 1, 1)) {
		t.Fatal("expected link-local")
	}
	if isLinkLocal(net.IPv4(192, 168, 1, 1)) {
		t.Fatal("expected not link-local")
	}
}
