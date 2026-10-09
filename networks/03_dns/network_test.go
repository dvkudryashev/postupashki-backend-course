package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"
)

func testDNSResponse(query []byte, answerCount int) []byte {
	response := make([]byte, dnsHeaderSize)
	binary.BigEndian.PutUint16(response[0:2], binary.BigEndian.Uint16(query[0:2]))
	binary.BigEndian.PutUint16(response[2:4], 0x8180)
	binary.BigEndian.PutUint16(response[4:6], 1)
	binary.BigEndian.PutUint16(response[6:8], uint16(answerCount))
	response = append(response, query[dnsHeaderSize:]...)

	for i := range answerCount {
		response = append(response, 0xc0, 0x0c)
		response = binary.BigEndian.AppendUint16(response, uint16(TypeA))
		response = binary.BigEndian.AppendUint16(response, classIN)
		response = binary.BigEndian.AppendUint32(response, 60)
		response = binary.BigEndian.AppendUint16(response, 4)
		response = append(response, 192, 0, 2, byte(i+1))
	}

	return response
}

func TestReceiveLargeResponseAndNextQuery(t *testing.T) {
	server, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { server.Close() })

	address := server.LocalAddr().(*net.UDPAddr)
	conn, err := openConnection(address.IP.String(), strconv.Itoa(address.Port))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })

	for _, tc := range []struct {
		id          uint16
		name        string
		answerCount int
	}{
		{17, "example.com", 40},
		{18, "second.example", 1},
	} {
		query, err := buildQuery(tc.id, tc.name, TypeA)
		if err != nil {
			t.Fatal(err)
		}
		if err := sendQuery(conn, query); err != nil {
			t.Fatal(err)
		}
		if err := server.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
			t.Fatal(err)
		}

		buffer := make([]byte, 1024)
		n, peer, err := server.ReadFromUDP(buffer)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(buffer[:n], query) {
			t.Fatalf("received query = %x; want %x", buffer[:n], query)
		}

		response := testDNSResponse(buffer[:n], tc.answerCount)
		if tc.answerCount == 40 && len(response) <= 512 {
			t.Fatalf("large response is only %d bytes", len(response))
		}
		if _, err := server.WriteToUDP(response, peer); err != nil {
			t.Fatal(err)
		}

		data, err := receiveResponse(conn, tc.id, time.Now().Add(3*time.Second))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(data, response) {
			t.Fatalf("response for %s = %d bytes; want %d bytes", tc.name, len(data), len(response))
		}
		parsedResponse, err := parseResponse(data)
		if err != nil {
			t.Fatal(err)
		}
		if parsedResponse.Status != NOERROR || int(parsedResponse.AnswerCount) != tc.answerCount || len(parsedResponse.Answers) != tc.answerCount || parsedResponse.MinTTL != 60 {
			t.Fatalf("response for %s = %+v; want %d A records with TTL 60", tc.name, parsedResponse, tc.answerCount)
		}
		for i, answer := range parsedResponse.Answers {
			want := fmt.Sprintf("192.0.2.%d", i+1)
			if answer.Type != TypeA || answer.Value != want || answer.TTL != 60 {
				t.Fatalf("answer %d for %s = %+v; want A %s 60", i, tc.name, answer, want)
			}
		}
	}
}

func TestClosedUDPPortReturnsTimeout(t *testing.T) {
	server, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	address := server.LocalAddr().(*net.UDPAddr)
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}

	conn, err := openConnection(address.IP.String(), strconv.Itoa(address.Port))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })

	query, err := buildQuery(1, "example.com", TypeA)
	if err != nil {
		t.Fatal(err)
	}
	if err := sendQuery(conn, query); err != nil {
		t.Fatal(err)
	}
	data, err := receiveResponse(conn, 1, time.Now().Add(time.Second))
	if !errors.Is(err, errTimeout) || data != nil {
		t.Fatalf("closed UDP port: response=%x, error=%v; want timeout without response", data, err)
	}
}
