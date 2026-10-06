package main

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

type EtherType uint16

const (
	EtherTypeIPv4 EtherType = 0x0800
)

const (
	IPProtocolTCP uint8 = 6
	IPProtocolUDP uint8 = 17
)

const (
	EthernetHeaderSize = 14
	IPv4HeaderSize     = 20
	TCPHeaderSize      = 20
	IPv4WordSize       = 4
	FragmentUnitSize   = 8
	UDPHeaderSize      = 8
)

const (
	IPFlagDF             uint16 = 1 << 14
	IPFlagMF             uint16 = 1 << 13
	IPFragmentOffsetMask uint16 = 0x1fff
)

const (
	FlagFIN uint8 = 1 << 0
	FlagSYN uint8 = 1 << 1
	FlagRST uint8 = 1 << 2
	FlagPSH uint8 = 1 << 3
	FlagACK uint8 = 1 << 4
	FlagURG uint8 = 1 << 5
)

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(in io.Reader, out io.Writer) error {
	frame, err := readFrame(in)
	if err != nil {
		return err
	}
	if err := validatePacket(frame); err != nil {
		return err
	}

	writer := bufio.NewWriter(out)
	printPacket(writer, frame)
	return writer.Flush()
}

func readFrame(in io.Reader) ([]byte, error) {
	data, err := io.ReadAll(in)
	if err != nil {
		return nil, err
	}

	dump := strings.ReplaceAll(strings.Join(strings.Fields(string(data)), ""), ":", "")
	return hex.DecodeString(dump)
}

func validatePacket(frame []byte) error {
	if len(frame) < EthernetHeaderSize {
		return fmt.Errorf("truncated Ethernet header: %d bytes", len(frame))
	}
	if EtherType(binary.BigEndian.Uint16(frame[12:14])) != EtherTypeIPv4 {
		return nil
	}

	ip := frame[EthernetHeaderSize:]
	if len(ip) < IPv4HeaderSize {
		return fmt.Errorf("truncated IPv4 header: %d bytes", len(ip))
	}
	if ip[0]>>4 != 4 {
		return fmt.Errorf("invalid IPv4 version: %d", ip[0]>>4)
	}
	ihlBytes := int(ip[0]&0x0f) * IPv4WordSize
	if ihlBytes < IPv4HeaderSize || ihlBytes > len(ip) {
		return fmt.Errorf("invalid IPv4 header length: %d", ihlBytes)
	}
	totalLength := int(binary.BigEndian.Uint16(ip[2:4]))
	if totalLength < ihlBytes || totalLength > len(ip) {
		return fmt.Errorf("invalid IPv4 total length: %d", totalLength)
	}
	ip = ip[:totalLength]
	flagsAndOffset := binary.BigEndian.Uint16(ip[6:8])
	if flagsAndOffset&IPFragmentOffsetMask != 0 {
		return nil
	}

	transport := ip[ihlBytes:]
	switch ip[9] {
	case IPProtocolTCP:
		if len(transport) < TCPHeaderSize {
			return fmt.Errorf("truncated TCP header: %d bytes", len(transport))
		}
		dataOffsetBytes := int(transport[12]>>4) * IPv4WordSize
		if dataOffsetBytes < TCPHeaderSize || dataOffsetBytes > len(transport) {
			return fmt.Errorf("invalid TCP header length: %d", dataOffsetBytes)
		}
	case IPProtocolUDP:
		if len(transport) < UDPHeaderSize {
			return fmt.Errorf("truncated UDP header: %d bytes", len(transport))
		}
		length := int(binary.BigEndian.Uint16(transport[4:6]))
		if length < UDPHeaderSize || (flagsAndOffset&IPFlagMF == 0 && length > len(transport)) {
			return fmt.Errorf("invalid UDP length: %d", length)
		}
	}
	return nil
}

func formatMAC(data []byte) string {
	var buffer []byte

	for i, b := range data {
		if i > 0 {
			buffer = append(buffer, ':')
		}
		buffer = fmt.Appendf(buffer, "%02x", b)
	}

	return string(buffer)
}

func formatIPv4(data []byte) string {
	var buffer []byte

	for i, b := range data {
		if i > 0 {
			buffer = append(buffer, '.')
		}
		buffer = fmt.Appendf(buffer, "%d", b)
	}

	return string(buffer)
}

func stringifyIPFlags(field uint16) string {
	var flags []string

	if field&IPFlagDF != 0 {
		flags = append(flags, "DF")
	}

	if field&IPFlagMF != 0 {
		flags = append(flags, "MF")
	}

	if len(flags) == 0 {
		return "none"
	}

	return strings.Join(flags, ",")
}

func parseIPOffset(field uint16) uint16 {
	return (field & IPFragmentOffsetMask) * FragmentUnitSize
}

