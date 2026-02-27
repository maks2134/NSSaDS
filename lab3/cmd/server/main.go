package main

import (
	"NSSaDS/lab3/internal/domain"
	"NSSaDS/lab3/internal/infrastructure/network"
	"NSSaDS/lab3/internal/usecase"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	host := flag.String("host", "localhost", "Server host")
	port := flag.Int("port", 8080, "Server port")
	maxClients := flag.Int("max-clients", 100, "Maximum number of clients")
	pingTimeout := flag.Duration("ping-timeout", 30*time.Second, "Ping timeout duration")
	chunkSize := flag.Int("chunk-size", 1024, "Default chunk size in bytes")
	selectTimeout := flag.Duration("select-timeout", 10*time.Millisecond, "Select timeout duration")
	flag.Parse()

	commandHandler := usecase.NewCommandHandler()

	multiplexer := network.NewSelectMultiplexer(commandHandler, nil, nil)

	server := network.NewTCPServer(multiplexer, commandHandler)

	config := &domain.ServerConfig{
		Host:          *host,
		Port:          *port,
		MaxClients:    *maxClients,
		PingTimeout:   *pingTimeout,
		ChunkSize:     *chunkSize,
		SelectTimeout: *selectTimeout,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error, 1)
	go func() {
		log.Printf("Starting server on %s:%d", config.Host, config.Port)
		if err := server.Start(ctx, config); err != nil {
			errChan <- fmt.Errorf("server error: %w", err)
		}
	}()

	fmt.Println("\n=== Lab3: TCP Server with Select() Multiplexing ===")
	fmt.Printf("Host: %s\n", config.Host)
	fmt.Printf("Port: %d\n", config.Port)
	fmt.Printf("Max Clients: %d\n", config.MaxClients)
	fmt.Printf("Ping Timeout: %v\n", config.PingTimeout)
	fmt.Printf("Chunk Size: %d bytes\n", config.ChunkSize)
	fmt.Printf("Select Timeout: %v\n", config.SelectTimeout)
	fmt.Println("\nMultiplexing Method: select() system call")
	fmt.Println("Single-threaded concurrent client handling")
	fmt.Println("\nSupported commands:")
	fmt.Println("  ECHO <text>     - Echo the provided text")
	fmt.Println("  TIME            - Get current server time")
	fmt.Println("  CLOSE/EXIT/QUIT - Close connection")
	fmt.Println("  HELP            - Show this help message")
	fmt.Println("\nUse telnet or netcat to connect:")
	fmt.Printf("  telnet %s %d\n", config.Host, config.Port)
	fmt.Printf("  nc %s %d\n", config.Host, config.Port)
	fmt.Println("\nFeatures:")
	fmt.Println("  ✓ Single-threaded operation")
	fmt.Println("  ✓ select() I/O multiplexing")
	fmt.Println("  ✓ Dynamic chunk sizing (t = ping * 10)")
	fmt.Println("  ✓ Non-blocking file transfers")
	fmt.Println("  ✓ Concurrent client handling")

	select {
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down gracefully...", sig)
		cancel()

		if err := server.Stop(); err != nil {
			log.Printf("Error stopping server: %v", err)
		}

		log.Println("Server stopped")
		os.Exit(0)

	case err := <-errChan:
		log.Printf("Server error: %v", err)
		os.Exit(1)
	}
}
