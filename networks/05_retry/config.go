package main

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type HTTPMethod uint8

const (
	MethodUnknown HTTPMethod = iota
	MethodGET
	MethodHEAD
	MethodPOST
	MethodPUT
	MethodDELETE
	MethodOPTIONS
	MethodTRACE
)

func (m HTTPMethod) String() string {
	switch m {
	case MethodGET:
		return "GET"
	case MethodHEAD:
		return "HEAD"
	case MethodPOST:
		return "POST"
	case MethodPUT:
		return "PUT"
	case MethodDELETE:
		return "DELETE"
	case MethodOPTIONS:
		return "OPTIONS"
	case MethodTRACE:
		return "TRACE"
	default:
		return ""
	}
}

func parseHTTPMethod(raw string) (HTTPMethod, error) {
	switch strings.ToUpper(raw) {
	case "GET":
		return MethodGET, nil
	case "HEAD":
		return MethodHEAD, nil
	case "POST":
		return MethodPOST, nil
	case "PUT":
		return MethodPUT, nil
	case "DELETE":
		return MethodDELETE, nil
	case "OPTIONS":
		return MethodOPTIONS, nil
	case "TRACE":
		return MethodTRACE, nil
	default:
		return MethodUnknown, fmt.Errorf("unsupported HTTP method: %s", raw)
	}
}

type Config struct {
	URL            string
	Method         HTTPMethod
	MaxAttempts    int
	IdempotencyKey string
}

func parseArgs(args []string) (Config, error) {
	if len(args) == 0 {
		return Config{}, errors.New("not enough args: URL is required")
	}

	cfg := Config{
		URL:         args[0],
		Method:      MethodGET,
		MaxAttempts: 5,
	}

	for i := 1; i < len(args); {
		switch args[i] {
		case "--method":
			if i+1 >= len(args) {
				return Config{}, errors.New("--method requires a value")
			}

			method, err := parseHTTPMethod(args[i+1])
			if err != nil {
				return Config{}, err
			}

			cfg.Method = method
			i += 2

		case "--max-attempts":
			if i+1 >= len(args) {
				return Config{}, errors.New("--max-attempts requires a value")
			}

			maxAttempts, err := strconv.Atoi(args[i+1])
			if err != nil {
				return Config{}, fmt.Errorf("invalid max attempts: %w", err)
			}

			if maxAttempts <= 0 {
				return Config{}, errors.New("max attempts must be greater than zero")
			}

			cfg.MaxAttempts = maxAttempts
			i += 2

		case "--idempotency-key":
			if i+1 >= len(args) {
				return Config{}, errors.New("--idempotency-key requires a value")
			}

			cfg.IdempotencyKey = args[i+1]
			i += 2

		default:
			return Config{}, fmt.Errorf("unknown argument: %s", args[i])
		}
	}

	parsedURL, err := url.Parse(cfg.URL)
	if err != nil {
		return Config{}, fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme != "http" {
		return Config{}, errors.New("only http scheme is supported")
	}

	if parsedURL.Host == "" {
		return Config{}, errors.New("URL host is required")
	}

	return cfg, nil
}
