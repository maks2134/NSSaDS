package domain

// Transport is the point-to-point MPI surface used by matrix multiply.
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

// Collective is the collective MPI surface (Bcast, Gatherv, Allreduce).
type Collective interface {
	BcastFloat64(buf []float64, root int)
	BcastInt32(buf []int32, root int)
	GathervFloat64(send []float64, recv []float64, recvCounts, displs []int, root int)
	AllreduceMax(val float64) float64
}

// GroupComm combines Transport and Collective for a group communicator.
type GroupComm interface {
	Transport
	Collective
}

// Request is an opaque handle for a non-blocking transfer.
type Request any

// MPIFile is a collective MPI-IO file handle.
type MPIFile interface {
	ReadAtAll(buf []byte, offset int64)
	WriteAtAll(buf []byte, offset int64)
	ReadSubarrayAll(buf []float64, globalRows, globalCols, rowStart, colStart, localRows, localCols int)
	Close()
}

// FileOpener opens an MPI file on a communicator.
type FileOpener interface {
	FileOpen(path string, create bool) (MPIFile, error)
}
