package main

import (
	"errors"
	"net/netip"
)

type Mode string

const (
	ModeSubnet Mode = "subnet"
	ModeRoute  Mode = "route"
)

type Config struct {
	Mode        Mode
	Subnet      string
	RouteFile   string
	Destination string
}

type SubnetInfo struct {
	Network   netip.Addr
	Broadcast *netip.Addr
	Netmask   netip.Addr
	Prefix    int
	First     netip.Addr
	Last      netip.Addr
	Hosts     uint64
}

type Route struct {
	Network   netip.Prefix
	Interface string
}

var errUnreachable = errors.New("destination is unreachable")
