package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	cfg, err := parseArgs(args)
	if err != nil {
		return fmt.Errorf("parseArgs: %w", err)
	}

	switch cfg.Mode {
	case ModeSubnet:
		prefix, err := parsePrefix(cfg.Subnet)
		if err != nil {
			return fmt.Errorf("parsePrefix: %w", err)
		}

		info := calculateSubnet(prefix)
		return printSubnet(out, info)

	case ModeRoute:
		destination, err := parseIPv4(cfg.Destination)
		if err != nil {
			return fmt.Errorf("parseIPv4: %w", err)
		}
		routes, err := loadRoutes(cfg.RouteFile)
		if err != nil {
			return fmt.Errorf("loadRoutes: %w", err)
		}
		best, found := selectRoute(routes, destination)
		if found {
			return printRoute(out, best)
		}
		if err := printUnreachable(out); err != nil {
			return err
		}
		return errUnreachable
	}
	return nil
}
