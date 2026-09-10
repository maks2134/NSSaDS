package usecase

import (
	"fmt"

	"NSSaDS/lab7/internal/domain"
)

const (
	tagARows  = 10
	tagBPanel = 20
	tagCRows  = 30
)

// Result holds timing and verification output from a run.
type Result struct {
	Mode     string
	N        int
	Panel    int
	Procs    int
	Seconds  float64
	Checksum float64
}

// MultiplyBlocking multiplies N×N matrices with blocking point-to-point MPI.
func MultiplyBlocking(t domain.Transport, n, panel int) (Result, error) {
	if n <= 0 || panel <= 0 {
		return Result{}, fmt.Errorf("n and panel must be positive")
	}
	rank := t.Rank()
	size := t.Size()
	if size < 1 {
		return Result{}, fmt.Errorf("MPI size < 1")
	}

	rowStart, rowEnd := domain.RowBounds(n, rank, size)
	localRows := rowEnd - rowStart

	aLocal := domain.NewMatrix(localRows, n)
	cLocal := domain.NewMatrix(localRows, n)

	var bFull *domain.Matrix
	if rank == 0 {
		aFull := domain.NewMatrix(n, n)
		bFull = domain.NewMatrix(n, n)
		domain.FillSequential(aFull)
		for i := range bFull.Data {
			bFull.Data[i] = float64((i*3)%89) * 0.02
		}
		copy(aLocal.Data, aFull.RowSlice(rowStart, rowEnd))
		for r := 1; r < size; r++ {
			s, e := domain.RowBounds(n, r, size)
			t.Send(aFull.RowSlice(s, e), r, tagARows)
		}
	} else {
		t.Recv(aLocal.Data, 0, tagARows)
	}

	t.Barrier()
	t0 := t.Wtime()

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
				t.Send(buf, 1, tagBPanel)
			}
		} else {
			t.Recv(buf, rank-1, tagBPanel)
			if rank < size-1 {
				t.Send(buf, rank+1, tagBPanel)
			}
		}
		domain.PanelGEMM(aLocal, cLocal, buf, n, col, pw)
	}

	t.Barrier()
	elapsed := t.Wtime() - t0

	checksum := gatherChecksum(t, cLocal, n, rank, size)
	return Result{
		Mode:     "blocking",
		N:        n,
		Panel:    panel,
		Procs:    size,
		Seconds:  elapsed,
		Checksum: checksum,
	}, nil
}

func gatherChecksum(t domain.Transport, cLocal *domain.Matrix, n, rank, size int) float64 {
	if rank == 0 {
		cFull := domain.NewMatrix(n, n)
		s0, e0 := domain.RowBounds(n, 0, size)
		copy(cFull.RowSlice(s0, e0), cLocal.Data)
		for r := 1; r < size; r++ {
			s, e := domain.RowBounds(n, r, size)
			t.Recv(cFull.RowSlice(s, e), r, tagCRows)
		}
		return domain.Checksum(cFull)
	}
	t.Send(cLocal.Data, 0, tagCRows)
	return 0
}
