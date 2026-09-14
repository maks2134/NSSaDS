package netiface

import (
	"net"
	"testing"

	"NSSaDS/lab6/internal/domain"
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

func TestIsVirtualName(t *testing.T) {
	cases := map[string]bool{
		"vEthernet (Default Switch)":    true,
		"Ethernet":                      false,
		"Wi-Fi":                         false,
		"en0":                           false,
		"docker0":                       true,
		"VMware Network Adapter VMnet8": true,
	}
	for name, want := range cases {
		if got := IsVirtualName(name); got != want {
			t.Fatalf("%q: got %v want %v", name, got, want)
		}
	}
}

func TestPickBestPrefersLAN(t *testing.T) {
	cands := []struct {
		Name string
		IP   string
		Mask int
	}{
		{Name: "vEthernet (Default Switch)", IP: "172.25.240.1", Mask: 20},
		{Name: "Wi-Fi", IP: "192.168.0.50", Mask: 24},
	}
	infos := make([]domain.IfaceInfo, 0, len(cands))
	for _, c := range cands {
		ip := net.ParseIP(c.IP).To4()
		mask := net.CIDRMask(c.Mask, 32)
		infos = append(infos, domain.IfaceInfo{
			Name:      c.Name,
			IP:        ip,
			Mask:      mask,
			Broadcast: Broadcast(ip, mask),
		})
	}
	best, ok := pickBest(infos)
	if !ok {
		t.Fatal("expected candidate")
	}
	if best.Name != "Wi-Fi" {
		t.Fatalf("got %s, want Wi-Fi", best.Name)
	}
}
