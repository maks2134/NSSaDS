package network

import (
	"NSSaDS/lab2/internal/domain"
	"NSSaDS/lab2/pkg/config"
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type UDPClient struct {
	config      *config.ClientConfig
	udpConfig   *config.UDPConfig
	conn        *net.UDPConn
	serverAddr  *net.UDPAddr
	relMgr      *ReliabilityManager
	connMgr     *UDPConnectionManager
	fileMgr     domain.FileManager
	perfMonitor *PerformanceMonitor
	connected   bool
}

func NewUDPClient(cfg *config.ClientConfig, udpCfg *config.UDPConfig, fileMgr domain.FileManager) *UDPClient {
	return &UDPClient{
		config:      cfg,
		udpConfig:   udpCfg,
		fileMgr:     fileMgr,
		perfMonitor: NewPerformanceMonitor(),
	}
}

func (c *UDPClient) Connect(ctx context.Context, addr string) error {
	var err error
	c.conn, err = net.DialUDP("udp", nil, nil)
	if err != nil {
		return fmt.Errorf("failed to create UDP connection: %w", err)
	}

	c.serverAddr, err = net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to resolve server address: %w", err)
	}

	c.relMgr = NewReliabilityManager(c.conn, c.udpConfig.PacketTimeout,
		c.udpConfig.RetransmissionTimeout, c.udpConfig.MaxRetransmissions)

	c.connMgr = NewUDPConnectionManager(c.conn, c.relMgr, c.udpConfig)

	c.connected = true
	fmt.Printf("Connected to UDP server: %s\n", addr)

	return nil
}

func (c *UDPClient) Disconnect() error {
	if c.conn != nil {
		c.connected = false
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

func (c *UDPClient) SendCommand(cmd string, args []string) (string, error) {
	if !c.connected {
		return "", fmt.Errorf("not connected to server")
	}

	command := cmd
	if len(args) > 0 {
		command += " " + strings.Join(args, " ")
	}

	packet := domain.NewPacket(domain.PacketTypeCommand, 0, []byte(command))
	if err := c.relMgr.SendPacket(packet, c.serverAddr); err != nil {
		return "", fmt.Errorf("failed to send command: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("command timeout")
		default:
			responsePacket, _, err := c.relMgr.ReceivePacket()
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				return "", fmt.Errorf("failed to receive response: %w", err)
			}

			if responsePacket.Type == domain.PacketTypeResponse {
				return string(responsePacket.Data), nil
			}
		}
	}
}

func (c *UDPClient) UploadFile(localPath, remoteName string) (*domain.TransferProgress, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected to server")
	}

	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	c.perfMonitor.StartTransfer(localPath, fileInfo.Size())

	cmd := fmt.Sprintf("UPLOAD %s", remoteName)
	response, err := c.SendCommand(cmd, []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to send upload command: %w", err)
	}

	if !strings.Contains(response, "READY") {
		return nil, fmt.Errorf("server not ready: %s", response)
	}

	return c.sendFile(localPath, fileInfo.Size())
}

func (c *UDPClient) DownloadFile(remoteName, localPath string) (*domain.TransferProgress, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected to server")
	}

	cmd := fmt.Sprintf("DOWNLOAD %s", remoteName)
	response, err := c.SendCommand(cmd, []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to send download command: %w", err)
	}

	if strings.HasPrefix(response, "ERROR") {
		return nil, fmt.Errorf("server error: %s", response)
	}

	parts := strings.Fields(response)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid file info response: %s", response)
	}

	fileSize, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid file size: %w", err)
	}

	c.perfMonitor.StartTransfer(remoteName, fileSize)

	return c.receiveFile(localPath, fileSize)
}

