package main

import (
	"bufio"
	"fmt"
	"io"
	"net/netip"
	"os"
	"strings"
)

func loadRoutes(path string) ([]Route, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("os.Open: %w", err)
	}
	defer file.Close()

	return parseRoutes(file)
}

func parseRoutes(in io.Reader) ([]Route, error) {
	var routes []Route

	scanner := bufio.NewScanner(in)
	scanner.Split(bufio.ScanLines)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++

		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, fmt.Errorf("line %d: expected 2 fields, got %d", lineNumber, len(fields))
		}

		network, err := parsePrefix(fields[0])
		if err != nil {
			return nil, fmt.Errorf("parsePrefix: %w, on line %d", err, lineNumber)
		}

		routes = append(routes, Route{
			Network:   network,
			Interface: fields[1],
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner: %w", err)
	}
	return routes, nil
}

func selectRoute(routes []Route, destination netip.Addr) (Route, bool) {
	found := false
	var best Route
	for _, route := range routes {
		if !route.Network.Contains(destination) {
			continue
		}
		if !found || route.Network.Bits() > best.Network.Bits() {
			best = route
			found = true
		}
		if route.Network.Bits() == best.Network.Bits() && route.Interface < best.Interface {
			best = route
		}
	}
	return best, found
}
