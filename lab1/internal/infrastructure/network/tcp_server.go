package network

import (
	"NSSaDS/internal/domain"
	"NSSaDS/pkg/config"
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

type TCPServer struct {
	config   *config.ServerConfig
	listener net.Listener
	handler  domain.CommandHandler
	connMgr  domain.ConnectionManager
}

func NewTCPServer(cfg *config.ServerConfig, handler domain.CommandHandler, connMgr domain.ConnectionManager) *TCPServer {
	return &TCPServer{
		config:  cfg,
		handler: handler,
		connMgr: connMgr,
	}
}

func (s *TCPServer) Start(ctx context.Context, addr string) error {
	var err error
	s.listener, err = net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	fmt.Printf("Server started on %s\n", addr)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return nil
				default:
					return fmt.Errorf("accept error: %w", err)
				}
			}

			go s.connMgr.HandleConnection(ctx, conn)
		}
	}
}

func (s *TCPServer) Stop() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

func (s *TCPServer) SetHandler(handler domain.CommandHandler) {
	s.handler = handler
}

type TCPConnectionManager struct {
	config  *config.ServerConfig
	fileMgr domain.FileManager
	handler domain.CommandHandler
}

func NewTCPConnectionManager(cfg *config.ServerConfig, fileMgr domain.FileManager) *TCPConnectionManager {
	return &TCPConnectionManager{
		config:  cfg,
		fileMgr: fileMgr,
	}
}

func (cm *TCPConnectionManager) SetCommandHandler(handler domain.CommandHandler) {
	cm.handler = handler
}

func (cm *TCPConnectionManager) HandleConnection(ctx context.Context, conn net.Conn) error {
	defer conn.Close()

	clientAddr := conn.RemoteAddr().String()
	fmt.Printf("Client connected: %s\n", clientAddr)

	if err := cm.SetKeepAlive(conn); err != nil {
		fmt.Printf("Warning: failed to set keepalive: %v\n", err)
	}

	buffer := make([]byte, cm.config.BufferSize)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			conn.SetReadDeadline(time.Now().Add(cm.config.SessionTimeout))

			n, readErr := conn.Read(buffer)
			if readErr != nil {
				if netErr, ok := readErr.(net.Error); ok && netErr.Timeout() {
					fmt.Printf("Client %s timeout\n", clientAddr)
					return nil
				}
				if readErr.Error() != "EOF" {
					fmt.Printf("Read error from %s: %v\n", clientAddr, readErr)
				}
				return nil
			}

			data := string(buffer[:n])
			data = strings.TrimRight(data, "\r\n")

			if data == "" {
				continue
			}

			parts := strings.Fields(data)
			if len(parts) == 0 {
				continue
			}

			cmd := strings.ToUpper(parts[0])
			args := parts[1:]

			var response string
			var err error

			switch cmd {
			case "UPLOAD", "DOWNLOAD":
				response, err = cm.handleCommand(ctx, cmd, args, conn, clientAddr)
			default:
				if cm.handler != nil {
					response, err = cm.handler.HandleCommand(ctx, cmd, args)
				} else {
					response, err = "", fmt.Errorf("command handler not set")
				}
			}

			if err != nil {
				response = fmt.Sprintf("ERROR: %v", err)
			}

			response += "\r\n"
			_, writeErr := conn.Write([]byte(response))
			if writeErr != nil {
				fmt.Printf("Write error to %s: %v\n", clientAddr, writeErr)
				return nil
			}

			if cmd == "CLOSE" || cmd == "EXIT" || cmd == "QUIT" {
				fmt.Printf("Client %s disconnected\n", clientAddr)
				return nil
			}
		}
	}
}

func (cm *TCPConnectionManager) SetKeepAlive(conn net.Conn) error {
	return setKeepAlive(conn, cm.config.KeepAlive, cm.config.KeepAliveIdle, cm.config.KeepAliveCount, cm.config.KeepAliveIntvl)
}

func (cm *TCPConnectionManager) handleCommand(ctx context.Context, cmd string, args []string, conn net.Conn, clientAddr string) (string, error) {
	switch cmd {
	case "UPLOAD":
		return cm.handleUpload(ctx, args, conn, clientAddr)
	case "DOWNLOAD":
		return cm.handleDownload(ctx, args, conn, clientAddr)
	case "ECHO", "TIME", "CLOSE", "EXIT", "QUIT":
		return "", fmt.Errorf("basic commands should be handled by command handler")
	default:
		return "", fmt.Errorf("unknown command: %s", cmd)
	}
}

