package domain

import "math"

// Matrix is a dense row-major float64 matrix of size Rows×Cols.
type Matrix struct {
	Rows, Cols int
	Data       []float64
}

// NewMatrix allocates a zero matrix.
func NewMatrix(rows, cols int) *Matrix {
	return &Matrix{
		Rows: rows,
		Cols: cols,
		Data: make([]float64, rows*cols),
	}
}

// At returns the element at (i, j).
func (m *Matrix) At(i, j int) float64 {
	return m.Data[i*m.Cols+j]
}

// Set sets the element at (i, j).
func (m *Matrix) Set(i, j int, v float64) {
	m.Data[i*m.Cols+j] = v
}

// FillSequential fills A[i,j] = float64((i*cols+j)%97) * 0.01 for reproducible checksums.
func FillSequential(m *Matrix) {
	for i := 0; i < m.Rows; i++ {
		for j := 0; j < m.Cols; j++ {
			m.Data[i*m.Cols+j] = float64((i*m.Cols+j)%97) * 0.01
		}
	}
}

// FillB fills B with the lab7 pattern (i*3)%89 * 0.02.
func FillB(m *Matrix) {
	for i := range m.Data {
		m.Data[i] = float64((i*3)%89) * 0.02
	}
}

// RowSlice returns a view of rows [rowStart, rowEnd) as a contiguous slice
// (row-major). The returned slice aliases m.Data.
func (m *Matrix) RowSlice(rowStart, rowEnd int) []float64 {
	return m.Data[rowStart*m.Cols : rowEnd*m.Cols]
}

// ExtractPanel extracts columns [colStart, colEnd) into dst (size Rows×panelW),
// packed row-major as rows×panelW: dst[i*panelW+k] = m[i, colStart+k].
func ExtractPanel(m *Matrix, colStart, colEnd int, dst []float64) {
	panelW := colEnd - colStart
	for i := 0; i < m.Rows; i++ {
		src := m.Data[i*m.Cols+colStart : i*m.Cols+colEnd]
		copy(dst[i*panelW:(i+1)*panelW], src)
	}
}

// PanelGEMM accumulates C[:, colStart:colEnd] += A_local * B_panel
// where A is aLocalRows×N, B_panel is N×panelW packed, C is aLocalRows×N.
func PanelGEMM(aLocal, cLocal *Matrix, bPanel []float64, n, colStart, panelW int) {
	rows := aLocal.Rows
	for i := 0; i < rows; i++ {
		aRow := aLocal.Data[i*n : (i+1)*n]
		cRow := cLocal.Data[i*n : (i+1)*n]
		for k := 0; k < n; k++ {
			aik := aRow[k]
			if aik == 0 {
				continue
			}
			bp := bPanel[k*panelW : (k+1)*panelW]
			for j := 0; j < panelW; j++ {
				cRow[colStart+j] += aik * bp[j]
			}
		}
	}
}

// NaiveMultiply computes C = A*B for square (or conforming) matrices. Used in tests.
func NaiveMultiply(a, b *Matrix) *Matrix {
	c := NewMatrix(a.Rows, b.Cols)
	for i := 0; i < a.Rows; i++ {
		for k := 0; k < a.Cols; k++ {
			aik := a.At(i, k)
			for j := 0; j < b.Cols; j++ {
				c.Data[i*c.Cols+j] += aik * b.At(k, j)
			}
		}
	}
	return c
}

// Checksum returns a stable scalar for comparing results across modes.
func Checksum(m *Matrix) float64 {
	var s float64
	for i, v := range m.Data {
		s += v * float64(i%17+1)
	}
	if s == 0 {
		return 0
	}
	return s
}

// AlmostEqual reports whether two checksums match within absTol.
func AlmostEqual(a, b, absTol float64) bool {
	return math.Abs(a-b) <= absTol
}

// RowBounds returns [start, end) row range for rank among size processes over n rows.
func RowBounds(n, rank, size int) (start, end int) {
	base := n / size
	extra := n % size
	start = rank*base + min(rank, extra)
	rows := base
	if rank < extra {
		rows++
	}
	end = start + rows
	return start, end
}
