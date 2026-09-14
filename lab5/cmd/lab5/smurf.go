package main

import (
	"flag"
	"fmt"
	"os"

	"NSSaDS/lab5/internal/usecase"
	"NSSaDS/lab5/pkg/config"
)

func runSmurf(args []string) error {
	fs := flag.NewFlagSet("smurf", flag.ContinueOnError)
	victimFlag := fs.String("victim", "", "victim IPv4 (or pass as the first argument)")
	bcast := fs.String("bcast", "", "broadcast (optional, auto from LAN)")
	count := fs.Int("c", config.DefaultSmurfCount, "rounds to send (max 5)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	victim := *victimFlag
	if victim == "" && fs.NArg() >= 1 {
		victim = fs.Arg(0)
	}
	if victim == "" {
		return fmt.Errorf("smurf: victim ip required, e.g. lab5 smurf 192.168.1.10")
	}

	cfg := config.New()
	out := usecase.NewPrinter(os.Stdout)
	fmt.Fprintln(os.Stderr, "warning: educational demo on your own lab network only")
	return usecase.NewSmurfer(cfg, out).Run(victim, *bcast, *count)
}
