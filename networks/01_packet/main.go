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

type EthernetInfo struct {
	Src       [6]byte
	Dst       [6]byte
	EtherType EtherType
}

type IPv4Info struct {
	Version             uint8
	TTL                 uint8
	Protocol            uint8
	IHLBytes            int
	TotalLength         int
	ID                  uint16
	Flags               uint16
	FragmentOffsetBytes uint16
	Src                 [4]byte
	Dst                 [4]byte
	ChecksumValid       bool
}

type TCPInfo struct {
	SrcPort         uint16
	DstPort         uint16
	Window          uint16
	Seq             uint32
	Ack             uint32
	DataOffsetBytes int
	Flags           uint8
}

type UDPInfo struct {
	SrcPort uint16
	DstPort uint16
	Length  uint16
}

type Packet struct {
	Ethernet      EthernetInfo
	IPv4          *IPv4Info
	TCP           *TCPInfo
	UDP           *UDPInfo
	PayloadLength int
}

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
	packet, err := parseFrame(frame)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(out)
	printPacket(writer, packet)
	return writer.Flush()
}

func parseFrame(raw []byte) (Packet, error) {
	if len(raw) < EthernetHeaderSize {
		return Packet{}, fmt.Errorf("truncated Ethernet header: %d bytes", len(raw))
	}

	var packet Packet
	packet.Ethernet.Dst = [6]byte(raw[0:6])
	packet.Ethernet.Src = [6]byte(raw[6:12])
	packet.Ethernet.EtherType = EtherType(binary.BigEndian.Uint16(raw[12:14]))

	if packet.Ethernet.EtherType != EtherTypeIPv4 {
		return packet, nil
	}

	ip := raw[EthernetHeaderSize:]

	if len(ip) < IPv4HeaderSize {
		return Packet{}, fmt.Errorf("truncated IPv4 header: %d bytes", len(ip))
	}

	version := ip[0] >> 4
	if version != 4 {
		return Packet{}, fmt.Errorf("invalid IPv4 version: %d", version)
	}

	ihlBytes := int(ip[0]&0x0f) * IPv4WordSize
	if ihlBytes < IPv4HeaderSize || ihlBytes > len(ip) {
		return Packet{}, fmt.Errorf("invalid IPv4 header length: %d", ihlBytes)
	}

	totalLength := int(binary.BigEndian.Uint16(ip[2:4]))
	if totalLength < ihlBytes || totalLength > len(ip) {
		return Packet{}, fmt.Errorf("invalid IPv4 total length: %d", totalLength)
	}

	ip = ip[:totalLength]

	flagsAndOffset := binary.BigEndian.Uint16(ip[6:8])
	packet.IPv4 = &IPv4Info{
		Version:             version,
		TTL:                 ip[8],
		Protocol:            ip[9],
		IHLBytes:            ihlBytes,
		TotalLength:         totalLength,
		ID:                  binary.BigEndian.Uint16(ip[4:6]),
		Flags:               flagsAndOffset & (IPFlagDF | IPFlagMF),
		FragmentOffsetBytes: parseIPOffset(flagsAndOffset),
		Src:                 [4]byte(ip[12:16]),
		Dst:                 [4]byte(ip[16:20]),
		ChecksumValid:       ipChecksumValid(ip[:ihlBytes]),
	}
	packet.PayloadLength = totalLength - ihlBytes
	if packet.IPv4.FragmentOffsetBytes != 0 {
		return packet, nil
	}

	transport := ip[ihlBytes:]
	switch packet.IPv4.Protocol {
	case IPProtocolTCP:
		if len(transport) < TCPHeaderSize {
			return Packet{}, fmt.Errorf("truncated TCP header: %d bytes", len(transport))
		}
		dataOffsetBytes := int(transport[12]>>4) * IPv4WordSize
		if dataOffsetBytes < TCPHeaderSize || dataOffsetBytes > len(transport) {
			return Packet{}, fmt.Errorf("invalid TCP header length: %d", dataOffsetBytes)
		}
		packet.TCP = &TCPInfo{
			SrcPort:         binary.BigEndian.Uint16(transport[0:2]),
			DstPort:         binary.BigEndian.Uint16(transport[2:4]),
			Window:          binary.BigEndian.Uint16(transport[14:16]),
			Seq:             binary.BigEndian.Uint32(transport[4:8]),
			Ack:             binary.BigEndian.Uint32(transport[8:12]),
			DataOffsetBytes: dataOffsetBytes,
			Flags:           transport[13],
		}
		packet.PayloadLength -= dataOffsetBytes

	case IPProtocolUDP:
		if len(transport) < UDPHeaderSize {
			return Packet{}, fmt.Errorf("truncated UDP header: %d bytes", len(transport))
		}
		length := binary.BigEndian.Uint16(transport[4:6])
		if length < UDPHeaderSize || (packet.IPv4.Flags&IPFlagMF == 0 && int(length) > len(transport)) {
			return Packet{}, fmt.Errorf("invalid UDP length: %d", length)
		}
		packet.UDP = &UDPInfo{
			SrcPort: binary.BigEndian.Uint16(transport[0:2]),
			DstPort: binary.BigEndian.Uint16(transport[2:4]),
			Length:  length,
		}
		packet.PayloadLength = min(int(length), len(transport)) - UDPHeaderSize
	}
	return packet, nil
}

