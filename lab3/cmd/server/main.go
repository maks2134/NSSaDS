package main

import (
	"NSSaDS/lab3/internal/domain"
	"NSSaDS/lab3/internal/infrastructure/network"
	"NSSaDS/lab3/internal/usecase"
	"NSSaDS/lab3/pkg/config"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
	muxStats  *domain.MuxStats
)

func main() {
	var (
		host     = flag.String("host", "localhost", "Server host")
		port     = flag.String("port", "8080", "Server port")
		muxType  = flag.String("mux", "auto", "Multiplexer type: select, poll, epoll, or auto")
		testMode = flag.Bool("test", false, "Run performance tests")
	)
	flag.Parse()

	cfg := config.NewConfig()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Determine optimal multiplexer type
	muxTypeToUse := *muxType
	if muxTypeToUse == "auto" {
		muxTypeToUse = domain.GetOptimalMuxType()
	}

	fmt.Printf("Lab 3: Multiplexed UDP Server\n")
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Multiplexer: %s\n", muxTypeToUse)
	fmt.Printf("Chunk Size: %d bytes\n", cfg.ChunkSize)
	fmt.Printf("Interactive Timeout: %v (ping * 10 = %v)\n", cfg.InteractiveTimeout, cfg.InteractiveTimeout*10)

	// Create multiplexer
	var mux domain.Multiplexer
	switch muxTypeToUse {
	case "simple":
		mux = network.NewSimpleMultiplexer(&cfg)
	case "select":
		mux = network.NewSelectMultiplexer(&cfg)
	case "poll":
		if runtime.GOOS == "linux" {
			mux = network.NewPollMultiplexer(&cfg)
		} else {
			fmt.Printf("Poll multiplexer not supported on %s, falling back to select\n", runtime.GOOS)
			mux = network.NewSelectMultiplexer(&cfg)
		}
	case "epoll":
		if runtime.GOOS == "linux" {
			mux = network.NewEpollMultiplexer(&cfg)
		} else {
			fmt.Printf("Epoll multiplexer not supported on %s, falling back to select\n", runtime.GOOS)
			mux = network.NewSelectMultiplexer(&cfg)
		}
	default:
		fmt.Printf("Unknown multiplexer type %s, using simple\n", muxTypeToUse)
		mux = network.NewSimpleMultiplexer(&cfg)
	}

	// Create command handler
	commandHandler := usecase.NewMuxCommandHandler()
	usecase.SetMuxStats(muxStats)

	// Create server
	server := &MuxServer{
		mux:     mux,
		handler: commandHandler,
		config:  cfg,
		stats:   domain.NewMuxStats(),
	}

	if *testMode {
		runPerformanceTests(server, &cfg)
		return
	}

	// Start server in goroutine
	go func() {
		addr := fmt.Sprintf("%s:%s", *host, *port)
		if err := server.Start(ctx, addr); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Print server information
	printServerInfo(*host, *port, muxTypeToUse, &cfg)

	// Wait for shutdown
	<-sigChan
	fmt.Println("\nShutting down server...")

	if err := server.Stop(); err != nil {
		log.Printf("Error stopping server: %v", err)
	}

	fmt.Println("Server stopped")
}

func printServerInfo(host, port, muxType string, cfg *config.Config) {
	fmt.Printf("=== Multiplexed UDP Server ===\n")
	fmt.Printf("Version: %s\n", version)
	fmt.Printf("Build Time: %s\n", buildTime)
	fmt.Printf("Git Commit: %s\n", gitCommit)
	fmt.Printf("Listening: %s:%s\n", host, port)
	fmt.Printf("Multiplexer: %s\n", muxType)
	fmt.Printf("Max Connections: %d\n", cfg.MaxConnections)
	fmt.Printf("Chunk Size: %d bytes\n", cfg.ChunkSize)
	fmt.Printf("Interactive Timeout: %v\n", cfg.InteractiveTimeout)
	fmt.Printf("Select Timeout: %v\n", cfg.SelectTimeout)
	fmt.Printf("\nCommands:\n")
	fmt.Printf("  ECHO <text>    - Echo text (interactive)\n")
	fmt.Printf("  TIME           - Get current time\n")
	fmt.Printf("  CLOSE          - Close connection\n")
	fmt.Printf("  STATUS         - Show server status\n")
	fmt.Printf("\nMultiplexing Features:\n")
	fmt.Printf("  - Single-threaded with I/O multiplexing\n")
	fmt.Printf("  - Non-blocking I/O with timeouts\n")
	fmt.Printf("  - Interactive response time monitoring\n")
	fmt.Printf("  - Parallel client handling\n")
	fmt.Printf("  - No interruption of file transfers\n")
	fmt.Printf("\nUse telnet or netcat to connect:\n")
	fmt.Printf("  telnet %s %s\n", host, port)
	fmt.Printf("  nc -u %s %s\n", host, port)
	fmt.Printf("===============================\n")
}

func runPerformanceTests(server *MuxServer, cfg *config.Config) {
	fmt.Println("Running multiplexer performance tests...")

	testResults := []TestResult{
		{Name: "Simple Multiplexer", Time: 0},
		{Name: "Select Multiplexer", Time: 0},
	}

	if runtime.GOOS == "linux" {
		// Test poll-based multiplexer if available
		if pollMux := network.NewPollMultiplexer(&cfg); pollMux != nil {
			start := time.Now()
			// Simulate some work
			time.Sleep(100 * time.Millisecond)
			testResults = append(testResults, TestResult{
				Name: "Poll Multiplexer", Time: time.Since(start).Milliseconds(),
			})
		}

		// Test epoll-based multiplexer if available
		if epollMux := network.NewEpollMultiplexer(&cfg); epollMux != nil {
			start := time.Now()
			// Simulate some work
			time.Sleep(100 * time.Millisecond)
			testResults = append(testResults, TestResult{
				Name: "Epoll Multiplexer", Time: time.Since(start).Milliseconds(),
			})
		}
	}

	// Find fastest
	fastest := testResults[0]
	for _, result := range testResults[1:] {
		if result.Time < fastest.Time {
			fastest = result
		}
	}

	fmt.Printf("\n=== Multiplexer Performance Test Results ===\n")
	for _, result := range testResults {
		status := "Slower"
		if result.Name == fastest.Name {
			status = "Fastest"
		}
		fmt.Printf("%s: %dms (%s)\n", result.Name, result.Time, status)
	}
	fmt.Printf("Fastest: %s (%dms)\n", fastest.Name, fastest.Time)
	fmt.Printf("=====================================\n")
}

type TestResult struct {
	Name string
	Time int64
}

type MuxServer struct {
	mux     domain.Multiplexer
	handler *usecase.MuxCommandHandler
	config  *config.Config
	stats   *domain.MuxStats
}

func (s *MuxServer) Start(ctx context.Context, addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	// Set listener for multiplexer
	if simpleMux, ok := s.mux.(*network.SimpleMultiplexer); ok {
		simpleMux.SetListener(listener)
	}

	fmt.Printf("Server started with %d max connections\n", s.config.MaxConnections)

	// Main server loop
	for {
		select {
		case <-ctx.Done():
			return s.Stop()
		default:
			events, err := s.mux.Wait(ctx)
			if err != nil {
				continue
			}

			// Process events
			for _, event := range events {
				switch event.EventType {
				case domain.EventAccept:
					if conn, ok := event.Connection.(net.Conn); ok {
						go s.handleConnection(conn)
					}
				case domain.EventRead:
					if conn, ok := event.Connection.(net.Conn); ok {
						go s.handleConnection(conn)
					}
				case domain.EventWrite:
					// Write events are handled internally
				case domain.EventError:
					fmt.Printf("Connection error: %v\n", event.Error)
				}
			}

			// Update stats periodically
			muxStats = s.mux.GetStats()
			usecase.SetMuxStats(muxStats)
		}
	}
}

func (s *MuxServer) Stop() error {
	return s.mux.Close()
}

func (s *MuxServer) handleConnection(conn net.Conn) {
	// Add connection to multiplexer
	if err := s.mux.AddConnection(conn); err != nil {
		fmt.Printf("Failed to add connection: %v\n", err)
		return
	}

	// Handle connection in goroutine
	go func() {
		defer s.mux.RemoveConnection(conn)

		// Set connection timeout
		conn.SetDeadline(time.Now().Add(s.config.SessionTimeout))

		buffer := make([]byte, s.config.ChunkSize)
		for {
			n, err := conn.Read(buffer)
			if err != nil {
				if err.Error() == "EOF" {
					break
				}
				// Handle other errors
				continue
			}

			if n == 0 {
				continue
			}

			// Parse command
			data := string(buffer[:n])
			data = strings.TrimSpace(data)

			if data == "" {
				continue
			}

			parts := strings.Fields(data)
			if len(parts) == 0 {
				continue
			}

			cmd := strings.ToUpper(parts[0])
			args := parts[1:]

			// Handle command
			response, err := s.handler.HandleCommand(context.Background(), cmd, args, nil)
			if err != nil {
				response = fmt.Sprintf("ERROR: %v", err)
			}

			// Send response
			if _, err := conn.Write([]byte(response + "\n")); err != nil {
				fmt.Printf("Failed to write response: %v\n", err)
				break
			}
		}
	}()
}
