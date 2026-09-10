//go:build !cgo || windows

package mpi

import (
	"fmt"
	"runtime"

	"NSSaDS/lab7/internal/domain"
)

// Session is a no-op placeholder when OpenMPI/CGO is unavailable.
type Session struct{}

// World is a stub transport (methods panic if called after a failed Start).
type World struct{}

// Request is an opaque stub handle.
type Request struct{}

// Start returns an error: real MPI needs CGO + OpenMPI on Unix.
func Start(args []string) (*Session, *World, error) {
	_ = args
	return nil, nil, fmt.Errorf(
		"MPI unavailable on %s/%s (need CGO=1, OpenMPI, non-Windows); build on the target host with: make build",
		runtime.GOOS, runtime.GOARCH,
	)
}

func (s *Session) Stop() {}

func (w *World) Rank() int      { panic("mpi stub") }
func (w *World) Size() int      { panic("mpi stub") }
func (w *World) Wtime() float64 { panic("mpi stub") }
func (w *World) Barrier()       { panic("mpi stub") }

func (w *World) Send(buf []float64, dest, tag int)   { panic("mpi stub") }
func (w *World) Recv(buf []float64, source, tag int) { panic("mpi stub") }

func (w *World) Isend(buf []float64, dest, tag int) domain.Request {
	panic("mpi stub")
}
func (w *World) Irecv(buf []float64, source, tag int) domain.Request {
	panic("mpi stub")
}
func (w *World) Wait(req domain.Request)       { panic("mpi stub") }
func (w *World) WaitAll(reqs []domain.Request) { panic("mpi stub") }

var _ domain.Transport = (*World)(nil)
