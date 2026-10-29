package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

func main() {
	if err := run(os.Args[1:], os.Stdin); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, in io.Reader) error {
	if len(args) != 2 {
		return errors.New("usage: dns <server address> <port>")
	}

	server := args[0]
	port := args[1]

	conn, err := openConnection(server, port)
	if err != nil {
		return err
	}
	defer conn.Close()

	scanner := bufio.NewScanner(in)
	cache := make(map[CacheKey]CacheEntry)

	var requestID uint16 = 1

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		lineFields := strings.Fields(line)
		if len(lineFields) != 2 {
			return fmt.Errorf("invalid query: %q", line)
		}

		domainName, rawType := lineFields[0], lineFields[1]

		queryType, ok := parseDNSType(rawType)
		if !ok {
			return fmt.Errorf("unsupported DNS type: %s", rawType)
		}

		key := makeCacheKey(domainName, queryType)

		cacheEntry, cached := getCached(cache, key, time.Now())
		if cached {
			printResponse(domainName, rawType, NOERROR, cacheEntry.Answers)
			continue
		}

		id := requestID
		requestID++

		queryData, err := buildQuery(id, domainName, queryType)
		if err != nil {
			return err
		}

		err = sendQuery(conn, queryData)
		if err != nil {
			return err
		}

		deadline := time.Now().Add(responseTimeout)

		responseData, err := receiveResponse(conn, id, deadline)
		if errors.Is(err, errTimeout) {
			printTimeout(domainName, rawType)
			return err
		}
		if err != nil {
			return err
		}

		parsedResponse, err := parseResponse(responseData)
		if err != nil {
			return err
		}

		printResponse(domainName, rawType, parsedResponse.Status, parsedResponse.Answers)
		putCache(cache, key, parsedResponse, time.Now())
	}

	return scanner.Err()
}
