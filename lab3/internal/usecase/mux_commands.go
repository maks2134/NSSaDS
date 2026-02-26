package usecase

import (
	"NSSaDS/lab3/internal/domain"
	"context"
	"fmt"
	"strings"
	"time"
)

type EchoCommand struct{}

func (c *EchoCommand) Execute(ctx context.Context, args []string, conn *domain.Connection) (string, error) {
	response := strings.Join(args, " ")

	// Send response in chunks for interactive timing
	chunkSize := conn.ChunkSize
	if len(response) > chunkSize {
		for i := 0; i < len(response); i += chunkSize {
			end := i + chunkSize
			if end > len(response) {
				end = len(response)
			}

			chunk := response[i:end]
			if _, err := conn.Conn.Write([]byte(chunk)); err != nil {
				return "", fmt.Errorf("failed to write response chunk: %w", err)
			}

			// Small delay to simulate network latency
			time.Sleep(1 * time.Millisecond)
		}
	} else {
		if _, err := conn.Conn.Write([]byte(response)); err != nil {
			return "", fmt.Errorf("failed to write response: %w", err)
		}
	}

	return response, nil
}

func (c *EchoCommand) Name() string {
	return "ECHO"
}

func (c *EchoCommand) IsInteractive() bool {
	return true
}

func (c *EchoCommand) GetChunkSize() int {
	return 512 // Small chunks for interactive response
}

type TimeCommand struct{}

func (c *TimeCommand) Execute(ctx context.Context, args []string, conn *domain.Connection) (string, error) {
	response := time.Now().Format(time.RFC3339)

	if _, err := conn.Conn.Write([]byte(response)); err != nil {
		return "", fmt.Errorf("failed to write time response: %w", err)
	}

	return response, nil
}

func (c *TimeCommand) Name() string {
	return "TIME"
}

func (c *TimeCommand) IsInteractive() bool {
	return true
}

func (c *TimeCommand) GetChunkSize() int {
	return 512
}

type CloseCommand struct{}

func (c *CloseCommand) Execute(ctx context.Context, args []string, conn *domain.Connection) (string, error) {
	response := "Connection closing..."

	if _, err := conn.Conn.Write([]byte(response)); err != nil {
		return "", fmt.Errorf("failed to write close response: %w", err)
	}

	return response, nil
}

func (c *CloseCommand) Name() string {
	return "CLOSE"
}

func (c *CloseCommand) IsInteractive() bool {
	return true
}

func (c *CloseCommand) GetChunkSize() int {
	return 512
}

type StatusCommand struct{}

func (c *StatusCommand) Execute(ctx context.Context, args []string, conn *domain.Connection) (string, error) {
	// Return server status including multiplexer stats
	stats := getMuxStats()

	response := fmt.Sprintf(
		"Server Status:\n"+
			"Total Connections: %d\n"+
			"Active Connections: %d\n"+
			"Bytes Read: %d\n"+
			"Bytes Written: %d\n"+
			"Events Processed: %d\n"+
			"Average Select Time: %v\n"+
			"Chunk Size: %d",
		stats.TotalConnections,
		stats.ActiveConnections,
		stats.BytesRead,
		stats.BytesWritten,
		stats.EventsProcessed,
		stats.AverageSelectTime,
		stats.ChunkSize,
	)

	if _, err := conn.Conn.Write([]byte(response)); err != nil {
		return "", fmt.Errorf("failed to write status response: %w", err)
	}

	return response, nil
}

func (c *StatusCommand) Name() string {
	return "STATUS"
}

func (c *StatusCommand) IsInteractive() bool {
	return true
}

func (c *StatusCommand) GetChunkSize() int {
	return 512
}

// Global reference to multiplexer stats (would be injected by server)
var muxStats *domain.MuxStats

func SetMuxStats(stats *domain.MuxStats) {
	muxStats = stats
}

func getMuxStats() *domain.MuxStats {
	if muxStats == nil {
		return &domain.MuxStats{}
	}
	return muxStats
}

type MuxCommandHandler struct {
	commands map[string]domain.Command
}

func NewMuxCommandHandler() *MuxCommandHandler {
	handler := &MuxCommandHandler{
		commands: make(map[string]domain.Command),
	}

	handler.RegisterCommand(&EchoCommand{})
	handler.RegisterCommand(&TimeCommand{})
	handler.RegisterCommand(&CloseCommand{})
	handler.RegisterCommand(&StatusCommand{})

	return handler
}

func (h *MuxCommandHandler) RegisterCommand(command domain.Command) {
	h.commands[command.Name()] = command
}

func (h *MuxCommandHandler) HandleCommand(ctx context.Context, cmd string, args []string, conn *domain.Connection) (string, error) {
	command, exists := h.commands[cmd]
	if !exists {
		return "", fmt.Errorf("unknown command: %s", cmd)
	}

	return command.Execute(ctx, args, conn)
}
