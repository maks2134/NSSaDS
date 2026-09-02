package icmp

import (
	"errors"
	"net"
	"time"
)

var (
	ErrNotForWorker = errors.New("packet belongs to another worker")
	ErrTimeout      = errors.New("recv timeout")
	errDropped      = errors.New("unrelated packet dropped")
)

func mapRecvError(err error) error {
	if err == nil {
		return nil
	}
	if isTimeout(err) {
		return ErrTimeout
	}
	return err
}

func sockaddr4(dst net.IP) ([4]byte, error) {
	ip := dst.To4()
	if ip == nil {
		return [4]byte{}, ErrNotIPv4
	}
	var addr [4]byte
	copy(addr[:], ip)
	return addr, nil
}

func ipFrom4(addr [4]byte) net.IP {
	return net.IPv4(addr[0], addr[1], addr[2], addr[3])
}

func durationToNsec(d time.Duration) int64 {
	if d < 0 {
		return 0
	}
	return d.Nanoseconds()
}
