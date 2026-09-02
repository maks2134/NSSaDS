package usecase

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

func NewNodeID() (uint32, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, fmt.Errorf("random node id: %w", err)
	}
	return binary.BigEndian.Uint32(b[:]), nil
}
