package network

import (
	"NSSaDS/lab2/internal/domain"
	"NSSaDS/lab2/pkg/config"
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

type UDPConnectionManager struct {
	conn      *net.UDPConn
	relMgr    *ReliabilityManager
	udpConfig *config.UDPConfig
	window    *domain.SlidingWindow
	timeout   time.Duration
	clients   map[string]*ClientSession
	clientsMu sync.RWMutex
}

type ClientSession struct {
	Addr     *net.UDPAddr
	LastSeen time.Time
	Window   *domain.SlidingWindow
	SeqNum   uint32
	AckNum   uint32
}

func NewUDPConnectionManager(conn *net.UDPConn, relMgr *ReliabilityManager, udpConfig *config.UDPConfig) *UDPConnectionManager {
	return &UDPConnectionManager{
		conn:      conn,
		relMgr:    relMgr,
		udpConfig: udpConfig,
		window:    domain.NewSlidingWindow(udpConfig.WindowSize),
		timeout:   udpConfig.PacketTimeout,
		clients:   make(map[string]*ClientSession),
	}
}

func (ucm *UDPConnectionManager) HandleConnection(ctx context.Context, conn *net.UDPConn, clientAddr *net.UDPAddr) error {
	// This is handled by the UDP server's packet routing
	return nil
}

func (ucm *UDPConnectionManager) SetPacketTimeout(timeout time.Duration) {
	ucm.timeout = timeout
}

func (ucm *UDPConnectionManager) SetWindowSize(size uint16) {
	ucm.udpConfig.WindowSize = size
	ucm.window.WindowSize = size
}

func (ucm *UDPConnectionManager) GetOrCreateClient(addr *net.UDPAddr) *ClientSession {
	ucm.clientsMu.Lock()
	defer ucm.clientsMu.Unlock()

	clientKey := addr.String()
	session, exists := ucm.clients[clientKey]
	if !exists {
		session = &ClientSession{
			Addr:     addr,
			LastSeen: time.Now(),
			Window:   domain.NewSlidingWindow(ucm.udpConfig.WindowSize),
			SeqNum:   0,
			AckNum:   0,
		}
		ucm.clients[clientKey] = session
	}

	session.LastSeen = time.Now()
	return session
}

func (ucm *UDPConnectionManager) SendReliablePacket(packet *domain.Packet, addr *net.UDPAddr) error {
	session := ucm.GetOrCreateClient(addr)

	// Add to sliding window
	if session.Window.CanSend() {
		session.Window.AddPacket(packet)
		return ucm.relMgr.SendPacket(packet, addr)
	}

	return fmt.Errorf("sliding window full")
}

func (ucm *UDPConnectionManager) HandleAckPacket(packet *domain.Packet, addr *net.UDPAddr) {
	session := ucm.GetOrCreateClient(addr)
	session.Window.AckPacket(packet.AckNum)
	session.AckNum = packet.AckNum
}

func (ucm *UDPConnectionManager) CleanupExpiredClients() {
	ucm.clientsMu.Lock()
	defer ucm.clientsMu.Unlock()

	now := time.Now()
	for key, session := range ucm.clients {
		if now.Sub(session.LastSeen) > 5*time.Minute {
			delete(ucm.clients, key)
		}
	}
}

func (ucm *UDPConnectionManager) GetClientStatistics(addr *net.UDPAddr) (uint32, uint32, uint16) {
	ucm.clientsMu.RLock()
	defer ucm.clientsMu.RUnlock()

	session := ucm.clients[addr.String()]
	if session == nil {
		return 0, 0, 0
	}

	return session.SeqNum, session.AckNum, session.Window.WindowSize
}