func stringifyTCPFlags(flags uint8) string {
	var result []string

	if flags&FlagFIN != 0 {
		result = append(result, "FIN")
	}
	if flags&FlagSYN != 0 {
		result = append(result, "SYN")
	}
	if flags&FlagRST != 0 {
		result = append(result, "RST")
	}
	if flags&FlagPSH != 0 {
		result = append(result, "PSH")
	}
	if flags&FlagACK != 0 {
		result = append(result, "ACK")
	}
	if flags&FlagURG != 0 {
		result = append(result, "URG")
	}

	if len(result) == 0 {
		return "none"
	}

	return strings.Join(result, ",")
}

func ipChecksumValid(header []byte) bool {
	storedChecksum := binary.BigEndian.Uint16(header[10:12])

	var sum uint32

	for i := 0; i < len(header); i += 2 {
		if i == 10 {
			continue
		}

		sum += uint32(binary.BigEndian.Uint16(header[i : i+2]))
	}

	for sum>>16 != 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}

	return ^uint16(sum) == storedChecksum
}

func printPacket(out io.Writer, frame []byte) {
	dst := frame[0:6]
	src := frame[6:12]
	etherType := EtherType(binary.BigEndian.Uint16(frame[12:14]))

	fmt.Fprintf(out, "eth.dst %s\n", formatMAC(dst))
	fmt.Fprintf(out, "eth.src %s\n", formatMAC(src))
	fmt.Fprintf(out, "eth.ethertype 0x%04x\n", etherType)

	if etherType != EtherTypeIPv4 {
		return
	}

	ip := frame[EthernetHeaderSize:]

	version := ip[0] >> 4
	ihlBytes := int(ip[0]&0x0f) * IPv4WordSize
	totalLength := int(binary.BigEndian.Uint16(ip[2:4]))
	ip = ip[:totalLength]
	id := binary.BigEndian.Uint16(ip[4:6])
	flagsAndOffset := binary.BigEndian.Uint16(ip[6:8])
	protocol := ip[9]

	fmt.Fprintf(out, "ip.version %d\n", version)
	fmt.Fprintf(out, "ip.ihl_bytes %d\n", ihlBytes)
	fmt.Fprintf(out, "ip.total_length %d\n", totalLength)
	fmt.Fprintf(out, "ip.id 0x%04x\n", id)
	fmt.Fprintf(out, "ip.flags %s\n", stringifyIPFlags(flagsAndOffset))
	fmt.Fprintf(out, "ip.frag_offset %d\n", parseIPOffset(flagsAndOffset))
	fmt.Fprintf(out, "ip.ttl %d\n", ip[8])
	fmt.Fprintf(out, "ip.protocol %d\n", protocol)
	fmt.Fprintf(out, "ip.src %s\n", formatIPv4(ip[12:16]))
	fmt.Fprintf(out, "ip.dst %s\n", formatIPv4(ip[16:20]))
	fmt.Fprintf(out, "ip.checksum_valid %t\n", ipChecksumValid(ip[:ihlBytes]))
	if flagsAndOffset&IPFragmentOffsetMask != 0 {
		fmt.Fprintf(out, "payload.length %d\n", totalLength-ihlBytes)
		return
	}

	switch protocol {
	case IPProtocolTCP:
		tcp := ip[ihlBytes:]

		srcPort := binary.BigEndian.Uint16(tcp[0:2])
		dstPort := binary.BigEndian.Uint16(tcp[2:4])
		seq := binary.BigEndian.Uint32(tcp[4:8])
		ack := binary.BigEndian.Uint32(tcp[8:12])
		dataOffsetBytes := int(tcp[12]>>4) * IPv4WordSize
		window := binary.BigEndian.Uint16(tcp[14:16])

		fmt.Fprintf(out, "tcp.src_port %d\n", srcPort)
		fmt.Fprintf(out, "tcp.dst_port %d\n", dstPort)
		fmt.Fprintf(out, "tcp.seq %d\n", seq)
		fmt.Fprintf(out, "tcp.ack %d\n", ack)
		fmt.Fprintf(out, "tcp.data_offset_bytes %d\n", dataOffsetBytes)
		fmt.Fprintf(out, "tcp.flags %s\n", stringifyTCPFlags(tcp[13]))
		fmt.Fprintf(out, "tcp.window %d\n", window)
		fmt.Fprintf(out, "payload.length %d\n", totalLength-ihlBytes-dataOffsetBytes)

	case IPProtocolUDP:
		udp := ip[ihlBytes:]

		srcPort := binary.BigEndian.Uint16(udp[0:2])
		dstPort := binary.BigEndian.Uint16(udp[2:4])
		length := binary.BigEndian.Uint16(udp[4:6])

		fmt.Fprintf(out, "udp.src_port %d\n", srcPort)
		fmt.Fprintf(out, "udp.dst_port %d\n", dstPort)
		fmt.Fprintf(out, "udp.length %d\n", length)
		payloadLength := min(int(length), len(udp)) - UDPHeaderSize
		fmt.Fprintf(out, "payload.length %d\n", payloadLength)

	default:
		fmt.Fprintf(out, "payload.length %d\n", totalLength-ihlBytes)
	}
}
