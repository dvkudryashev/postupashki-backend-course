package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net/netip"
)

var errNotIPv4 = errors.New("expected IPv4")

func parseIPv4(raw string) (netip.Addr, error) {
	address, err := netip.ParseAddr(raw)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("netip.ParseAddr: %w", err)
	}
	if !address.Is4() {
		return netip.Addr{}, errNotIPv4
	}

	return address, nil
}

func parsePrefix(raw string) (netip.Prefix, error) {
	prefix, err := netip.ParsePrefix(raw)
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("netip.ParsePrefix: %w", err)
	}
	if !prefix.Addr().Is4() {
		return netip.Prefix{}, errNotIPv4
	}
	return prefix.Masked(), nil
}

func ipv4ToUint32(address netip.Addr) uint32 {
	octets := address.As4()
	return binary.BigEndian.Uint32(octets[:])
}

func uint32ToIPv4(value uint32) netip.Addr {
	var octets [4]byte
	binary.BigEndian.PutUint32(octets[:], value)
	return netip.AddrFrom4(octets)
}

func prefixMask(bits int) uint32 {
	if bits == 0 {
		return 0
	}
	return ^uint32(0) << (32 - bits)
}
