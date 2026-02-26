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

type UDPServer struct {
	config      *config.ServerConfig
	udpConfig   *config.UDPConfig
	conn        *net.UDPConn
	handler     domain.CommandHandler
	connMgr     domain.UDPConnectionManager
	relMgr      *ReliabilityManager
	fileMgr     domain.FileManager
	perfMonitor *PerformanceMonitor
	sessions    map[string]*domain.TransferSession
	sessionsMu  sync.RWMutex
}

func NewUDPServer(cfg *config.ServerConfig, udpCfg *config.UDPConfig, handler domain.CommandHandler,
	fileMgr domain.FileManager) *UDPServer {

	return &UDPServer{
		config:    cfg,
		udpConfig: udpCfg,
		handler:   handler,
		fileMgr:   fileMgr,
		sessions:  make(map[string]*domain.TransferSession),
	}
}

func (s *UDPServer) Start(ctx context.Context, addr string) error {
	var err error
	packetConn, err := net.ListenPacket("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to start UDP server: %w", err)
	}

	var ok bool
	s.conn, ok = packetConn.(*net.UDPConn)
	if !ok {
		return fmt.Errorf("failed to get UDP connection")
	}

	s.relMgr = NewReliabilityManager(s.conn, s.udpConfig.PacketTimeout,
		s.udpConfig.RetransmissionTimeout, s.udpConfig.MaxRetransmissions)

	s.connMgr = NewUDPConnectionManager(s.conn, s.relMgr, s.udpConfig)
	s.perfMonitor = NewPerformanceMonitor()

	fmt.Printf("UDP Server started on %s\n", addr)

	// Start cleanup goroutine
	go s.cleanupRoutine(ctx)

	for {
		select {
		case <-ctx.Done():
			return s.Stop()
		default:
			packet, clientAddr, err := s.relMgr.ReceivePacket()
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				fmt.Printf("Error receiving packet: %v\n", err)
				continue
			}

			go s.handlePacket(ctx, packet, clientAddr)
		}
	}
}

func (s *UDPServer) Stop() error {
	if s.conn != nil {
		s.conn.Close()
	}
	if s.relMgr != nil {
		s.relMgr.Stop()
	}
	return nil
}

func (s *UDPServer) SetHandler(handler domain.CommandHandler) {
	s.handler = handler
}

func (s *UDPServer) handlePacket(ctx context.Context, packet *domain.Packet, clientAddr *net.UDPAddr) {
	switch packet.Type {
	case domain.PacketTypeCommand:
		s.handleCommand(ctx, packet, clientAddr)
	case domain.PacketTypeData:
		s.handleDataPacket(ctx, packet, clientAddr)
	case domain.PacketTypeAck, domain.PacketTypeNack:
		// Handled by reliability manager
	case domain.PacketTypeSyn:
		s.handleSynPacket(ctx, packet, clientAddr)
	case domain.PacketTypeFin:
		s.handleFinPacket(ctx, packet, clientAddr)
	}
}

func (s *UDPServer) handleCommand(ctx context.Context, packet *domain.Packet, clientAddr *net.UDPAddr) {
	cmd := string(packet.Data)
	args := []string{}

	// Parse command (simplified)
	if len(cmd) > 0 {
		parts := parseCommand(cmd)
		if len(parts) > 0 {
			cmd = parts[0]
			args = parts[1:]
		}
	}

	response, err := s.handler.HandleCommand(ctx, cmd, args, clientAddr)
	if err != nil {
		response = fmt.Sprintf("ERROR: %v", err)
	}

	responsePacket := domain.NewPacket(domain.PacketTypeResponse, packet.SeqNum+1, []byte(response))
	if err := s.relMgr.SendPacket(responsePacket, clientAddr); err != nil {
		fmt.Printf("Failed to send response: %v\n", err)
	}
}

func (s *UDPServer) handleDataPacket(ctx context.Context, packet *domain.Packet, clientAddr *net.UDPAddr) {
	sessionID := fmt.Sprintf("%s_%d", clientAddr.String(), packet.SeqNum)

	s.sessionsMu.RLock()
	session, exists := s.sessions[sessionID]
	s.sessionsMu.RUnlock()

	if !exists {
		// Start new transfer session
		session = &domain.TransferSession{
			ID:          sessionID,
			ClientAddr:  clientAddr.String(),
			Transferred: 0,
			LastUpdate:  time.Now(),
			WindowSize:  s.udpConfig.WindowSize,
			BufferSize:  s.udpConfig.BufferSizes[len(s.udpConfig.BufferSizes)/2], // Use middle buffer size
		}

		s.sessionsMu.Lock()
		s.sessions[sessionID] = session
		s.sessionsMu.Unlock()
	}

	// Save data
	if err := s.fileMgr.SaveFile(session.FileName, packet.Data, int64(packet.SeqNum)); err != nil {
		fmt.Printf("Failed to save data: %v\n", err)
		return
	}

	session.Transferred += int64(len(packet.Data))
	session.LastUpdate = time.Now()

	// Send ACK
	ackPacket := domain.NewAckPacket(packet.SeqNum, packet.SeqNum+1, s.udpConfig.WindowSize)
	if err := s.relMgr.SendPacket(ackPacket, clientAddr); err != nil {
		fmt.Printf("Failed to send ACK: %v\n", err)
	}

	// Update progress
	s.perfMonitor.UpdateProgress(session.Transferred)
}

func (s *UDPServer) handleSynPacket(ctx context.Context, packet *domain.Packet, clientAddr *net.UDPAddr) {
	// Handle connection initiation
	synAck := domain.NewPacket(domain.PacketTypeAck, packet.SeqNum+1, []byte("SYN-ACK"))
	s.relMgr.SendPacket(synAck, clientAddr)
}

func (s *UDPServer) handleFinPacket(ctx context.Context, packet *domain.Packet, clientAddr *net.UDPAddr) {
	// Handle connection termination
	sessionID := fmt.Sprintf("%s_%d", clientAddr.String(), packet.SeqNum)

	s.sessionsMu.Lock()
	delete(s.sessions, sessionID)
	s.sessionsMu.Unlock()

	finAck := domain.NewPacket(domain.PacketTypeAck, packet.SeqNum+1, []byte("FIN-ACK"))
	s.relMgr.SendPacket(finAck, clientAddr)
}

func (s *UDPServer) cleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.cleanupExpiredSessions()
		}
	}
}

func (s *UDPServer) cleanupExpiredSessions() {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()

	now := time.Now()
	for id, session := range s.sessions {
		if now.Sub(session.LastUpdate) > s.config.SessionTimeout {
			delete(s.sessions, id)
		}
	}
}

func parseCommand(cmd string) []string {
	// Simple command parsing - split by spaces
	parts := []string{}
	current := ""

	for _, char := range cmd {
		if char == ' ' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}
