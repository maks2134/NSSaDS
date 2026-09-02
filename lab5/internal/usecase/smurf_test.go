package usecase

import (
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"NSSaDS/lab5/internal/domain"
	"NSSaDS/lab5/pkg/config"
)

func TestClampSmurfCount(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		n        int
		expected int
	}{
		{name: "default on zero", n: 0, expected: config.DefaultSmurfCount},
		{name: "default on negative", n: -1, expected: config.DefaultSmurfCount},
		{name: "within cap", n: 2, expected: 2},
		{name: "at cap", n: 5, expected: config.MaxSmurfCount},
		{name: "above cap", n: 100, expected: config.MaxSmurfCount},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ClampSmurfCount(tc.n, config.DefaultSmurfCount, config.MaxSmurfCount)
			if got != tc.expected {
				t.Fatalf("got %d want %d", got, tc.expected)
			}
		})
	}
}

func TestWorkerIDNonZero(t *testing.T) {
	t.Parallel()
	for i := 0; i < 16; i++ {
		if id := WorkerID(i); id == 0 {
			t.Fatalf("worker %d id is 0", i)
		}
	}
}

func TestHandleEchoReply(t *testing.T) {
	t.Parallel()
	pkt := &domain.Packet{
		From: net.IPv4(8, 8, 8, 8),
		TTL:  117,
		Seq:  1,
		Size: 84,
	}
	got := HandleEchoReply("dns.google", pkt, 12*time.Millisecond)
	for _, want := range []string{"dns.google", "icmp_seq=1", "ttl=117", "12.000 ms"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
}

func TestHandleUnreachable(t *testing.T) {
	t.Parallel()
	pkt := &domain.Packet{From: net.IPv4(192, 168, 0, 1), Code: 1}
	got := HandleUnreachable("10.0.0.1", pkt)
	if !strings.Contains(got, "unreachable") {
		t.Fatalf("got %q", got)
	}
}

func TestFormatStats(t *testing.T) {
	t.Parallel()
	st := domain.HostStats{
		Host:        "example",
		Transmitted: 4,
		Received:    2,
		RTTs:        []time.Duration{time.Millisecond, 3 * time.Millisecond},
	}
	got := FormatStats(st)
	if !strings.Contains(got, "50%") {
		t.Fatalf("loss: %q", got)
	}
}

type fakeRaw struct {
	packets [][]byte
}

func (f *fakeRaw) SendIP(_ net.IP, packet []byte) error {
	f.packets = append(f.packets, append([]byte(nil), packet...))
	return nil
}

func (f *fakeRaw) Close() error { return nil }

func TestSendOneSpoofsSource(t *testing.T) {
	t.Parallel()
	raw := &fakeRaw{}
	s := NewSmurfer(config.New(), NewPrinter(io.Discard))
	victim := net.IPv4(192, 168, 1, 10)
	bcast := net.IPv4(192, 168, 1, 255)
	if err := s.sendOne(raw, victim, bcast, 1); err != nil {
		t.Fatal(err)
	}
	if len(raw.packets) != 1 {
		t.Fatalf("sent %d", len(raw.packets))
	}
	pkt := raw.packets[0]
	src := net.IPv4(pkt[12], pkt[13], pkt[14], pkt[15])
	dst := net.IPv4(pkt[16], pkt[17], pkt[18], pkt[19])
	if !src.Equal(victim) {
		t.Fatalf("source %s want %s", src, victim)
	}
	if !dst.Equal(bcast) {
		t.Fatalf("dest %s want %s", dst, bcast)
	}
	if pkt[9] != 1 {
		t.Fatalf("protocol %d want icmp", pkt[9])
	}
}
