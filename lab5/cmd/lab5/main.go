package main

import (
	"fmt"
	"os"

	"NSSaDS/lab5/pkg/config"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}
	if err := dispatch(os.Args[1], os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func dispatch(cmd string, args []string) error {
	switch cmd {
	case "ping":
		return runPing(args)
	case "traceroute", "trace":
		return runTraceroute(args)
	case "smurf":
		return runSmurf(args)
	case "-h", "-help", "--help", "help":
		printUsage()
		return nil
	default:
		printUsage()
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `lab5 — parallel ICMP ping, traceroute, educational smurf

Usage:
  lab5 ping [-c N] [-W duration] [-i interval] [--trace] host [host...]
  lab5 traceroute [-m maxhops] [-W duration] host [host...]
  lab5 smurf -victim IP -bcast IP [-c N]

Raw ICMP sockets require root (Unix) or Administrator (Windows).
Smurf is a lab demonstration: at most %d packets, own network only.
`, config.MaxSmurfCount)
}
