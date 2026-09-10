//go:build !cgo || windows

package mpi

import (
	"fmt"
	"runtime"

	"NSSaDS/lab8/internal/domain"
)

// Session is a no-op placeholder when OpenMPI/CGO is unavailable.
type Session struct{}

// Comm is a stub communicator.
type Comm struct{}

// Request is an opaque stub handle.
type Request struct{}

// File is a stub MPI file.
type File struct{}

// Start returns an error: real MPI needs CGO + OpenMPI on Unix.
func Start(args []string) (*Session, *Comm, error) {
	_ = args
	return nil, nil, fmt.Errorf(
		"MPI unavailable on %s/%s (need CGO=1, OpenMPI, non-Windows); build on the target host with: make build",
		runtime.GOOS, runtime.GOARCH,
	)
}

func (s *Session) Stop() {}

func (c *Comm) Rank() int      { panic("mpi stub") }
func (c *Comm) Size() int      { panic("mpi stub") }
func (c *Comm) Wtime() float64 { panic("mpi stub") }
func (c *Comm) Barrier()       { panic("mpi stub") }
func (c *Comm) Split(color, key int) *Comm {
	panic("mpi stub")
}
func (c *Comm) Free() {}

func (c *Comm) Send(buf []float64, dest, tag int)   { panic("mpi stub") }
func (c *Comm) Recv(buf []float64, source, tag int) { panic("mpi stub") }

func (c *Comm) Isend(buf []float64, dest, tag int) domain.Request {
	panic("mpi stub")
}
func (c *Comm) Irecv(buf []float64, source, tag int) domain.Request {
	panic("mpi stub")
}
func (c *Comm) Wait(req domain.Request)       { panic("mpi stub") }
func (c *Comm) WaitAll(reqs []domain.Request) { panic("mpi stub") }

func (c *Comm) BcastFloat64(buf []float64, root int) { panic("mpi stub") }
func (c *Comm) BcastInt32(buf []int32, root int)     { panic("mpi stub") }
func (c *Comm) GathervFloat64(send []float64, recv []float64, recvCounts, displs []int, root int) {
	panic("mpi stub")
}
func (c *Comm) AllreduceMax(val float64) float64 { panic("mpi stub") }

func (c *Comm) FileOpen(path string, create bool) (domain.MPIFile, error) {
	panic("mpi stub")
}

func (f *File) ReadAtAll(buf []byte, offset int64)  { panic("mpi stub") }
func (f *File) WriteAtAll(buf []byte, offset int64) { panic("mpi stub") }
func (f *File) ReadSubarrayAll(buf []float64, globalRows, globalCols, rowStart, colStart, localRows, localCols int) {
	panic("mpi stub")
}
func (f *File) Close() {}

var (
	_ domain.Transport  = (*Comm)(nil)
	_ domain.Collective = (*Comm)(nil)
	_ domain.GroupComm  = (*Comm)(nil)
	_ domain.FileOpener = (*Comm)(nil)
	_ domain.MPIFile    = (*File)(nil)
)
