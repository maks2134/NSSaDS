package domain

import (
	"net"
	"time"
)

type Socket interface {
	Send(dst net.IP, payload []byte) error
	Peek(buf []byte) (n int, from net.IP, err error)
	Recv(buf []byte) (n int, from net.IP, err error)
	SetTTL(ttl int) error
	SetReadTimeout(d time.Duration) error
	Close() error
}

type RawIPSocket interface {
	SendIP(dst net.IP, packet []byte) error
	Close() error
}

type Printer interface {
	Printf(format string, args ...any)
}
