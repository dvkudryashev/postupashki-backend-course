package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

func encodeName(name string) ([]byte, error) {
	if name == "." {
		return []byte{0}, nil
	}
	name = strings.TrimSuffix(name, ".")

	labels := strings.Split(name, ".")

	var result []byte

	for _, label := range labels {
		if label == "" {
			return nil, errors.New("label is empty")
		}

		if len(label) > maxLabelLength {
			return nil, errors.New("label length exceeds 63 bytes")
		}

		result = append(result, byte(len(label)))
		result = append(result, label...)
	}

	result = append(result, 0)
	if len(result) > maxNameLength {
		return nil, errors.New("name length exceeds 255 bytes")
	}

	return result, nil
}

func buildQuery(id uint16, name string, queryType DNSType) ([]byte, error) {
	encodedName, err := encodeName(name)
	if err != nil {
		return nil, fmt.Errorf("encodeName: %w", err)
	}

	query := make([]byte, dnsHeaderSize)

	binary.BigEndian.PutUint16(query[0:2], id)
	binary.BigEndian.PutUint16(query[2:4], flagRD)
	binary.BigEndian.PutUint16(query[4:6], questionCount)

	query = append(query, encodedName...)
	query = binary.BigEndian.AppendUint16(query, uint16(queryType))
	query = binary.BigEndian.AppendUint16(query, classIN)

	return query, nil
}
