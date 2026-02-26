package domain

import (
	"context"
	"net"
	"time"
)

type UDPServer interface {
	Start(ctx context.Context, addr string) error
	Stop() error
	SetHandler(handler CommandHandler)
}

type UDPClient interface {
	Connect(ctx context.Context, addr string) error
	Disconnect() error
	SendCommand(cmd string, args []string) (string, error)
	UploadFile(localPath, remoteName string) (*TransferProgress, error)
	DownloadFile(remoteName, localPath string) (*TransferProgress, error)
}

type UDPConnectionManager interface {
	HandleConnection(ctx context.Context, conn *net.UDPConn, clientAddr *net.UDPAddr) error
	SetPacketTimeout(timeout time.Duration)
	SetWindowSize(size uint16)
}

type CommandHandler interface {
	HandleCommand(ctx context.Context, cmd string, args []string, clientAddr *net.UDPAddr) (string, error)
	RegisterCommand(command Command)
}

type Command interface {
	Execute(ctx context.Context, args []string, clientAddr *net.UDPAddr) (string, error)
	Name() string
}

type FileManager interface {
	SaveFile(filename string, data []byte, offset int64) error
	ReadFile(filename string) ([]byte, error)
	GetFileInfo(filename string) (*FileInfo, error)
	DeleteFile(filename string) error
	CreateTransferSession(session *TransferSession) error
	GetTransferSession(clientAddr string, filename string) (*TransferSession, error)
	UpdateTransferSession(session *TransferSession) error
	DeleteTransferSession(sessionID string) error
	CleanupExpiredSessions() error
}

type ReliabilityManager interface {
	SendPacket(packet *Packet, addr *net.UDPAddr) error
	ReceivePacket() (*Packet, *net.UDPAddr, error)
	HandleRetransmissions()
	GetStatistics() (packetsSent, packetsLost, retransmits uint32)
}

type PerformanceMonitor interface {
	StartTransfer(filename string, totalBytes int64)
	UpdateProgress(transferred int64)
	GetProgress() *TransferProgress
	CalculateOptimalBufferSize() (int, float64)
	CompareWithTCP(tcpBitrate float64) (float64, bool)
}
