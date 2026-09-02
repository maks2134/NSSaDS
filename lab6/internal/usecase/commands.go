package usecase

import (
	"fmt"
	"net"
	"strings"

	"NSSaDS/lab6/internal/domain"
)

type Command struct {
	Name string
	Arg  string
}

func ParseCommand(line string) (Command, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return Command{}, false
	}
	if !strings.HasPrefix(line, "/") {
		return Command{}, false
	}
	parts := strings.Fields(line)
	cmd := Command{Name: strings.ToLower(parts[0])}
	if len(parts) > 1 {
		cmd.Arg = strings.Join(parts[1:], " ")
	}
	return cmd, true
}

func ParseIgnoreIP(arg string) (net.IP, error) {
	ip := net.ParseIP(strings.TrimSpace(arg))
	if ip == nil {
		return nil, fmt.Errorf("invalid ip %q", arg)
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return nil, fmt.Errorf("ipv4 required")
	}
	return ip4, nil
}

func ParseModeArg(arg string) (domain.Mode, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return 0, fmt.Errorf("mode required: broadcast or multicast")
	}
	return domain.ParseMode(arg)
}
