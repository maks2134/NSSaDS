package domain

import (
	"context"
	"net"
	"time"
)

type Server interface {
	Start(ctx context.Context, config *ServerConfig) error
	Stop() error
	SetHandler(handler CommandHandler)
}

type Client interface {
	Connect(ctx context.Context, addr string) error
	Disconnect() error
	SendCommand(cmd string, args []string) (string, error)
	UploadFile(localPath, remoteName string) (*TransferProgress, error)
	DownloadFile(remoteName, localPath string) (*TransferProgress, error)
}

type ConnectionManager interface {
	HandleConnection(ctx context.Context, conn net.Conn) error
	SetKeepAlive(conn net.Conn) error
	GetClient(clientID string) (*ClientConnection, error)
	AddClient(client *ClientConnection) error
	RemoveClient(clientID string) error
	GetAllClients() map[string]*ClientConnection
}

type FileManager interface {
	SaveFile(filename string, data []byte, offset int64) error
	ReadFile(filename string) ([]byte, error)
	GetFileInfo(filename string) (*FileInfo, error)
	DeleteFile(filename string) error
	CreateTransferSession(session *TransferSession) error
	GetTransferSession(clientAddr, filename string) (*TransferSession, error)
	UpdateTransferSession(session *TransferSession) error
	DeleteTransferSession(sessionID string) error
	CleanupExpiredSessions() error
}

type Multiplexer interface {
	Start(ctx context.Context, config *ServerConfig) error
	Stop() error
	AddConnection(conn net.Conn) error
	RemoveConnection(clientID string) error
	ProcessConnections() error
	CalculateOptimalChunkSize(ping time.Duration) int
	SetHandler(handler CommandHandler)
}

type SelectSystem interface {
	Select(nfd int, readFds, writeFds, exceptFds *FdSet, timeout *Timeval) (int, error)
	FDSet(fd int, set *FdSet)
	FDIsSet(fd int, set *FdSet) bool
	FDZero(set *FdSet)
}

type FdSet struct {
	Bits [32]int32
}

type Timeval struct {
	Sec  int64
	Usec int32
}
