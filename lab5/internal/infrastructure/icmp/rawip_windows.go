//go:build windows

package icmp

import (
	"fmt"
	"net"

	"golang.org/x/sys/windows"
)

type RawIP struct {
	fd windows.Handle
}

func OpenRawIP() (*RawIP, error) {
	fd, err := windows.Socket(windows.AF_INET, windows.SOCK_RAW, windows.IPPROTO_IP)
	if err != nil {
		return nil, fmt.Errorf("open raw ip socket: %w", err)
	}
	if err := windows.SetsockoptInt(fd, windows.IPPROTO_IP, windows.IP_HDRINCL, 1); err != nil {
		_ = windows.Closesocket(fd)
		return nil, fmt.Errorf("set ip_hdrincl: %w", err)
	}
	if err := windows.SetsockoptInt(fd, windows.SOL_SOCKET, windows.SO_BROADCAST, 1); err != nil {
		_ = windows.Closesocket(fd)
		return nil, fmt.Errorf("set so_broadcast: %w", err)
	}
	return &RawIP{fd: fd}, nil
}

func (r *RawIP) SendIP(dst net.IP, packet []byte) error {
	addr, err := sockaddr4(dst)
	if err != nil {
		return err
	}
	sa := &windows.SockaddrInet4{Addr: addr}
	if err := windows.Sendto(r.fd, packet, 0, sa); err != nil {
		return fmt.Errorf("raw ip send: %w", err)
	}
	return nil
}

func (r *RawIP) Close() error {
	return windows.Closesocket(r.fd)
}
