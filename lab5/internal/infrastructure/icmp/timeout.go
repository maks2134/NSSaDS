package icmp

import (
	"errors"
	"net"
	"os"
	"syscall"
)

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrDeadlineExceeded) || errors.Is(err, ErrTimeout) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return errno.Timeout() || errno == syscall.EAGAIN || errno == syscall.EWOULDBLOCK || errno == syscall.ETIMEDOUT
	}
	return false
}
