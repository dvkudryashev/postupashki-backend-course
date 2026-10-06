package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	cfg, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	client := newHTTPClient()

	attempt := 1

	for {
		result := doAttempt(client, cfg)

		printAttempt(attempt, result)

		if result.Err == nil && isSuccessStatus(result.StatusCode) {
			printResult(true, attempt)
			return
		}

		if !shouldRetry(result, cfg, attempt) {
			printResult(false, attempt)
			os.Exit(1)
		}

		nextAttempt := attempt + 1

		delay := retryDelay(result, nextAttempt)

		printSleep(delay)
		time.Sleep(delay)

		attempt = nextAttempt
	}
}
