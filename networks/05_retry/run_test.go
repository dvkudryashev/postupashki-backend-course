package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunRetriesUntilSuccess(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(out http.ResponseWriter, req *http.Request) {
		if requests.Add(1) < 3 {
			out.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		out.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var out bytes.Buffer
	var delays []time.Duration
	result := run(Config{URL: server.URL, Method: MethodGET, MaxAttempts: 3}, newHTTPClient(), func(delay time.Duration) {
		delays = append(delays, delay)
	}, &out)

	if !result.Success || result.Attempts != 3 || requests.Load() != 3 || len(delays) != 2 {
		t.Fatalf("result=%+v, requests=%d, delays=%v", result, requests.Load(), delays)
	}
	for i, limit := range []time.Duration{400 * time.Millisecond, 800 * time.Millisecond} {
		if delays[i] < 0 || delays[i] > limit {
			t.Fatalf("delay %d = %v, limit=%v", i, delays[i], limit)
		}
	}

	want := fmt.Sprintf("attempt 1 status 503\nsleep_ms %d\nattempt 2 status 503\nsleep_ms %d\nattempt 3 status 200\nresult success attempts 3\n", delays[0]/time.Millisecond, delays[1]/time.Millisecond)
	if out.String() != want {
		t.Fatalf("output=%q, want=%q", out.String(), want)
	}
}

func TestRunRespectsRetryPolicy(t *testing.T) {
	for _, tc := range []struct {
		name         string
		method       HTTPMethod
		key          string
		status       HTTPStatus
		maxAttempts  int
		wantSuccess  bool
		wantAttempts int
	}{
		{"first success", MethodGET, "", http.StatusOK, 3, true, 1},
		{"redirect is success", MethodGET, "", http.StatusFound, 3, true, 1},
		{"terminal status", MethodGET, "", http.StatusNotFound, 3, false, 1},
		{"single attempt", MethodGET, "", http.StatusServiceUnavailable, 1, false, 1},
		{"attempt limit", MethodGET, "", http.StatusServiceUnavailable, 3, false, 3},
		{"post without key", MethodPOST, "", http.StatusServiceUnavailable, 3, false, 1},
		{"post with key", MethodPOST, "request-key", http.StatusServiceUnavailable, 3, false, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(out http.ResponseWriter, req *http.Request) {
				requests.Add(1)
				if req.Method != tc.method.String() || req.Header.Get("Idempotency-Key") != tc.key {
					t.Errorf("method=%s, key=%q", req.Method, req.Header.Get("Idempotency-Key"))
				}
				out.WriteHeader(int(tc.status))
			}))
			defer server.Close()

			var out bytes.Buffer
			var delays []time.Duration
			cfg := Config{URL: server.URL, Method: tc.method, IdempotencyKey: tc.key, MaxAttempts: tc.maxAttempts}
			result := run(cfg, newHTTPClient(), func(delay time.Duration) {
				delays = append(delays, delay)
			}, &out)

			if result.Success != tc.wantSuccess || result.Attempts != tc.wantAttempts || int(requests.Load()) != tc.wantAttempts {
				t.Fatalf("result=%+v, requests=%d", result, requests.Load())
			}
			if len(delays) != tc.wantAttempts-1 {
				t.Fatalf("unexpected sleeps: %v", delays)
			}
			status := "failure"
			if tc.wantSuccess {
				status = "success"
			}
			wantResult := fmt.Sprintf("result %s attempts %d\n", status, tc.wantAttempts)
			if strings.Count(out.String(), "result ") != 1 || !strings.HasSuffix(out.String(), wantResult) {
				t.Fatalf("unexpected output: %q", out.String())
			}
		})
	}
}

func TestRunUsesRetryAfter(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(out http.ResponseWriter, req *http.Request) {
		if requests.Add(1) == 1 {
			out.Header().Set("Retry-After", "3")
			out.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		out.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var out bytes.Buffer
	var delays []time.Duration
	result := run(Config{URL: server.URL, Method: MethodGET, MaxAttempts: 2}, newHTTPClient(), func(delay time.Duration) {
		delays = append(delays, delay)
	}, &out)

	if !result.Success || result.Attempts != 2 || len(delays) != 1 || delays[0] != 3*time.Second {
		t.Fatalf("result=%+v, delays=%v", result, delays)
	}
	if !strings.Contains(out.String(), "sleep_ms 3000\n") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}
