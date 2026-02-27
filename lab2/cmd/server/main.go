package main

import (
	"NSSaDS/lab2/internal/infrastructure/network"
	"NSSaDS/lab2/internal/infrastructure/repository"
	"NSSaDS/lab2/internal/usecase"
	"NSSaDS/lab2/pkg/config"
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
	var (
		host = flag.String("host", "localhost", "Server host")
		port = flag.String("port", "8080", "Server port")
		test = flag.Bool("test", false, "Run performance tests")
	)
	flag.Parse()

	cfg := config.NewConfig()
	cfg.Server.Host = *host
	cfg.Server.Port = *port

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fileMgr := repository.NewFileManager(cfg.Server.UploadDir)
	defer fileMgr.Close()

	commandHandler := usecase.NewCommandHandler()

	server := network.NewUDPServer(&cfg.Server, &cfg.UDP, commandHandler, fileMgr)

	if *test {
		runPerformanceTests(server, &cfg.UDP)
		return
	}

	go func() {
		addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
		if err := server.Start(ctx, addr); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	fmt.Printf("UDP Server started on %s:%s\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Println("Supported commands:")
	fmt.Println("  ECHO <text>     - Echo the provided text")
	fmt.Println("  TIME            - Get current server time")
	fmt.Println("  CLOSE/EXIT/QUIT - Close connection")
	fmt.Println("  UPLOAD <file>   - Upload a file to server")
	fmt.Println("  DOWNLOAD <file> - Download a file from server")
	fmt.Println("\nUDP Features:")
	fmt.Println("  - Sliding window protocol")
	fmt.Println("  - Packet acknowledgment and retransmission")
	fmt.Println("  - Performance monitoring and optimization")
	fmt.Println("  - Connection recovery and timeout handling")
	fmt.Println("\nUse the UDP client to connect:")
	fmt.Printf("  ./client %s:%s\n", cfg.Server.Host, cfg.Server.Port)

	<-sigChan
	fmt.Println("\nShutting down server...")

	if err := server.Stop(); err != nil {
		log.Printf("Error stopping server: %v", err)
	}

	fmt.Println("Server stopped")
}

func runPerformanceTests(server *network.UDPServer, udpConfig *config.UDPConfig) {
	fmt.Println("Running UDP performance tests...")

	fmt.Printf("Testing buffer sizes: %v\n", udpConfig.BufferSizes)

	for _, bufferSize := range udpConfig.BufferSizes {
		fmt.Printf("Testing buffer size: %d bytes\n", bufferSize)

		start := time.Now()

		elapsed := time.Since(start).Seconds()
		if elapsed > 0 {
			bitrate := float64(bufferSize) / elapsed / 1024 / 1024
			fmt.Printf("  Buffer size %d: %.2f MB/s\n", bufferSize, bitrate)
		}
	}

	fmt.Println("Performance tests completed")
}
