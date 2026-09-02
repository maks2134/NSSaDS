//go:build windows

package icmp

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/sys/windows"
)

type Socket struct {
	fd windows.Handle
}

func Open() (*Socket, error) {
	fd, err := windows.Socket(windows.AF_INET, windows.SOCK_RAW, windows.IPPROTO_ICMP)
	if err != nil {
		return nil, fmt.Errorf("open icmp socket: %w", err)
	}
	s := &Socket{fd: fd}
	if err := s.SetReadTimeout(100 * time.Millisecond); err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}

func (s *Socket) Send(dst net.IP, payload []byte) error {
	addr, err := sockaddr4(dst)
	if err != nil {
		return err
	}
	sa := &windows.SockaddrInet4{Addr: addr}
	if err := windows.Sendto(s.fd, payload, 0, sa); err != nil {
		return fmt.Errorf("icmp send: %w", err)
	}
	return nil
}

func (s *Socket) Peek(buf []byte) (int, net.IP, error) {
	return s.recv(buf, windows.MSG_PEEK)
}

func (s *Socket) Recv(buf []byte) (int, net.IP, error) {
	return s.recv(buf, 0)
}

func (s *Socket) recv(buf []byte, flags int) (int, net.IP, error) {
	n, from, err := windows.Recvfrom(s.fd, buf, flags)
	if err != nil {
		return 0, nil, mapRecvError(err)
	}
	return n, ipFromSockaddr(from), nil
}

func (s *Socket) SetTTL(ttl int) error {
	if err := windows.SetsockoptInt(s.fd, windows.IPPROTO_IP, windows.IP_TTL, ttl); err != nil {
		return fmt.Errorf("set ttl: %w", err)
	}
	return nil
}

func (s *Socket) SetReadTimeout(d time.Duration) error {
	ms := int(d / time.Millisecond)
	if ms <= 0 {
		ms = 1
	}
	if err := windows.SetsockoptInt(s.fd, windows.SOL_SOCKET, windows.SO_RCVTIMEO, ms); err != nil {
		return fmt.Errorf("set recv timeout: %w", err)
	}
	return nil
}

func (s *Socket) Close() error {
	return windows.Closesocket(s.fd)
}

func ipFromSockaddr(sa windows.Sockaddr) net.IP {
	in4, ok := sa.(*windows.SockaddrInet4)
	if !ok {
		return nil
	}
	return ipFrom4(in4.Addr)
}
