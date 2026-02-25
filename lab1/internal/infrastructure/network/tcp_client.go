package network

import (
	"NSSaDS/internal/domain"
	"NSSaDS/pkg/config"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type TCPClient struct {
	config  *config.ClientConfig
	conn    net.Conn
	fileMgr domain.FileManager
}

func NewTCPClient(cfg *config.ClientConfig, fileMgr domain.FileManager) *TCPClient {
	return &TCPClient{
		config:  cfg,
		fileMgr: fileMgr,
	}
}

func (c *TCPClient) Connect(ctx context.Context, addr string) error {
	var err error
	c.conn, err = net.DialTimeout("tcp", addr, c.config.Timeout)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	if err := c.SetKeepAlive(); err != nil {
		fmt.Printf("Warning: failed to set keepalive: %v\n", err)
	}

	fmt.Printf("Connected to server: %s\n", addr)
	return nil
}

func (c *TCPClient) Disconnect() error {
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

func (c *TCPClient) SendCommand(cmd string, args []string) (string, error) {
	if c.conn == nil {
		return "", fmt.Errorf("not connected to server")
	}

	command := cmd
	if len(args) > 0 {
		command += " " + strings.Join(args, " ")
	}
	command += "\r\n"

	_, err := c.conn.Write([]byte(command))
	if err != nil {
		return "", fmt.Errorf("failed to send command: %w", err)
	}

	response, err := c.readResponse()
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return response, nil
}

func (c *TCPClient) UploadFile(localPath, remoteName string) (*domain.TransferProgress, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("not connected to server")
	}

	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	response, err := c.SendCommand("UPLOAD", []string{remoteName})
	if err != nil {
		return nil, fmt.Errorf("failed to send upload command: %w", err)
	}

	if !strings.HasPrefix(response, "READY_TO_RECEIVE") {
		return nil, fmt.Errorf("server not ready to receive file: %s", response)
	}

	return c.sendFile(localPath, fileInfo.Size())
}

func (c *TCPClient) DownloadFile(remoteName, localPath string) (*domain.TransferProgress, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("not connected to server")
	}

	response, err := c.SendCommand("DOWNLOAD", []string{remoteName})
	if err != nil {
		return nil, fmt.Errorf("failed to send download command: %w", err)
	}

	if strings.HasPrefix(response, "ERROR") {
		return nil, fmt.Errorf("server error: %s", response)
	}

	if !strings.HasPrefix(response, "FILE_INFO") {
		return nil, fmt.Errorf("unexpected response: %s", response)
	}

	parts := strings.Fields(response)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid file info response: %s", response)
	}

	fileSize, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid file size: %w", err)
	}

	return c.receiveFile(localPath, fileSize)
}

func (c *TCPClient) SetKeepAlive() error {
	if c.conn == nil {
		return fmt.Errorf("no connection")
	}

	return setKeepAlive(c.conn, c.config.KeepAlive, c.config.KeepAliveIdle, c.config.KeepAliveCount, c.config.KeepAliveIntvl)
}

func (c *TCPClient) readResponse() (string, error) {
	c.conn.SetReadDeadline(time.Now().Add(c.config.Timeout))

	buffer := make([]byte, c.config.BufferSize)
	var response strings.Builder

	for {
		n, err := c.conn.Read(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return "", fmt.Errorf("read timeout")
			}
			return "", fmt.Errorf("read error: %w", err)
		}

		data := string(buffer[:n])
		response.WriteString(data)

		if strings.Contains(data, "\r\n") {
			break
		}
	}

	return strings.TrimRight(response.String(), "\r\n"), nil
}

func (c *TCPClient) sendFile(localPath string, fileSize int64) (*domain.TransferProgress, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	buffer := make([]byte, c.config.BufferSize)
	var totalBytes int64
	startTime := time.Now()

	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("file read error: %w", err)
		}

		if n == 0 {
			break
		}

		_, err = c.conn.Write(buffer[:n])
		if err != nil {
			return nil, fmt.Errorf("network write error: %w", err)
		}

		totalBytes += int64(n)

		percentage := float64(totalBytes) / float64(fileSize) * 100
		bitrate := float64(totalBytes) / time.Since(startTime).Seconds() / 1024 / 1024

		fmt.Printf("Upload progress: %.2f%% (%.2f MB/s)\n", percentage, bitrate)
	}

	duration := time.Since(startTime)
	avgBitrate := float64(totalBytes) / duration.Seconds() / 1024 / 1024

	progress := &domain.TransferProgress{
		FileName:    localPath,
		TotalBytes:  fileSize,
		Transferred: totalBytes,
		StartTime:   startTime,
		Bitrate:     avgBitrate,
		Percentage:  100.0,
	}

	return progress, nil
}

func (c *TCPClient) receiveFile(localPath string, fileSize int64) (*domain.TransferProgress, error) {
	file, err := os.Create(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	buffer := make([]byte, c.config.BufferSize)
	var totalBytes int64
	startTime := time.Now()

	for totalBytes < fileSize {
		remaining := fileSize - totalBytes
		if remaining < int64(len(buffer)) {
			buffer = make([]byte, remaining)
		}

		n, err := c.conn.Read(buffer)
		if err != nil {
			return nil, fmt.Errorf("network read error: %w", err)
		}

		_, err = file.Write(buffer[:n])
		if err != nil {
			return nil, fmt.Errorf("file write error: %w", err)
		}

		totalBytes += int64(n)

		percentage := float64(totalBytes) / float64(fileSize) * 100
		bitrate := float64(totalBytes) / time.Since(startTime).Seconds() / 1024 / 1024

		fmt.Printf("Download progress: %.2f%% (%.2f MB/s)\n", percentage, bitrate)
	}

	duration := time.Since(startTime)
	avgBitrate := float64(totalBytes) / duration.Seconds() / 1024 / 1024

	progress := &domain.TransferProgress{
		FileName:    localPath,
		TotalBytes:  fileSize,
		Transferred: totalBytes,
		StartTime:   startTime,
		Bitrate:     avgBitrate,
		Percentage:  100.0,
	}

	return progress, nil
}
