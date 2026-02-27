package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	host := flag.String("host", "localhost", "Server host")
	port := flag.Int("port", 8080, "Server port")
	flag.Parse()

	addr := fmt.Sprintf("%s:%d", *host, *port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Printf("Failed to connect to %s: %v\n", addr, err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Printf("Connected to %s\n", addr)
	fmt.Println("Type commands (ECHO, TIME, HELP, CLOSE, EXIT, QUIT)")
	fmt.Println("Example: ECHO Hello World")
	fmt.Print("> ")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	inputChan := make(chan string)
	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				close(inputChan)
				return
			}
			inputChan <- strings.TrimSpace(line)
		}
	}()

	responseChan := make(chan string)
	go func() {
		reader := bufio.NewReader(conn)
		for {
			response, err := reader.ReadString('\n')
			if err != nil {
				close(responseChan)
				return
			}
			responseChan <- strings.TrimSpace(response)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nShutting down...")
			return

		case sig := <-sigChan:
			fmt.Printf("\nReceived signal %v, shutting down...\n", sig)
			return

		case input, ok := <-inputChan:
			if !ok {
				fmt.Println("\nInput closed, shutting down...")
				return
			}

			if input == "" {
				fmt.Print("> ")
				continue
			}

			_, err := conn.Write([]byte(input + "\n"))
			if err != nil {
				fmt.Printf("Error sending command: %v\n", err)
				return
			}

		case response, ok := <-responseChan:
			if !ok {
				fmt.Println("\nServer disconnected")
				return
			}

			fmt.Printf("Server: %s\n", response)
			fmt.Print("> ")
		}
	}
}
