package main

import (
	"errors"
	"fmt"
)

func parseArgs(args []string) (Config, error) {
	if len(args) < 1 {
		return Config{}, errors.New("no args provided")
	}

	if args[0] == "" {
		return Config{}, errors.New("no mode provided")
	}

	cfg := Config{}
	switch args[0] {
	case "subnet":
		if len(args) != 2 {
			return Config{}, fmt.Errorf("for subnet expected 2 args, got: %d", len(args))
		}

		cfg.Mode = ModeSubnet
		cfg.Subnet = args[1]
	case "route":
		if len(args) != 3 {
			return Config{}, fmt.Errorf("for route expected 3 args, got: %d", len(args))
		}

		cfg.Mode = ModeRoute
		cfg.RouteFile = args[1]
		cfg.Destination = args[2]
	default:
		return Config{}, fmt.Errorf("unknown mode: %q", args[0])
	}

	return cfg, nil
}
