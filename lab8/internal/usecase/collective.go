package usecase

import "NSSaDS/lab8/internal/domain"

// TimedResult is timing + checksum for one multiply algorithm.
type TimedResult struct {
	Mode     string
	Seconds  float64
	Checksum float64
	CLocal   *domain.Matrix
}

// MultiplyCollective multiplies using Bcast of each rank's B column slab.
func MultiplyCollective(g domain.GroupComm, data *LocalData) (TimedResult, error) {
	n := data.N
	size := g.Size()
	rank := g.Rank()
	cLocal := domain.NewMatrix(data.LocalRows, n)

	g.Barrier()
	t0 := g.Wtime()

	for root := 0; root < size; root++ {
		cs, ce := domain.RowBounds(n, root, size)
		pw := ce - cs
		buf := make([]float64, n*pw)
		if rank == root {
			packBLocal(data.BLocal, buf, n, pw)
		}
		g.BcastFloat64(buf, root)
		domain.PanelGEMM(data.ALocal, cLocal, buf, n, cs, pw)
	}

	g.Barrier()
	elapsed := g.AllreduceMax(g.Wtime() - t0)
	checksum := gatherChecksum(g, cLocal, n, rank, size)
	return TimedResult{Mode: "collective", Seconds: elapsed, Checksum: checksum, CLocal: cLocal}, nil
}

func packBLocal(bLocal *domain.Matrix, dst []float64, n, pw int) {
	for i := 0; i < n; i++ {
		copy(dst[i*pw:(i+1)*pw], bLocal.Data[i*pw:(i+1)*pw])
	}
}

func gatherChecksum(g domain.GroupComm, cLocal *domain.Matrix, n, rank, size int) float64 {
	counts := make([]int, size)
	displs := make([]int, size)
	total := 0
	for r := 0; r < size; r++ {
		s, e := domain.RowBounds(n, r, size)
		counts[r] = (e - s) * n
		displs[r] = total
		total += counts[r]
	}
	var recv []float64
	if rank == 0 {
		recv = make([]float64, total)
	}
	g.GathervFloat64(cLocal.Data, recv, counts, displs, 0)
	if rank != 0 {
		return 0
	}
	cFull := &domain.Matrix{Rows: n, Cols: n, Data: recv}
	return domain.Checksum(cFull)
}

// AssembleFullB gathers column slabs to rank 0 for P2P ring multiply.
func AssembleFullB(g domain.GroupComm, data *LocalData) *domain.Matrix {
	n := data.N
	rank := g.Rank()
	size := g.Size()

	counts := make([]int, size)
	displs := make([]int, size)
	total := 0
	for r := 0; r < size; r++ {
		cs, ce := domain.RowBounds(n, r, size)
		counts[r] = n * (ce - cs)
		displs[r] = total
		total += counts[r]
		_ = cs
	}

	var recv []float64
	if rank == 0 {
		recv = make([]float64, total)
	}
	g.GathervFloat64(data.BLocal.Data, recv, counts, displs, 0)
	if rank != 0 {
		return nil
	}

	bFull := domain.NewMatrix(n, n)
	offset := 0
	for r := 0; r < size; r++ {
		cs, ce := domain.RowBounds(n, r, size)
		pw := ce - cs
		block := recv[offset : offset+n*pw]
		for i := 0; i < n; i++ {
			copy(bFull.Data[i*n+cs:i*n+ce], block[i*pw:(i+1)*pw])
		}
		offset += n * pw
	}
	return bFull
}
