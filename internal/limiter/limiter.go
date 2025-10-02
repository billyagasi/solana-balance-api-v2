package limiter

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

type tokenBucket struct {
	tokens int
	last   time.Time
}

type IPLimiter struct {
	limit   int
	per     time.Duration
	mu      sync.Mutex
	buckets map[string]*tokenBucket
}

func NewIPLimiter(limit int, per time.Duration) *IPLimiter {
	return &IPLimiter{limit: limit, per: per, buckets: make(map[string]*tokenBucket)}
}

func (l *IPLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	b := l.buckets[ip]
	now := time.Now()
	if b == nil {
		b = &tokenBucket{tokens: l.limit - 1, last: now}
		l.buckets[ip] = b
		return true
	}
	elapsed := now.Sub(b.last)
	if elapsed >= l.per {
		b.tokens = l.limit - 1
		b.last = now
		return true
	}
	if b.tokens <= 0 {
		return false
	}
	b.tokens--
	return true
}

func (l *IPLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !l.allow(ip) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	// Prefer X-Forwarded-For; fallback to RemoteAddr
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		return xff
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func ParseInt(s string, def int) int {
	if s == "" {
		return def
	}
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil {
		return def
	}
	return n
}

