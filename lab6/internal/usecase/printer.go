package usecase

import (
	"fmt"
	"io"
	"net"
	"time"

	"NSSaDS/lab6/internal/domain"
	"NSSaDS/lab6/internal/infrastructure/netiface"
	"NSSaDS/lab6/internal/infrastructure/udp"
	"NSSaDS/lab6/pkg/config"
)

func PrintNet(out io.Writer, info domain.IfaceInfo, mode domain.Mode, group string) {
	mask := net.IP(info.Mask).String()
	fmt.Fprintf(out, "interface: %s\n", info.Name)
	fmt.Fprintf(out, "ip:        %s\n", info.IP)
	fmt.Fprintf(out, "mask:      %s\n", mask)
	fmt.Fprintf(out, "broadcast: %s\n", info.Broadcast)
	fmt.Fprintf(out, "mode:      %s\n", domain.ModeName(mode))
	if mode == domain.ModeMulticast {
		fmt.Fprintf(out, "group:     %s\n", group)
	}
}

func PrintPeers(out io.Writer, peers []domain.Peer, now time.Time) {
	if len(peers) == 0 {
		fmt.Fprintln(out, "no peers in current mode")
		return
	}
	fmt.Fprintln(out, "peers:")
	for _, p := range peers {
		age := now.Sub(p.LastSeen).Round(time.Second)
		flag := ""
		if p.Ignored {
			flag = " [ignored]"
		}
		fmt.Fprintf(out, "  %s  nick=%s  last=%s ago%s\n",
			p.IP, p.Nick, age, flag)
	}
}

func PrintHelp(out io.Writer) {
	fmt.Fprintln(out, "commands:")
	fmt.Fprintln(out, "  /net")
	fmt.Fprintln(out, "  /peers")
	fmt.Fprintln(out, "  /mode broadcast|multicast")
	fmt.Fprintln(out, "  /leave")
	fmt.Fprintln(out, "  /join")
	fmt.Fprintln(out, "  /ignore <ip>")
	fmt.Fprintln(out, "  /unignore <ip>")
	fmt.Fprintln(out, "  /help")
	fmt.Fprintln(out, "  /quit")
}

func PrintStartup(out io.Writer, cfg *config.Config, info domain.IfaceInfo, mode domain.Mode) {
	fmt.Fprintf(out, "lab6 chat started on port %d as %q\n", cfg.Port, cfg.Nick)
	PrintNet(out, info, mode, cfg.Group)
	fmt.Fprintln(out, "type /help for commands")
}

func DetectIface(name string) (domain.IfaceInfo, error) {
	return netiface.Detect(name)
}

func OpenSocket(info domain.IfaceInfo, cfg *config.Config) (*udp.Socket, error) {
	return udp.Open(info, cfg.Port, cfg.Group, config.RecvTimeout)
}
