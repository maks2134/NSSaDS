package domain

import "net"

type Transport interface {
	Send(data []byte) error
	Recv(buf []byte) (int, net.IP, error)
	JoinGroup(group net.IP) error
	LeaveGroup(group net.IP) error
	Close() error
}
