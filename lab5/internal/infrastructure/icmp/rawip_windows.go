//go:build windows

package icmp

import (
	"errors"
	"fmt"
	"net"

	"golang.org/x/sys/windows"
)

type RawIP struct {
	fd windows.Handle
}

func OpenRawIP() (*RawIP, error) {
	protos := []int{windows.IPPROTO_ICMP, windows.IPPROTO_IP}
	var last error
	for _, proto := range protos {
		r, err := openRawIPProto(proto)
		if err == nil {
			return r, nil
		}
		last = err
	}
	return nil, denyHint(fmt.Errorf("open raw ip socket: %w", last))
}

func openRawIPProto(proto int) (*RawIP, error) {
	fd, err := windows.Socket(windows.AF_INET, windows.SOCK_RAW, proto)
	if err != nil {
		return nil, err
	}
	if err := windows.SetsockoptInt(fd, windows.IPPROTO_IP, windows.IP_HDRINCL, 1); err != nil {
		_ = windows.Closesocket(fd)
		return nil, err
	}
	_ = windows.SetsockoptInt(fd, windows.SOL_SOCKET, windows.SO_BROADCAST, 1)
	return &RawIP{fd: fd}, nil
}

func denyHint(err error) error {
	if errors.Is(err, windows.WSAEACCES) || errors.Is(err, windows.ERROR_ACCESS_DENIED) {
		return fmt.Errorf("%w; run cmd.exe as Administrator", err)
	}
	return err
}

func (r *RawIP) SendIP(dst net.IP, packet []byte) error {
	addr, err := sockaddr4(dst)
	if err != nil {
		return err
	}
	sa := &windows.SockaddrInet4{Addr: addr}
	if err := windows.Sendto(r.fd, packet, 0, sa); err != nil {
		return denyHint(fmt.Errorf("raw ip send: %w", err))
	}
	return nil
}

func (r *RawIP) Close() error {
	return windows.Closesocket(r.fd)
}
