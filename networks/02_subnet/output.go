package main

import (
	"fmt"
	"io"
)

func printSubnet(out io.Writer, info SubnetInfo) error {
	broadcast := "none"
	if info.Broadcast != nil {
		broadcast = info.Broadcast.String()
	}

	_, err := fmt.Fprintf(out,
		"network %s\nbroadcast %s\nnetmask %s\nprefix %d\nfirst %s\nlast %s\nhosts %d\n",
		info.Network.String(), broadcast, info.Netmask.String(), info.Prefix,
		info.First.String(), info.Last.String(), info.Hosts,
	)
	return err
}

func printRoute(out io.Writer, route Route) error {
	_, err := fmt.Fprintf(out, "via %s\nprefix %d\n", route.Interface, route.Network.Bits())
	return err
}

func printUnreachable(out io.Writer) error {
	_, err := fmt.Fprintln(out, "unreachable true")
	return err
}
