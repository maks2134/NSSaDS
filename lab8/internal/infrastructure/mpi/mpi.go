//go:build cgo && !windows

package mpi

/*
#cgo pkg-config: ompi-c
#include <mpi.h>
#include <stdlib.h>
static inline char* mpi_native_datarep(void) { return "native"; }
*/
import "C"

import (
	"fmt"
	"unsafe"

	"NSSaDS/lab8/internal/domain"
)

// Session wraps an initialized MPI runtime.
type Session struct{}

// Comm is an MPI communicator adapter.
type Comm struct {
	c    C.MPI_Comm
	owns bool
}

// Request holds an MPI_Request for non-blocking ops.
type Request struct {
	req C.MPI_Request
}

// File wraps MPI_File.
type File struct {
	fh   C.MPI_File
	comm C.MPI_Comm
}

// Start initializes MPI and returns COMM_WORLD.
func Start(args []string) (*Session, *Comm, error) {
	_ = args
	var initialized C.int
	if C.MPI_Initialized(&initialized) != C.MPI_SUCCESS {
		return nil, nil, fmt.Errorf("MPI_Initialized failed")
	}
	if initialized == 0 {
		if rc := C.MPI_Init(nil, nil); rc != C.MPI_SUCCESS {
			return nil, nil, fmt.Errorf("MPI_Init failed: %d", int(rc))
		}
	}
	return &Session{}, &Comm{c: C.MPI_COMM_WORLD, owns: false}, nil
}

// Stop finalizes MPI.
func (s *Session) Stop() {
	var finalized C.int
	C.MPI_Finalized(&finalized)
	if finalized == 0 {
		C.MPI_Finalize()
	}
}

func (c *Comm) Rank() int {
	var r C.int
	C.MPI_Comm_rank(c.c, &r)
	return int(r)
}

func (c *Comm) Size() int {
	var s C.int
	C.MPI_Comm_size(c.c, &s)
	return int(s)
}

func (c *Comm) Wtime() float64 {
	return float64(C.MPI_Wtime())
}

func (c *Comm) Barrier() {
	C.MPI_Barrier(c.c)
}

// Split creates a subgroup communicator (caller must Free).
func (c *Comm) Split(color, key int) *Comm {
	var out C.MPI_Comm
	C.MPI_Comm_split(c.c, C.int(color), C.int(key), &out)
	return &Comm{c: out, owns: true}
}

// Free releases a communicator created by Split.
func (c *Comm) Free() {
	if c == nil || !c.owns {
		return
	}
	C.MPI_Comm_free(&c.c)
	c.owns = false
}

func floatPtr(buf []float64) unsafe.Pointer {
	if len(buf) == 0 {
		return nil
	}
	return unsafe.Pointer(&buf[0])
}

func bytePtr(buf []byte) unsafe.Pointer {
	if len(buf) == 0 {
		return nil
	}
	return unsafe.Pointer(&buf[0])
}

func (c *Comm) Send(buf []float64, dest, tag int) {
	C.MPI_Send(floatPtr(buf), C.int(len(buf)), C.MPI_DOUBLE, C.int(dest), C.int(tag), c.c)
}

func (c *Comm) Recv(buf []float64, source, tag int) {
	C.MPI_Recv(floatPtr(buf), C.int(len(buf)), C.MPI_DOUBLE, C.int(source), C.int(tag), c.c, C.MPI_STATUS_IGNORE)
}

func (c *Comm) Isend(buf []float64, dest, tag int) domain.Request {
	r := &Request{}
	C.MPI_Isend(floatPtr(buf), C.int(len(buf)), C.MPI_DOUBLE, C.int(dest), C.int(tag), c.c, &r.req)
	return r
}

func (c *Comm) Irecv(buf []float64, source, tag int) domain.Request {
	r := &Request{}
	C.MPI_Irecv(floatPtr(buf), C.int(len(buf)), C.MPI_DOUBLE, C.int(source), C.int(tag), c.c, &r.req)
	return r
}

func (c *Comm) Wait(req domain.Request) {
	r := req.(*Request)
	C.MPI_Wait(&r.req, C.MPI_STATUS_IGNORE)
}

