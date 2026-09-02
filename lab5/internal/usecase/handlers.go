package usecase

import (
	"fmt"
	"strconv"
	"time"

	"NSSaDS/lab5/internal/domain"
	icmphost "NSSaDS/lab5/internal/infrastructure/icmp"
)

func HandleEchoReply(host string, pkt *domain.Packet, rtt time.Duration) string {
	return fmt.Sprintf(
		"%d bytes from %s (%s): icmp_seq=%d ttl=%d time=%s",
		pkt.Size,
		host,
		pkt.From,
		pkt.Seq,
		pkt.TTL,
		FormatRTT(rtt),
	)
}

func HandleTimeExceeded(host string, hop int, pkt *domain.Packet, rtt time.Duration) string {
	return fmt.Sprintf(
		"%2d  %s  %s  (ttl expired, dest %s)",
		hop,
		pkt.From,
		FormatRTT(rtt),
		host,
	)
}

func HandleUnreachable(host string, pkt *domain.Packet) string {
	return fmt.Sprintf(
		"From %s: destination %s unreachable (code %d)",
		pkt.From,
		host,
		pkt.Code,
	)
}

func FormatRTT(d time.Duration) string {
	ms := float64(d.Microseconds()) / 1000
	return strconv.FormatFloat(ms, 'f', 3, 64) + " ms"
}

func FormatStats(st domain.HostStats) string {
	loss := 0.0
	if st.Transmitted > 0 {
		loss = float64(st.Transmitted-st.Received) * 100 / float64(st.Transmitted)
	}
	line := fmt.Sprintf(
		"--- %s ping statistics ---\n%d packets transmitted, %d received, %.0f%% packet loss",
		st.Host,
		st.Transmitted,
		st.Received,
		loss,
	)
	if len(st.RTTs) == 0 {
		return line
	}
	min, avg, max := rttSummary(st.RTTs)
	return line + fmt.Sprintf(
		"\nrtt min/avg/max = %s/%s/%s",
		FormatRTT(min),
		FormatRTT(avg),
		FormatRTT(max),
	)
}

func rttSummary(rtts []time.Duration) (min, avg, max time.Duration) {
	min, max = rtts[0], rtts[0]
	var sum time.Duration
	for _, r := range rtts {
		sum += r
		if r < min {
			min = r
		}
		if r > max {
			max = r
		}
	}
	avg = sum / time.Duration(len(rtts))
	return min, avg, max
}

func ProbeRTT(pkt *domain.Packet, now time.Time) time.Duration {
	rtt, err := icmphost.ComputeRTT(pkt.Payload, now)
	if err != nil {
		return 0
	}
	return rtt
}
