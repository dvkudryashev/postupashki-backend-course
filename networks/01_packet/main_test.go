package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"strings"
	"testing"
)

func testFrame(protocol uint8, transport []byte) []byte {
	frame := make([]byte, EthernetHeaderSize+IPv4HeaderSize+len(transport))
	binary.BigEndian.PutUint16(frame[12:14], uint16(EtherTypeIPv4))
	ip := frame[EthernetHeaderSize:]
	ip[0] = 0x45
	ip[9] = protocol
	binary.BigEndian.PutUint16(ip[2:4], uint16(len(ip)))
	copy(ip[IPv4HeaderSize:], transport)
	return frame
}

func TestPacketRejectsMalformedInput(t *testing.T) {
	tcp := make([]byte, TCPHeaderSize)
	tcp[12] = 0x50
	base := testFrame(IPProtocolTCP, tcp)
	cases := []struct {
		name   string
		change func([]byte) []byte
	}{
		{"short Ethernet", func(f []byte) []byte { return f[:13] }},
		{"short IPv4", func(f []byte) []byte { return f[:33] }},
		{"wrong version", func(f []byte) []byte { f[14] = 0x65; return f }},
		{"short IHL", func(f []byte) []byte { f[14] = 0x44; return f }},
		{"long IHL", func(f []byte) []byte { f[14] = 0x4f; return f }},
		{"total below IHL", func(f []byte) []byte { f[17] = 19; return f }},
		{"truncated packet", func(f []byte) []byte { return f[:len(f)-4] }},
		{"short TCP", func(f []byte) []byte { f[17] = 36; return f[:50] }},
		{"short TCP offset", func(f []byte) []byte { f[46] = 0x40; return f }},
		{"long TCP offset", func(f []byte) []byte { f[46] = 0xf0; return f }},
		{"padding is not TCP", func(f []byte) []byte { f[17] = 20; return f }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			frame := tc.change(bytes.Clone(base))
			var out bytes.Buffer
			if err := run(strings.NewReader(hex.EncodeToString(frame)), &out); err == nil {
				t.Fatal("malformed packet was accepted")
			}
			if out.Len() != 0 {
				t.Fatalf("malformed packet produced output: %q", out.String())
			}
		})
	}
	for _, input := range []string{"", "zz", "0"} {
		if err := run(strings.NewReader(input), &bytes.Buffer{}); err == nil {
			t.Errorf("invalid input %q was accepted", input)
		}
	}
}

func TestUDPLengthsAndFragments(t *testing.T) {
	for _, tc := range []struct {
		name       string
		bytes      int
		length     uint16
		flags      uint16
		wantError  bool
		wantLength string
	}{
		{"short header", 7, 8, 0, true, ""},
		{"short length", 8, 7, 0, true, ""},
		{"long length", 8, 9, 0, true, ""},
		{"complete datagram", 12, 12, 0, false, "4"},
		{"first fragment", 16, 32, IPFlagMF, false, "8"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			udp := make([]byte, tc.bytes)
			binary.BigEndian.PutUint16(udp[4:6], tc.length)
			frame := testFrame(IPProtocolUDP, udp)
			binary.BigEndian.PutUint16(frame[20:22], tc.flags)
			var out bytes.Buffer
			err := run(strings.NewReader(hex.EncodeToString(frame)), &out)
			if (err != nil) != tc.wantError {
				t.Fatalf("error = %v, wantError = %v", err, tc.wantError)
			}
			if !tc.wantError && !strings.Contains(out.String(), "payload.length "+tc.wantLength+"\n") {
				t.Fatalf("unexpected payload length: %s", out.String())
			}
		})
	}
}

func TestNoninitialFragmentHasNoTransportHeader(t *testing.T) {
	frame := testFrame(IPProtocolTCP, []byte{1, 2, 3})
	binary.BigEndian.PutUint16(frame[20:22], 1)
	var out bytes.Buffer
	if err := run(strings.NewReader(hex.EncodeToString(frame)), &out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "tcp.") || !strings.Contains(out.String(), "payload.length 3\n") {
		t.Fatalf("fragment was parsed as a TCP header: %s", out.String())
	}
}
