package main

import (
	"io"
	"net/http"
	"time"
)

const requestTimeout = 5 * time.Second

type HTTPStatus int

type AttemptResult struct {
	StatusCode HTTPStatus
	RetryAfter string
	Err        error
}

func newHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: requestTimeout,
	}
}

func doAttempt(client *http.Client, cfg Config) AttemptResult {
	req, err := http.NewRequest(cfg.Method.String(), cfg.URL, nil)
	if err != nil {
		return AttemptResult{Err: err}
	}

	if cfg.IdempotencyKey != "" {
		req.Header.Set("Idempotency-Key", cfg.IdempotencyKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return AttemptResult{Err: err}
	}
	defer resp.Body.Close()

	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		return AttemptResult{Err: err}
	}

	return AttemptResult{
		StatusCode: HTTPStatus(resp.StatusCode),
		RetryAfter: resp.Header.Get("Retry-After"),
	}
}
