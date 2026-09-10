package domain

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
)

const MatrixHeaderSize = 8 // int64 N, little-endian

// WriteMatrixFile writes N (int64 LE) then N*N float64 row-major.
func WriteMatrixFile(path string, m *Matrix) error {
	if m.Rows != m.Cols {
		return fmt.Errorf("matrix must be square, got %dx%d", m.Rows, m.Cols)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	n := int64(m.Rows)
	if err := binary.Write(f, binary.LittleEndian, n); err != nil {
		return err
	}
	return binary.Write(f, binary.LittleEndian, m.Data)
}

// ReadMatrixFile reads a matrix written by WriteMatrixFile.
func ReadMatrixFile(path string) (*Matrix, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var n int64
	if err := binary.Read(f, binary.LittleEndian, &n); err != nil {
		return nil, err
	}
	if n <= 0 {
		return nil, fmt.Errorf("invalid N=%d in %s", n, path)
	}
	m := NewMatrix(int(n), int(n))
	if err := binary.Read(f, binary.LittleEndian, m.Data); err != nil {
		return nil, err
	}
	return m, nil
}

// ReadMatrixHeader reads only the N header from path.
func ReadMatrixHeader(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	var n int64
	if err := binary.Read(f, binary.LittleEndian, &n); err != nil {
		return 0, err
	}
	if n <= 0 {
		return 0, fmt.Errorf("invalid N=%d", n)
	}
	return int(n), nil
}

// Float64SliceAsBytes reinterprets []float64 as little-endian bytes (same machine).
func Float64SliceAsBytes(data []float64) []byte {
	if len(data) == 0 {
		return nil
	}
	nbytes := len(data) * 8
	b := make([]byte, nbytes)
	for i, v := range data {
		binary.LittleEndian.PutUint64(b[i*8:], math.Float64bits(v))
	}
	return b
}

// BytesAsFloat64Slice converts little-endian bytes to []float64.
func BytesAsFloat64Slice(b []byte) ([]float64, error) {
	if len(b)%8 != 0 {
		return nil, fmt.Errorf("byte length %d not multiple of 8", len(b))
	}
	out := make([]float64, len(b)/8)
	for i := range out {
		out[i] = math.Float64frombits(binary.LittleEndian.Uint64(b[i*8:]))
	}
	return out, nil
}

// WriteHeaderOnly creates/truncates path and writes N header (data region zero-filled by writer later).
func WriteHeaderOnly(path string, n int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := binary.Write(f, binary.LittleEndian, int64(n)); err != nil {
		return err
	}
	// Pre-size file so collective writes have a valid extent.
	size := int64(MatrixHeaderSize) + int64(n)*int64(n)*8
	return f.Truncate(size)
}

// EnsureDir creates directory if needed.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}
