package usecase

import (
	"fmt"

	"NSSaDS/lab8/internal/domain"
	"NSSaDS/lab8/internal/infrastructure/mpi"
	"NSSaDS/lab8/pkg/config"
)

// GroupReport is printed by WORLD rank 0 for one group.
type GroupReport struct {
	Color       int
	Ranks       []int
	Procs       int
	Collective  float64
	P2P         float64
	P2PMode     string
	Checksum    float64
	ChecksumP2P float64
}

// RunReport aggregates all group reports.
type RunReport struct {
	Groups  int
	N       int
	Panel   int
	Seed    int64
	Mode    string
	Reports []GroupReport
}

// Run executes lab8: assign groups, multiply, write results, gather reports.
func Run(world *mpi.Comm, cfg config.Config) (RunReport, error) {
	rank := world.Rank()
	size := world.Size()

	if cfg.Gen && rank == 0 {
		if err := GenerateInputFiles(cfg); err != nil {
			return RunReport{}, err
		}
	}
	world.Barrier()

	colors := broadcastColors(world, size, cfg.Groups, cfg.Seed)
	myColor := int(colors[rank])
	group := world.Split(myColor, rank)
	defer group.Free()

	data, err := ReadLocalMatrices(group, group, cfg.APath, cfg.BPath, cfg.N)
	if err != nil {
		return RunReport{}, fmt.Errorf("group %d read: %w", myColor, err)
	}

	report, cOut, err := runGroupModes(group, data, cfg)
	if err != nil {
		return RunReport{}, err
	}
	report.Color = myColor
	report.Ranks = domain.GroupMembers(colors, int32(myColor))
	report.Procs = group.Size()

	if cOut != nil {
		if err := WriteGroupResult(group, group, cfg.OutDir, myColor, data, cOut); err != nil {
			return RunReport{}, fmt.Errorf("group %d write: %w", myColor, err)
		}
	}

	return gatherReports(world, cfg, report)
}

func broadcastColors(world *mpi.Comm, size, groups int, seed int64) []int32 {
	colors := make([]int32, size)
	if world.Rank() == 0 {
		colors = domain.AssignColors(size, groups, seed)
	}
	world.BcastInt32(colors, 0)
	return colors
}

func runGroupModes(g *mpi.Comm, data *LocalData, cfg config.Config) (GroupReport, *domain.Matrix, error) {
	var rep GroupReport
	var cOut *domain.Matrix

	switch cfg.Mode {
	case "collective":
		tr, err := MultiplyCollective(g, data)
		if err != nil {
			return rep, nil, err
		}
		rep.Collective = tr.Seconds
		rep.Checksum = tr.Checksum
		cOut = tr.CLocal
	case "blocking", "nonblocking":
		tr, err := runP2P(g, data, cfg.Mode, cfg.Panel)
		if err != nil {
			return rep, nil, err
		}
		rep.P2P = tr.Seconds
		rep.P2PMode = tr.Mode
		rep.ChecksumP2P = tr.Checksum
		rep.Checksum = tr.Checksum
		cOut = tr.CLocal
	default: // compare
		trC, err := MultiplyCollective(g, data)
		if err != nil {
			return rep, nil, err
		}
		trP, err := runP2P(g, data, "blocking", cfg.Panel)
		if err != nil {
			return rep, nil, err
		}
		rep.Collective = trC.Seconds
		rep.P2P = trP.Seconds
		rep.P2PMode = trP.Mode
		rep.Checksum = trC.Checksum
		rep.ChecksumP2P = trP.Checksum
		cOut = trC.CLocal
	}
	return rep, cOut, nil
}

func runP2P(g *mpi.Comm, data *LocalData, mode string, panel int) (TimedResult, error) {
	bFull := AssembleFullB(g, data)
	if mode == "nonblocking" {
		return MultiplyNonBlocking(g, data, bFull, panel)
	}
	return MultiplyBlocking(g, data, bFull, panel)
}

func gatherReports(world *mpi.Comm, cfg config.Config, local GroupReport) (RunReport, error) {
	out := RunReport{
		Groups: cfg.Groups, N: cfg.N, Panel: cfg.Panel, Seed: cfg.Seed, Mode: cfg.Mode,
	}
	rank := world.Rank()
	size := world.Size()
	isLeader := len(local.Ranks) > 0 && local.Ranks[0] == rank

	const tagMeta = 100
	const tagRanks = 101

	meta := make([]float64, 8)
	if isLeader {
		meta[0] = 1
		meta[1] = float64(local.Color)
		meta[2] = float64(local.Procs)
		meta[3] = local.Collective
		meta[4] = local.P2P
		meta[5] = float64(len(local.Ranks))
		meta[6] = local.Checksum
		meta[7] = local.ChecksumP2P
	}

	if rank == 0 {
		if isLeader {
			out.Reports = append(out.Reports, local)
		}
		for src := 1; src < size; src++ {
			m := make([]float64, 8)
			world.Recv(m, src, tagMeta)
			if m[0] < 0.5 {
				continue
			}
			nRanks := int(m[5])
			ranksBuf := make([]float64, nRanks)
			world.Recv(ranksBuf, src, tagRanks)
			ranks := make([]int, nRanks)
			for i, v := range ranksBuf {
				ranks[i] = int(v)
			}
			out.Reports = append(out.Reports, GroupReport{
				Color:       int(m[1]),
				Ranks:       ranks,
				Procs:       int(m[2]),
				Collective:  m[3],
				P2P:         m[4],
				Checksum:    m[6],
				ChecksumP2P: m[7],
				P2PMode:     "blocking",
			})
		}
		sortReports(out.Reports)
		return out, nil
	}

	world.Send(meta, 0, tagMeta)
	if isLeader {
		ranksBuf := make([]float64, len(local.Ranks))
		for i, r := range local.Ranks {
			ranksBuf[i] = float64(r)
		}
		world.Send(ranksBuf, 0, tagRanks)
	}
	return out, nil
}

func sortReports(reps []GroupReport) {
	for i := 0; i < len(reps); i++ {
		for j := i + 1; j < len(reps); j++ {
			if reps[j].Color < reps[i].Color {
				reps[i], reps[j] = reps[j], reps[i]
			}
		}
	}
}

// FormatReport returns a human-readable multi-line summary.
func FormatReport(r RunReport) string {
	s := fmt.Sprintf("groups=%d  N=%d  panel=%d  seed=%d  mode=%s\n",
		r.Groups, r.N, r.Panel, r.Seed, r.Mode)
	for _, g := range r.Reports {
		s += fmt.Sprintf("group=%d  ranks=%v  P=%d", g.Color, g.Ranks, g.Procs)
		if r.Mode == "compare" || r.Mode == "collective" {
			s += fmt.Sprintf("  collective=%.3fs", g.Collective)
		}
		if r.Mode == "compare" || r.Mode == "blocking" || r.Mode == "nonblocking" {
			mode := g.P2PMode
			if mode == "" {
				mode = "blocking"
			}
			if r.Mode == "nonblocking" {
				mode = "nonblocking"
			}
			s += fmt.Sprintf("  %s=%.3fs", mode, g.P2P)
		}
		s += fmt.Sprintf("  checksum=%.6e", g.Checksum)
		if r.Mode == "compare" && g.ChecksumP2P != 0 {
			s += fmt.Sprintf("  checksum_p2p=%.6e", g.ChecksumP2P)
		}
		s += "\n"
	}
	return s
}
