//go:build windows

package udp

import (
	"syscall"

	"golang.org/x/sys/windows"
)

func applySocketOptions(fd int) error {
	if err := windows.SetsockoptInt(windows.Handle(fd), windows.SOL_SOCKET, windows.SO_REUSEADDR, 1); err != nil {
		return err
	}
	return windows.SetsockoptInt(windows.Handle(fd), windows.SOL_SOCKET, windows.SO_BROADCAST, 1)
}

func rawFD(fd int) int {
	return fd
}

func isTimeout(err error) bool {
	return err == syscall.EWOULDBLOCK || err == syscall.WSAEWOULDBLOCK
}
