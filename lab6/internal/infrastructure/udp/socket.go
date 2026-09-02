package udp

import (
	"context"
	"fmt"
	"net"
	"syscall"
	"time"

	"NSSaDS/lab6/internal/domain"
	"NSSaDS/lab6/pkg/config"

	"golang.org/x/net/ipv4"
)

type Socket struct {
	conn      *net.UDPConn
	pc        *ipv4.PacketConn
	iface     domain.IfaceInfo
	port      int
	group     net.IP
	bcastAddr *net.UDPAddr
	mcastAddr *net.UDPAddr
}

func Open(iface domain.IfaceInfo, port int, group string, timeout time.Duration) (*Socket, error) {
	lc := net.ListenConfig{
		Control: func(_ string, _ string, c syscall.RawConn) error {
			var opErr error
			err := c.Control(func(fd uintptr) {
				opErr = applySocketOptions(rawFD(int(fd)))
			})
			if err != nil {
				return err
			}
			return opErr
		},
	}
	addr := &net.UDPAddr{IP: net.IPv4zero, Port: port}
	conn, err := lc.ListenPacket(context.Background(), "udp4", addr.String())
	if err != nil {
		return nil, fmt.Errorf("listen udp on port %d: %w (stop other lab6 or use -port)", port, err)
	}
	udpConn, ok := conn.(*net.UDPConn)
	if !ok {
		_ = conn.Close()
		return nil, fmt.Errorf("expected UDPConn")
	}
	if err := udpConn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		_ = udpConn.Close()
		return nil, fmt.Errorf("set read deadline: %w", err)
	}
	groupIP := net.ParseIP(group).To4()
	if groupIP == nil {
		_ = udpConn.Close()
		return nil, fmt.Errorf("invalid multicast group %q", group)
	}
	s := &Socket{
		conn:      udpConn,
		pc:        ipv4.NewPacketConn(udpConn),
		iface:     iface,
		port:      port,
		group:     groupIP,
		bcastAddr: &net.UDPAddr{IP: iface.Broadcast, Port: port},
		mcastAddr: &net.UDPAddr{IP: groupIP, Port: port},
	}
	if err := s.configureMulticast(); err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}

func (s *Socket) configureMulticast() error {
	iface, err := net.InterfaceByName(s.iface.Name)
	if err != nil {
		return fmt.Errorf("interface by name: %w", err)
	}
	if err := s.pc.SetMulticastInterface(iface); err != nil {
		return fmt.Errorf("set multicast interface: %w", err)
	}
	if err := s.pc.SetMulticastTTL(config.MulticastTTL); err != nil {
		return fmt.Errorf("set multicast ttl: %w", err)
	}
	if err := s.pc.SetMulticastLoopback(false); err != nil {
		return fmt.Errorf("set multicast loopback: %w", err)
	}
	return nil
}

func (s *Socket) JoinGroup(group net.IP) error {
	iface, err := net.InterfaceByName(s.iface.Name)
	if err != nil {
		return fmt.Errorf("interface by name: %w", err)
	}
	if err := s.pc.JoinGroup(iface, &net.UDPAddr{IP: group}); err != nil {
		return fmt.Errorf("join group: %w", err)
	}
	return nil
}

func (s *Socket) LeaveGroup(group net.IP) error {
	iface, err := net.InterfaceByName(s.iface.Name)
	if err != nil {
		return fmt.Errorf("interface by name: %w", err)
	}
	if err := s.pc.LeaveGroup(iface, &net.UDPAddr{IP: group}); err != nil {
		return fmt.Errorf("leave group: %w", err)
	}
	return nil
}

func (s *Socket) SendBroadcast(data []byte) error {
	_, err := s.conn.WriteToUDP(data, s.bcastAddr)
	if err != nil {
		return fmt.Errorf("send broadcast: %w", err)
	}
	return nil
}

func (s *Socket) SendMulticast(data []byte) error {
	_, err := s.conn.WriteToUDP(data, s.mcastAddr)
	if err != nil {
		return fmt.Errorf("send multicast: %w", err)
	}
	return nil
}

func (s *Socket) Send(data []byte, mode domain.Mode) error {
	switch mode {
	case domain.ModeBroadcast:
		return s.SendBroadcast(data)
	case domain.ModeMulticast:
		return s.SendMulticast(data)
	default:
		return fmt.Errorf("unknown mode")
	}
}

func (s *Socket) Recv(buf []byte) (int, net.IP, error) {
	if err := s.conn.SetReadDeadline(time.Now().Add(config.RecvTimeout)); err != nil {
		return 0, nil, err
	}
	n, addr, err := s.conn.ReadFromUDP(buf)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return 0, nil, err
		}
		return 0, nil, fmt.Errorf("recv: %w", err)
	}
	return n, addr.IP.To4(), nil
}

func (s *Socket) Close() error {
	return s.conn.Close()
}

func (s *Socket) Group() net.IP {
	return s.group
}
