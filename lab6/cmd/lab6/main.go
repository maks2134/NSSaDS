package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"NSSaDS/lab6/internal/usecase"
	"NSSaDS/lab6/pkg/config"
)

var version = "dev"

func main() {
	cfg := config.New()
	listIfaces := false
	flag.IntVar(&cfg.Port, "port", cfg.Port, "UDP port")
	flag.StringVar(&cfg.IfaceName, "iface", "", "network interface name")
	flag.StringVar(&cfg.Group, "group", cfg.Group, "multicast group address")
	flag.StringVar(&cfg.Nick, "nick", "", "chat nickname")
	flag.BoolVar(&listIfaces, "list", false, "list network interfaces and exit")
	flag.Parse()

	if listIfaces {
		if err := usecase.PrintIfaces(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if cfg.Nick == "" {
		host, err := os.Hostname()
		if err != nil {
			fmt.Fprintf(os.Stderr, "hostname: %v\n", err)
			os.Exit(1)
		}
		cfg.Nick = host
	}

	if err := usecase.Run(context.Background(), cfg, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