func readFrame(in io.Reader) ([]byte, error) {
	data, err := io.ReadAll(in)
	if err != nil {
		return nil, err
	}

	dump := strings.ReplaceAll(strings.Join(strings.Fields(string(data)), ""), ":", "")
	return hex.DecodeString(dump)
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

func (e EthernetInfo) String() string {
	return fmt.Sprintf("eth.dst %s\neth.src %s\neth.ethertype 0x%04x\n",
		formatMAC(e.Dst[:]), formatMAC(e.Src[:]), e.EtherType)
}

func (ip IPv4Info) String() string {
	return fmt.Sprintf("ip.version %d\nip.ihl_bytes %d\nip.total_length %d\nip.id 0x%04x\nip.flags %s\nip.frag_offset %d\nip.ttl %d\nip.protocol %d\nip.src %s\nip.dst %s\nip.checksum_valid %t\n",
		ip.Version, ip.IHLBytes, ip.TotalLength, ip.ID, stringifyIPFlags(ip.Flags), ip.FragmentOffsetBytes,
		ip.TTL, ip.Protocol, formatIPv4(ip.Src[:]), formatIPv4(ip.Dst[:]), ip.ChecksumValid)
}

func (tcp TCPInfo) String() string {
	return fmt.Sprintf("tcp.src_port %d\ntcp.dst_port %d\ntcp.seq %d\ntcp.ack %d\ntcp.data_offset_bytes %d\ntcp.flags %s\ntcp.window %d\n",
		tcp.SrcPort, tcp.DstPort, tcp.Seq, tcp.Ack, tcp.DataOffsetBytes, stringifyTCPFlags(tcp.Flags), tcp.Window)
}

func (udp UDPInfo) String() string {
	return fmt.Sprintf("udp.src_port %d\nudp.dst_port %d\nudp.length %d\n", udp.SrcPort, udp.DstPort, udp.Length)
}

func (packet Packet) String() string {
	var buffer strings.Builder
	fmt.Fprint(&buffer, packet.Ethernet)

	if packet.IPv4 == nil {
		return buffer.String()
	}

	fmt.Fprint(&buffer, packet.IPv4)
	if packet.TCP != nil {
		fmt.Fprint(&buffer, packet.TCP)
	}
	if packet.UDP != nil {
		fmt.Fprint(&buffer, packet.UDP)
	}
	fmt.Fprintf(&buffer, "payload.length %d\n", packet.PayloadLength)
	return buffer.String()
}

func printPacket(out io.Writer, packet Packet) {
	fmt.Fprint(out, packet)
}
