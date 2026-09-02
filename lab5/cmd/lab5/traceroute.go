package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"NSSaDS/lab5/internal/usecase"
	"NSSaDS/lab5/pkg/config"
)

func runTraceroute(args []string) error {
	fs := flag.NewFlagSet("traceroute", flag.ContinueOnError)
	maxTTL := fs.Int("m", config.DefaultMaxTTL, "max hop count")
	timeout := fs.Duration("W", time.Second, "wait time per hop")
	if err := fs.Parse(args); err != nil {
		return err
	}
	hosts := fs.Args()
	if len(hosts) == 0 {
		return fmt.Errorf("traceroute: at least one host is required")
	}

	cfg := config.New()
	cfg.MaxTTL = *maxTTL
	cfg.Timeout = *timeout

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	out := usecase.NewPrinter(os.Stdout)
	return usecase.NewTracer(cfg, out).Run(ctx, hosts)
}
