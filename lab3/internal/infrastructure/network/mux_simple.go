package network

import (
	"NSSaDS/lab3/internal/domain"
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// Simple cross-platform multiplexer using goroutines and channels
type SimpleMultiplexer struct {
	connections map[string]*domain.Connection
	config      *domain.MuxConfig
	stats       *domain.MuxStats
	listener    net.Listener
	ctx         context.Context
	cancel      context.CancelFunc
	eventChan   chan domain.ReadyEvent
	mu          sync.RWMutex
}

func NewSimpleMultiplexer(config *domain.MuxConfig) *SimpleMultiplexer {
	ctx, cancel := context.WithCancel(context.Background())

	return &SimpleMultiplexer{
		connections: make(map[string]*domain.Connection),
		config:      config,
		stats:       domain.NewMuxStats(),
		listener:    nil,
		ctx:         ctx,
		cancel:      cancel,
		eventChan:   make(chan domain.ReadyEvent, 1000),
	}
}

func (sm *SimpleMultiplexer) AddConnection(conn net.Conn) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	connection := &domain.Connection{
		Conn:        conn,
		LastActive:  time.Now(),
		Buffer:      make([]byte, sm.config.BufferSize),
		WriteBuffer: make([]byte, 0),
		ChunkSize:   sm.config.ChunkSize,
		ClientID:    fmt.Sprintf("conn_%d", time.Now().UnixNano()),
	}

	sm.connections[connection.ClientID] = connection
	sm.stats.TotalConnections++
	sm.stats.ActiveConnections++

	return nil
}

func (sm *SimpleMultiplexer) RemoveConnection(conn net.Conn) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Find connection by comparing Conn references
	for id, connection := range sm.connections {
		if connection.Conn == conn {
			conn.Close()
			delete(sm.connections, id)
			sm.stats.ActiveConnections--
			break
		}
	}

	return nil
}

func (sm *SimpleMultiplexer) Wait(ctx context.Context) ([]domain.ReadyEvent, error) {
	sm.mu.Lock()
	listener := sm.listener
	sm.mu.Unlock()

	if listener == nil {
		return nil, fmt.Errorf("listener not set")
	}

	// Start accept goroutine
	go func() {
		for {
			select {
			case <-sm.ctx.Done():
				return
			default:
				conn, err := listener.Accept()
				if err != nil {
					if !isTimeoutError(err) {
						sm.eventChan <- domain.ReadyEvent{
							EventType: domain.EventError,
							Error:     err,
						}
					}
					time.Sleep(100 * time.Millisecond)
					continue
				}

				sm.eventChan <- domain.ReadyEvent{
					Connection: conn,
					EventType:  domain.EventAccept,
				}
			}
		}
	}()

	// Start connection handler goroutines
	for _, connection := range sm.connections {
		go sm.handleConnection(connection)
	}

	var events []domain.ReadyEvent

	for {
		select {
		case <-ctx.Done():
			return events, nil
		case event := <-sm.eventChan:
			events = append(events, event)
			sm.stats.EventsProcessed++
		}
	}
}

func (sm *SimpleMultiplexer) Close() error {
	sm.cancel()

	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Close all connections
	for _, connection := range sm.connections {
		if connection.Conn != nil {
			connection.Conn.Close()
		}
	}

	// Close listener
	if sm.listener != nil {
		sm.listener.Close()
	}

	close(sm.eventChan)
	return nil
}

func (sm *SimpleMultiplexer) GetConnectionCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.connections)
}

func (sm *SimpleMultiplexer) SetChunkSize(size int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.config.ChunkSize = size
	sm.stats.ChunkSize = size

	// Update all connections with new chunk size
	for _, connection := range sm.connections {
		connection.ChunkSize = size
	}
}

func (sm *SimpleMultiplexer) SetListener(listener net.Listener) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.listener = listener
}

func (sm *SimpleMultiplexer) GetStats() *domain.MuxStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.stats
}

func (sm *SimpleMultiplexer) handleConnection(connection *domain.Connection) {
	for {
		select {
		case <-sm.ctx.Done():
			return
		default:
			// Check for read events
			if !connection.IsWriting {
				connection.Conn.SetReadDeadline(time.Now().Add(sm.config.InteractiveTimeout))
				buffer := make([]byte, connection.ChunkSize)
				n, err := connection.Conn.Read(buffer)
				if err != nil {
					if !isTimeoutError(err) && err.Error() != "EOF" {
						// Handle error
						time.Sleep(sm.config.SelectTimeout)
						continue
					}
				} else if n > 0 {
					connection.BytesRead += int64(n)
					sm.stats.BytesRead += int64(n)
					connection.LastActive = time.Now()

					// Check if this is an interactive command
					if isInteractiveCommand(buffer[:n]) {
						// Calculate response time
						responseTime := time.Since(connection.LastActive)
						maxResponseTime := sm.config.PingTime * 10

						if responseTime > maxResponseTime {
							fmt.Printf("Warning: Interactive command response time %v exceeds limit %v\n",
								responseTime, maxResponseTime)
						}
					}
				}
			}

			// Check for write events
			if connection.IsWriting && len(connection.WriteBuffer) > 0 {
				connection.Conn.SetWriteDeadline(time.Now().Add(sm.config.SelectTimeout))
				n, err := connection.Conn.Write(connection.WriteBuffer)
				if err != nil && !isTimeoutError(err) {
					// Handle error
					time.Sleep(sm.config.SelectTimeout)
					continue
				} else if n > 0 {
					connection.BytesWritten += int64(n)
					sm.stats.BytesWritten += int64(n)
					connection.WriteBuffer = connection.WriteBuffer[:0]
					connection.IsWriting = false
					connection.LastActive = time.Now()
				}
			}
		}
	}
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}

	return false
}

func isInteractiveCommand(data []byte) bool {
	// Simple heuristic: if it looks like a command, it's interactive
	commands := []string{"ECHO", "TIME", "HELP", "STATUS"}
	dataStr := string(data)

	for _, cmd := range commands {
		if len(dataStr) >= len(cmd) && dataStr[:len(cmd)] == cmd {
			return true
		}
	}

	return false
}
