package icmp

import (
	"net"
	"testing"
	"time"

	"golang.org/x/net/ipv4"

	"NSSaDS/lab5/internal/domain"
)

type fakeSock struct {
	packets [][]byte
	from    []net.IP
}

func (f *fakeSock) Send(net.IP, []byte) error { return nil }

func (f *fakeSock) Peek(buf []byte) (int, net.IP, error) {
	if len(f.packets) == 0 {
		return 0, nil, ErrTimeout
	}
	n := copy(buf, f.packets[0])
	return n, f.from[0], nil
}

func (f *fakeSock) Recv(buf []byte) (int, net.IP, error) {
	if len(f.packets) == 0 {
		return 0, nil, ErrTimeout
	}
	n := copy(buf, f.packets[0])
	from := f.from[0]
	f.packets = f.packets[1:]
	f.from = f.from[1:]
	return n, from, nil
}

func (f *fakeSock) SetTTL(int) error                   { return nil }
func (f *fakeSock) SetReadTimeout(time.Duration) error { return nil }
func (f *fakeSock) Close() error                       { return nil }

func TestPeekAndClaimTakesOwnPacket(t *testing.T) {
	t.Parallel()
	src := net.IPv4(8, 8, 8, 8)
	body := mustEcho(t, ipv4.ICMPTypeEchoReply, 11, 1, BuildPayload(time.Now(), 8))
	sock := &fakeSock{
		packets: [][]byte{wrapIPv4(t, src, body, 64)},
		from:    []net.IP{src},
	}
	c := NewClaimer(sock)
	c.Register(11)

	pkt, err := c.PeekAndClaim(11, 1)
	if err != nil {
		t.Fatal(err)
	}
	if pkt.Kind != domain.KindEchoReply || pkt.ID != 11 {
		t.Fatalf("unexpected packet: %+v", pkt)
	}
	if len(sock.packets) != 0 {
		t.Fatal("packet was not consumed")
	}
}

func TestPeekAndClaimLeavesForeignPacket(t *testing.T) {
	t.Parallel()
	src := net.IPv4(1, 1, 1, 1)
	body := mustEcho(t, ipv4.ICMPTypeEchoReply, 22, 1, BuildPayload(time.Now(), 8))
	sock := &fakeSock{
		packets: [][]byte{wrapIPv4(t, src, body, 64)},
		from:    []net.IP{src},
	}
	c := NewClaimer(sock)
	c.Register(11)
	c.Register(22)

	_, err := c.PeekAndClaim(11, 1)
	if err != ErrNotForWorker {
		t.Fatalf("got %v want %v", err, ErrNotForWorker)
	}
	if len(sock.packets) != 1 {
		t.Fatal("foreign packet must stay in the buffer")
	}
}

func TestPeekAndClaimDropsUnrelated(t *testing.T) {
	t.Parallel()
	src := net.IPv4(9, 9, 9, 9)
	body := mustEcho(t, ipv4.ICMPTypeEchoReply, 99, 1, BuildPayload(time.Now(), 8))
	sock := &fakeSock{
		packets: [][]byte{wrapIPv4(t, src, body, 64)},
		from:    []net.IP{src},
	}
	c := NewClaimer(sock)
	c.Register(11)

	_, err := c.PeekAndClaim(11, 1)
	if err != errDropped {
		t.Fatalf("got %v want %v", err, errDropped)
	}
	if len(sock.packets) != 0 {
		t.Fatal("unrelated packet must be dropped")
	}
}
