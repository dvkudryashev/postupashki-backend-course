package main

import (
	"fmt"
	"io"
	"time"
)

func printAttempt(out io.Writer, attempt int, result AttemptResult) {
	if result.Err != nil {
		fmt.Fprintf(out, "attempt %d error %v\n", attempt, result.Err)
		return
	}

	fmt.Fprintf(out, "attempt %d status %d\n", attempt, result.StatusCode)
}

func printSleep(out io.Writer, delay time.Duration) {
	fmt.Fprintf(out, "sleep_ms %d\n", delay/time.Millisecond)
}

func printResult(out io.Writer, success bool, attempts int) {
	if success {
		fmt.Fprintf(out, "result success attempts %d\n", attempts)
		return
	}

	fmt.Fprintf(out, "result failure attempts %d\n", attempts)
}
