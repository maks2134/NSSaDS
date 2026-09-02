package domain

import "net"

type IfaceInfo struct {
	Name      string
	IP        net.IP
	Mask      net.IPMask
	Broadcast net.IP
}
