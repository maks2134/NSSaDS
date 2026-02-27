package network

import (
	"NSSaDS/lab3/internal/domain"
	"context"
	"fmt"
	"net"
	"sync"
	"syscall"
	"time"
)

type selectMultiplexer struct {
	listener     net.Listener
	clients      map[string]*domain.ClientConnection
	clientsMutex sync.RWMutex
	handler      domain.CommandHandler
	connManager  domain.ConnectionManager
	fileManager  domain.FileManager
	config       *domain.ServerConfig
	running      bool
	listenerFD   int
	selectSystem domain.SelectSystem
}

func NewSelectMultiplexer(handler domain.CommandHandler, connManager domain.ConnectionManager, fileManager domain.FileManager) domain.Multiplexer {
	return &selectMultiplexer{
		clients:      make(map[string]*domain.ClientConnection),
		handler:      handler,
		connManager:  connManager,
		fileManager:  fileManager,
		selectSystem: &UnixSelectSystem{},
	}
}

func (sm *selectMultiplexer) Start(ctx context.Context, config *domain.ServerConfig) error {
	sm.config = config
	sm.running = true

	if sm.config.SelectTimeout == 0 {
		sm.config.SelectTimeout = domain.DefaultSelectTimeout
	}
	if sm.config.ChunkSize == 0 {
		sm.config.ChunkSize = domain.DefaultChunkSize
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", config.Host, config.Port))
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	sm.listener = listener

	tcpListener := listener.(*net.TCPListener)
	file, err := tcpListener.File()
	if err != nil {
		return fmt.Errorf("failed to get listener file descriptor: %w", err)
	}
	file.Close()
	sm.listenerFD = int(file.Fd())

	fmt.Printf("Server started on %s:%d (using select() multiplexing)\n", config.Host, config.Port)
	fmt.Printf("Select timeout: %v, Default chunk size: %d\n", sm.config.SelectTimeout, sm.config.ChunkSize)

	for sm.running {
		select {
		case <-ctx.Done():
			return sm.Stop()
		default:
			if err := sm.processSelectLoop(); err != nil {
				if sm.running {
					fmt.Printf("Select loop error: %v\n", err)
				}
			}
		}
	}

	return nil
}

func (sm *selectMultiplexer) processSelectLoop() error {
	readFds := &domain.FdSet{}
	writeFds := &domain.FdSet{}
	exceptFds := &domain.FdSet{}

	sm.selectSystem.FDZero(readFds)
	sm.selectSystem.FDZero(writeFds)
	sm.selectSystem.FDZero(exceptFds)

	sm.selectSystem.FDSet(sm.listenerFD, readFds)
	maxFD := sm.listenerFD

	sm.clientsMutex.RLock()
	for _, client := range sm.clients {
		if client.IsActive && client.FD != 0 {
			fd := int(client.FD)
			sm.selectSystem.FDSet(fd, readFds)

			if client.FileTransfer != nil && client.FileTransfer.IsActive {
				sm.selectSystem.FDSet(fd, writeFds)
			}

			if fd > maxFD {
				maxFD = fd
			}
		}
	}
	sm.clientsMutex.RUnlock()

	timeout := &domain.Timeval{
		Sec:  0,
		Usec: int32(sm.config.SelectTimeout.Microseconds()),
	}

	n, err := sm.selectSystem.Select(maxFD+1, readFds, writeFds, exceptFds, timeout)
	if err != nil {
		if err == syscall.EINTR {
			return nil
		}
		return fmt.Errorf("select error: %w", err)
	}

	if n == 0 {
		sm.checkPingTimeouts()
		return nil
	}

	return sm.processReadyFDs(readFds, writeFds, exceptFds)
}

func (sm *selectMultiplexer) processReadyFDs(readFds, writeFds, exceptFds *domain.FdSet) error {
	if sm.selectSystem.FDIsSet(sm.listenerFD, readFds) {
		if err := sm.handleNewConnection(); err != nil {
			fmt.Printf("Error handling new connection: %v\n", err)
		}
	}

	sm.clientsMutex.RLock()
	clientsCopy := make(map[string]*domain.ClientConnection)
	for id, client := range sm.clients {
		clientsCopy[id] = client
	}
	sm.clientsMutex.RUnlock()

	for clientID, client := range clientsCopy {
		if !client.IsActive || client.FD == 0 {
			continue
		}

		fd := int(client.FD)

		if sm.selectSystem.FDIsSet(fd, readFds) {
			if err := sm.handleClientRead(clientID, client); err != nil {
				fmt.Printf("Error handling read from client %s: %v\n", clientID, err)
				sm.RemoveConnection(clientID)
				continue
			}
		}

		if sm.selectSystem.FDIsSet(fd, writeFds) {
			if err := sm.handleClientWrite(clientID, client); err != nil {
				fmt.Printf("Error handling write to client %s: %v\n", clientID, err)
				sm.RemoveConnection(clientID)
				continue
			}
		}
	}

	return nil
}

func (sm *selectMultiplexer) handleNewConnection() error {
	conn, err := sm.listener.Accept()
	if err != nil {
		return fmt.Errorf("accept error: %w", err)
	}

	sm.clientsMutex.RLock()
	clientCount := len(sm.clients)
	sm.clientsMutex.RUnlock()

	if clientCount >= sm.config.MaxClients {
		fmt.Printf("Max clients reached, rejecting connection\n")
		conn.Close()
		return nil
	}

	clientID := fmt.Sprintf("client_%d", time.Now().UnixNano())
	client := &domain.ClientConnection{
		ID:        clientID,
		Conn:      conn,
		LastPing:  time.Now(),
		Buffer:    make([]byte, 0, sm.config.ChunkSize),
		IsActive:  true,
		ChunkSize: sm.calculateOptimalChunkSize(sm.config.PingTimeout),
	}

	tcpConn := conn.(*net.TCPConn)
	file, err := tcpConn.File()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to get connection file descriptor: %w", err)
	}
	file.Close()
	client.FD = file.Fd()

	sm.clientsMutex.Lock()
	sm.clients[clientID] = client
	sm.clientsMutex.Unlock()

	fmt.Printf("New client connected: %s (FD: %d, chunk size: %d)\n", clientID, client.FD, client.ChunkSize)

	return nil
}

