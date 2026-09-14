package icmp

import (
	"net"

	"NSSaDS/lab5/internal/domain"
)

func PacketWorkerID(pkt *domain.Packet) (int, bool) {
	switch pkt.Kind {
	case domain.KindEchoReply:
		return pkt.ID, true
	case domain.KindTimeExceeded, domain.KindUnreachable:
		return pkt.OrigID, true
	default:
		return 0, false
	}
}

func MatchesWorker(pkt *domain.Packet, id, seq int) bool {
	wid, ok := PacketWorkerID(pkt)
	if !ok || wid != id {
		return false
	}
	return PacketSequence(pkt) == seq
}

func MatchesTarget(pkt *domain.Packet, target net.IP, seq int) bool {
	if target == nil || PacketSequence(pkt) != seq {
		return false
	}
	switch pkt.Kind {
	case domain.KindEchoReply:
		return ipEqual(pkt.From, target)
	case domain.KindTimeExceeded, domain.KindUnreachable:
		return ipEqual(pkt.OrigDst, target)
	default:
		return false
	}
}

func ipEqual(a, b net.IP) bool {
	if a == nil || b == nil {
		return false
	}
	a4, b4 := a.To4(), b.To4()
	if a4 != nil && b4 != nil {
		return a4.Equal(b4)
	}
	return a.Equal(b)
}
