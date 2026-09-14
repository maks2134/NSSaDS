//go:build unix

package icmp

import (
	"fmt"
	"net"
	"runtime"
	"time"

	"golang.org/x/sys/unix"
)

type Socket struct {
	fd int
}

func Open() (*Socket, error) {
	fd, err := unix.Socket(unix.AF_INET, icmpSockType(), unix.IPPROTO_ICMP)
	if err != nil {
		return nil, fmt.Errorf("open icmp socket: %w", err)
	}
	s := &Socket{fd: fd}
	if err := s.configure(); err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}

func icmpSockType() int {
	switch runtime.GOOS {
	case "darwin", "ios":
		// macOS does not deliver Echo Reply to SOCK_RAW; SOCK_DGRAM does.
		return unix.SOCK_DGRAM
	default:
		return unix.SOCK_RAW
	}
}

func (s *Socket) configure() error {
	if err := s.SetReadTimeout(100 * time.Millisecond); err != nil {
		return err
	}
	_ = unix.SetsockoptInt(s.fd, unix.IPPROTO_IP, unix.IP_RECVTTL, 1)
	_ = unix.Bind(s.fd, &unix.SockaddrInet4{})
	return nil
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

func (s *Socket) Peek(buf []byte) (int, net.IP, int, error) {
	return s.recv(buf, unix.MSG_PEEK)
}

func (s *Socket) Recv(buf []byte) (int, net.IP, int, error) {
	return s.recv(buf, 0)
}

func (s *Socket) recv(buf []byte, flags int) (int, net.IP, int, error) {
	oob := make([]byte, unix.CmsgSpace(16))
	n, oobn, _, from, err := unix.Recvmsg(s.fd, buf, oob, flags)
	if err != nil {
		return 0, nil, 0, mapRecvError(err)
	}
	return n, ipFromSockaddr(from), ttlFromOOB(oob[:oobn]), nil
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

func ttlFromOOB(oob []byte) int {
	if len(oob) == 0 {
		return 0
	}
	msgs, err := unix.ParseSocketControlMessage(oob)
	if err != nil {
		return 0
	}
	for _, msg := range msgs {
		if msg.Header.Level == unix.IPPROTO_IP && len(msg.Data) > 0 {
			if msg.Header.Type == unix.IP_TTL || msg.Header.Type == unix.IP_RECVTTL {
				return int(msg.Data[0])
			}
		}
	}
	return 0
}
