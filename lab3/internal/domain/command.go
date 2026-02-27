package domain

import (
	"context"
	"net"
	"time"
)

type Command interface {
	Execute(ctx context.Context, args []string) (string, error)
	Name() string
}

type CommandHandler interface {
	HandleCommand(ctx context.Context, cmd string, args []string) (string, error)
	RegisterCommand(command Command)
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
	IsActive    bool
}

type ClientConnection struct {
	ID           string
	Conn         net.Conn
	LastPing     time.Time
	Buffer       []byte
	IsActive     bool
	ChunkSize    int
	FileTransfer *TransferSession
	FD           uintptr
}

type ServerConfig struct {
	Host          string
	Port          int
	MaxClients    int
	PingTimeout   time.Duration
	ChunkSize     int
	SelectTimeout time.Duration
}

type SelectResult struct {
	ReadyRead   []int
	ReadyWrite  []int
	ReadyExcept []int
}

const (
	DefaultChunkSize     = 1024
	DefaultPingTimeout   = 30 * time.Second
	DefaultSelectTimeout = 10 * time.Millisecond
	MaxChunkSize         = 8192
	MinChunkSize         = 512
)
