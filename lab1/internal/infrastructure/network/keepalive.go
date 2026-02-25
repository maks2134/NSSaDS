package network

import (
	"net"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

// TCP keepalive constants for different platforms
const (
	tcpKeepIdle  = 4    // TCP_KEEPIDLE (Linux)
	tcpKeepIntvl = 5    // TCP_KEEPINTVL (Linux)
	tcpKeepCnt   = 6    // TCP_KEEPCNT (Linux)
	tcpKeepAlive = 0x10 // TCP_KEEPALIVE (BSD)
)

func setKeepAlive(conn net.Conn, keepAlive bool, keepAliveIdle time.Duration, keepAliveCount int, keepAliveIntvl time.Duration) error {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return nil // Not a TCP connection, skip keepalive
	}

	if err := tcpConn.SetKeepAlive(keepAlive); err != nil {
		return err
	}

	if !keepAlive {
		return nil
	}

	// Set keepalive period (works on Unix-like systems)
	if runtime.GOOS != "windows" {
		if err := tcpConn.SetKeepAlivePeriod(keepAliveIdle); err != nil {
			return err
		}
	}

	// Platform-specific socket options
	file, err := tcpConn.File()
	if err != nil {
		return nil // Not critical, continue without additional options
	}
	defer file.Close()

	fd := int(file.Fd())

	switch runtime.GOOS {
	case "linux":
		// Linux-specific TCP keepalive options using raw syscalls
		// TCP_KEEPIDLE = 4, TCP_KEEPINTVL = 5, TCP_KEEPCNT = 6
		syscall.Syscall6(syscall.SYS_SETSOCKOPT, uintptr(fd), syscall.IPPROTO_TCP, 4, uintptr(keepAliveIdle.Seconds()), 0, 0)
		syscall.Syscall6(syscall.SYS_SETSOCKOPT, uintptr(fd), syscall.IPPROTO_TCP, 5, uintptr(keepAliveIntvl.Seconds()), 0, 0)
		syscall.Syscall6(syscall.SYS_SETSOCKOPT, uintptr(fd), syscall.IPPROTO_TCP, 6, uintptr(keepAliveCount), 0, 0)

	case "darwin", "freebsd", "netbsd", "openbsd":
		// BSD systems use different constants
		// TCP_KEEPALIVE = 0x10, TCP_KEEPINTVL = 0x101
		syscall.Syscall6(syscall.SYS_SETSOCKOPT, uintptr(fd), syscall.IPPROTO_TCP, 0x10, uintptr(keepAliveIdle.Seconds()), 0, 0)
		syscall.Syscall6(syscall.SYS_SETSOCKOPT, uintptr(fd), syscall.IPPROTO_TCP, 0x101, uintptr(keepAliveIntvl.Seconds()), 0, 0)

	case "windows":
		// Windows uses different socket options
		// The basic SetKeepAlive() call is usually sufficient for Windows
		// Additional Windows-specific options could be set using winsock API if needed
		_ = fd // Suppress unused warning

	default:
		// Other platforms - basic keepalive should work
	}

	return nil
}

// setsockoptString is a helper for platforms that need string-based socket options
func setsockoptString(fd, level, opt int, value string) error {
	val := []byte(value)
	return syscall.SetsockoptString(fd, level, opt, string(val))
}

// setsockoptInt is a helper for platforms that might need different int handling
func setsockoptInt(fd, level, opt int, value int) error {
	// For some platforms, we might need to use unsafe pointer conversion
	// This is a fallback method
	var buf [4]byte
	*(*int32)(unsafe.Pointer(&buf[0])) = int32(value)
	return syscall.SetsockoptInt(fd, level, opt, value)
}
