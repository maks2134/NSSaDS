package icmp

import (
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
