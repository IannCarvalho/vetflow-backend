package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// okHandler is a trivial next handler that always returns 200.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

// doRequest sends a request with the given RemoteAddr through the handler and
// returns the response status code.
func doRequest(h http.Handler, remoteAddr string) int {
	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.RemoteAddr = remoteAddr
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

func TestRateLimiterAllowsUpToLimitThenBlocks(t *testing.T) {
	const limit = 3
	h := NewRateLimiter(limit)(okHandler)

	for i := 1; i <= limit; i++ {
		if code := doRequest(h, "10.0.0.1:1234"); code != http.StatusOK {
			t.Fatalf("request %d: expected %d, got %d", i, http.StatusOK, code)
		}
	}

	if code := doRequest(h, "10.0.0.1:1234"); code != http.StatusTooManyRequests {
		t.Fatalf("request %d: expected %d, got %d", limit+1, http.StatusTooManyRequests, code)
	}
}

func TestRateLimiterIsolatesPerIP(t *testing.T) {
	const limit = 2
	h := NewRateLimiter(limit)(okHandler)

	// Exhaust the limit for the first IP.
	for i := 0; i < limit; i++ {
		doRequest(h, "10.0.0.1:1111")
	}
	if code := doRequest(h, "10.0.0.1:1111"); code != http.StatusTooManyRequests {
		t.Fatalf("first IP should be blocked, got %d", code)
	}

	// A different IP must still be allowed.
	if code := doRequest(h, "10.0.0.2:2222"); code != http.StatusOK {
		t.Fatalf("second IP should be allowed, got %d", code)
	}
}

func TestRateLimiterDisabledWhenLimitZero(t *testing.T) {
	h := NewRateLimiter(0)(okHandler)

	for i := 0; i < 100; i++ {
		if code := doRequest(h, "10.0.0.1:1234"); code != http.StatusOK {
			t.Fatalf("request %d: disabled limiter should allow all, got %d", i, code)
		}
	}
}

func TestRateLimiterPrefersXForwardedFor(t *testing.T) {
	const limit = 1
	h := NewRateLimiter(limit)(okHandler)

	req := func(xff, remote string) int {
		r := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		r.RemoteAddr = remote
		r.Header.Set("X-Forwarded-For", xff)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		return rec.Code
	}

	// Same forwarded client IP, different proxy RemoteAddr: second call is blocked.
	if code := req("203.0.113.5, 10.0.0.9", "10.0.0.9:5555"); code != http.StatusOK {
		t.Fatalf("first forwarded request: expected %d, got %d", http.StatusOK, code)
	}
	if code := req("203.0.113.5, 10.0.0.10", "10.0.0.10:6666"); code != http.StatusTooManyRequests {
		t.Fatalf("second forwarded request from same client should be blocked, got %d", code)
	}
}
