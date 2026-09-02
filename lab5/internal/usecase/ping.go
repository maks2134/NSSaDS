package usecase

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"NSSaDS/lab5/internal/domain"
	"NSSaDS/lab5/internal/infrastructure/icmp"
	"NSSaDS/lab5/pkg/config"
)

type Pinger struct {
	cfg   *config.Config
	out   domain.Printer
	trace bool
}

func NewPinger(cfg *config.Config, out domain.Printer, trace bool) *Pinger {
	return &Pinger{cfg: cfg, out: out, trace: trace}
}

func (p *Pinger) Run(ctx context.Context, hosts []string) error {
	targets, err := ResolveAll(hosts)
	if err != nil {
		return err
	}
	sock, err := icmp.Open()
	if err != nil {
		return fmt.Errorf("open icmp: %w", err)
	}
	defer sock.Close()
	if err := sock.SetReadTimeout(p.cfg.PeekTimeout); err != nil {
		return fmt.Errorf("set timeout: %w", err)
	}

	claimer := icmp.NewClaimer(sock)
	var wg sync.WaitGroup
	for i, tgt := range targets {
		wg.Add(1)
		go func(index int, tgt Target) {
			defer wg.Done()
			p.runHost(ctx, claimer, index, tgt)
		}(i, tgt)
	}
	wg.Wait()
	return nil
}

func (p *Pinger) runHost(
	ctx context.Context,
	claimer *icmp.Claimer,
	index int,
	tgt Target,
) {
	id := WorkerID(index)
	claimer.Register(id)
	defer claimer.Unregister(id)

	if p.trace {
		NewTracer(p.cfg, p.out).runHost(ctx, claimer, id, tgt)
	}
	p.pingHost(ctx, claimer, id, tgt)
}

func (p *Pinger) pingHost(
	ctx context.Context,
	claimer *icmp.Claimer,
	id int,
	tgt Target,
) {
	stats := domain.HostStats{
		Host: tgt.Host,
		Addr: tgt.IP,
		RTTs: []time.Duration{},
	}
	p.out.Printf("PING %s (%s)\n", tgt.Host, tgt.IP)

	seqBase := 0
	if p.trace {
		seqBase = p.cfg.MaxTTL
	}
	for n := 1; n <= p.cfg.Count; n++ {
		if ctx.Err() != nil {
			return
		}
		p.probe(ctx, claimer, id, seqBase+n, tgt, &stats)
		if n < p.cfg.Count {
			sleepCtx(ctx, p.cfg.Interval)
		}
	}
	p.out.Printf("%s\n", FormatStats(stats))
}

func (p *Pinger) probe(
	ctx context.Context,
	claimer *icmp.Claimer,
	id, seq int,
	tgt Target,
	stats *domain.HostStats,
) {
	pkt, err := sendAndWait(ctx, claimer, p.cfg, id, seq, tgt.IP, p.cfg.DefaultTTL)
	stats.Transmitted++
	if err != nil {
		p.reportProbeError(tgt.Host, seq, err)
		return
	}
	dispatchReply(p.out, tgt.Host, pkt, stats)
}

func (p *Pinger) reportProbeError(host string, seq int, err error) {
	if errors.Is(err, icmp.ErrTimeout) || errors.Is(err, context.DeadlineExceeded) {
		p.out.Printf("Request timeout for %s icmp_seq=%d\n", host, seq)
		return
	}
	p.out.Printf("%s icmp_seq=%d: %v\n", host, seq, err)
}

func sendAndWait(
	ctx context.Context,
	claimer *icmp.Claimer,
	cfg *config.Config,
	id, seq int,
	dst net.IP,
	ttl int,
) (*domain.Packet, error) {
	sent := time.Now()
	payload, err := icmp.BuildEchoRequest(id, seq, sent, cfg.PayloadSize)
	if err != nil {
		return nil, fmt.Errorf("build echo: %w", err)
	}
	probe := domain.Probe{Target: dst, ID: id, Seq: seq, TTL: ttl}
	if err := claimer.SendProbe(probe, payload); err != nil {
		return nil, fmt.Errorf("send: %w", err)
	}
	return claimer.Wait(ctx, id, seq, cfg.Timeout)
}
