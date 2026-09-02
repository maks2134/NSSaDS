//go:build unix

package icmp

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/sys/unix"
)

type Socket struct {
	fd int
}

func Open() (*Socket, error) {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_RAW, unix.IPPROTO_ICMP)
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
	sa := &unix.SockaddrInet4{Addr: addr}
	if err := unix.Sendto(s.fd, payload, 0, sa); err != nil {
		return fmt.Errorf("icmp send: %w", err)
	}
	return nil
}

func (s *Socket) Peek(buf []byte) (int, net.IP, error) {
	return s.recv(buf, unix.MSG_PEEK)
}

func (s *Socket) Recv(buf []byte) (int, net.IP, error) {
	return s.recv(buf, 0)
}

func (s *Socket) recv(buf []byte, flags int) (int, net.IP, error) {
	n, from, err := unix.Recvfrom(s.fd, buf, flags)
	if err != nil {
		return 0, nil, mapRecvError(err)
	}
	return n, ipFromSockaddr(from), nil
}

func (s *Socket) SetTTL(ttl int) error {
	if err := unix.SetsockoptInt(s.fd, unix.IPPROTO_IP, unix.IP_TTL, ttl); err != nil {
		return fmt.Errorf("set ttl: %w", err)
	}
	return nil
}

func (s *Socket) SetReadTimeout(d time.Duration) error {
	tv := unix.NsecToTimeval(durationToNsec(d))
	if err := unix.SetsockoptTimeval(s.fd, unix.SOL_SOCKET, unix.SO_RCVTIMEO, &tv); err != nil {
		return fmt.Errorf("set recv timeout: %w", err)
	}
	return nil
}

func (s *Socket) Close() error {
	return unix.Close(s.fd)
}

func ipFromSockaddr(sa unix.Sockaddr) net.IP {
	in4, ok := sa.(*unix.SockaddrInet4)
	if !ok {
		return nil
	}
	return ipFrom4(in4.Addr)
}
