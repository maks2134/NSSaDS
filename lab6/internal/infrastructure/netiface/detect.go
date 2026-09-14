package netiface

import (
	"fmt"
	"net"
	"strings"

	"NSSaDS/lab6/internal/domain"
)

func Detect(name string) (domain.IfaceInfo, error) {
	cands, err := Candidates()
	if err != nil {
		return domain.IfaceInfo{}, err
	}
	if name != "" {
		for _, c := range cands {
			if c.Name == name {
				return c, nil
			}
		}
		return domain.IfaceInfo{}, fmt.Errorf(
			"interface %q not found or has no IPv4; try -list", name)
	}
	best, ok := pickBest(cands)
	if !ok {
		return domain.IfaceInfo{}, fmt.Errorf(
			"no suitable IPv4 interface found; try -list and -iface")
	}
	return best, nil
}

func Candidates() ([]domain.IfaceInfo, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("list interfaces: %w", err)
	}
	out := make([]domain.IfaceInfo, 0)
	for _, iface := range ifaces {
		info, ok, err := fromInterface(iface)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, info)
		}
	}
	return out, nil
}

func pickBest(cands []domain.IfaceInfo) (domain.IfaceInfo, bool) {
	bestScore := -1
	var best domain.IfaceInfo
	found := false
	for _, c := range cands {
		if IsVirtualName(c.Name) {
			continue
		}
		score := scoreIface(c)
		if !found || score > bestScore {
			best = c
			bestScore = score
			found = true
		}
	}
	if found {
		return best, true
	}
	// fall back to any non-empty candidate if all look virtual
	for _, c := range cands {
		score := scoreIface(c)
		if !found || score > bestScore {
			best = c
			bestScore = score
			found = true
		}
	}
	return best, found
}

func fromInterface(iface net.Interface) (domain.IfaceInfo, bool, error) {
	if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
		return domain.IfaceInfo{}, false, nil
	}
	if iface.Flags&net.FlagPointToPoint != 0 {
		return domain.IfaceInfo{}, false, nil
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return domain.IfaceInfo{}, false, fmt.Errorf("list addrs for %s: %w", iface.Name, err)
	}
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		ip4 := ipNet.IP.To4()
		if ip4 == nil || isLinkLocal(ip4) {
			continue
		}
		return domain.IfaceInfo{
			Name:      iface.Name,
			IP:        append(net.IP(nil), ip4...),
			Mask:      append(net.IPMask(nil), ipNet.Mask...),
			Broadcast: Broadcast(ip4, ipNet.Mask),
		}, true, nil
	}
	return domain.IfaceInfo{}, false, nil
}

func IsVirtualName(name string) bool {
	n := strings.ToLower(name)
	needles := []string{
		"vethernet", "hyper-v", "vmware", "virtualbox", "vbox",
		"docker", "veth", "wsl", "bluetooth", "tailscale", "zerotier",
		"utun", "awdl", "llw", "bridge", "appletalk", "hamachi",
		"npcap", "loopback", "pseudo",
	}
	for _, s := range needles {
		if strings.Contains(n, s) {
			return true
		}
	}
	return strings.HasPrefix(n, "br-")
}

func scoreIface(info domain.IfaceInfo) int {
	score := 0
	ip := info.IP.To4()
	if ip == nil {
		return -100
	}
	ones, bits := info.Mask.Size()
	if bits == 32 && ones == 24 {
		score += 30
	}
	if bits == 32 && ones >= 20 && ones <= 24 {
		score += 10
	}
	switch {
	case ip[0] == 192 && ip[1] == 168:
		score += 50
	case ip[0] == 10:
		score += 30
	case ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31:
		score += 5 // often Hyper-V / Docker
	default:
		score -= 10
	}
	if IsVirtualName(info.Name) {
		score -= 100
	}
	return score
}

func isLinkLocal(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	return ip4[0] == 169 && ip4[1] == 254
}

func Broadcast(ip net.IP, mask net.IPMask) net.IP {
	ip4 := ip.To4()
	if ip4 == nil || len(mask) != net.IPv4len {
		return nil
	}
	out := make(net.IP, net.IPv4len)
	for i := range ip4 {
		out[i] = ip4[i] | ^mask[i]
	}
	return out
}
