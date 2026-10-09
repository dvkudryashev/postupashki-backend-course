package main

import (
	"strconv"
	"strings"
)

type DNSType uint16

const (
	TypeA     DNSType = 1
	TypeNS    DNSType = 2
	TypeCNAME DNSType = 5
	TypeMX    DNSType = 15
	TypeTXT   DNSType = 16
	TypeAAAA  DNSType = 28
)

type DNSStatus uint16

const (
	NOERROR  DNSStatus = 0
	FORMERR  DNSStatus = 1
	SERVFAIL DNSStatus = 2
	NXDOMAIN DNSStatus = 3
	REFUSED  DNSStatus = 5
)

type DNSAnswer struct {
	Type  DNSType
	Value string
	TTL   uint32
}

type DNSResponse struct {
	Status      DNSStatus
	Answers     []DNSAnswer
	AnswerCount uint16
	MinTTL      uint32
}

func (t DNSType) String() string {
	switch t {
	case TypeA:
		return "A"
	case TypeNS:
		return "NS"
	case TypeCNAME:
		return "CNAME"
	case TypeMX:
		return "MX"
	case TypeTXT:
		return "TXT"
	case TypeAAAA:
		return "AAAA"
	default:
		return ""
	}
}

func (t DNSType) Valid() bool {
	switch t {
	case TypeA, TypeNS, TypeCNAME, TypeMX, TypeTXT, TypeAAAA:
		return true
	default:
		return false
	}
}

func parseDNSType(name string) (DNSType, bool) {
	switch strings.ToUpper(name) {
	case "A":
		return TypeA, true
	case "NS":
		return TypeNS, true
	case "CNAME":
		return TypeCNAME, true
	case "MX":
		return TypeMX, true
	case "TXT":
		return TypeTXT, true
	case "AAAA":
		return TypeAAAA, true
	default:
		return 0, false
	}
}

func (s DNSStatus) String() string {
	switch s {
	case NOERROR:
		return "NOERROR"
	case FORMERR:
		return "FORMERR"
	case SERVFAIL:
		return "SERVFAIL"
	case NXDOMAIN:
		return "NXDOMAIN"
	case REFUSED:
		return "REFUSED"
	default:
		return "RCODE" + strconv.Itoa(int(s))
	}
}

const (
	dnsHeaderSize              = 12
	dnsRecordHeaderSize        = 10
	dnsQuestionTailSize        = 4
	dnsRCodeMask        uint16 = 0x000F
	maxLabelLength             = 63
	maxNameLength              = 255
	flagRD              uint16 = 1 << 8
	questionCount       uint16 = 1
	classIN             uint16 = 1
)
