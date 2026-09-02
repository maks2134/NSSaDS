package netiface

import (
	"fmt"
	"net"

	"NSSaDS/lab6/internal/domain"
)

func Detect(name string) (domain.IfaceInfo, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return domain.IfaceInfo{}, fmt.Errorf("list interfaces: %w", err)
	}
	for _, iface := range ifaces {
		if name != "" && iface.Name != name {
			continue
		}
		info, ok, err := fromInterface(iface)
		if err != nil {
			return domain.IfaceInfo{}, err
		}
		if ok {
			return info, nil
		}
	}
	if name != "" {
		return domain.IfaceInfo{}, fmt.Errorf("interface %q not found or has no IPv4", name)
	}
	return domain.IfaceInfo{}, fmt.Errorf("no suitable IPv4 interface found")
}

func fromInterface(iface net.Interface) (domain.IfaceInfo, bool, error) {
	if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
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
			IP:        ip4,
			Mask:      ipNet.Mask,
			Broadcast: Broadcast(ip4, ipNet.Mask),
		}, true, nil
	}
	return domain.IfaceInfo{}, false, nil
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
