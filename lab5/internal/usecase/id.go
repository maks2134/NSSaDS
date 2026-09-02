package usecase

import "os"

func WorkerID(index int) int {
	id := (os.Getpid() + index*997) & 0xffff
	if id == 0 {
		return 1
	}
	return id
}
