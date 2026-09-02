//go:build unix

package icmp

import (
	"fmt"
	"net"

	"golang.org/x/sys/unix"
)

type RawIP struct {
	fd int
}

func OpenRawIP() (*RawIP, error) {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_RAW, unix.IPPROTO_RAW)
	if err != nil {
		return nil, fmt.Errorf("open raw ip socket: %w", err)
	}
	if err := unix.SetsockoptInt(fd, unix.IPPROTO_IP, unix.IP_HDRINCL, 1); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("set ip_hdrincl: %w", err)
	}
	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_BROADCAST, 1); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("set so_broadcast: %w", err)
	}
	return &RawIP{fd: fd}, nil
}

func (r *RawIP) SendIP(dst net.IP, packet []byte) error {
	addr, err := sockaddr4(dst)
	if err != nil {
		return err
	}
	sa := &unix.SockaddrInet4{Addr: addr}
	if err := unix.Sendto(r.fd, packet, 0, sa); err != nil {
		return fmt.Errorf("raw ip send: %w", err)
	}
	return nil
}

func (r *RawIP) Close() error {
	return unix.Close(r.fd)
}
