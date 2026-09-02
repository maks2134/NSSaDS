package icmp

import (
	"encoding/binary"
	"net"
	"os"
)

func BuildIPv4(src, dst net.IP, payload []byte, ttl int) ([]byte, error) {
	src4 := src.To4()
	dst4 := dst.To4()
	if src4 == nil || dst4 == nil {
		return nil, ErrNotIPv4
	}
	if ttl <= 0 {
		ttl = 64
	}

	total := minIPv4Header + len(payload)
	hdr := make([]byte, total)
	hdr[0] = 0x45
	binary.BigEndian.PutUint16(hdr[2:4], uint16(total))
	binary.BigEndian.PutUint16(hdr[4:6], uint16(os.Getpid()&0xffff))
	hdr[8] = byte(ttl)
	hdr[9] = ianaProtocolICMP
	copy(hdr[12:16], src4)
	copy(hdr[16:20], dst4)
	binary.BigEndian.PutUint16(hdr[10:12], IPChecksum(hdr[:minIPv4Header]))
	copy(hdr[minIPv4Header:], payload)
	return hdr, nil
}

func IPChecksum(header []byte) uint16 {
	var sum uint32
	for i := 0; i+1 < len(header); i += 2 {
		sum += uint32(header[i])<<8 | uint32(header[i+1])
	}
	if len(header)%2 == 1 {
		sum += uint32(header[len(header)-1]) << 8
	}
	for sum > 0xffff {
		sum = (sum & 0xffff) + (sum >> 16)
	}
	return ^uint16(sum)
}
