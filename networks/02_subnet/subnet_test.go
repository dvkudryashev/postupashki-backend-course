package main

import "testing"

func TestSubnetFromCanonicalPrefix(t *testing.T) {
	for _, tc := range []struct {
		input     string
		network   string
		broadcast string
		netmask   string
		first     string
		last      string
		prefix    int
		hosts     uint64
	}{
		{"192.168.10.37/24", "192.168.10.0", "192.168.10.255", "255.255.255.0", "192.168.10.1", "192.168.10.254", 24, 254},
		{"10.1.2.200/22", "10.1.0.0", "10.1.3.255", "255.255.252.0", "10.1.0.1", "10.1.3.254", 22, 1022},
		{"10.0.0.5/31", "10.0.0.4", "none", "255.255.255.254", "10.0.0.4", "10.0.0.5", 31, 2},
		{"192.168.10.37/0", "0.0.0.0", "255.255.255.255", "0.0.0.0", "0.0.0.1", "255.255.255.254", 0, 4294967294},
		{"203.0.113.9/32", "203.0.113.9", "none", "255.255.255.255", "203.0.113.9", "203.0.113.9", 32, 1},
	} {
		t.Run(tc.input, func(t *testing.T) {
			prefix, err := parsePrefix(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			if prefix.Addr().String() != tc.network {
				t.Fatalf("parsed network = %s; want %s", prefix.Addr(), tc.network)
			}

			info := calculateSubnet(prefix)
			broadcast := "none"
			if info.Broadcast != nil {
				broadcast = info.Broadcast.String()
			}
			if info.Network.String() != tc.network || broadcast != tc.broadcast || info.Netmask.String() != tc.netmask || info.First.String() != tc.first || info.Last.String() != tc.last || info.Prefix != tc.prefix || info.Hosts != tc.hosts {
				t.Fatalf("subnet = %+v, broadcast=%s; want %+v", info, broadcast, tc)
			}
		})
	}
}
