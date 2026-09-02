package usecase

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"NSSaDS/lab5/internal/domain"
	"NSSaDS/lab5/internal/infrastructure/icmp"
	"NSSaDS/lab5/pkg/config"
)

type Tracer struct {
	cfg *config.Config
	out domain.Printer
}

func NewTracer(cfg *config.Config, out domain.Printer) *Tracer {
	return &Tracer{cfg: cfg, out: out}
}

func (t *Tracer) Run(ctx context.Context, hosts []string) error {
	targets, err := ResolveAll(hosts)
	if err != nil {
		return err
	}
	sock, err := icmp.Open()
	if err != nil {
		return fmt.Errorf("open icmp: %w", err)
	}
	defer sock.Close()
	if err := sock.SetReadTimeout(t.cfg.PeekTimeout); err != nil {
		return fmt.Errorf("set timeout: %w", err)
	}

	claimer := icmp.NewClaimer(sock)
	var wg sync.WaitGroup
	for i, tgt := range targets {
		wg.Add(1)
		go func(index int, tgt Target) {
			defer wg.Done()
			id := WorkerID(index)
			claimer.Register(id)
			defer claimer.Unregister(id)
			t.runHost(ctx, claimer, id, tgt)
		}(i, tgt)
	}
	wg.Wait()
	return nil
}

func (t *Tracer) runHost(
	ctx context.Context,
	claimer *icmp.Claimer,
	id int,
	tgt Target,
) {
	t.out.Printf("=== traceroute %s (%s) ===\n", tgt.Host, tgt.IP)
	for hop := 1; hop <= t.cfg.MaxTTL; hop++ {
		if ctx.Err() != nil {
			return
		}
		hopRes := t.probeHop(ctx, claimer, id, hop, tgt)
		if hopRes.Kind == domain.KindEchoReply || hopRes.Kind == domain.KindUnreachable {
			return
		}
	}
}

func (t *Tracer) probeHop(
	ctx context.Context,
	claimer *icmp.Claimer,
	id, hop int,
	tgt Target,
) domain.Hop {
	pkt, err := sendAndWait(ctx, claimer, t.cfg, id, hop, tgt.IP, hop)
	if err != nil {
		t.reportHopError(hop, tgt.Host, err)
		return domain.Hop{Number: hop, Missed: true}
	}
	now := time.Now()
	result := domain.Hop{
		Number: hop,
		Addr:   pkt.From,
		RTT:    ProbeRTT(pkt, now),
		Kind:   pkt.Kind,
	}
	t.printHop(tgt.Host, hop, pkt, result.RTT)
	return result
}

func (t *Tracer) reportHopError(hop int, host string, err error) {
	if errors.Is(err, icmp.ErrTimeout) || errors.Is(err, context.DeadlineExceeded) {
		t.out.Printf("%2d  *  (timeout, dest %s)\n", hop, host)
		return
	}
	t.out.Printf("%2d  error: %v\n", hop, err)
}

func (t *Tracer) printHop(host string, hop int, pkt *domain.Packet, rtt time.Duration) {
	switch pkt.Kind {
	case domain.KindTimeExceeded:
		t.out.Printf("%s\n", HandleTimeExceeded(host, hop, pkt, rtt))
	case domain.KindEchoReply:
		t.out.Printf("%2d  %s  %s  (echo-reply)\n", hop, pkt.From, FormatRTT(rtt))
	case domain.KindUnreachable:
		t.out.Printf("%2d  %s\n", hop, HandleUnreachable(host, pkt))
	default:
		t.out.Printf("%2d  %s  unexpected icmp\n", hop, pkt.From)
	}
}
