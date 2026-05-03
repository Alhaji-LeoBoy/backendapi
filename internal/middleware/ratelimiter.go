package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type RateLimiterConfig struct {
	RequestsPerSecond float64
	Burst             int
	CleanupInterval   time.Duration
	ClientTTL         time.Duration
	TrustProxy        bool
}

func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		RequestsPerSecond: 10,
		Burst:             20,
		CleanupInterval:   5 * time.Minute,
		ClientTTL:         10 * time.Minute,
		TrustProxy:        false,
	}
}

type bucket struct {
	tokens   float64
	lastSeen time.Time
	mu       sync.Mutex
}

type RateLimiter struct {
	cfg     RateLimiterConfig
	clients map[string]*bucket
	mu      sync.RWMutex
	stop    chan struct{}
}

func NewRateLimiter(cfg RateLimiterConfig) *RateLimiter {
	rl := &RateLimiter{
		cfg:     cfg,
		clients: make(map[string]*bucket),
		stop:    make(chan struct{}),
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) Close() {
	close(rl.stop)
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cfg.CleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			rl.evictStale()
		case <-rl.stop:
			return
		}
	}
}

func (rl *RateLimiter) evictStale() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	cutoff := time.Now().Add(-rl.cfg.ClientTTL)
	for ip, b := range rl.clients {
		b.mu.Lock()
		if b.lastSeen.Before(cutoff) {
			delete(rl.clients, ip)
		}
		b.mu.Unlock()
	}
}

func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.RLock()
	b, exists := rl.clients[ip]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		b, exists = rl.clients[ip]
		if !exists {
			b = &bucket{
				tokens:   float64(rl.cfg.Burst),
				lastSeen: time.Now(),
			}
			rl.clients[ip] = b
		}
		rl.mu.Unlock()
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastSeen).Seconds()
	b.lastSeen = now

	b.tokens += elapsed * rl.cfg.RequestsPerSecond
	if b.tokens > float64(rl.cfg.Burst) {
		b.tokens = float64(rl.cfg.Burst)
	}

	if b.tokens < 1 {
		return false
	}

	b.tokens--
	return true
}

func (rl *RateLimiter) clientIP(r *http.Request) string {
	if rl.cfg.TrustProxy {
		if ip := r.Header.Get("X-Real-IP"); ip != "" {
			return strings.TrimSpace(ip)
		}
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			if idx := strings.IndexByte(forwarded, ','); idx != -1 {
				return strings.TrimSpace(forwarded[:idx])
			}
			return strings.TrimSpace(forwarded)
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func rateLimitedResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", "1")
	w.WriteHeader(http.StatusTooManyRequests)
	json.NewEncoder(w).Encode(map[string]string{
		"error":   "rate limit exceeded",
		"message": "You are sending too many requests. Please slow down and try again shortly.",
	})
}

func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := rl.clientIP(r)

		if !rl.allow(ip) {
			rateLimitedResponse(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}
