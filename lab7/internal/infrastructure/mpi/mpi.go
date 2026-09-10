//go:build cgo && !windows

package mpi

/*
#cgo pkg-config: ompi-c
#include <mpi.h>
*/
import "C"

import (
	"fmt"
	"unsafe"

	"NSSaDS/lab7/internal/domain"
)

// Session wraps an initialized MPI world.
type Session struct{}

// World is a communicator adapter for MPI_COMM_WORLD.
type World struct{}

// Request holds an MPI_Request for non-blocking ops.
type Request struct {
	req C.MPI_Request
}

// Start initializes MPI. args is unused (MPI_Init(NULL, NULL)); kept for call-site clarity.
func Start(args []string) (*Session, *World, error) {
	_ = args
	var initialized C.int
	if C.MPI_Initialized(&initialized) != C.MPI_SUCCESS {
		return nil, nil, fmt.Errorf("MPI_Initialized failed")
	}
	if initialized != 0 {
		return &Session{}, &World{}, nil
	}
	if rc := C.MPI_Init(nil, nil); rc != C.MPI_SUCCESS {
		return nil, nil, fmt.Errorf("MPI_Init failed: %d", int(rc))
	}
	return &Session{}, &World{}, nil
}

// Stop finalizes MPI.
func (s *Session) Stop() {
	var finalized C.int
	C.MPI_Finalized(&finalized)
	if finalized == 0 {
		C.MPI_Finalize()
	}
}

func (w *World) Rank() int {
	var r C.int
	C.MPI_Comm_rank(C.MPI_COMM_WORLD, &r)
	return int(r)
}

func (w *World) Size() int {
	var s C.int
	C.MPI_Comm_size(C.MPI_COMM_WORLD, &s)
	return int(s)
}

func (w *World) Wtime() float64 {
	return float64(C.MPI_Wtime())
}

func (w *World) Barrier() {
	C.MPI_Barrier(C.MPI_COMM_WORLD)
}

func (w *World) Send(buf []float64, dest, tag int) {
	ptr := unsafe.Pointer(nil)
	if len(buf) > 0 {
		ptr = unsafe.Pointer(&buf[0])
	}
	C.MPI_Send(ptr, C.int(len(buf)), C.MPI_DOUBLE, C.int(dest), C.int(tag), C.MPI_COMM_WORLD)
}

func (w *World) Recv(buf []float64, source, tag int) {
	ptr := unsafe.Pointer(nil)
	if len(buf) > 0 {
		ptr = unsafe.Pointer(&buf[0])
	}
	C.MPI_Recv(ptr, C.int(len(buf)), C.MPI_DOUBLE, C.int(source), C.int(tag), C.MPI_COMM_WORLD, C.MPI_STATUS_IGNORE)
}

func (w *World) Isend(buf []float64, dest, tag int) domain.Request {
	r := &Request{}
	ptr := unsafe.Pointer(nil)
	if len(buf) > 0 {
		ptr = unsafe.Pointer(&buf[0])
	}
	C.MPI_Isend(ptr, C.int(len(buf)), C.MPI_DOUBLE, C.int(dest), C.int(tag), C.MPI_COMM_WORLD, &r.req)
	return r
}

func (w *World) Irecv(buf []float64, source, tag int) domain.Request {
	r := &Request{}
	ptr := unsafe.Pointer(nil)
	if len(buf) > 0 {
		ptr = unsafe.Pointer(&buf[0])
	}
	C.MPI_Irecv(ptr, C.int(len(buf)), C.MPI_DOUBLE, C.int(source), C.int(tag), C.MPI_COMM_WORLD, &r.req)
	return r
}

func (w *World) Wait(req domain.Request) {
	r := req.(*Request)
	C.MPI_Wait(&r.req, C.MPI_STATUS_IGNORE)
}

func (w *World) WaitAll(reqs []domain.Request) {
	if len(reqs) == 0 {
		return
	}
	creqs := make([]C.MPI_Request, len(reqs))
	for i, rq := range reqs {
		creqs[i] = rq.(*Request).req
	}
	C.MPI_Waitall(C.int(len(creqs)), &creqs[0], C.MPI_STATUSES_IGNORE)
	for i, rq := range reqs {
		rq.(*Request).req = creqs[i]
	}
}

// Ensure World implements domain.Transport.
var _ domain.Transport = (*World)(nil)
