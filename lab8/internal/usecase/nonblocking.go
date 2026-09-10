package usecase

import (
	"fmt"

	"NSSaDS/lab8/internal/domain"
)

// MultiplyNonBlocking multiplies with double-buffered non-blocking ring transfers.
func MultiplyNonBlocking(g domain.GroupComm, data *LocalData, bFull *domain.Matrix, panel int) (TimedResult, error) {
	if panel <= 0 {
		return TimedResult{}, fmt.Errorf("panel must be positive")
	}
	n := data.N
	rank := g.Rank()
	size := g.Size()
	cLocal := domain.NewMatrix(data.LocalRows, n)
	widths := panelWidths(n, panel)
	bufs := [2][]float64{make([]float64, n*panel), make([]float64, n*panel)}

	g.Barrier()
	t0 := g.Wtime()
	runNonBlockingRing(g, data.ALocal, bFull, cLocal, widths, bufs, n, panel, rank, size)
	g.Barrier()

	elapsed := g.AllreduceMax(g.Wtime() - t0)
	cs := gatherChecksum(g, cLocal, n, rank, size)
	return TimedResult{Mode: "nonblocking", Seconds: elapsed, Checksum: cs, CLocal: cLocal}, nil
}

func panelWidths(n, panel int) []int {
	num := (n + panel - 1) / panel
	w := make([]int, num)
	for p := 0; p < num; p++ {
		end := (p + 1) * panel
		if end > n {
			end = n
		}
		w[p] = end - p*panel
	}
	return w
}

func runNonBlockingRing(
	g domain.GroupComm, aLocal, bFull, cLocal *domain.Matrix,
	widths []int, buf [2][]float64, n, panel, rank, size int,
) {
	var recvReq, sendReq domain.Request
	haveRecv, haveSend := false, false
	cur := stageFirstPanel(g, bFull, buf[0], widths[0], n, rank, &recvReq, &haveRecv)

	for p := 0; p < len(widths); p++ {
		pw := widths[p]
		curBuf := buf[cur][:n*pw]
		if haveRecv {
			g.Wait(recvReq)
			haveRecv = false
		}
		if rank < size-1 {
			sendReq = g.Isend(curBuf, rank+1, tagBPanel)
			haveSend = true
		}
		next := 1 - cur
		if p+1 < len(widths) {
			startNextPanel(g, bFull, buf[next], widths[p+1], n, (p+1)*panel, rank, &recvReq, &haveRecv)
		}
		domain.PanelGEMM(aLocal, cLocal, curBuf, n, p*panel, pw)
		if haveSend {
			g.Wait(sendReq)
			haveSend = false
		}
		cur = next
	}
}

func stageFirstPanel(
	g domain.GroupComm, bFull *domain.Matrix, dst []float64, pw, n, rank int,
	recvReq *domain.Request, haveRecv *bool,
) int {
	if rank == 0 {
		domain.ExtractPanel(bFull, 0, pw, dst[:n*pw])
	} else {
		*recvReq = g.Irecv(dst[:n*pw], rank-1, tagBPanel)
		*haveRecv = true
	}
	return 0
}

func startNextPanel(
	g domain.GroupComm, bFull *domain.Matrix, dst []float64, pw, n, col, rank int,
	recvReq *domain.Request, haveRecv *bool,
) {
	if rank == 0 {
		domain.ExtractPanel(bFull, col, col+pw, dst[:n*pw])
		return
	}
	*recvReq = g.Irecv(dst[:n*pw], rank-1, tagBPanel)
	*haveRecv = true
}
