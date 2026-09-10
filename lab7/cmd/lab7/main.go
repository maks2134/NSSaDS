package main

import (
	"flag"
	"fmt"
	"os"

	"NSSaDS/lab7/internal/infrastructure/mpi"
	"NSSaDS/lab7/internal/usecase"
	"NSSaDS/lab7/pkg/config"
)

func main() {
	cfg := config.Config{
		N:     config.DefaultN,
		Panel: config.DefaultPanel,
		Mode:  config.DefaultMode,
	}

	flag.IntVar(&cfg.N, "n", cfg.N, "matrix size N (N×N)")
	flag.IntVar(&cfg.Panel, "panel", cfg.Panel, "B panel width in columns")
	flag.StringVar(&cfg.Mode, "mode", cfg.Mode, "blocking | nonblocking")
	flag.Parse()

	if cfg.N <= 0 || cfg.Panel <= 0 {
		fmt.Fprintln(os.Stderr, "n and panel must be positive")
		os.Exit(2)
	}
	if cfg.Mode != "blocking" && cfg.Mode != "nonblocking" {
		fmt.Fprintln(os.Stderr, "mode must be blocking or nonblocking")
		os.Exit(2)
	}

	session, world, err := mpi.Start(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer session.Stop()

	rank := world.Rank()
	size := world.Size()
	if rank == 0 && size < config.MinProcessesHint {
		fmt.Printf("warning: running with %d process(es); lab expects >= %d hosts/processes\n",
			size, config.MinProcessesHint)
	}

	var res usecase.Result
	switch cfg.Mode {
	case "blocking":
		res, err = usecase.MultiplyBlocking(world, cfg.N, cfg.Panel)
	case "nonblocking":
		res, err = usecase.MultiplyNonBlocking(world, cfg.N, cfg.Panel)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "rank %d: %v\n", rank, err)
		os.Exit(1)
	}

	if rank == 0 {
		fmt.Printf("mode=%s  P=%d  N=%d  panel=%d  time=%.3f s  checksum=%.6e\n",
			res.Mode, res.Procs, res.N, res.Panel, res.Seconds, res.Checksum)
	}
}
