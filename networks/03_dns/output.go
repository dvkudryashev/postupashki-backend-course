package main

import "fmt"

func printResponse(domainName, rawType string, status DNSStatus, answers []DNSAnswer) {
	fmt.Printf("query %s %s\n", domainName, rawType)
	fmt.Printf("status %s\n", status)

	for _, answer := range answers {
		fmt.Printf("answer %s %s %d\n", answer.Type, answer.Value, answer.TTL)
	}
	fmt.Printf("end\n")
}

func printTimeout(domainName, rawType string) {
	fmt.Printf("query %s %s\n", domainName, rawType)
	fmt.Printf("status TIMEOUT\n")
	fmt.Printf("end\n")
}
