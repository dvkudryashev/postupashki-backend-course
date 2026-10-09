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
		{"shorter declared datagram", 16, 12, 0, false, "4"},
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

func TestParsedPacketOptionalLayers(t *testing.T) {
	tcp := make([]byte, TCPHeaderSize)
	tcp[12] = 0x50
	udp := make([]byte, UDPHeaderSize)
	binary.BigEndian.PutUint16(udp[4:6], UDPHeaderSize)
	ethernet := make([]byte, EthernetHeaderSize)
	binary.BigEndian.PutUint16(ethernet[12:14], 0x0806)
	fragment := testFrame(IPProtocolTCP, []byte{1, 2, 3})
	binary.BigEndian.PutUint16(fragment[20:22], 1)

	for _, tc := range []struct {
		name          string
		frame         []byte
		ipv4          bool
		tcp           bool
		udp           bool
		payloadLength int
	}{
		{"Ethernet only", ethernet, false, false, false, 0},
		{"unknown IP protocol", testFrame(1, []byte{1, 2, 3}), true, false, false, 3},
		{"TCP", testFrame(IPProtocolTCP, tcp), true, true, false, 0},
		{"UDP", testFrame(IPProtocolUDP, udp), true, false, true, 0},
		{"noninitial fragment", fragment, true, false, false, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			packet, err := parseFrame(tc.frame)
			if err != nil {
				t.Fatal(err)
			}
			if (packet.IPv4 != nil) != tc.ipv4 || (packet.TCP != nil) != tc.tcp || (packet.UDP != nil) != tc.udp {
				t.Fatalf("unexpected parsed layers: %+v", packet)
			}
			if packet.PayloadLength != tc.payloadLength {
				t.Fatalf("payload length = %d, want %d", packet.PayloadLength, tc.payloadLength)
			}
			var out bytes.Buffer
			printPacket(&out, packet)
			if strings.Contains(out.String(), "payload.length") != tc.ipv4 {
				t.Fatalf("unexpected payload output: %s", out.String())
			}
		})
	}
}

func TestPacketOptionsAndPadding(t *testing.T) {
	ihlBytes := IPv4HeaderSize + IPv4WordSize
	dataOffsetBytes := TCPHeaderSize + IPv4WordSize
	payload := []byte("abc")
	totalLength := ihlBytes + dataOffsetBytes + len(payload)
	frame := make([]byte, EthernetHeaderSize+totalLength+18)
	binary.BigEndian.PutUint16(frame[12:14], uint16(EtherTypeIPv4))
	ip := frame[EthernetHeaderSize : EthernetHeaderSize+totalLength]
	ip[0] = 0x46
	ip[9] = IPProtocolTCP
	binary.BigEndian.PutUint16(ip[2:4], uint16(totalLength))
	copy(ip[IPv4HeaderSize:ihlBytes], []byte{1, 1, 1, 0})
	tcp := ip[ihlBytes:]
	tcp[12] = 0x60
	copy(tcp[TCPHeaderSize:dataOffsetBytes], []byte{1, 1, 1, 0})
	copy(tcp[dataOffsetBytes:], payload)

	var sum uint32
	for i := 0; i < ihlBytes; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(ip[i : i+2]))
	}
	for sum>>16 != 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}
	binary.BigEndian.PutUint16(ip[10:12], ^uint16(sum))

	packet, err := parseFrame(frame)
	if err != nil {
		t.Fatal(err)
	}
	if packet.IPv4 == nil || packet.TCP == nil {
		t.Fatalf("missing parsed headers: %+v", packet)
	}
	if packet.IPv4.IHLBytes != ihlBytes || packet.IPv4.TotalLength != totalLength || packet.TCP.DataOffsetBytes != dataOffsetBytes {
		t.Fatalf("unexpected header lengths: IP=%+v TCP=%+v", packet.IPv4, packet.TCP)
	}
	if packet.PayloadLength != len(payload) || !packet.IPv4.ChecksumValid {
		t.Fatalf("payload length = %d, checksum valid = %t", packet.PayloadLength, packet.IPv4.ChecksumValid)
	}

	ip[IPv4HeaderSize] ^= 1
	packet, err = parseFrame(frame)
	if err != nil {
		t.Fatal(err)
	}
	if packet.IPv4.ChecksumValid {
		t.Fatal("changed IP option did not invalidate checksum")
	}
}

func TestParsedPacketDoesNotKeepFrameBytes(t *testing.T) {
	tcp := make([]byte, TCPHeaderSize)
	binary.BigEndian.PutUint16(tcp[0:2], 1234)
	binary.BigEndian.PutUint16(tcp[2:4], 80)
	binary.BigEndian.PutUint32(tcp[4:8], 42)
	tcp[12] = 0x50
	tcp[13] = FlagACK
	frame := testFrame(IPProtocolTCP, tcp)
	copy(frame[0:6], []byte{1, 2, 3, 4, 5, 6})
	copy(frame[6:12], []byte{6, 5, 4, 3, 2, 1})
	copy(frame[26:30], []byte{192, 168, 1, 2})
	copy(frame[30:34], []byte{192, 168, 1, 3})

	packet, err := parseFrame(frame)
	if err != nil {
		t.Fatal(err)
	}
	var before bytes.Buffer
	printPacket(&before, packet)
	clear(frame)
	var after bytes.Buffer
	printPacket(&after, packet)
	if after.String() != before.String() {
		t.Fatalf("parsed packet changed with source bytes:\nbefore:\n%safter:\n%s", before.String(), after.String())
	}
}
