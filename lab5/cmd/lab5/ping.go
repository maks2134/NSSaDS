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

func runPing(args []string) error {
	fs := flag.NewFlagSet("ping", flag.ContinueOnError)
	count := fs.Int("c", config.DefaultCount, "stop after N probes")
	timeout := fs.Duration("W", time.Second, "wait time for each reply")
	interval := fs.Duration("i", time.Second, "interval between probes")
	trace := fs.Bool("trace", false, "run traceroute before ping")
	if err := fs.Parse(args); err != nil {
		return err
	}
	hosts := fs.Args()
	if len(hosts) == 0 {
		return fmt.Errorf("ping: at least one host is required")
	}

	cfg := config.New()
	cfg.Count = *count
	cfg.Timeout = *timeout
	cfg.Interval = *interval

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	out := usecase.NewPrinter(os.Stdout)
	return usecase.NewPinger(cfg, out, *trace).Run(ctx, hosts)
}
