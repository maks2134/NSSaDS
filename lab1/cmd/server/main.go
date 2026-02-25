package main

import (
	"NSSaDS/internal/infrastructure/network"
	"NSSaDS/internal/infrastructure/repository"
	"NSSaDS/internal/usecase"
	"NSSaDS/pkg/config"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var (
		host = flag.String("host", "localhost", "Server host")
		port = flag.String("port", "8080", "Server port")
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

	connMgr := network.NewTCPConnectionManager(&cfg.Server, fileMgr)
	connMgr.SetCommandHandler(commandHandler)

	server := network.NewTCPServer(&cfg.Server, commandHandler, connMgr)

	go func() {
		addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
		if err := server.Start(ctx, addr); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	fmt.Printf("TCP Server started on %s:%s\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Println("Supported commands:")
	fmt.Println("  ECHO <text>     - Echo the provided text")
	fmt.Println("  TIME            - Get current server time")
	fmt.Println("  CLOSE/EXIT/QUIT - Close connection")
	fmt.Println("  UPLOAD <file>   - Upload a file to server")
	fmt.Println("  DOWNLOAD <file> - Download a file from server")
	fmt.Println("\nUse telnet or netcat to connect:")
	fmt.Printf("  telnet %s %s\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("  nc %s %s\n", cfg.Server.Host, cfg.Server.Port)

	<-sigChan
	fmt.Println("\nShutting down server...")

	if err := server.Stop(); err != nil {
		log.Printf("Error stopping server: %v", err)
	}

	fmt.Println("Server stopped")
}
