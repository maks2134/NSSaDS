package usecase

import (
	"NSSaDS/internal/domain"
	"context"
	"fmt"
	"strings"
	"time"
)

type EchoCommand struct{}

func (c *EchoCommand) Execute(ctx context.Context, args []string) (string, error) {
	return strings.Join(args, " "), nil
}

func (c *EchoCommand) Name() string {
	return "ECHO"
}

type TimeCommand struct{}

func (c *TimeCommand) Execute(ctx context.Context, args []string) (string, error) {
	return time.Now().Format(time.RFC3339), nil
}

func (c *TimeCommand) Name() string {
	return "TIME"
}

type CloseCommand struct{}

func (c *CloseCommand) Execute(ctx context.Context, args []string) (string, error) {
	return "Connection closing...", nil
}

func (c *CloseCommand) Name() string {
	return "CLOSE"
}

type CommandHandler struct {
	commands map[string]domain.Command
}

func NewCommandHandler() *CommandHandler {
	handler := &CommandHandler{
		commands: make(map[string]domain.Command),
	}

	handler.RegisterCommand(&EchoCommand{})
	handler.RegisterCommand(&TimeCommand{})
	handler.RegisterCommand(&CloseCommand{})

	return handler
}

func (h *CommandHandler) RegisterCommand(command domain.Command) {
	h.commands[command.Name()] = command
}

func (h *CommandHandler) HandleCommand(ctx context.Context, cmd string, args []string) (string, error) {
	command, exists := h.commands[cmd]
	if !exists {
		return "", fmt.Errorf("unknown command: %s", cmd)
	}

	return command.Execute(ctx, args)
}
