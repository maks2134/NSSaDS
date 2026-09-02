package domain

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	MagicSize  = 4
	HeaderSize = 12
	MaxPayload = 1400
)

var Magic = [4]byte{'N', 'S', '6', 0x01}

type MsgType uint8

const (
	MsgHello    MsgType = 1
	MsgBye      MsgType = 2
	MsgChat     MsgType = 3
	MsgIgnore   MsgType = 4
	MsgUnignore MsgType = 5
)

type Mode uint8

const (
	ModeBroadcast Mode = 1
	ModeMulticast Mode = 2
)

type Message struct {
	Type    MsgType
	Mode    Mode
	NodeID  uint32
	Payload []byte
}

func Encode(msg Message) ([]byte, error) {
	if len(msg.Payload) > MaxPayload {
		return nil, fmt.Errorf("payload too large: %d", len(msg.Payload))
	}
	buf := make([]byte, HeaderSize+len(msg.Payload))
	copy(buf[:MagicSize], Magic[:])
	buf[4] = byte(msg.Type)
	buf[5] = byte(msg.Mode)
	binary.BigEndian.PutUint32(buf[6:10], msg.NodeID)
	binary.BigEndian.PutUint16(buf[10:12], uint16(len(msg.Payload)))
	copy(buf[HeaderSize:], msg.Payload)
	return buf, nil
}

func Decode(data []byte) (Message, error) {
	if len(data) < HeaderSize {
		return Message{}, errors.New("packet too short")
	}
	if data[0] != Magic[0] || data[1] != Magic[1] || data[2] != Magic[2] || data[3] != Magic[3] {
		return Message{}, errors.New("invalid magic")
	}
	payloadLen := int(binary.BigEndian.Uint16(data[10:12]))
	if HeaderSize+payloadLen > len(data) {
		return Message{}, errors.New("invalid payload length")
	}
	payload := make([]byte, payloadLen)
	copy(payload, data[HeaderSize:HeaderSize+payloadLen])
	return Message{
		Type:    MsgType(data[4]),
		Mode:    Mode(data[5]),
		NodeID:  binary.BigEndian.Uint32(data[6:10]),
		Payload: payload,
	}, nil
}

func ModeName(m Mode) string {
	switch m {
	case ModeBroadcast:
		return "broadcast"
	case ModeMulticast:
		return "multicast"
	default:
		return "unknown"
	}
}

func ParseMode(name string) (Mode, error) {
	switch name {
	case "broadcast", "bcast":
		return ModeBroadcast, nil
	case "multicast", "mcast":
		return ModeMulticast, nil
	default:
		return 0, fmt.Errorf("unknown mode %q", name)
	}
}
