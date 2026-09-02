package domain

import (
	"testing"
)

func TestEncodeDecodeHello(t *testing.T) {
	msg := Message{
		Type:    MsgHello,
		Mode:    ModeBroadcast,
		NodeID:  42,
		Payload: []byte("alice"),
	}
	data, err := Encode(msg)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != msg.Type || got.Mode != msg.Mode || got.NodeID != msg.NodeID {
		t.Fatalf("header mismatch: %+v", got)
	}
	if string(got.Payload) != "alice" {
		t.Fatalf("payload %q", got.Payload)
	}
}

func TestDecodeInvalidMagic(t *testing.T) {
	data := make([]byte, HeaderSize)
	_, err := Decode(data)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDecodeTooShort(t *testing.T) {
	_, err := Decode([]byte{1, 2, 3})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseMode(t *testing.T) {
	mode, err := ParseMode("broadcast")
	if err != nil || mode != ModeBroadcast {
		t.Fatalf("got %v err %v", mode, err)
	}
	mode, err = ParseMode("multicast")
	if err != nil || mode != ModeMulticast {
		t.Fatalf("got %v err %v", mode, err)
	}
}
