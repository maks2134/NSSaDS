package usecase

import (
	"fmt"
	"path/filepath"

	"NSSaDS/lab8/internal/domain"
	"NSSaDS/lab8/pkg/config"
)

// GenerateInputFiles writes A.bin and B.bin on the calling process (rank 0).
func GenerateInputFiles(cfg config.Config) error {
	if err := domain.EnsureDir(filepath.Dir(cfg.APath)); err != nil {
		return err
	}
	if err := domain.EnsureDir(filepath.Dir(cfg.BPath)); err != nil {
		return err
	}
	a := domain.NewMatrix(cfg.N, cfg.N)
	b := domain.NewMatrix(cfg.N, cfg.N)
	domain.FillSequential(a)
	domain.FillB(b)
	if err := domain.WriteMatrixFile(cfg.APath, a); err != nil {
		return fmt.Errorf("write A: %w", err)
	}
	if err := domain.WriteMatrixFile(cfg.BPath, b); err != nil {
		return fmt.Errorf("write B: %w", err)
	}
	return nil
}

// LocalData holds the portions of A and B owned by this group rank.
type LocalData struct {
	N         int
	RowStart  int
	RowEnd    int
	ColStart  int
	ColEnd    int
	ALocal    *domain.Matrix // localRows × N
	BLocal    *domain.Matrix // N × localCols
	LocalRows int
	LocalCols int
}

// ReadLocalMatrices reads A row strip and B column strip via collective MPI-IO.
func ReadLocalMatrices(g domain.GroupComm, opener domain.FileOpener, aPath, bPath string, n int) (*LocalData, error) {
	rank := g.Rank()
	size := g.Size()
	rowStart, rowEnd := domain.RowBounds(n, rank, size)
	colStart, colEnd := domain.RowBounds(n, rank, size)
	localRows := rowEnd - rowStart
	localCols := colEnd - colStart

	aLocal := domain.NewMatrix(localRows, n)
	bLocal := domain.NewMatrix(n, localCols)

	if err := readARows(opener, aPath, aLocal, n, rowStart, localRows); err != nil {
		return nil, err
	}
	if err := readBCols(opener, bPath, bLocal, n, colStart, localCols); err != nil {
		return nil, err
	}

	return &LocalData{
		N: n, RowStart: rowStart, RowEnd: rowEnd,
		ColStart: colStart, ColEnd: colEnd,
		ALocal: aLocal, BLocal: bLocal,
		LocalRows: localRows, LocalCols: localCols,
	}, nil
}

func readARows(opener domain.FileOpener, path string, aLocal *domain.Matrix, n, rowStart, localRows int) error {
	f, err := opener.FileOpen(path, false)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := make([]byte, localRows*n*8)
	offset := int64(domain.MatrixHeaderSize) + int64(rowStart)*int64(n)*8
	f.ReadAtAll(buf, offset)
	vals, err := domain.BytesAsFloat64Slice(buf)
	if err != nil {
		return err
	}
	copy(aLocal.Data, vals)
	return nil
}

func readBCols(opener domain.FileOpener, path string, bLocal *domain.Matrix, n, colStart, localCols int) error {
	f, err := opener.FileOpen(path, false)
	if err != nil {
		return err
	}
	defer f.Close()
	f.ReadSubarrayAll(bLocal.Data, n, n, 0, colStart, n, localCols)
	return nil
}

// WriteGroupResult writes C rows into outdir/group-{color}.bin via collective MPI-IO.
func WriteGroupResult(g domain.GroupComm, opener domain.FileOpener, outDir string, color int, data *LocalData, cLocal *domain.Matrix) error {
	path := filepath.Join(outDir, fmt.Sprintf("group-%d.bin", color))
	if g.Rank() == 0 {
		if err := domain.EnsureDir(outDir); err != nil {
			return err
		}
		if err := domain.WriteHeaderOnly(path, data.N); err != nil {
			return err
		}
	}
	g.Barrier()

	f, err := opener.FileOpen(path, false)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := domain.Float64SliceAsBytes(cLocal.Data)
	offset := int64(domain.MatrixHeaderSize) + int64(data.RowStart)*int64(data.N)*8
	f.WriteAtAll(buf, offset)
	return nil
}
