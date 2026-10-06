package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strings"
)

func readName(packet []byte, offset int) (name string, nextOffset int, err error) {
	return readNameWithin(packet, offset, len(packet))
}

func readNameWithin(packet []byte, offset, endOffset int) (name string, nextOffset int, err error) {
	var labels []string

	currentOffset := offset
	jumped := false

	visitedOffsets := make(map[int]struct{})

	for {
		if currentOffset < 0 || currentOffset >= len(packet) {
			return "", 0, fmt.Errorf("offset out of range: %d", currentOffset)
		}
		if !jumped && currentOffset >= endOffset {
			return "", 0, errors.New("name exceeds its data boundary")
		}

		if _, isVisited := visitedOffsets[currentOffset]; isVisited {
			return "", 0, errors.New("compression loop")
		}

		visitedOffsets[currentOffset] = struct{}{}

		lengthByte := packet[currentOffset]

		if lengthByte == 0 {
			if !jumped {
				nextOffset = currentOffset + 1
			}
			break
		}

		if lengthByte&0xC0 == 0xC0 {
			if currentOffset+1 >= len(packet) || (!jumped && currentOffset+1 >= endOffset) {
				return "", 0, errors.New("truncated compression pointer")
			}
			firstPart := uint16(lengthByte&0x3F) << 8
			secondPart := uint16(packet[currentOffset+1])

			pointerOffset := int(firstPart | secondPart)

			if pointerOffset >= len(packet) {
				return "", 0, errors.New("compression pointer out of range")
			}

			if !jumped {
				nextOffset = currentOffset + 2
			}

			currentOffset = pointerOffset
			jumped = true
			continue
		}

		labelLength := int(lengthByte)
		if labelLength > maxLabelLength {
			return "", 0, errors.New("label length exceeds 63 bytes")
		}

		labelStart := currentOffset + 1
		labelEnd := labelStart + labelLength

		if labelEnd > len(packet) || (!jumped && labelEnd > endOffset) {
			return "", 0, errors.New("truncated label end")
		}

		label := packet[labelStart:labelEnd]

		labels = append(labels, string(label))

		currentOffset = labelEnd
	}

	name = strings.Join(labels, ".") + "."

	return name, nextOffset, nil
}

func readRecordName(packet []byte, startOffset, endOffset int) (string, error) {
	if startOffset < 0 || startOffset >= endOffset || endOffset > len(packet) {
		return "", errors.New("record name is empty")
	}
	name, nextOffset, err := readNameWithin(packet, startOffset, endOffset)
	if err != nil {
		return "", err
	}
	if nextOffset != endOffset {
		return "", errors.New("record name does not match RDLENGTH")
	}
	return name, nil
}

