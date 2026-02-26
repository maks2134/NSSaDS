package domain

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"syscall"
	"time"
)

// Multiplexer interface for different I/O multiplexing methods
type Multiplexer interface {
	AddConnection(conn net.Conn) error
	RemoveConnection(conn net.Conn) error
	Wait(ctx context.Context) ([]ReadyEvent, error)
	Close() error
	GetConnectionCount() int
	SetChunkSize(size int)
}

// ReadyEvent represents a connection that's ready for I/O
type ReadyEvent struct {
	Connection net.Conn
	EventType  EventType // Read, Write, or Accept
	Error      error
}

type EventType int

const (
	EventRead   EventType = iota
	EventWrite  EventType = iota
	EventAccept EventType = iota
	EventError  EventType = iota
)

// Connection represents a client connection with metadata
type Connection struct {
	Conn         net.Conn
	LastActive   time.Time
	Buffer       []byte
	WriteBuffer  []byte
	IsWriting    bool
	ClientID     string
	BytesRead    int64
	BytesWritten int64
	ChunkSize    int
	Fd           int
}

// Config for multiplexer tuning
type MuxConfig struct {
	MaxConnections     int
	ChunkSize          int
	InteractiveTimeout time.Duration // ping * 10
	SelectTimeout      time.Duration
	BufferSize         int
	PingTime           time.Duration
}

func NewMuxConfig() *MuxConfig {
	return &MuxConfig{
		MaxConnections:     1000,
		ChunkSize:          512,                    // Small for interactive response (ping * 10 = 5ms for 512 bytes)
		InteractiveTimeout: 100 * time.Millisecond, // ping * 10ms
		SelectTimeout:      10 * time.Millisecond,
		BufferSize:         8192,
		PingTime:           10 * time.Millisecond,
	}
}

// Statistics for monitoring multiplexer performance
type MuxStats struct {
	TotalConnections    int64
	ActiveConnections   int64
	BytesRead           int64
	BytesWritten        int64
	EventsProcessed     int64
	SelectCalls         int64
	AverageSelectTime   time.Duration
	MaxSelectTime       time.Duration
	InteractiveCommands int64
	FileTransfers       int64
	ChunkSize           int
}

func NewMuxStats() *MuxStats {
	return &MuxStats{
		MaxSelectTime: 0,
		ChunkSize:     512,
	}
}

// FileTransfer represents an active file transfer
type FileTransfer struct {
	ID          string
	ClientID    string
	FileName    string
	FileSize    int64
	Transferred int64
	IsUpload    bool
	StartTime   time.Time
	LastUpdate  time.Time
	ChunkSize   int
}

// Command represents a server command with execution context
type Command interface {
	Execute(ctx context.Context, args []string, conn *Connection) (string, error)
	Name() string
	IsInteractive() bool
	GetChunkSize() int
}

// PollFd for cross-platform compatibility
type PollFd struct {
	Fd      int
	Events  int16
	Revents int16
}

// EpollEvent for cross-platform compatibility
type EpollEvent struct {
	Events uint32
	Fd     int32
	Pad    int32
}

// Select-based multiplexer
type SelectMultiplexer struct {
	connections map[int]*Connection
	maxFd       int
	config      *MuxConfig
	stats       *MuxStats
	listener    net.Listener
	ctx         context.Context
	cancel      context.CancelFunc
}

// Poll-based multiplexer (for Linux)
type PollMultiplexer struct {
	connections map[int]*Connection
	pollFds     []PollFd
	config      *MuxConfig
	stats       *MuxStats
	listener    net.Listener
	ctx         context.Context
	cancel      context.CancelFunc
}

// Epoll-based multiplexer (for Linux, high performance)
type EpollMultiplexer struct {
	connections map[int]*Connection
	epollFd     int
	events      []EpollEvent
	config      *MuxConfig
	stats       *MuxStats
	listener    net.Listener
	ctx         context.Context
	cancel      context.CancelFunc
}

// Platform-specific multiplexer creation
func NewMultiplexer(muxType string, config *MuxConfig) Multiplexer {
	switch muxType {
	case "select":
		return NewSelectMultiplexer(config)
	case "poll":
		if runtime.GOOS == "linux" {
			return NewPollMultiplexer(config)
		}
		fallthrough
	case "epoll":
		if runtime.GOOS == "linux" {
			return NewEpollMultiplexer(config)
		}
		fallthrough
	default:
		// Default to select for cross-platform compatibility
		return NewSelectMultiplexer(config)
	}
}

// Get optimal multiplexer type for current platform
func GetOptimalMuxType() string {
	switch runtime.GOOS {
	case "linux":
		return "epoll" // Best performance on Linux
	case "windows":
		return "select" // Windows has good select support
	case "darwin", "freebsd", "netbsd", "openbsd":
		return "select" // BSD systems use kqueue, but select is portable
	default:
		return "select" // Safe default
	}
}

// Calculate optimal chunk size based on ping time
func CalculateOptimalChunkSize(pingTime time.Duration, bandwidth int64) int {
	// Formula: chunk_size = (ping_time * bandwidth) / 8
	// Ensure chunk is small enough for interactive response
	maxInteractiveChunk := int64(pingTime) * bandwidth / 8 / int64(time.Millisecond)

	if maxInteractiveChunk > 512 {
		return 512 // Cap at 512 bytes for interactive use
	}

	return int(maxInteractiveChunk)
}

// Platform-specific syscall wrappers
func SetNonBlocking(fd int) error {
	return syscall.SetNonblock(fd, true)
}

func GetFd(conn net.Conn) (int, error) {
	var fd int
	var err error

	switch c := conn.(type) {
	case *net.TCPConn:
		file, e := c.File()
		if e != nil {
			return 0, e
		}
		fd = int(file.Fd())
		defer file.Close()
	case *net.UDPConn:
		file, e := c.File()
		if e != nil {
			return 0, e
		}
		fd = int(file.Fd())
		defer file.Close()
	default:
		// Try to get file descriptor through reflection for other types
		return 0, fmt.Errorf("unsupported connection type: %T", conn)
	}

	return fd, nil
}
