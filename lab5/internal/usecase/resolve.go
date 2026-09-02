package usecase

import (
	"fmt"
	"net"
)

type Target struct {
	Host string
	IP   net.IP
}

func ResolveIPv4(host string) (net.IP, error) {
	if ip := net.ParseIP(host); ip != nil {
		v4 := ip.To4()
		if v4 == nil {
			return nil, fmt.Errorf("not an ipv4 address: %s", host)
		}
		return v4, nil
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", host, err)
	}
	for _, ip := range ips {
		if v4 := ip.To4(); v4 != nil {
			return v4, nil
		}
	}
	return nil, fmt.Errorf("no ipv4 address for %s", host)
}

func ResolveAll(hosts []string) ([]Target, error) {
	targets := make([]Target, 0, len(hosts))
	for _, host := range hosts {
		ip, err := ResolveIPv4(host)
		if err != nil {
			return nil, err
		}
		targets = append(targets, Target{Host: host, IP: ip})
	}
	return targets, nil
}
