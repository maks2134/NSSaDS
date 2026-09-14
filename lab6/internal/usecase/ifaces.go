package usecase

import (
	"fmt"
	"io"
	"net"

	"NSSaDS/lab6/internal/infrastructure/netiface"
)

func PrintIfaces(out io.Writer) error {
	cands, err := netiface.Candidates()
	if err != nil {
		return err
	}
	if len(cands) == 0 {
		fmt.Fprintln(out, "no IPv4 interfaces found")
		return nil
	}
	fmt.Fprintln(out, "available interfaces:")
	for _, c := range cands {
		virt := ""
		if netiface.IsVirtualName(c.Name) {
			virt = " [virtual]"
		}
		fmt.Fprintf(out, "  %-32s %s  mask=%s  bcast=%s%s\n",
			c.Name, c.IP, net.IP(c.Mask).String(), c.Broadcast, virt)
	}
	fmt.Fprintln(out, `use: lab6 -iface "Wi-Fi" -nick bob`)
	return nil
}