func parseRecord(packet []byte, offset int) (answer DNSAnswer, nextOffset int, ttl uint32, supported bool, err error) {
	if offset < 0 || offset >= len(packet) {
		return DNSAnswer{}, 0, 0, false, fmt.Errorf("invalid record offset: %d", offset)
	}

	_, headerOffset, err := readName(packet, offset)
	if err != nil {
		return DNSAnswer{}, 0, 0, false, fmt.Errorf("read record name: %w", err)
	}

	if headerOffset+dnsRecordHeaderSize > len(packet) {
		return DNSAnswer{}, 0, 0, false, errors.New("record header exceeds packet")
	}

	recordType := DNSType(binary.BigEndian.Uint16(packet[headerOffset : headerOffset+2]))
	ttl = binary.BigEndian.Uint32(packet[headerOffset+4 : headerOffset+8])
	rdataLength := int(binary.BigEndian.Uint16(packet[headerOffset+8 : headerOffset+dnsRecordHeaderSize]))

	rdataStart := headerOffset + dnsRecordHeaderSize
	rdataEnd := rdataStart + rdataLength

	if rdataEnd > len(packet) {
		return DNSAnswer{}, 0, 0, false, errors.New("record data exceeds packet")
	}

	nextOffset = rdataEnd

	if !recordType.Valid() {
		return DNSAnswer{}, nextOffset, ttl, false, nil
	}

	var value string

	switch recordType {
	case TypeA:
		if rdataLength != net.IPv4len {
			return DNSAnswer{}, 0, 0, false, fmt.Errorf("invalid A record length: %d", rdataLength)
		}
		value = net.IP(packet[rdataStart:rdataEnd]).String()

	case TypeAAAA:
		if rdataLength != net.IPv6len {
			return DNSAnswer{}, 0, 0, false, fmt.Errorf("invalid AAAA record length: %d", rdataLength)
		}
		data := packet[rdataStart:rdataEnd]
		address := netip.AddrFrom16([16]byte(data))
		value = address.String()
		if address.Is4In6() {
			value = fmt.Sprintf("::ffff:%x:%x", binary.BigEndian.Uint16(data[12:14]), binary.BigEndian.Uint16(data[14:16]))
		}

	case TypeCNAME, TypeNS:
		value, err = readRecordName(packet, rdataStart, rdataEnd)
		if err != nil {
			return DNSAnswer{}, 0, 0, false, fmt.Errorf("read record value: %w", err)
		}

	case TypeMX:
		if rdataLength < 3 {
			return DNSAnswer{}, 0, 0, false, fmt.Errorf("invalid MX record length: %d", rdataLength)
		}

		preference := binary.BigEndian.Uint16(packet[rdataStart : rdataStart+2])

		mailServer, err := readRecordName(packet, rdataStart+2, rdataEnd)
		if err != nil {
			return DNSAnswer{}, 0, 0, false, fmt.Errorf("read MX name: %w", err)
		}

		value = fmt.Sprintf("%d %s", preference, mailServer)

	case TypeTXT:
		var builder strings.Builder

		for pos := rdataStart; pos < rdataEnd; {
			length := int(packet[pos])
			pos++

			if pos+length > rdataEnd {
				return DNSAnswer{}, 0, 0, false, errors.New("invalid TXT record")
			}

			builder.Write(packet[pos : pos+length])
			pos += length
		}

		value = builder.String()
	}

	answer = DNSAnswer{
		Type:  recordType,
		Value: value,
		TTL:   ttl,
	}

	return answer, nextOffset, ttl, true, nil
}

func parseResponse(packet []byte) (DNSResponse, error) {
	if len(packet) < dnsHeaderSize {
		return DNSResponse{}, errors.New("truncated packet")
	}

	flags := binary.BigEndian.Uint16(packet[2:4])

	status := DNSStatus(flags & dnsRCodeMask)

	questionsCount := binary.BigEndian.Uint16(packet[4:6])
	answerCount := binary.BigEndian.Uint16(packet[6:8])
	offset := dnsHeaderSize

	for range questionsCount {
		_, nextOffset, err := readName(packet, offset)
		if err != nil {
			return DNSResponse{}, fmt.Errorf("readName: %w", err)
		}

		offset = nextOffset

		if offset+dnsQuestionTailSize > len(packet) {
			return DNSResponse{}, errors.New("offset exceed packet length")
		}

		offset += dnsQuestionTailSize
	}

	answers := make([]DNSAnswer, 0, answerCount)
	var minTTL uint32
	var hasTTL bool

	for range answerCount {
		answer, nextOffset, ttl, supported, err := parseRecord(packet, offset)
		if err != nil {
			return DNSResponse{}, fmt.Errorf("parse answer record: %w", err)
		}

		offset = nextOffset

		if !hasTTL {
			minTTL = ttl
			hasTTL = true
		} else if ttl < minTTL {
			minTTL = ttl
		}

		if supported {
			answers = append(answers, answer)
		}
	}

	return DNSResponse{
		Status:      status,
		Answers:     answers,
		AnswerCount: answerCount,
		MinTTL:      minTTL,
	}, nil
}
