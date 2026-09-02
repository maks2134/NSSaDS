//go:build unix

package udp

import (
	"syscall"

	"golang.org/x/sys/unix"
)

func applySocketOptions(fd int) error {
	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
		return err
	}
	return unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_BROADCAST, 1)
}

func rawFD(fd int) int {
	return fd
}

func isTimeout(err error) bool {
	return err == syscall.EAGAIN || err == syscall.EWOULDBLOCK
}
