package domain_test

import (
	"testing"

	"NSSaDS/lab7/internal/domain"
)

func TestPanelGEMMMatchesNaive(t *testing.T) {
	const n = 32
	const panel = 8

	a := domain.NewMatrix(n, n)
	b := domain.NewMatrix(n, n)
	domain.FillSequential(a)
	for i := range b.Data {
		b.Data[i] = float64((i*3)%89) * 0.02
	}

	want := domain.NaiveMultiply(a, b)

	c := domain.NewMatrix(n, n)
	bPanel := make([]float64, n*panel)
	for col := 0; col < n; col += panel {
		end := col + panel
		if end > n {
			end = n
		}
		pw := end - col
		domain.ExtractPanel(b, col, end, bPanel[:n*pw])
		domain.PanelGEMM(a, c, bPanel[:n*pw], n, col, pw)
	}

	if !domain.AlmostEqual(domain.Checksum(c), domain.Checksum(want), 1e-6) {
		t.Fatalf("checksum mismatch: got %v want %v", domain.Checksum(c), domain.Checksum(want))
	}
}

func TestRowBoundsPartition(t *testing.T) {
	n, size := 100, 3
	covered := 0
	for r := 0; r < size; r++ {
		s, e := domain.RowBounds(n, r, size)
		if s < 0 || e > n || e < s {
			t.Fatalf("bad bounds rank=%d: [%d,%d)", r, s, e)
		}
		covered += e - s
	}
	if covered != n {
		t.Fatalf("rows covered=%d want=%d", covered, n)
	}
}

func TestRowBoundsUneven(t *testing.T) {
	// 10 rows, 3 ranks → 4, 3, 3
	cases := []struct{ rank, start, end int }{
		{0, 0, 4},
		{1, 4, 7},
		{2, 7, 10},
	}
	for _, tc := range cases {
		s, e := domain.RowBounds(10, tc.rank, 3)
		if s != tc.start || e != tc.end {
			t.Fatalf("rank %d: got [%d,%d) want [%d,%d)", tc.rank, s, e, tc.start, tc.end)
		}
	}
}
