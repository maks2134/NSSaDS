package usecase

import (
	"time"

	"NSSaDS/lab5/internal/domain"
)

func dispatchReply(
	out domain.Printer,
	host string,
	pkt *domain.Packet,
	stats *domain.HostStats,
) {
	now := time.Now()
	switch pkt.Kind {
	case domain.KindEchoReply:
		onEchoReply(out, host, pkt, stats, now)
	case domain.KindTimeExceeded:
		onTimeExceeded(out, host, pkt, now)
	case domain.KindUnreachable:
		onUnreachable(out, host, pkt, stats)
	default:
		out.Printf("%s: unexpected icmp from %s\n", host, pkt.From)
	}
}

func onEchoReply(
	out domain.Printer,
	host string,
	pkt *domain.Packet,
	stats *domain.HostStats,
	now time.Time,
) {
	rtt := ProbeRTT(pkt, now)
	stats.Received++
	stats.RTTs = append(stats.RTTs, rtt)
	out.Printf("%s\n", HandleEchoReply(host, pkt, rtt))
}

func onTimeExceeded(
	out domain.Printer,
	host string,
	pkt *domain.Packet,
	now time.Time,
) {
	out.Printf("%s\n", HandleTimeExceeded(host, 0, pkt, ProbeRTT(pkt, now)))
}

func onUnreachable(
	out domain.Printer,
	host string,
	pkt *domain.Packet,
	stats *domain.HostStats,
) {
	stats.Unreachable++
	out.Printf("%s\n", HandleUnreachable(host, pkt))
}
