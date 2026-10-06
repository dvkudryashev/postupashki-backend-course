package main

import (
	"bytes"
	"encoding/binary"
	"net/netip"
	"strings"
	"testing"
)

func testRecord(recordType DNSType, data []byte) []byte {
	packet := []byte{0}
	packet = binary.BigEndian.AppendUint16(packet, uint16(recordType))
	packet = binary.BigEndian.AppendUint16(packet, classIN)
	packet = binary.BigEndian.AppendUint32(packet, 60)
	packet = binary.BigEndian.AppendUint16(packet, uint16(len(data)))
	return append(packet, data...)
}

func TestRootName(t *testing.T) {
	encoded, err := encodeName(".")
	if err != nil || !bytes.Equal(encoded, []byte{0}) {
		t.Fatalf("encode root = %v, %v", encoded, err)
	}
	name, next, err := readName(encoded, 0)
	if err != nil || name != "." || next != 1 {
		t.Fatalf("decode root = %q, %d, %v", name, next, err)
	}
	answer, _, _, _, err := parseRecord(testRecord(TypeMX, []byte{0, 0, 0}), 0)
	if err != nil || answer.Value != "0 ." {
		t.Fatalf("root MX = %q, %v", answer.Value, err)
	}
}

func TestRecordNameRespectsRDLENGTH(t *testing.T) {
	for _, recordType := range []DNSType{TypeCNAME, TypeNS, TypeMX} {
		t.Run(recordType.String(), func(t *testing.T) {
			var prefix []byte
			if recordType == TypeMX {
				prefix = []byte{0, 10}
			}
			for _, data := range [][]byte{nil, {0xc0}, {3, 'a'}, {0, 0}} {
				rdata := append(bytes.Clone(prefix), data...)
				packet := testRecord(recordType, rdata)
				packet = append(packet, testRecord(TypeA, []byte{192, 0, 2, 1})...)
				if _, _, _, _, err := parseRecord(packet, 0); err == nil {
					t.Errorf("accepted invalid RDATA %x", rdata)
				}
			}
			packet := testRecord(recordType, append(bytes.Clone(prefix), 0xc0, 0))
			answer, next, _, supported, err := parseRecord(packet, 0)
			want := "."
			if recordType == TypeMX {
				want = "10 ."
			}
			if err != nil || !supported || answer.Value != want || next != len(packet) {
				t.Fatalf("compressed record = %+v, next=%d, supported=%v, error=%v", answer, next, supported, err)
			}
		})
	}
}

func TestAAAAKeepsIPv6Form(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"2001:db8::1", "2001:db8::1"},
		{"::", "::"},
		{"::ffff:c000:201", "::ffff:c000:201"},
		{"::ffff:0:1", "::ffff:0:1"},
	} {
		data := netip.MustParseAddr(tc.input).As16()
		answer, _, _, _, err := parseRecord(testRecord(TypeAAAA, data[:]), 0)
		if err != nil || answer.Value != tc.want {
			t.Errorf("AAAA %s = %q, %v; want %q", tc.input, answer.Value, err, tc.want)
		}
	}
}

func TestRejectsOversizedName(t *testing.T) {
	name := strings.Repeat(strings.Repeat("a", 63)+".", 4)
	if _, err := encodeName(name); err == nil {
		t.Fatal("accepted a name larger than 255 bytes")
	}
}

func TestRunRejectsInvalidArguments(t *testing.T) {
	for _, args := range [][]string{nil, {"127.0.0.1"}, {"127.0.0.1", "-1"}} {
		if err := run(args, strings.NewReader("")); err == nil {
			t.Errorf("accepted invalid arguments: %v", args)
		}
	}
}
