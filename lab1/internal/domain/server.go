package domain

import (
	"context"
	"net"
)

type Server interface {
	Start(ctx context.Context, addr string) error
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
