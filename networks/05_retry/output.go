package main

import (
	"fmt"
	"time"
)

func printAttempt(attempt int, result AttemptResult) {
	if result.Err != nil {
		fmt.Printf("attempt %d error %v\n", attempt, result.Err)
		return
	}

	fmt.Printf("attempt %d status %d\n", attempt, result.StatusCode)
}

func printSleep(delay time.Duration) {
	fmt.Printf("sleep_ms %d\n", delay/time.Millisecond)
}

func printResult(success bool, attempts int) {
	if success {
		fmt.Printf("result success attempts %d\n", attempts)
		return
	}

	fmt.Printf("result failure attempts %d\n", attempts)
}
