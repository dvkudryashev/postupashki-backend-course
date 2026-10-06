package main

import (
	"testing"
	"time"
)

func TestRetryAfterBounds(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  time.Duration
		valid bool
	}{
		{"0", 0, true},
		{" 1 ", time.Second, true},
		{"3", 3 * time.Second, true},
		{"9223372036", 9223372036 * time.Second, true},
		{"9223372037", 0, false},
		{"18446744073709551615", 0, false},
		{"18446744073709551616", 0, false},
		{"-1", 0, false},
		{"+1", 0, false},
		{"", 0, false},
		{"Wed, 21 Oct 2015 07:28:00 GMT", 0, false},
	} {
		t.Run(tc.value, func(t *testing.T) {
			delay, valid := parseRetryAfter(tc.value)
			if valid != tc.valid || delay != tc.want {
				t.Fatalf("parseRetryAfter = %v, %v; want %v, %v", delay, valid, tc.want, tc.valid)
			}
		})
	}
}

func TestInvalidRetryAfterFallsBackToJitter(t *testing.T) {
	delay := retryDelay(AttemptResult{RetryAfter: "9223372037"}, 2)
	if delay < 0 || delay > 400*time.Millisecond {
		t.Fatalf("invalid Retry-After produced delay %v", delay)
	}
}
