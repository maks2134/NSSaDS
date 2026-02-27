package usecase

import (
	"NSSaDS/lab3/internal/domain"
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

type QuitCommand struct{}

func (c *QuitCommand) Execute(ctx context.Context, args []string) (string, error) {
	return "Connection closing...", nil
}

func (c *QuitCommand) Name() string {
	return "QUIT"
}

type ExitCommand struct{}

func (c *ExitCommand) Execute(ctx context.Context, args []string) (string, error) {
	return "Connection closing...", nil
}

func (c *ExitCommand) Name() string {
	return "EXIT"
}

type HelpCommand struct{}

func (c *HelpCommand) Execute(ctx context.Context, args []string) (string, error) {
	help := `Available commands:
  ECHO <text>     - Echo the provided text
  TIME            - Get current server time
  CLOSE/EXIT/QUIT - Close connection
  HELP            - Show this help message`
	return help, nil
}

func (c *HelpCommand) Name() string {
	return "HELP"
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
	handler.RegisterCommand(&QuitCommand{})
	handler.RegisterCommand(&ExitCommand{})
	handler.RegisterCommand(&HelpCommand{})

	return handler
}

func (h *CommandHandler) RegisterCommand(command domain.Command) {
	h.commands[command.Name()] = command
}

func (h *CommandHandler) HandleCommand(ctx context.Context, cmd string, args []string) (string, error) {
	parts := strings.Fields(strings.TrimSpace(cmd))
	if len(parts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	commandName := strings.ToUpper(parts[0])
	var commandArgs []string
	if len(parts) > 1 {
		commandArgs = parts[1:]
	}

	command, exists := h.commands[commandName]
	if !exists {
		return "", fmt.Errorf("unknown command: %s. Type HELP for available commands", commandName)
	}

	return command.Execute(ctx, commandArgs)
}
