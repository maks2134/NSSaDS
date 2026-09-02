package usecase

import (
	"testing"

	"NSSaDS/lab6/internal/domain"
)

func TestParseCommand(t *testing.T) {
	cmd, ok := ParseCommand("/mode multicast")
	if !ok || cmd.Name != "/mode" || cmd.Arg != "multicast" {
		t.Fatalf("got %+v ok=%v", cmd, ok)
	}
	_, ok = ParseCommand("hello")
	if ok {
		t.Fatal("expected not a command")
	}
}

func TestParseModeArg(t *testing.T) {
	mode, err := ParseModeArg("broadcast")
	if err != nil || mode != domain.ModeBroadcast {
		t.Fatalf("got %v err %v", mode, err)
	}
}

func TestParseIgnoreIP(t *testing.T) {
	ip, err := ParseIgnoreIP("192.168.1.5")
	if err != nil || ip.String() != "192.168.1.5" {
		t.Fatalf("got %v err %v", ip, err)
	}
	_, err = ParseIgnoreIP("bad")
	if err == nil {
		t.Fatal("expected error")
	}
}
