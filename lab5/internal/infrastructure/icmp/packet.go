package icmp

import (
	"encoding/binary"
	"errors"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"

	"NSSaDS/lab5/internal/domain"
)

const (
	ianaProtocolICMP = 1
	minICMPHeader    = 8
	minIPv4Header    = 20
)

var (
	ErrShortPacket = errors.New("packet too short")
	ErrNotIPv4     = errors.New("not an ipv4 address")
)

func BuildEchoRequest(id, seq int, sent time.Time, payloadSize int) ([]byte, error) {
	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   id,
			Seq:  seq,
			Data: BuildPayload(sent, payloadSize),
		},
	}
	return msg.Marshal(nil)
}

func ParsePacket(buf []byte, from net.IP) (*domain.Packet, error) {
	body, ttl, src, err := extractICMP(buf)
	if err != nil {
		return nil, err
	}
	if src == nil {
		src = cloneIP(from)
	}

	msg, err := icmp.ParseMessage(ianaProtocolICMP, body)
	if err != nil {
		return nil, err
	}

	pkt := &domain.Packet{
		Kind: ClassifyReply(msg.Type),
		From: src,
		TTL:  ttl,
		Code: msg.Code,
		Size: len(buf),
	}
	if err := fillPacketBody(pkt, msg); err != nil {
		return nil, err
	}
	return pkt, nil
}

func ClassifyReply(typ icmp.Type) domain.ReplyKind {
	switch typ {
	case ipv4.ICMPTypeEchoReply:
		return domain.KindEchoReply
	case ipv4.ICMPTypeTimeExceeded:
		return domain.KindTimeExceeded
	case ipv4.ICMPTypeDestinationUnreachable:
		return domain.KindUnreachable
	default:
		return domain.KindUnknown
	}
}

func fillPacketBody(pkt *domain.Packet, msg *icmp.Message) error {
	switch body := msg.Body.(type) {
	case *icmp.Echo:
		pkt.ID = body.ID
		pkt.Seq = body.Seq
		pkt.Payload = body.Data
		return nil
	case *icmp.TimeExceeded:
		return fillInnerEcho(pkt, body.Data)
	case *icmp.DstUnreach:
		return fillInnerEcho(pkt, body.Data)
	default:
		return nil
	}
}

func fillInnerEcho(pkt *domain.Packet, data []byte) error {
	inner, _, _, err := extractICMP(data)
	if err != nil {
		return err
	}
	if len(inner) < minICMPHeader {
		return ErrShortPacket
	}
	pkt.OrigID = int(binary.BigEndian.Uint16(inner[4:6]))
	pkt.OrigSeq = int(binary.BigEndian.Uint16(inner[6:8]))
	if len(inner) > minICMPHeader {
		pkt.Payload = inner[minICMPHeader:]
	}
	return nil
}

func extractICMP(buf []byte) (body []byte, ttl int, src net.IP, err error) {
	if len(buf) < minICMPHeader {
		return nil, 0, nil, ErrShortPacket
	}
	if buf[0]>>4 != 4 {
		return buf, 0, nil, nil
	}
	ihl := int(buf[0]&0x0f) * 4
	if ihl < minIPv4Header || len(buf) < ihl+minICMPHeader {
		return nil, 0, nil, ErrShortPacket
	}
	ttl = int(buf[8])
	src = net.IPv4(buf[12], buf[13], buf[14], buf[15])
	return buf[ihl:], ttl, src, nil
}

func cloneIP(ip net.IP) net.IP {
	if ip == nil {
		return nil
	}
	return append(net.IP(nil), ip...)
}

func PacketSequence(pkt *domain.Packet) int {
	if pkt.Kind == domain.KindEchoReply {
		return pkt.Seq
	}
	return pkt.OrigSeq
}