func (c *UDPClient) sendFile(localPath string, fileSize int64) (*domain.TransferProgress, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	buffer := make([]byte, c.udpConfig.BufferSizes[len(c.udpConfig.BufferSizes)/2])
	var totalBytes int64
	seqNum := uint32(0)

	for {
		n, err := file.Read(buffer)
		if err != nil && err.Error() != "EOF" {
			return nil, fmt.Errorf("file read error: %w", err)
		}

		if n == 0 {
			break
		}

		packet := domain.NewPacket(domain.PacketTypeData, seqNum, buffer[:n])

		if err := c.connMgr.SendReliablePacket(packet, c.serverAddr); err != nil {
			return nil, fmt.Errorf("failed to send data packet: %w", err)
		}

		totalBytes += int64(n)
		seqNum++

		c.perfMonitor.UpdateProgress(totalBytes)

		c.testBufferSizes(totalBytes, fileSize)
	}

	progress := c.perfMonitor.GetProgress()
	return progress, nil
}

func (c *UDPClient) receiveFile(localPath string, fileSize int64) (*domain.TransferProgress, error) {
	file, err := os.Create(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	var totalBytes int64
	expectedSeq := uint32(0)

	for totalBytes < fileSize {
		packet, _, err := c.relMgr.ReceivePacket()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return nil, fmt.Errorf("failed to receive packet: %w", err)
		}

		if packet.Type == domain.PacketTypeData && packet.SeqNum == expectedSeq {
			_, err = file.Write(packet.Data)
			if err != nil {
				return nil, fmt.Errorf("failed to write file: %w", err)
			}

			totalBytes += int64(len(packet.Data))
			expectedSeq++

			ackPacket := domain.NewAckPacket(packet.SeqNum, expectedSeq, c.udpConfig.WindowSize)
			c.relMgr.SendPacket(ackPacket, c.serverAddr)

			c.perfMonitor.UpdateProgress(totalBytes)
		}
	}

	progress := c.perfMonitor.GetProgress()
	return progress, nil
}

func (c *UDPClient) testBufferSizes(transferred, total int64) {
	if transferred%int64(total/10) == 0 {
		for _, bufferSize := range c.udpConfig.BufferSizes {
			start := time.Now()

			testData := make([]byte, bufferSize)
			packet := domain.NewPacket(domain.PacketTypeData, 0, testData)

			if err := c.relMgr.SendPacket(packet, c.serverAddr); err == nil {
				elapsed := time.Since(start).Seconds()
				if elapsed > 0 {
					bitrate := float64(bufferSize) / elapsed / 1024 / 1024
					c.perfMonitor.RecordBufferTest(bufferSize, bitrate)
				}
			}
		}
	}
}

func (c *UDPClient) GetPerformanceReport() {
	c.perfMonitor.PrintReport()

	packetsSent, packetsLost, retransmits, avgBitrate := c.perfMonitor.GetStatistics()
	fmt.Printf("\n=== UDP Performance Statistics ===\n")
	fmt.Printf("Packets Sent: %d\n", packetsSent)
	fmt.Printf("Packets Lost: %d\n", packetsLost)
	fmt.Printf("Retransmissions: %d\n", retransmits)

	if packetsSent > 0 {
		lossRate := float64(packetsLost) / float64(packetsSent) * 100
		fmt.Printf("Packet Loss Rate: %.2f%%\n", lossRate)
	}

	fmt.Printf("Average Bitrate: %.2f MB/s\n", avgBitrate)

	tcpBitrate := 10.0
	ratio, isFaster := c.perfMonitor.CompareWithTCP(tcpBitrate)

	fmt.Printf("UDP vs TCP Performance Ratio: %.2f\n", ratio)
	if isFaster {
		fmt.Printf("UDP is %.2fx faster than TCP (meets requirement)\n", ratio)
	} else {
		fmt.Printf("UDP is %.2fx faster than TCP (does not meet 1.5x requirement)\n", ratio)
	}

	optimalSize, optimalBitrate := c.perfMonitor.CalculateOptimalBufferSize()
	fmt.Printf("Optimal Buffer Size: %d bytes (%.2f MB/s)\n", optimalSize, optimalBitrate)

	fmt.Printf("===============================\n")
}
