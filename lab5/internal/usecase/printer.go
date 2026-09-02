package usecase

import (
	"fmt"
	"io"
	"sync"
)

type SyncPrinter struct {
	mu sync.Mutex
	w  io.Writer
}

func NewPrinter(w io.Writer) *SyncPrinter {
	return &SyncPrinter{w: w}
}

func (p *SyncPrinter) Printf(format string, args ...any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintf(p.w, format, args...)
}
