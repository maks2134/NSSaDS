package domain

import (
	"net"
	"testing"
	"time"

	"NSSaDS/lab6/pkg/config"
)

func TestPeerRegistry(t *testing.T) {
	r := NewPeerRegistry()
	ip := net.IPv4(10, 0, 0, 2)
	now := time.Now()

	r.Upsert(ip, "bob", ModeBroadcast, now)
	peers := r.List(ModeBroadcast)
	if len(peers) != 1 || peers[0].Nick != "bob" {
		t.Fatalf("unexpected peers: %+v", peers)
	}

	r.SetIgnored(ip, true)
	if !r.IsIgnored(ip) {
		t.Fatal("expected ignored")
	}

	r.Expire(now.Add(config.PeerTimeout + time.Second))
	if len(r.List(ModeBroadcast)) != 0 {
		t.Fatal("expected expired peer removed")
	}
}

func TestPeerRegistryModeFilter(t *testing.T) {
	r := NewPeerRegistry()
	now := time.Now()
	r.Upsert(net.IPv4(10, 0, 0, 1), "a", ModeBroadcast, now)
	r.Upsert(net.IPv4(10, 0, 0, 2), "b", ModeMulticast, now)

	if len(r.List(ModeBroadcast)) != 1 {
		t.Fatal("broadcast filter failed")
	}
	if len(r.List(ModeMulticast)) != 1 {
		t.Fatal("multicast filter failed")
	}
}
