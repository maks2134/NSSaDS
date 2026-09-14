package usecase

import (
	"fmt"
	"net"
	"strings"
)

func BroadcastAddr(ip net.IP, mask net.IPMask) net.IP {
	ip4 := ip.To4()
	if ip4 == nil || len(mask) < 4 {
		return nil
	}
	if len(mask) > 4 {
		mask = mask[len(mask)-4:]
	}
	out := make(net.IP, 4)
	for i := 0; i < 4; i++ {
		out[i] = ip4[i] | ^mask[i]
	}
	return out
}

func LANToward(dst net.IP) (local, bcast net.IP, err error) {
	dst4 := dst.To4()
	if dst4 == nil {
		return nil, nil, fmt.Errorf("not an ipv4 address")
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, nil, err
	}
	var fallbackLocal, fallbackBcast net.IP
	for _, iface := range ifaces {
		localIP, mask, ok := firstIPv4(iface)
		if !ok {
			continue
		}
		bcastIP := BroadcastAddr(localIP, mask)
		if ipNetContains(localIP, mask, dst4) {
			return localIP, bcastIP, nil
		}
		if fallbackLocal == nil && !isTunnel(iface.Name) {
			fallbackLocal, fallbackBcast = localIP, bcastIP
		}
	}
	if fallbackLocal != nil {
		return fallbackLocal, fallbackBcast, nil
	}
	return nil, nil, fmt.Errorf("no ipv4 interface")
}

func firstIPv4(iface net.Interface) (net.IP, net.IPMask, bool) {
	if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
		return nil, nil, false
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, nil, false
	}
	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		ip4 := ipnet.IP.To4()
		if ip4 == nil || ipnet.Mask == nil {
			continue
		}
		return ip4, ipnet.Mask, true
	}
	return nil, nil, false
}

func ipNetContains(ip net.IP, mask net.IPMask, dst net.IP) bool {
	network := &net.IPNet{IP: ip, Mask: mask}
	return network.Contains(dst)
}

func isTunnel(name string) bool {
	n := strings.ToLower(name)
	for _, p := range []string{"utun", "tun", "wg", "ppp", "ipsec", "tailscale"} {
		if strings.HasPrefix(n, p) {
			return true
		}
	}
	return false
}

func uniqueIPs(ips ...net.IP) []net.IP {
	out := make([]net.IP, 0, len(ips))
	seen := map[string]struct{}{}
	for _, ip := range ips {
		if ip == nil || ip.To4() == nil {
			continue
		}
		key := ip.To4().String()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, ip.To4())
	}
	return out
}
