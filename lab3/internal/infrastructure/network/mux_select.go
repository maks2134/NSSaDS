package network

import (
	"NSSaDS/lab3/internal/domain"
	"context"
	"fmt"
	"net"
	"syscall"
	"time"
)

type SelectMultiplexer struct {
	connections map[int]*domain.Connection
	maxFd       int
	config      *domain.MuxConfig
	stats       *domain.MuxStats
	listener    net.Listener
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewSelectMultiplexer(config *domain.MuxConfig) *SelectMultiplexer {
	ctx, cancel := context.WithCancel(context.Background())

	return &SelectMultiplexer{
		connections: make(map[int]*domain.Connection),
		config:      config,
		stats:       domain.NewMuxStats(),
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (sm *SelectMultiplexer) AddConnection(conn net.Conn) error {
	fd, err := domain.GetFd(conn)
	if err != nil {
		return fmt.Errorf("failed to get file descriptor: %w", err)
	}

	// Set non-blocking mode
	if err := domain.SetNonBlocking(fd); err != nil {
		return fmt.Errorf("failed to set non-blocking: %w", err)
	}

	connection := &domain.Connection{
		Conn:        conn,
		LastActive:  time.Now(),
		Buffer:      make([]byte, sm.config.BufferSize),
		WriteBuffer: make([]byte, 0),
		ChunkSize:   sm.config.ChunkSize,
		Fd:          fd,
		ClientID:    fmt.Sprintf("conn_%d", fd),
	}

	sm.connections[fd] = connection
	sm.stats.TotalConnections++
	sm.stats.ActiveConnections++

	if fd > sm.maxFd {
		sm.maxFd = fd
	}

	return nil
}

func (sm *SelectMultiplexer) RemoveConnection(conn net.Conn) error {
	fd, err := domain.GetFd(conn)
	if err != nil {
		return err
	}

	if connection, exists := sm.connections[fd]; exists {
		conn.Close()
		delete(sm.connections, fd)
		sm.stats.ActiveConnections--

		// Recalculate maxFd
		sm.maxFd = 0
		for fd := range sm.connections {
			if fd > sm.maxFd {
				sm.maxFd = fd
			}
		}

		_ = connection // Suppress unused warning
	}

	return nil
}

func (sm *SelectMultiplexer) Wait(ctx context.Context) ([]domain.ReadyEvent, error) {
	start := time.Now()
	sm.stats.SelectCalls++

	// Build fd sets for select
	readFds := make([]syscall.FdSet, 1)
	writeFds := make([]syscall.FdSet, 1)

	syscall.FD_ZERO(&readFds[0])
	syscall.FD_ZERO(&writeFds[0])

	// Add listener fd for accepting new connections
	if sm.listener != nil {
		listenerFd, err := domain.GetFd(sm.listener)
		if err == nil {
			syscall.FD_SET(listenerFd, &readFds[0])
			if listenerFd > sm.maxFd {
				sm.maxFd = listenerFd
			}
		}
	}

	// Add all connection fds
	for fd, conn := range sm.connections {
		if !conn.IsWriting {
			syscall.FD_SET(fd, &readFds[0])
		} else if len(conn.WriteBuffer) > 0 {
			syscall.FD_SET(fd, &writeFds[0])
		}
	}

	// Set timeout
	timeout := &syscall.Timeval{
		Sec:  int(sm.config.SelectTimeout.Seconds()),
		Usec: int(sm.config.SelectTimeout.Microseconds()) % 1000000,
	}

	// Call select
	n, err := syscall.Select(sm.maxFd+1, &readFds[0], &writeFds[0], nil, timeout)
	if err != nil {
		return nil, fmt.Errorf("select failed: %w", err)
	}

	// Update statistics
	selectTime := time.Since(start)
	sm.stats.EventsProcessed += int64(n)
	if selectTime > sm.stats.MaxSelectTime {
		sm.stats.MaxSelectTime = selectTime
	}

	// Calculate average select time
	totalCalls := sm.stats.SelectCalls
	if totalCalls > 0 {
		sm.stats.AverageSelectTime = time.Duration(
			(int64(sm.stats.AverageSelectTime)*totalCalls + int64(selectTime)) / (totalCalls + 1),
		)
	}

	var events []domain.ReadyEvent

	// Check for new connections
	if sm.listener != nil {
		listenerFd, err := domain.GetFdOrZero(sm.listener)
		if err == nil && syscall.FD_ISSET(listenerFd, &readFds[0]) {
			conn, err := sm.listener.Accept()
			if err != nil {
				events = append(events, domain.ReadyEvent{
					EventType: domain.EventError,
					Error:     err,
				})
			} else {
				events = append(events, domain.ReadyEvent{
					Connection: conn,
					EventType:  domain.EventAccept,
				})
			}
		}
	}

	// Check for ready connections
	for fd, conn := range sm.connections {
		if syscall.FD_ISSET(fd, &readFds[0]) && !conn.IsWriting {
			events = append(events, domain.ReadyEvent{
				Connection: conn.Conn,
				EventType:  domain.EventRead,
			})
			conn.LastActive = time.Now()
		}

		if syscall.FD_ISSET(fd, &writeFds[0]) && conn.IsWriting && len(conn.WriteBuffer) > 0 {
			events = append(events, domain.ReadyEvent{
				Connection: conn.Conn,
				EventType:  domain.EventWrite,
			})
		}
	}

	return events, nil
}

func (sm *SelectMultiplexer) Close() error {
	sm.cancel()

	// Close all connections
	for _, conn := range sm.connections {
		if conn.Conn != nil {
			conn.Conn.Close()
		}
	}

	// Close listener
	if sm.listener != nil {
		sm.listener.Close()
	}

	return nil
}

func (sm *SelectMultiplexer) GetConnectionCount() int {
	return len(sm.connections)
}

func (sm *SelectMultiplexer) SetChunkSize(size int) {
	sm.config.ChunkSize = size
	sm.stats.ChunkSize = size

	// Update all connections with new chunk size
	for _, conn := range sm.connections {
		conn.ChunkSize = size
	}
}

func (sm *SelectMultiplexer) GetStats() *domain.MuxStats {
	return sm.stats
}

// Helper function to get fd or return 0 for select
func getFdOrZero(conn interface{}) int {
	fd, err := domain.GetFd(conn.(net.Conn))
	if err != nil {
		return 0
	}
	return fd
}
