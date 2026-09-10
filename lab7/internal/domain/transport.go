package domain

// Transport is the point-to-point MPI surface used by matrix multiply.
// Collective ops are intentionally absent (except Barrier for timing sync).
type Transport interface {
	Rank() int
	Size() int
	Wtime() float64
	Barrier()

	Send(buf []float64, dest, tag int)
	Recv(buf []float64, source, tag int)

	Isend(buf []float64, dest, tag int) Request
	Irecv(buf []float64, source, tag int) Request
	Wait(req Request)
	WaitAll(reqs []Request)
}

// Request is an opaque handle for a non-blocking transfer
// (concrete type lives in infrastructure/mpi).
type Request any
