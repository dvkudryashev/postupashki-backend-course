package main

import "net/netip"

func calculateSubnet(prefix netip.Prefix) SubnetInfo {
	bits := prefix.Bits()
	mask := prefixMask(bits)
	network := ipv4ToUint32(prefix.Addr())
	upper := network | ^mask
	blockSize := uint64(1) << (32 - bits)

	info := SubnetInfo{
		Network: uint32ToIPv4(network),
		Netmask: uint32ToIPv4(mask),
		Prefix:  bits,
	}

	if bits == 32 {
		info.Broadcast = nil
		info.First = uint32ToIPv4(network)
		info.Last = uint32ToIPv4(network)
		info.Hosts = 1
		return info
	}
	if bits == 31 {
		info.Broadcast = nil
		info.First = uint32ToIPv4(network)
		info.Last = uint32ToIPv4(upper)
		info.Hosts = 2
		return info
	}
	broadcast := uint32ToIPv4(upper)
	info.Broadcast = &broadcast
	info.First = uint32ToIPv4(network + 1)
	info.Last = uint32ToIPv4(upper - 1)
	info.Hosts = blockSize - 2
	return info
}
