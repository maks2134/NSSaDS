package main

import (
	"NSSaDS/internal/infrastructure/network"
	"NSSaDS/internal/infrastructure/repository"
	"NSSaDS/pkg/config"
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	var (
		host = flag.String("host", "localhost", "Server host")
		port = flag.String("port", "8080", "Server port")
	)
	flag.Parse()

	cfg := config.NewConfig()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fileMgr := repository.NewFileManager("./downloads")
	defer fileMgr.Close()

	client := network.NewTCPClient(&cfg.Client, fileMgr)

	addr := fmt.Sprintf("%s:%s", *host, *port)
	if err := client.Connect(ctx, addr); err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer client.Disconnect()

	fmt.Printf("Connected to server %s\n", addr)
	fmt.Println("Available commands:")
	fmt.Println("  ECHO <text>           - Echo the provided text")
	fmt.Println("  TIME                  - Get current server time")
	fmt.Println("  CLOSE/EXIT/QUIT       - Close connection")
	fmt.Println("  UPLOAD <local> <remote> - Upload a file to server")
	fmt.Println("  DOWNLOAD <remote> <local> - Download a file from server")
	fmt.Println("  HELP                  - Show this help")
	fmt.Println()

	go func() {
		<-sigChan
		fmt.Println("\nDisconnecting...")
		client.Disconnect()
		cancel()
	}()

	scanner := bufio.NewScanner(os.Stdin)

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
		cmd := strings.ToUpper(parts[0])
		args := parts[1:]

		switch cmd {
		case "HELP":
			showHelp()
		case "UPLOAD":
			if len(args) < 2 {
				fmt.Println("Usage: UPLOAD <local_path> <remote_name>")
				continue
			}
			handleUpload(client, args[0], args[1])
		case "DOWNLOAD":
			if len(args) < 2 {
				fmt.Println("Usage: DOWNLOAD <remote_name> <local_path>")
				continue
			}
			handleDownload(client, args[0], args[1])
		case "EXIT", "QUIT":
			client.SendCommand("CLOSE", []string{})
			return
		default:
			response, err := client.SendCommand(cmd, args)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Printf("Server: %s\n", response)
			}
		}
	}
}

func showHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  ECHO <text>           - Echo the provided text")
	fmt.Println("  TIME                  - Get current server time")
	fmt.Println("  CLOSE/EXIT/QUIT       - Close connection")
	fmt.Println("  UPLOAD <local> <remote> - Upload a file to server")
	fmt.Println("  DOWNLOAD <remote> <local> - Download a file from server")
	fmt.Println("  HELP                  - Show this help")
}

func handleUpload(client *network.TCPClient, localPath, remoteName string) {
	progress, err := client.UploadFile(localPath, remoteName)
	if err != nil {
		fmt.Printf("Upload error: %v\n", err)
		return
	}

	fmt.Printf("Upload completed: %s (%.2f MB, %.2f MB/s)\n",
		progress.FileName,
		float64(progress.Transferred)/1024/1024,
		progress.Bitrate)
}

func handleDownload(client *network.TCPClient, remoteName, localPath string) {
	progress, err := client.DownloadFile(remoteName, localPath)
	if err != nil {
		fmt.Printf("Download error: %v\n", err)
		return
	}

	fmt.Printf("Download completed: %s (%.2f MB, %.2f MB/s)\n",
		progress.FileName,
		float64(progress.Transferred)/1024/1024,
		progress.Bitrate)
}
