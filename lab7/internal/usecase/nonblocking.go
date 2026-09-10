package usecase

import (
	"fmt"

	"NSSaDS/lab7/internal/domain"
)

// MultiplyNonBlocking multiplies with double-buffered non-blocking ring
// transfers (Isend/Irecv) overlapped with PanelGEMM — analogous to CUDA
// streams + cudaMemcpyAsync.
func MultiplyNonBlocking(t domain.Transport, n, panel int) (Result, error) {
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

	numPanels := (n + panel - 1) / panel
	panelWidths := make([]int, numPanels)
	for p := 0; p < numPanels; p++ {
		col := p * panel
		end := col + panel
		if end > n {
			end = n
		}
		panelWidths[p] = end - col
	}

	// Double buffers for B panels (packed N×pw).
	buf := [2][]float64{
		make([]float64, n*panel),
		make([]float64, n*panel),
	}

	t.Barrier()
	t0 := t.Wtime()

	var recvReq, sendReq domain.Request
	haveRecv := false
	haveSend := false

	// Stage panel 0 into buffer 0.
	cur := 0
	pw0 := panelWidths[0]
	if rank == 0 {
		domain.ExtractPanel(bFull, 0, pw0, buf[0][:n*pw0])
	} else {
		recvReq = t.Irecv(buf[0][:n*pw0], rank-1, tagBPanel)
		haveRecv = true
	}

	for p := 0; p < numPanels; p++ {
		pw := panelWidths[p]
		curBuf := buf[cur][:n*pw]

		if haveRecv {
			t.Wait(recvReq)
			haveRecv = false
		}

		// Forward current panel down the ring while we compute.
		if rank < size-1 {
			sendReq = t.Isend(curBuf, rank+1, tagBPanel)
			haveSend = true
		}

		// Prefetch next panel into the other buffer (overlap with GEMM).
		next := 1 - cur
		if p+1 < numPanels {
			pwNext := panelWidths[p+1]
			nextBuf := buf[next][:n*pwNext]
			colNext := (p + 1) * panel
			if rank == 0 {
				domain.ExtractPanel(bFull, colNext, colNext+pwNext, nextBuf)
			} else {
				recvReq = t.Irecv(nextBuf, rank-1, tagBPanel)
				haveRecv = true
			}
		}

		col := p * panel
		domain.PanelGEMM(aLocal, cLocal, curBuf, n, col, pw)

		if haveSend {
			t.Wait(sendReq)
			haveSend = false
		}

		cur = next
	}

	t.Barrier()
	elapsed := t.Wtime() - t0

	checksum := gatherChecksum(t, cLocal, n, rank, size)
	return Result{
		Mode:     "nonblocking",
		N:        n,
		Panel:    panel,
		Procs:    size,
		Seconds:  elapsed,
		Checksum: checksum,
	}, nil
}
