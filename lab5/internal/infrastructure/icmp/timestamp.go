package icmp

import (
	"encoding/binary"
	"errors"
	"time"
)

var ErrShortTimestamp = errors.New("payload shorter than timestamp")

func EncodeTimestamp(t time.Time) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(t.UnixNano()))
	return buf
}

func DecodeTimestamp(payload []byte) (time.Time, error) {
	if len(payload) < 8 {
		return time.Time{}, ErrShortTimestamp
	}
	ns := int64(binary.BigEndian.Uint64(payload[:8]))
	return time.Unix(0, ns), nil
}

func ComputeRTT(payload []byte, now time.Time) (time.Duration, error) {
	sent, err := DecodeTimestamp(payload)
	if err != nil {
		return 0, err
	}
	rtt := now.Sub(sent)
	if rtt < 0 {
		return 0, nil
	}
	return rtt, nil
}

func BuildPayload(sent time.Time, size int) []byte {
	if size < 8 {
		size = 8
	}
	data := make([]byte, size)
	copy(data, EncodeTimestamp(sent))
	for i := 8; i < len(data); i++ {
		data[i] = byte(i)
	}
	return data
}
