package network

import (
	"net"
	"runtime"
	"syscall"
	"time"
)

const (
	tcpKeepIdle  = 4
	tcpKeepIntvl = 5
	tcpKeepCnt   = 6
	tcpKeepAlive = 0x10
)

func setKeepAlive(conn net.Conn, keepAlive bool, keepAliveIdle time.Duration, keepAliveCount int, keepAliveIntvl time.Duration) error {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return nil
	}

	if err := tcpConn.SetKeepAlive(keepAlive); err != nil {
		return err
	}

	if !keepAlive {
		return nil
	}

	if runtime.GOOS != "windows" {
		if err := tcpConn.SetKeepAlivePeriod(keepAliveIdle); err != nil {
			return err
		}
	}

	file, err := tcpConn.File()
	if err != nil {
		return nil
	}
	defer file.Close()

	fd := int(file.Fd())

	switch runtime.GOOS {
	case "linux":
		syscall.Syscall6(syscall.SYS_SETSOCKOPT, uintptr(fd), syscall.IPPROTO_TCP, 4, uintptr(keepAliveIdle.Seconds()), 0, 0)
		syscall.Syscall6(syscall.SYS_SETSOCKOPT, uintptr(fd), syscall.IPPROTO_TCP, 5, uintptr(keepAliveIntvl.Seconds()), 0, 0)
		syscall.Syscall6(syscall.SYS_SETSOCKOPT, uintptr(fd), syscall.IPPROTO_TCP, 6, uintptr(keepAliveCount), 0, 0)

	case "darwin", "freebsd", "netbsd", "openbsd":
		syscall.Syscall6(syscall.SYS_SETSOCKOPT, uintptr(fd), syscall.IPPROTO_TCP, 0x10, uintptr(keepAliveIdle.Seconds()), 0, 0)
		syscall.Syscall6(syscall.SYS_SETSOCKOPT, uintptr(fd), syscall.IPPROTO_TCP, 0x101, uintptr(keepAliveIntvl.Seconds()), 0, 0)

	case "windows":
		_ = fd

	default:
	}

	return nil
}

func setsockoptString(fd, level, opt int, value string) error {
	val := []byte(value)
	return syscall.SetsockoptString(fd, level, opt, string(val))
}

func setsockoptInt(fd, level, opt int, value int) error {
	return syscall.SetsockoptInt(fd, level, opt, value)
}
