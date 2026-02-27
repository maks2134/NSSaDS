package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"NSSaDS/lab4/internal/domain"
	"NSSaDS/lab4/internal/infrastructure/network"
	"NSSaDS/lab4/internal/usecase"
	"NSSaDS/lab4/pkg/config"
)

func main() {
	var (
		host       = flag.String("host", "localhost", "Server host")
		configFile = flag.String("config", "", "Config file path (optional)")
	)
	flag.Parse()

	cfg := config.NewConfig()
	if *configFile != "" {
		log.Printf("Config file loading not implemented, using defaults")
	}

	cfg.Server.Host = *host

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	registry := network.NewServiceRegistry()
	threadPool := network.NewThreadPool(&domain.ThreadPoolConfig{
		MinWorkers:      cfg.ThreadPool.MinWorkers,
		MaxWorkers:      cfg.ThreadPool.MaxWorkers,
		QueueSize:       cfg.ThreadPool.QueueSize,
		WorkerTimeout:   cfg.ThreadPool.WorkerTimeout,
		ExpandThreshold: cfg.ThreadPool.ExpandThreshold,
	})
	server := network.NewUDPServer(cfg, registry, threadPool)

	echoService := usecase.NewEchoService(cfg.Services[domain.EchoService].Port)
	timeService := usecase.NewTimeService(cfg.Services[domain.TimeService].Port)
	calcService := usecase.NewCalcService(cfg.Services[domain.CalcService].Port)
	statsService := usecase.NewStatsService(cfg.Services[domain.StatsService].Port, server)

	services := []domain.Service{
		echoService,
		timeService,
		calcService,
		statsService,
	}

	for _, service := range services {
		if err := server.RegisterService(service); err != nil {
			log.Printf("Failed to register service: %v", err)
		}
	}

	if err := server.Start(ctx); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	fmt.Printf("UDP Multiservice Server started on %s\n", cfg.Server.Host)
	fmt.Println("Services:")
	for serviceType, serviceConfig := range cfg.Services {
		if serviceConfig.Enabled {
			fmt.Printf("  %s: %s:%d\n", serviceType, cfg.Server.Host, serviceConfig.Port)
		}
	}

	fmt.Println("\nAvailable services and commands:")
	fmt.Println("ECHO Service (port 8081):")
	fmt.Println("  Any text - Echo the text back")
	fmt.Println("\nTIME Service (port 8082):")
	fmt.Println("  Any command - Get current time")
	fmt.Println("  UNIX - Get Unix timestamp")
	fmt.Println("\nCALC Service (port 8084):")
	fmt.Println("  <num1> <op> <num2> - Perform calculation (+, -, *, /)")
	fmt.Println("  Example: 5 * 10")
	fmt.Println("\nSTATS Service (port 8085):")
	fmt.Println("  ALL - Show all service statistics")
	fmt.Println("  SERVICE <name> - Show specific service stats")
	fmt.Println("  HELP - Show help")

	<-sigChan
	fmt.Println("\nShutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	done := make(chan struct{})
	go func() {
		if err := server.Stop(); err != nil {
			log.Printf("Error stopping server: %v", err)
		}
		close(done)
	}()

	select {
	case <-done:
		fmt.Println("Server stopped gracefully")
	case <-shutdownCtx.Done():
		fmt.Println("Shutdown timeout exceeded")
	}
}
