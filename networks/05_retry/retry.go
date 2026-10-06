package main

import (
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	baseDelay            = 200 * time.Millisecond
	maxDelay             = 2000 * time.Millisecond
	maxRetryAfterSeconds = uint64(math.MaxInt64 / int64(time.Second))
)

func isSuccessStatus(status HTTPStatus) bool {
	return status >= http.StatusOK && status < http.StatusBadRequest
}

func isRetryableStatus(status HTTPStatus) bool {
	switch status {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func isRetryableMethod(method HTTPMethod, idempotencyKey string) bool {
	switch method {
	case MethodGET,
		MethodHEAD,
		MethodPUT,
		MethodDELETE,
		MethodOPTIONS,
		MethodTRACE:
		return true

	case MethodPOST:
		return idempotencyKey != ""

	default:
		return false
	}
}

func shouldRetry(result AttemptResult, cfg Config, attempt int) bool {
	if attempt >= cfg.MaxAttempts {
		return false
	}

	if !isRetryableMethod(cfg.Method, cfg.IdempotencyKey) {
		return false
	}

	if result.Err != nil {
		return true
	}

	return isRetryableStatus(result.StatusCode)
}

func parseRetryAfter(value string) (time.Duration, bool) {
	value = strings.TrimSpace(value)

	if value == "" {
		return 0, false
	}

	seconds, err := strconv.ParseUint(value, 10, 64)
	if err != nil || seconds > maxRetryAfterSeconds {
		return 0, false
	}

	return time.Duration(seconds) * time.Second, true
}

func backoffDelay(nextAttempt int) time.Duration {
	delay := baseDelay

	for attempt := 1; attempt < nextAttempt; attempt++ {
		delay *= 2

		if delay >= maxDelay {
			delay = maxDelay
			break
		}
	}

	maxMilliseconds := int(delay / time.Millisecond)
	randomMilliseconds := rand.Intn(maxMilliseconds + 1)

	return time.Duration(randomMilliseconds) * time.Millisecond
}

func retryDelay(result AttemptResult, nextAttempt int) time.Duration {
	if delay, ok := parseRetryAfter(result.RetryAfter); ok {
		return delay
	}

	return backoffDelay(nextAttempt)
}
