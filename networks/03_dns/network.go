package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"
)

const (
	dnsUDPBufferSize = 65535
	responseTimeout  = 5 * time.Second
)

var errTimeout = errors.New("timeout")

func openConnection(server, port string) (*net.UDPConn, error) {
	address := net.JoinHostPort(server, port)
	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, fmt.Errorf("net.ResolveUDPAddr: %w", err)
	}

	return net.DialUDP("udp", nil, udpAddr)
}

func sendQuery(conn *net.UDPConn, data []byte) error {
	_, err := conn.Write(data)
	if err != nil {
		return fmt.Errorf("conn.Write: %w", err)
	}
	return nil
}

func receiveResponse(conn *net.UDPConn, requestID uint16, deadline time.Time) ([]byte, error) {
	err := conn.SetReadDeadline(deadline)
	if err != nil {
		return nil, err
	}

	buffer := make([]byte, dnsUDPBufferSize)

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			return nil, errTimeout
		}

		if n < 2 {
			continue
		}

		if requestID != binary.BigEndian.Uint16(buffer[0:2]) {
			continue
		}
		return buffer[:n], nil
	}
}