func (c *Comm) WaitAll(reqs []domain.Request) {
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

func (c *Comm) BcastFloat64(buf []float64, root int) {
	C.MPI_Bcast(floatPtr(buf), C.int(len(buf)), C.MPI_DOUBLE, C.int(root), c.c)
}

func (c *Comm) BcastInt32(buf []int32, root int) {
	ptr := unsafe.Pointer(nil)
	if len(buf) > 0 {
		ptr = unsafe.Pointer(&buf[0])
	}
	C.MPI_Bcast(ptr, C.int(len(buf)), C.MPI_INT, C.int(root), c.c)
}

func (c *Comm) GathervFloat64(send []float64, recv []float64, recvCounts, displs []int, root int) {
	counts := make([]C.int, len(recvCounts))
	dis := make([]C.int, len(displs))
	for i := range recvCounts {
		counts[i] = C.int(recvCounts[i])
	}
	for i := range displs {
		dis[i] = C.int(displs[i])
	}
	var countsPtr, disPtr *C.int
	var recvPtr unsafe.Pointer
	if len(counts) > 0 {
		countsPtr = &counts[0]
		disPtr = &dis[0]
	}
	if len(recv) > 0 {
		recvPtr = floatPtr(recv)
	}
	C.MPI_Gatherv(floatPtr(send), C.int(len(send)), C.MPI_DOUBLE,
		recvPtr, countsPtr, disPtr, C.MPI_DOUBLE, C.int(root), c.c)
}

func (c *Comm) AllreduceMax(val float64) float64 {
	var in, out C.double
	in = C.double(val)
	C.MPI_Allreduce(unsafe.Pointer(&in), unsafe.Pointer(&out), 1, C.MPI_DOUBLE, C.MPI_MAX, c.c)
	return float64(out)
}

// FileOpen opens path for collective MPI-IO on this communicator.
func (c *Comm) FileOpen(path string, create bool) (domain.MPIFile, error) {
	amode := C.int(C.MPI_MODE_RDWR)
	if create {
		amode |= C.MPI_MODE_CREATE
	}
	cstr := C.CString(path)
	defer C.free(unsafe.Pointer(cstr))
	var fh C.MPI_File
	rc := C.MPI_File_open(c.c, cstr, amode, C.MPI_INFO_NULL, &fh)
	if rc != C.MPI_SUCCESS {
		return nil, fmt.Errorf("MPI_File_open %s: %d", path, int(rc))
	}
	return &File{fh: fh, comm: c.c}, nil
}

func (f *File) Close() {
	C.MPI_File_close(&f.fh)
}

func (f *File) ReadAtAll(buf []byte, offset int64) {
	C.MPI_File_read_at_all(f.fh, C.MPI_Offset(offset), bytePtr(buf), C.int(len(buf)), C.MPI_BYTE, C.MPI_STATUS_IGNORE)
}

func (f *File) WriteAtAll(buf []byte, offset int64) {
	C.MPI_File_write_at_all(f.fh, C.MPI_Offset(offset), bytePtr(buf), C.int(len(buf)), C.MPI_BYTE, C.MPI_STATUS_IGNORE)
}

// ReadSubarrayAll reads a contiguous subarray of a row-major N×N matrix after an 8-byte header.
func (f *File) ReadSubarrayAll(buf []float64, globalRows, globalCols, rowStart, colStart, localRows, localCols int) {
	var dtype C.MPI_Datatype
	sizes := [2]C.int{C.int(globalRows), C.int(globalCols)}
	subsizes := [2]C.int{C.int(localRows), C.int(localCols)}
	starts := [2]C.int{C.int(rowStart), C.int(colStart)}
	C.MPI_Type_create_subarray(2, &sizes[0], &subsizes[0], &starts[0], C.MPI_ORDER_C, C.MPI_DOUBLE, &dtype)
	C.MPI_Type_commit(&dtype)
	defer C.MPI_Type_free(&dtype)

	C.MPI_File_set_view(f.fh, C.MPI_Offset(domain.MatrixHeaderSize), C.MPI_DOUBLE, dtype,
		C.mpi_native_datarep(), C.MPI_INFO_NULL)
	C.MPI_File_read_all(f.fh, floatPtr(buf), C.int(localRows*localCols), C.MPI_DOUBLE, C.MPI_STATUS_IGNORE)

	// Restore default byte view for subsequent ReadAtAll/WriteAtAll.
	C.MPI_File_set_view(f.fh, 0, C.MPI_BYTE, C.MPI_BYTE, C.mpi_native_datarep(), C.MPI_INFO_NULL)
}

var (
	_ domain.Transport  = (*Comm)(nil)
	_ domain.Collective = (*Comm)(nil)
	_ domain.GroupComm  = (*Comm)(nil)
	_ domain.FileOpener = (*Comm)(nil)
	_ domain.MPIFile    = (*File)(nil)
)
