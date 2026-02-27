package network

import (
	"NSSaDS/lab3/internal/domain"
	"syscall"
)

type UnixSelectSystem struct{}

func (uss *UnixSelectSystem) Select(nfd int, readFds, writeFds, exceptFds *domain.FdSet, timeout *domain.Timeval) (int, error) {

	syscallReadFds := &syscall.FdSet{}
	syscallWriteFds := &syscall.FdSet{}
	syscallExceptFds := &syscall.FdSet{}
	syscallTimeout := &syscall.Timeval{
		Sec:  timeout.Sec,
		Usec: timeout.Usec,
	}

	for i := 0; i < 32; i++ {
		if i < len(readFds.Bits) {
			syscallReadFds.Bits[i] = readFds.Bits[i]
		}
		if i < len(writeFds.Bits) {
			syscallWriteFds.Bits[i] = writeFds.Bits[i]
		}
		if i < len(exceptFds.Bits) {
			syscallExceptFds.Bits[i] = exceptFds.Bits[i]
		}
	}

	err := syscall.Select(nfd, syscallReadFds, syscallWriteFds, syscallExceptFds, syscallTimeout)
	if err != nil {
		return 0, err
	}

	count := 0
	for i := 0; i < 32; i++ {
		if i < len(readFds.Bits) {
			readFds.Bits[i] = syscallReadFds.Bits[i]
			if syscallReadFds.Bits[i] != 0 {
				count++
			}
		}
		if i < len(writeFds.Bits) {
			writeFds.Bits[i] = syscallWriteFds.Bits[i]
			if syscallWriteFds.Bits[i] != 0 {
				count++
			}
		}
		if i < len(exceptFds.Bits) {
			exceptFds.Bits[i] = syscallExceptFds.Bits[i]
			if syscallExceptFds.Bits[i] != 0 {
				count++
			}
		}
	}

	return count, nil
}

func (uss *UnixSelectSystem) FDSet(fd int, set *domain.FdSet) {
	set.Bits[fd/32] |= 1 << (uint(fd) % 32)
}

func (uss *UnixSelectSystem) FDIsSet(fd int, set *domain.FdSet) bool {
	if fd/32 >= len(set.Bits) {
		return false
	}
	return set.Bits[fd/32]&(1<<(uint(fd)%32)) != 0
}

func (uss *UnixSelectSystem) FDZero(set *domain.FdSet) {
	for i := range set.Bits {
		set.Bits[i] = 0
	}
}
