package usecase

import (
	"fmt"

	"NSSaDS/lab8/internal/domain"
)

// MultiplyBlocking multiplies with blocking point-to-point ring of B panels.
// Rank 0 must hold full B (from AssembleFullB); others pass nil for bFull.
func MultiplyBlocking(g domain.GroupComm, data *LocalData, bFull *domain.Matrix, panel int) (TimedResult, error) {
	if panel <= 0 {
		return TimedResult{}, fmt.Errorf("panel must be positive")
	}
	n := data.N
	rank := g.Rank()
	size := g.Size()
	cLocal := domain.NewMatrix(data.LocalRows, n)

	g.Barrier()
	t0 := g.Wtime()

	bPanel := make([]float64, n*panel)
	for col := 0; col < n; col += panel {
		end := col + panel
		if end > n {
			end = n
		}
		pw := end - col
		buf := bPanel[:n*pw]

		if rank == 0 {
			domain.ExtractPanel(bFull, col, end, buf)
			if size > 1 {
				g.Send(buf, 1, tagBPanel)
			}
		} else {
			g.Recv(buf, rank-1, tagBPanel)
			if rank < size-1 {
				g.Send(buf, rank+1, tagBPanel)
			}
		}
		domain.PanelGEMM(data.ALocal, cLocal, buf, n, col, pw)
	}

	g.Barrier()
	elapsed := g.AllreduceMax(g.Wtime() - t0)
	checksum := gatherChecksum(g, cLocal, n, rank, size)
	return TimedResult{Mode: "blocking", Seconds: elapsed, Checksum: checksum, CLocal: cLocal}, nil
}
