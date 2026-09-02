package domain

import (
	"net"
	"time"
)

type ReplyKind int

const (
	KindUnknown ReplyKind = iota
	KindEchoReply
	KindTimeExceeded
	KindUnreachable
)

func (k ReplyKind) String() string {
	switch k {
	case KindEchoReply:
		return "echo-reply"
	case KindTimeExceeded:
		return "time-exceeded"
	case KindUnreachable:
		return "unreachable"
	default:
		return "unknown"
	}
}

type Packet struct {
	Kind    ReplyKind
	From    net.IP
	TTL     int
	ID      int
	Seq     int
	OrigID  int
	OrigSeq int
	Code    int
	Payload []byte
	Size    int
}

type Probe struct {
	Target net.IP
	ID     int
	Seq    int
	TTL    int
}

type Hop struct {
	Number int
	Addr   net.IP
	RTT    time.Duration
	Kind   ReplyKind
	Missed bool
}

type HostStats struct {
	Host        string
	Addr        net.IP
	Transmitted int
	Received    int
	Unreachable int
	RTTs        []time.Duration
}