func (cm *TCPConnectionManager) handleUpload(ctx context.Context, args []string, conn net.Conn, clientAddr string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("usage: UPLOAD <filename>")
	}

	filename := args[0]

	session := &domain.TransferSession{
		ID:         fmt.Sprintf("%s_%s_%d", clientAddr, filename, time.Now().Unix()),
		ClientAddr: clientAddr,
		FileName:   filename,
		IsUpload:   true,
		LastUpdate: time.Now(),
		FilePath:   cm.config.UploadDir + "/" + filename,
	}

	if err := cm.fileMgr.CreateTransferSession(session); err != nil {
		return "", fmt.Errorf("failed to create transfer session: %w", err)
	}

	response := fmt.Sprintf("READY_TO_RECEIVE %s", filename)
	_, err := conn.Write([]byte(response + "\r\n"))
	if err != nil {
		return "", fmt.Errorf("failed to send ready response: %w", err)
	}

	return cm.receiveFile(ctx, conn, session)
}

func (cm *TCPConnectionManager) handleDownload(ctx context.Context, args []string, conn net.Conn, clientAddr string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("usage: DOWNLOAD <filename>")
	}

	filename := args[0]

	fileInfo, err := cm.fileMgr.GetFileInfo(filename)
	if err != nil {
		return "", fmt.Errorf("file not found: %s", filename)
	}

	session := &domain.TransferSession{
		ID:         fmt.Sprintf("%s_%s_%d", clientAddr, filename, time.Now().Unix()),
		ClientAddr: clientAddr,
		FileName:   filename,
		FileSize:   fileInfo.Size,
		IsUpload:   false,
		LastUpdate: time.Now(),
		FilePath:   fileInfo.Path,
	}

	if err := cm.fileMgr.CreateTransferSession(session); err != nil {
		return "", fmt.Errorf("failed to create transfer session: %w", err)
	}

	return cm.sendFile(ctx, conn, session)
}

func (cm *TCPConnectionManager) receiveFile(ctx context.Context, conn net.Conn, session *domain.TransferSession) (string, error) {
	buffer := make([]byte, cm.config.BufferSize)
	var totalBytes int64
	startTime := time.Now()

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return "", fmt.Errorf("file receive error: %w", err)
		}

		if err := cm.fileMgr.SaveFile(session.FileName, buffer[:n], totalBytes); err != nil {
			return "", fmt.Errorf("failed to save file: %w", err)
		}

		totalBytes += int64(n)
		session.Transferred = totalBytes
		session.LastUpdate = time.Now()

		if err := cm.fileMgr.UpdateTransferSession(session); err != nil {
			fmt.Printf("Warning: failed to update session: %v\n", err)
		}

		percentage := float64(totalBytes) / float64(session.FileSize) * 100
		bitrate := float64(totalBytes) / time.Since(startTime).Seconds() / 1024 / 1024

		fmt.Printf("Upload progress: %s - %.2f%% (%.2f MB/s)\n", session.FileName, percentage, bitrate)
	}

	duration := time.Since(startTime)
	avgBitrate := float64(totalBytes) / duration.Seconds() / 1024 / 1024

	return fmt.Sprintf("File uploaded successfully: %s (%.2f MB, %.2f MB/s)",
		session.FileName, float64(totalBytes)/1024/1024, avgBitrate), nil
}

func (cm *TCPConnectionManager) sendFile(ctx context.Context, conn net.Conn, session *domain.TransferSession) (string, error) {
	fileData, err := cm.fileMgr.ReadFile(session.FileName)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	header := fmt.Sprintf("FILE_INFO %s %d", session.FileName, len(fileData))
	_, err = conn.Write([]byte(header + "\r\n"))
	if err != nil {
		return "", fmt.Errorf("failed to send file header: %w", err)
	}

	totalBytes := int64(0)
	startTime := time.Now()

	for i := 0; i < len(fileData); i += cm.config.BufferSize {
		end := i + cm.config.BufferSize
		if end > len(fileData) {
			end = len(fileData)
		}

		n, err := conn.Write(fileData[i:end])
		if err != nil {
			return "", fmt.Errorf("file send error: %w", err)
		}

		totalBytes += int64(n)
		session.Transferred = totalBytes
		session.LastUpdate = time.Now()

		if err := cm.fileMgr.UpdateTransferSession(session); err != nil {
			fmt.Printf("Warning: failed to update session: %v\n", err)
		}

		percentage := float64(totalBytes) / float64(len(fileData)) * 100
		bitrate := float64(totalBytes) / time.Since(startTime).Seconds() / 1024 / 1024

		fmt.Printf("Download progress: %s - %.2f%% (%.2f MB/s)\n", session.FileName, percentage, bitrate)
	}

	duration := time.Since(startTime)
	avgBitrate := float64(totalBytes) / duration.Seconds() / 1024 / 1024

	return fmt.Sprintf("File downloaded successfully: %s (%.2f MB, %.2f MB/s)",
		session.FileName, float64(totalBytes)/1024/1024, avgBitrate), nil
}
