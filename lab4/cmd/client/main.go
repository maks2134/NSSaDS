package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"NSSaDS/lab4/internal/domain"
	"NSSaDS/lab4/internal/infrastructure/network"
	"NSSaDS/lab4/pkg/config"
)

func main() {
	var (
		timeout = flag.Duration("timeout", 10*time.Second, "Request timeout")
	)
	flag.Parse()

	cfg := config.NewConfig()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	client := network.NewUDPClient(cfg)
	client.SetTimeout(*timeout)

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("UDP Multiservice Client")
	fmt.Println("======================")
	fmt.Println("Available services:")
	fmt.Println("  echo (port 8081) - Echo service")
	fmt.Println("  time (port 8082) - Time service")
	fmt.Println("  calc (port 8084) - Calculator service")
	fmt.Println("  stats (port 8085) - Statistics service")
	fmt.Println("\nCommands:")
	fmt.Println("  connect <host:port> - Connect to specific service")
	fmt.Println("  send <service> <command> [data] - Send request to service")
	fmt.Println("  echo <text> - Quick echo command")
	fmt.Println("  time [unix] - Quick time command")
	fmt.Println("  calc <num1> <op> <num2> - Quick calc command")
	fmt.Println("  stats <command> - Quick stats command")
	fmt.Println("  quit/exit - Exit client")
	fmt.Println()

	for {
		fmt.Print("client> ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		cmd := strings.ToLower(parts[0])
		args := parts[1:]

		switch cmd {
		case "connect":
			if len(args) < 1 {
				fmt.Println("Usage: connect <host:port>")
				continue
			}
			handleConnect(ctx, client, args[0])

		case "send":
			if len(args) < 2 {
				fmt.Println("Usage: send <service> <command> [data]")
				continue
			}
			service := domain.ServiceType(args[0])
			command := args[1]
			data := strings.Join(args[2:], " ")
			handleSend(client, service, command, data)

		case "echo":
			if len(args) < 1 {
				fmt.Println("Usage: echo <text>")
				continue
			}
			handleQuickCommand(client, "localhost:8081", "ECHO", strings.Join(args, " "))

		case "time":
			command := "GET"
			data := ""
			if len(args) > 0 && strings.ToLower(args[0]) == "unix" {
				command = "UNIX"
			}
			handleQuickCommand(client, "localhost:8082", command, data)

		case "calc":
			if len(args) < 3 {
				fmt.Println("Usage: calc <num1> <op> <num2>")
				continue
			}
			handleQuickCommand(client, "localhost:8084", "CALC", strings.Join(args, " "))

		case "stats":
			command := "ALL"
			data := ""
			if len(args) > 0 {
				command = strings.ToUpper(args[0])
				if len(args) > 1 {
					data = strings.Join(args[1:], " ")
				}
			}
			handleQuickCommand(client, "localhost:8085", command, data)

		case "help":
			showHelp()

		case "quit", "exit":
			fmt.Println("Goodbye!")
			return

		default:
			fmt.Printf("Unknown command: %s\n", cmd)
			fmt.Println("Type 'help' for available commands")
		}
	}
}

func handleConnect(ctx context.Context, client domain.UDPClient, addr string) {
	if err := client.Connect(ctx, addr); err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}
	fmt.Printf("Connected to %s\n", addr)
}

func handleSend(client domain.UDPClient, service domain.ServiceType, command, data string) {
	response, err := client.SendRequest(service, command, []byte(data))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if response.Error != nil {
		fmt.Printf("Server error: %v\n", response.Error)
	} else {
		fmt.Printf("Response: %s\n", string(response.Data))
	}
}

func handleQuickCommand(client domain.UDPClient, addr, command, data string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Connect(ctx, addr); err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}
	defer client.Disconnect()

	response, err := client.SendRequest("", command, []byte(data))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if response.Error != nil {
		fmt.Printf("Server error: %v\n", response.Error)
	} else {
		fmt.Printf("Response: %s\n", string(response.Data))
	}
}

func showHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  connect <host:port> - Connect to specific service")
	fmt.Println("  send <service> <command> [data] - Send request to service")
	fmt.Println("  echo <text> - Quick echo command")
	fmt.Println("  time [unix] - Quick time command")
	fmt.Println("  calc <num1> <op> <num2> - Quick calc command")
	fmt.Println("  stats <command> - Quick stats command")
	fmt.Println("  quit/exit - Exit client")
	fmt.Println("\nServices:")
	fmt.Println("  echo - Echo service (port 8081)")
	fmt.Println("  time - Time service (port 8082)")
	fmt.Println("  calc - Calculator service (port 8084)")
	fmt.Println("  stats - Statistics service (port 8085)")
}
