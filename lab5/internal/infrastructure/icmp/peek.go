package icmp

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"time"

	"NSSaDS/lab5/internal/domain"
	"NSSaDS/lab5/pkg/config"
)

type Claimer struct {
	mu      sync.Mutex
	sock    domain.Socket
	workers map[int]struct{}
	buf     []byte
}

func NewClaimer(sock domain.Socket) *Claimer {
	return &Claimer{
		sock:    sock,
		workers: make(map[int]struct{}),
		buf:     make([]byte, config.ReadBufferSize),
	}
}

func (c *Claimer) Register(id int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.workers[id] = struct{}{}
}

func (c *Claimer) Unregister(id int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.workers, id)
}

func (c *Claimer) isRegistered(id int) bool {
	_, ok := c.workers[id]
	return ok
}

func (c *Claimer) SendProbe(p domain.Probe, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if p.TTL > 0 {
		if err := c.sock.SetTTL(p.TTL); err != nil {
			return err
		}
	}
	return c.sock.Send(p.Target, payload)
}

func (c *Claimer) PeekAndClaim(id, seq int) (*domain.Packet, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	n, from, err := c.sock.Peek(c.buf)
	if err != nil {
		return nil, err
	}

	pkt, err := ParsePacket(c.buf[:n], from)
	if err != nil {
		return nil, c.dropCurrent(err)
	}
	if !c.shouldKeep(pkt) {
		return nil, c.dropCurrent(errDropped)
	}
	if !MatchesWorker(pkt, id, seq) {
		return nil, ErrNotForWorker
	}

	n, from, err = c.sock.Recv(c.buf)
	if err != nil {
		return nil, err
	}
	return ParsePacket(c.buf[:n], from)
}

func (c *Claimer) shouldKeep(pkt *domain.Packet) bool {
	wid, ok := PacketWorkerID(pkt)
	return ok && c.isRegistered(wid)
}

func (c *Claimer) dropCurrent(cause error) error {
	_, _, recvErr := c.sock.Recv(c.buf)
	if recvErr != nil {
		return recvErr
	}
	return cause
}

func (c *Claimer) Wait(
	ctx context.Context,
	id, seq int,
	timeout time.Duration,
) (*domain.Packet, error) {
	deadline := time.Now().Add(timeout)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, ErrTimeout
		}

		pkt, err := c.PeekAndClaim(id, seq)
		if err == nil {
			return pkt, nil
		}
		if errors.Is(err, ErrNotForWorker) {
			runtime.Gosched()
			continue
		}
		if !shouldRetryWait(err) {
			return nil, err
		}
	}
}

func shouldRetryWait(err error) bool {
	return errors.Is(err, ErrNotForWorker) ||
		errors.Is(err, errDropped) ||
		errors.Is(err, ErrTimeout) ||
		isTimeout(err)
}
