package icmp

import (
	"encoding/binary"
	"net"
	"testing"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"

	"NSSaDS/lab5/internal/domain"
)

func TestEncodeDecodeTimestamp(t *testing.T) {
	t.Parallel()
	sent := time.Unix(1_700_000_000, 123_456_789)
	got, err := DecodeTimestamp(EncodeTimestamp(sent))
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(sent) {
		t.Fatalf("got %v want %v", got, sent)
	}
}

func TestComputeRTT(t *testing.T) {
	t.Parallel()
	sent := time.Now().Add(-25 * time.Millisecond)
	rtt, err := ComputeRTT(EncodeTimestamp(sent), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if rtt < 20*time.Millisecond || rtt > 80*time.Millisecond {
		t.Fatalf("rtt out of range: %v", rtt)
	}
}

func TestComputeRTTShortPayload(t *testing.T) {
	t.Parallel()
	_, err := ComputeRTT([]byte{1, 2, 3}, time.Now())
	if err != ErrShortTimestamp {
		t.Fatalf("got %v want %v", err, ErrShortTimestamp)
	}
}

func TestParseEchoReply(t *testing.T) {
	t.Parallel()
	sent := time.Now()
	body := mustEcho(t, ipv4.ICMPTypeEchoReply, 42, 7, BuildPayload(sent, 16))
	src := net.IPv4(8, 8, 8, 8)
	pkt, err := ParsePacket(wrapIPv4(t, src, body, 117), src)
	if err != nil {
		t.Fatal(err)
	}
	if pkt.Kind != domain.KindEchoReply {
		t.Fatalf("kind %v", pkt.Kind)
	}
	if pkt.ID != 42 || pkt.Seq != 7 {
		t.Fatalf("id/seq %d/%d", pkt.ID, pkt.Seq)
	}
	if pkt.TTL != 117 {
		t.Fatalf("ttl %d", pkt.TTL)
	}
}

func TestParseTimeExceeded(t *testing.T) {
	t.Parallel()
	orig, err := BuildEchoRequest(9, 3, time.Now(), 16)
	if err != nil {
		t.Fatal(err)
	}
	inner, err := BuildIPv4(net.IPv4(10, 0, 0, 1), net.IPv4(8, 8, 8, 8), orig, 1)
	if err != nil {
		t.Fatal(err)
	}
	msg := icmp.Message{
		Type: ipv4.ICMPTypeTimeExceeded,
		Code: 0,
		Body: &icmp.TimeExceeded{Data: inner},
	}
	body, err := msg.Marshal(nil)
	if err != nil {
		t.Fatal(err)
	}
	hop := net.IPv4(192, 168, 1, 1)
	pkt, err := ParsePacket(wrapIPv4(t, hop, body, 64), hop)
	if err != nil {
		t.Fatal(err)
	}
	if pkt.Kind != domain.KindTimeExceeded {
		t.Fatalf("kind %v", pkt.Kind)
	}
	if pkt.OrigID != 9 || pkt.OrigSeq != 3 {
		t.Fatalf("orig %d/%d", pkt.OrigID, pkt.OrigSeq)
	}
}

func TestClassifyReply(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		typ      icmp.Type
		expected domain.ReplyKind
	}{
		{name: "echo", typ: ipv4.ICMPTypeEchoReply, expected: domain.KindEchoReply},
		{name: "ttl", typ: ipv4.ICMPTypeTimeExceeded, expected: domain.KindTimeExceeded},
		{name: "unreach", typ: ipv4.ICMPTypeDestinationUnreachable, expected: domain.KindUnreachable},
		{name: "other", typ: ipv4.ICMPTypeEcho, expected: domain.KindUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := ClassifyReply(tc.typ); got != tc.expected {
				t.Fatalf("got %v want %v", got, tc.expected)
			}
		})
	}
}

func TestMatchesWorker(t *testing.T) {
	t.Parallel()
	echo := &domain.Packet{Kind: domain.KindEchoReply, ID: 4, Seq: 2}
	ttl := &domain.Packet{Kind: domain.KindTimeExceeded, OrigID: 4, OrigSeq: 2}
	if !MatchesWorker(echo, 4, 2) {
		t.Fatal("echo should match")
	}
	if MatchesWorker(echo, 4, 3) {
		t.Fatal("wrong seq")
	}
	if !MatchesWorker(ttl, 4, 2) {
		t.Fatal("ttl should match orig")
	}
	if MatchesWorker(&domain.Packet{Kind: domain.KindUnknown, ID: 4, Seq: 2}, 4, 2) {
		t.Fatal("unknown should not match")
	}
}

func TestIPChecksumCompleteHeader(t *testing.T) {
	t.Parallel()
	hdr := make([]byte, 20)
	hdr[0] = 0x45
	binary.BigEndian.PutUint16(hdr[2:4], 20)
	hdr[8] = 64
	hdr[9] = 1
	copy(hdr[12:16], []byte{192, 168, 1, 1})
	copy(hdr[16:20], []byte{192, 168, 1, 255})
	cs := IPChecksum(hdr)
	binary.BigEndian.PutUint16(hdr[10:12], cs)
	if IPChecksum(hdr) != 0 {
		t.Fatal("checksum of complete header must be 0")
	}
}

func wrapIPv4(t *testing.T, src net.IP, payload []byte, ttl int) []byte {
	t.Helper()
	pkt, err := BuildIPv4(src, net.IPv4(8, 8, 8, 8), payload, ttl)
	if err != nil {
		t.Fatal(err)
	}
	return pkt
}

func mustEcho(t *testing.T, typ ipv4.ICMPType, id, seq int, data []byte) []byte {
	t.Helper()
	msg := icmp.Message{
		Type: typ,
		Code: 0,
		Body: &icmp.Echo{ID: id, Seq: seq, Data: data},
	}
	body, err := msg.Marshal(nil)
	if err != nil {
		t.Fatal(err)
	}
	return body
}
