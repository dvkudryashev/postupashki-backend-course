package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type RunResult struct {
	Success  bool
	Attempts int
}

func main() {
	cfg, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	result := run(cfg, newHTTPClient(), time.Sleep, os.Stdout)
	if !result.Success {
		os.Exit(1)
	}
}

func run(cfg Config, client *http.Client, sleep func(time.Duration), out io.Writer) RunResult {
	attempt := 1

	for {
		result := doAttempt(client, cfg)

		printAttempt(out, attempt, result)

		if result.Err == nil && isSuccessStatus(result.StatusCode) {
			printResult(out, true, attempt)
			return RunResult{Success: true, Attempts: attempt}
		}

		if !shouldRetry(result, cfg, attempt) {
			printResult(out, false, attempt)
			return RunResult{Success: false, Attempts: attempt}
		}

		nextAttempt := attempt + 1

		delay := retryDelay(result, nextAttempt)

		printSleep(out, delay)
		sleep(delay)

		attempt = nextAttempt
	}
}
