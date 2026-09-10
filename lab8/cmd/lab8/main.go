package main

import (
	"flag"
	"fmt"
	"os"

	"NSSaDS/lab8/internal/infrastructure/mpi"
	"NSSaDS/lab8/internal/usecase"
	"NSSaDS/lab8/pkg/config"
)

func main() {
	cfg := config.Config{
		N:      config.DefaultN,
		Panel:  config.DefaultPanel,
		Mode:   config.DefaultMode,
		Groups: config.DefaultGroups,
		Seed:   config.DefaultSeed,
		APath:  config.DefaultAPath,
		BPath:  config.DefaultBPath,
		OutDir: config.DefaultOutDir,
	}

	flag.IntVar(&cfg.N, "n", cfg.N, "matrix size N (N×N)")
	flag.IntVar(&cfg.Panel, "panel", cfg.Panel, "B panel width in columns (P2P modes)")
	flag.StringVar(&cfg.Mode, "mode", cfg.Mode, "blocking | nonblocking | collective | compare")
	flag.IntVar(&cfg.Groups, "groups", cfg.Groups, "number of MPI groups")
	flag.Int64Var(&cfg.Seed, "seed", cfg.Seed, "RNG seed for group assignment")
	flag.StringVar(&cfg.APath, "a", cfg.APath, "path to matrix A file")
	flag.StringVar(&cfg.BPath, "b", cfg.BPath, "path to matrix B file")
	flag.StringVar(&cfg.OutDir, "outdir", cfg.OutDir, "directory for group-*.bin results")
	flag.BoolVar(&cfg.Gen, "gen", false, "generate A/B input files on rank 0 before run")
	flag.Parse()

	if err := validateConfig(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
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

	report, err := usecase.Run(world, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "rank %d: %v\n", rank, err)
		os.Exit(1)
	}

	if rank == 0 {
		fmt.Print(usecase.FormatReport(report))
	}
}

func validateConfig(cfg config.Config) error {
	if cfg.N <= 0 || cfg.Panel <= 0 {
		return fmt.Errorf("n and panel must be positive")
	}
	if cfg.Groups <= 0 {
		return fmt.Errorf("groups must be positive")
	}
	switch cfg.Mode {
	case "blocking", "nonblocking", "collective", "compare":
	default:
		return fmt.Errorf("mode must be blocking|nonblocking|collective|compare")
	}
	return nil
}
