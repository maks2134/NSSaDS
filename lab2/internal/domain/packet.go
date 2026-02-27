package domain

import (
	"encoding/binary"
	"fmt"
	"time"
)

const (
	PacketTypeData     = 1
	PacketTypeAck      = 2
	PacketTypeNack     = 3
	PacketTypeSyn      = 4
	PacketTypeFin      = 5
	PacketTypeFileInfo = 6
	PacketTypeCommand  = 7
	PacketTypeResponse = 8
)

type Packet struct {
	Type      uint8
	SeqNum    uint32
	AckNum    uint32
	Data      []byte
	Checksum  uint16
	Window    uint16
	Flags     uint8
	Timestamp int64
}

type FileInfo struct {
	Name    string
	Size    int64
	ModTime time.Time
	Path    string
}

type TransferProgress struct {
	FileName    string
	TotalBytes  int64
	Transferred int64
	StartTime   time.Time
	Bitrate     float64
	Percentage  float64
	PacketsSent uint32
	PacketsLost uint32
	Retransmits uint32
}

type TransferSession struct {
	ID          string
	ClientAddr  string
	FileName    string
	FileSize    int64
	Transferred int64
	IsUpload    bool
	LastUpdate  time.Time
	FilePath    string
	WindowBase  uint32
	WindowSize  uint16
	LastAck     uint32
	BufferSize  int
}

type SlidingWindow struct {
	BaseSeq    uint32
	NextSeq    uint32
	WindowSize uint16
	Buffer     map[uint32]*Packet
	Acked      map[uint32]bool
	MaxSeq     uint32
}

func NewPacket(packetType uint8, seqNum uint32, data []byte) *Packet {
	return &Packet{
		Type:      packetType,
		SeqNum:    seqNum,
		Data:      data,
		Timestamp: time.Now().UnixNano(),
	}
}

func NewAckPacket(seqNum, ackNum uint32, window uint16) *Packet {
	return &Packet{
		Type:   PacketTypeAck,
		SeqNum: seqNum,
		AckNum: ackNum,
		Window: window,
	}
}

func NewNackPacket(seqNum uint32) *Packet {
	return &Packet{
		Type:   PacketTypeNack,
		SeqNum: seqNum,
	}
}

func (p *Packet) Serialize() []byte {
	buf := make([]byte, 23+len(p.Data))
	buf[0] = p.Type
	binary.BigEndian.PutUint32(buf[1:5], p.SeqNum)
	binary.BigEndian.PutUint32(buf[5:9], p.AckNum)
	binary.BigEndian.PutUint16(buf[9:11], p.Window)
	buf[11] = p.Flags
	binary.BigEndian.PutUint64(buf[12:20], uint64(p.Timestamp))
	binary.BigEndian.PutUint16(buf[20:22], uint16(len(p.Data)))
	copy(buf[22:], p.Data)

	p.Checksum = p.calculateChecksum(buf)
	binary.BigEndian.PutUint16(buf[21:23], p.Checksum)

	return buf
}

func DeserializePacket(data []byte) (*Packet, error) {
	if len(data) < 22 {
		return nil, fmt.Errorf("packet too short")
	}

	p := &Packet{
		Type:      data[0],
		SeqNum:    binary.BigEndian.Uint32(data[1:5]),
		AckNum:    binary.BigEndian.Uint32(data[5:9]),
		Window:    binary.BigEndian.Uint16(data[9:11]),
		Flags:     data[11],
		Timestamp: int64(binary.BigEndian.Uint64(data[12:20])),
		Checksum:  binary.BigEndian.Uint16(data[21:23]),
	}

	dataLen := binary.BigEndian.Uint16(data[20:22])
	if int(dataLen) > len(data)-22 {
		return nil, fmt.Errorf("invalid data length")
	}

	p.Data = make([]byte, dataLen)
	copy(p.Data, data[22:22+dataLen])

	if !p.verifyChecksum(data) {
		return nil, fmt.Errorf("checksum mismatch")
	}

	return p, nil
}

func (p *Packet) calculateChecksum(data []byte) uint16 {
	var sum uint32
	for i := 0; i < len(data); i += 2 {
		if i+1 < len(data) {
			sum += uint32(data[i])<<8 | uint32(data[i+1])
		} else {
			sum += uint32(data[i]) << 8
		}
	}

	for sum > 0xFFFF {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}

	return ^uint16(sum)
}

func (p *Packet) verifyChecksum(data []byte) bool {
	originalChecksum := p.Checksum
	p.Checksum = 0
	calculated := p.calculateChecksum(data)
	p.Checksum = originalChecksum

	return calculated == originalChecksum
}

func NewSlidingWindow(windowSize uint16) *SlidingWindow {
	return &SlidingWindow{
		WindowSize: windowSize,
		Buffer:     make(map[uint32]*Packet),
		Acked:      make(map[uint32]bool),
	}
}

func (sw *SlidingWindow) CanSend() bool {
	return sw.NextSeq-sw.BaseSeq < uint32(sw.WindowSize)
}

func (sw *SlidingWindow) AddPacket(packet *Packet) {
	sw.Buffer[packet.SeqNum] = packet
	sw.NextSeq++
}

func (sw *SlidingWindow) AckPacket(seqNum uint32) {
	sw.Acked[seqNum] = true

	for sw.Acked[sw.BaseSeq] {
		delete(sw.Buffer, sw.BaseSeq)
		delete(sw.Acked, sw.BaseSeq)
		sw.BaseSeq++
	}
}

func (sw *SlidingWindow) GetUnackedPackets() []*Packet {
	var packets []*Packet
	for seqNum := sw.BaseSeq; seqNum < sw.NextSeq; seqNum++ {
		if packet, exists := sw.Buffer[seqNum]; exists {
			packets = append(packets, packet)
		}
	}
	return packets
}

func (sw *SlidingWindow) GetRetransmissionPackets(timeout time.Duration) []*Packet {
	var packets []*Packet
	now := time.Now().UnixNano()

	for seqNum, packet := range sw.Buffer {
		if !sw.Acked[seqNum] && now-packet.Timestamp > timeout.Nanoseconds() {
			packets = append(packets, packet)
			packet.Timestamp = now
		}
	}

	return packets
}
