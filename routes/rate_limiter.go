package routes

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// rateLimitWindow is the fixed window over which requests are counted per IP.
const rateLimitWindow = time.Minute

// rateLimitEntry tracks how many requests an IP made in the current window.
type rateLimitEntry struct {
	count       int
	windowStart time.Time
}

// NewRateLimiter returns a chi-compatible middleware that limits the number of
// requests per client IP to limitPerMin within a fixed one-minute window.
// When limitPerMin <= 0 the returned middleware is a pass-through (no limiting).
func NewRateLimiter(limitPerMin int) func(http.Handler) http.Handler {
	if limitPerMin <= 0 {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	var mu sync.Mutex
	entries := make(map[string]*rateLimitEntry)

	// Lazily prune stale entries whenever we handle a request. We piggyback the
	// prune on access so no background goroutine lingers in tests.
	prune := func(now time.Time) {
		for ip, e := range entries {
			if now.Sub(e.windowStart) >= rateLimitWindow {
				delete(entries, ip)
			}
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			now := time.Now()

			mu.Lock()
			prune(now)
			e, ok := entries[ip]
			if !ok || now.Sub(e.windowStart) >= rateLimitWindow {
				e = &rateLimitEntry{count: 0, windowStart: now}
				entries[ip] = e
			}
			e.count++
			exceeded := e.count > limitPerMin
			mu.Unlock()

			if exceeded {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"rate limit exceeded, try again later"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// clientIP resolves the client IP for rate limiting. It prefers the first
// comma-separated value of the X-Forwarded-For header when present; otherwise
// it uses r.RemoteAddr with the port stripped.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		first := strings.TrimSpace(strings.Split(xff, ",")[0])
		if first != "" {
			return first
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
