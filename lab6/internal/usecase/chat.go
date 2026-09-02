package usecase

import (
	"fmt"
	"io"
	"net"
	"time"

	"NSSaDS/lab6/internal/domain"
	"NSSaDS/lab6/internal/infrastructure/udp"
	"NSSaDS/lab6/pkg/config"
)

type Chat struct {
	cfg     *config.Config
	iface   domain.IfaceInfo
	sock    *udp.Socket
	peers   *domain.PeerRegistry
	nodeID  uint32
	nick    string
	mode    domain.Mode
	inGroup bool
	out     io.Writer
}

func NewChat(cfg *config.Config, info domain.IfaceInfo, sock *udp.Socket, nodeID uint32, out io.Writer) *Chat {
	return &Chat{
		cfg:    cfg,
		iface:  info,
		sock:   sock,
		peers:  domain.NewPeerRegistry(),
		nodeID: nodeID,
		nick:   cfg.Nick,
		mode:   domain.ModeBroadcast,
		out:    out,
	}
}

func (c *Chat) Start() error {
	PrintStartup(c.out, c.cfg, c.iface, c.mode)
	return c.sendHello()
}

func (c *Chat) Mode() domain.Mode {
	return c.mode
}

func (c *Chat) SendBye() error {
	return c.send(domain.MsgBye, nil)
}

func (c *Chat) SendHello() error {
	return c.send(domain.MsgHello, []byte(c.nick))
}

func (c *Chat) SendChat(text string) error {
	return c.send(domain.MsgChat, []byte(text))
}

func (c *Chat) SendIgnore(ip net.IP) error {
	return c.send(domain.MsgIgnore, ip.To4())
}

func (c *Chat) SendUnignore(ip net.IP) error {
	return c.send(domain.MsgUnignore, ip.To4())
}

func (c *Chat) send(msgType domain.MsgType, payload []byte) error {
	msg := domain.Message{
		Type:    msgType,
		Mode:    c.mode,
		NodeID:  c.nodeID,
		Payload: payload,
	}
	data, err := domain.Encode(msg)
	if err != nil {
		return err
	}
	return c.sock.Send(data, c.mode)
}

func (c *Chat) HandleInput(line string) (done bool, err error) {
	cmd, ok := ParseCommand(line)
	if !ok {
		return false, c.SendChat(line)
	}
	switch cmd.Name {
	case "/net":
		PrintNet(c.out, c.iface, c.mode, c.cfg.Group)
	case "/peers":
		PrintPeers(c.out, c.peers.List(c.mode), time.Now())
	case "/mode":
		return false, c.switchMode(cmd.Arg)
	case "/leave":
		return false, c.leaveGroup()
	case "/join":
		return false, c.joinGroup()
	case "/ignore":
		return false, c.ignorePeer(cmd.Arg)
	case "/unignore":
		return false, c.unignorePeer(cmd.Arg)
	case "/help":
		PrintHelp(c.out)
	case "/quit", "/exit":
		return true, c.shutdown()
	default:
		fmt.Fprintf(c.out, "unknown command %s (try /help)\n", cmd.Name)
	}
	return false, nil
}

func (c *Chat) switchMode(arg string) error {
	mode, err := ParseModeArg(arg)
	if err != nil {
		return err
	}
	if mode == c.mode {
		fmt.Fprintf(c.out, "already in %s mode\n", domain.ModeName(mode))
		return nil
	}
	if err := c.SendBye(); err != nil {
		return err
	}
	if c.inGroup && c.mode == domain.ModeMulticast {
		_ = c.sock.LeaveGroup(c.sock.Group())
		c.inGroup = false
	}
	c.mode = mode
	c.peers.Clear()
	if mode == domain.ModeMulticast {
		if err := c.sock.JoinGroup(c.sock.Group()); err != nil {
			return err
		}
		c.inGroup = true
	}
	fmt.Fprintf(c.out, "switched to %s mode\n", domain.ModeName(mode))
	return c.SendHello()
}

func (c *Chat) leaveGroup() error {
	if c.mode != domain.ModeMulticast {
		fmt.Fprintln(c.out, "not in multicast mode")
		return nil
	}
	if c.inGroup {
		if err := c.SendBye(); err != nil {
			return err
		}
		if err := c.sock.LeaveGroup(c.sock.Group()); err != nil {
			return err
		}
		c.inGroup = false
	}
	return c.switchMode("broadcast")
}

func (c *Chat) joinGroup() error {
	if c.mode == domain.ModeMulticast && c.inGroup {
		fmt.Fprintln(c.out, "already in multicast group")
		return nil
	}
	return c.switchMode("multicast")
}

func (c *Chat) ignorePeer(arg string) error {
	ip, err := ParseIgnoreIP(arg)
	if err != nil {
		return err
	}
	c.peers.SetIgnored(ip, true)
	if err := c.SendIgnore(ip); err != nil {
		return err
	}
	fmt.Fprintf(c.out, "ignoring %s\n", ip)
	return nil
}

func (c *Chat) unignorePeer(arg string) error {
	ip, err := ParseIgnoreIP(arg)
	if err != nil {
		return err
	}
	c.peers.SetIgnored(ip, false)
	if err := c.SendUnignore(ip); err != nil {
		return err
	}
	fmt.Fprintf(c.out, "unignored %s\n", ip)
	return nil
}

func (c *Chat) shutdown() error {
	if err := c.SendBye(); err != nil {
		return err
	}
	if c.inGroup {
		_ = c.sock.LeaveGroup(c.sock.Group())
	}
	return nil
}

func (c *Chat) ExpirePeers(now time.Time) {
	c.peers.Expire(now.Add(-config.PeerTimeout))
}

func (c *Chat) HandlePacket(from net.IP, data []byte) error {
	msg, err := domain.Decode(data)
	if err != nil {
		return nil
	}
	if msg.NodeID == c.nodeID {
		return nil
	}
	if msg.Mode != c.mode {
		return nil
	}
	if c.peers.IsIgnored(from) && msg.Type == domain.MsgChat {
		return nil
	}
	now := time.Now()
	switch msg.Type {
	case domain.MsgHello:
		c.peers.Upsert(from, string(msg.Payload), msg.Mode, now)
	case domain.MsgBye:
		c.peers.Remove(from)
	case domain.MsgChat:
		fmt.Fprintf(c.out, "[%s] %s\n", from, string(msg.Payload))
	case domain.MsgIgnore:
		if ip := parseIPPayload(msg.Payload); ip != nil {
			c.peers.SetIgnored(ip, true)
			fmt.Fprintf(c.out, "peer %s ignored %s\n", from, ip)
		}
	case domain.MsgUnignore:
		if ip := parseIPPayload(msg.Payload); ip != nil {
			c.peers.SetIgnored(ip, false)
			fmt.Fprintf(c.out, "peer %s unignored %s\n", from, ip)
		}
	}
	return nil
}

func parseIPPayload(payload []byte) net.IP {
	if len(payload) != net.IPv4len {
		return nil
	}
	return net.IP(payload).To4()
}

func (c *Chat) sendHello() error {
	return c.SendHello()
}