func (sm *selectMultiplexer) handleClientRead(clientID string, client *domain.ClientConnection) error {
	client.LastPing = time.Now()

	buffer := make([]byte, client.ChunkSize)
	n, err := client.Conn.Read(buffer)
	if err != nil {
		return fmt.Errorf("read error: %w", err)
	}

	if n > 0 {
		if err := sm.processClientData(clientID, buffer[:n]); err != nil {
			return fmt.Errorf("process data error: %w", err)
		}
	}

	return nil
}

func (sm *selectMultiplexer) handleClientWrite(clientID string, client *domain.ClientConnection) error {
	if client.FileTransfer != nil && client.FileTransfer.IsActive {
		return sm.handleFileTransfer(clientID, client)
	}
	return nil
}

func (sm *selectMultiplexer) processClientData(clientID string, data []byte) error {
	commandStr := string(data)
	commandStr = trimString(commandStr)

	if commandStr == "" {
		return nil
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		response, err := sm.handler.HandleCommand(ctx, commandStr, []string{})
		if err != nil {
			response = fmt.Sprintf("Error: %v", err)
		}

		sm.clientsMutex.RLock()
		client, exists := sm.clients[clientID]
		sm.clientsMutex.RUnlock()

		if exists && client.IsActive {
			client.Conn.Write([]byte(response + "\n"))
		}
	}()

	return nil
}

func (sm *selectMultiplexer) handleFileTransfer(clientID string, client *domain.ClientConnection) error {
	if client.FileTransfer != nil {
		client.FileTransfer.Transferred += int64(client.ChunkSize)

		if client.FileTransfer.Transferred >= client.FileTransfer.FileSize {
			client.FileTransfer.IsActive = false
			fmt.Printf("File transfer completed for client %s: %s\n", clientID, client.FileTransfer.FileName)
		}
	}
	return nil
}

func (sm *selectMultiplexer) calculateOptimalChunkSize(ping time.Duration) int {
	targetLatency := ping * 10

	bytesPerMs := 1024 * 1024 / 1000
	chunkSize := int(targetLatency.Milliseconds()) * bytesPerMs

	if chunkSize < domain.MinChunkSize {
		chunkSize = domain.MinChunkSize
	}
	if chunkSize > domain.MaxChunkSize {
		chunkSize = domain.MaxChunkSize
	}

	return chunkSize
}

func (sm *selectMultiplexer) checkPingTimeouts() {
	now := time.Now()
	sm.clientsMutex.RLock()
	clientsCopy := make(map[string]*domain.ClientConnection)
	for id, client := range sm.clients {
		clientsCopy[id] = client
	}
	sm.clientsMutex.RUnlock()

	for clientID, client := range clientsCopy {
		if now.Sub(client.LastPing) > sm.config.PingTimeout {
			fmt.Printf("Client %s ping timeout, disconnecting\n", clientID)
			sm.RemoveConnection(clientID)
		}
	}
}

func (sm *selectMultiplexer) Stop() error {
	sm.running = false

	if sm.listener != nil {
		sm.listener.Close()
	}

	sm.clientsMutex.Lock()
	for clientID := range sm.clients {
		if client, exists := sm.clients[clientID]; exists {
			client.Conn.Close()
		}
	}
	sm.clientsMutex.Unlock()

	fmt.Println("Select multiplexer stopped")
	return nil
}

func (sm *selectMultiplexer) AddConnection(conn net.Conn) error {
	return sm.handleNewConnection()
}

func (sm *selectMultiplexer) RemoveConnection(clientID string) error {
	sm.clientsMutex.Lock()
	defer sm.clientsMutex.Unlock()

	client, exists := sm.clients[clientID]
	if !exists {
		return nil
	}

	client.Conn.Close()
	delete(sm.clients, clientID)

	fmt.Printf("Client disconnected: %s\n", clientID)
	return nil
}

func (sm *selectMultiplexer) ProcessConnections() error {
	return sm.processSelectLoop()
}

func (sm *selectMultiplexer) SetHandler(handler domain.CommandHandler) {
	sm.handler = handler
}

func (sm *selectMultiplexer) CalculateOptimalChunkSize(ping time.Duration) int {
	return sm.calculateOptimalChunkSize(ping)
}

func trimString(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == ' ') {
		s = s[:len(s)-1]
	}
	for len(s) > 0 && (s[0] == '\n' || s[0] == '\r' || s[0] == ' ') {
		s = s[1:]
	}
	return s
}
