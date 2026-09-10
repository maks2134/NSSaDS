package domain_test

import (
	"path/filepath"
	"testing"

	"NSSaDS/lab8/internal/domain"
)

func TestPanelGEMMMatchesNaive(t *testing.T) {
	const n = 32
	const panel = 8

	a := domain.NewMatrix(n, n)
	b := domain.NewMatrix(n, n)
	domain.FillSequential(a)
	domain.FillB(b)

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

func TestAssignColorsDisjoint(t *testing.T) {
	p, k := 8, 3
	colors := domain.AssignColors(p, k, 42)
	if len(colors) != p {
		t.Fatalf("len=%d want %d", len(colors), p)
	}
	counts := make([]int, k)
	seen := make([]bool, p)
	for r, c := range colors {
		if int(c) < 0 || int(c) >= k {
			t.Fatalf("rank %d color %d out of range", r, c)
		}
		counts[c]++
		if seen[r] {
			t.Fatalf("duplicate rank %d", r)
		}
		seen[r] = true
	}
	for color, n := range counts {
		if n == 0 {
			t.Fatalf("empty group color=%d", color)
		}
	}
}

func TestAssignColorsKGreaterThanP(t *testing.T) {
	colors := domain.AssignColors(3, 5, 1)
	for r, c := range colors {
		if int(c) != r {
			t.Fatalf("rank %d: color=%d want %d", r, c, r)
		}
	}
}

func TestAssignColorsReproducible(t *testing.T) {
	a := domain.AssignColors(10, 4, 99)
	b := domain.AssignColors(10, 4, 99)
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("not reproducible at %d", i)
		}
	}
}

func TestMatrixFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "M.bin")
	m := domain.NewMatrix(4, 4)
	domain.FillSequential(m)
	if err := domain.WriteMatrixFile(path, m); err != nil {
		t.Fatal(err)
	}
	got, err := domain.ReadMatrixFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Rows != 4 || len(got.Data) != 16 {
		t.Fatalf("bad shape %dx%d len=%d", got.Rows, got.Cols, len(got.Data))
	}
	if !domain.AlmostEqual(domain.Checksum(got), domain.Checksum(m), 0) {
		t.Fatalf("checksum mismatch after round-trip")
	}
	n, err := domain.ReadMatrixHeader(path)
	if err != nil || n != 4 {
		t.Fatalf("header N=%d err=%v", n, err)
	}
}

func TestFloat64BytesRoundTrip(t *testing.T) {
	in := []float64{1.5, -2.25, 0}
	b := domain.Float64SliceAsBytes(in)
	out, err := domain.BytesAsFloat64Slice(b)
	if err != nil {
		t.Fatal(err)
	}
	for i := range in {
		if in[i] != out[i] {
			t.Fatalf("at %d: %v != %v", i, in[i], out[i])
		}
	}
}

func TestGroupMembers(t *testing.T) {
	colors := []int32{0, 1, 0, 1, 0}
	m := domain.GroupMembers(colors, 0)
	if len(m) != 3 || m[0] != 0 || m[1] != 2 || m[2] != 4 {
		t.Fatalf("members=%v", m)
	}
}
