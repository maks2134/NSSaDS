package network

import (
	"NSSaDS/lab2/internal/domain"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"
)

type ReliabilityManager struct {
	conn                  *net.UDPConn
	packetsSent           uint32
	packetsLost           uint32
	retransmits           uint32
	pendingPackets        map[uint32]*domain.Packet
	pendingMutex          sync.RWMutex
	packetTimeout         time.Duration
	maxRetransmissions    int
	retransmissionTimeout time.Duration
	stopChan              chan struct{}
	wg                    sync.WaitGroup
}

func NewReliabilityManager(conn *net.UDPConn, packetTimeout, retransmissionTimeout time.Duration, maxRetransmissions int) *ReliabilityManager {
	rm := &ReliabilityManager{
		conn:                  conn,
		pendingPackets:        make(map[uint32]*domain.Packet),
		packetTimeout:         packetTimeout,
		maxRetransmissions:    maxRetransmissions,
		retransmissionTimeout: retransmissionTimeout,
		stopChan:              make(chan struct{}),
	}

	rm.wg.Add(1)
	go rm.retransmissionLoop()

	return rm
}

func (rm *ReliabilityManager) SendPacket(packet *domain.Packet, addr *net.UDPAddr) error {
	rm.pendingMutex.Lock()
	rm.pendingPackets[packet.SeqNum] = packet
	rm.packetsSent++
	rm.pendingMutex.Unlock()

	data := packet.Serialize()
	_, err := rm.conn.WriteToUDP(data, addr)
	if err != nil {
		return fmt.Errorf("failed to send packet: %w", err)
	}

	return nil
}

func (rm *ReliabilityManager) ReceivePacket() (*domain.Packet, *net.UDPAddr, error) {
	buf := make([]byte, 65536)
	n, addr, err := rm.conn.ReadFromUDP(buf)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read packet: %w", err)
	}

	packet, err := domain.DeserializePacket(buf[:n])
	if err != nil {
		return nil, nil, fmt.Errorf("failed to deserialize packet: %w", err)
	}

	if packet.Type == domain.PacketTypeAck {
		rm.pendingMutex.Lock()
		delete(rm.pendingPackets, packet.AckNum)
		rm.pendingMutex.Unlock()
	}

	return packet, addr, nil
}

func (rm *ReliabilityManager) HandleRetransmissions() {
	rm.retransmissionLoop()
}

func (rm *ReliabilityManager) GetStatistics() (packetsSent, packetsLost, retransmits uint32) {
	rm.pendingMutex.RLock()
	defer rm.pendingMutex.RUnlock()

	return rm.packetsSent, rm.packetsLost, rm.retransmits
}

func (rm *ReliabilityManager) retransmissionLoop() {
	defer rm.wg.Done()

	ticker := time.NewTicker(rm.retransmissionTimeout / 2)
	defer ticker.Stop()

	for {
		select {
		case <-rm.stopChan:
			return
		case <-ticker.C:
			rm.checkRetransmissions()
		}
	}
}

func (rm *ReliabilityManager) checkRetransmissions() {
	rm.pendingMutex.Lock()
	defer rm.pendingMutex.Unlock()

	now := time.Now().UnixNano()

	for seqNum, packet := range rm.pendingPackets {
		elapsed := time.Duration(now - packet.Timestamp)

		if elapsed > rm.retransmissionTimeout {
			retransmitCount := rm.getRetransmitCount(packet)
			if retransmitCount >= rm.maxRetransmissions {
				delete(rm.pendingPackets, seqNum)
				rm.packetsLost++
				continue
			}

			data := packet.Serialize()
			if _, err := rm.conn.WriteToUDP(data, nil); err != nil {
				fmt.Printf("Retransmission failed: %v\n", err)
			} else {
				rm.retransmits++
				packet.Timestamp = now
			}
		}
	}
}

func (rm *ReliabilityManager) getRetransmitCount(packet *domain.Packet) int {
	return 0
}

func (rm *ReliabilityManager) Stop() {
	close(rm.stopChan)
	rm.wg.Wait()
}

func (rm *ReliabilityManager) SimulatePacketLoss(lossRate float64) {
	rand.Seed(time.Now().UnixNano())

	if rand.Float64() < lossRate {
		rm.packetsLost++
	}
}
