package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"data-platform/internal/httpresp"
)

// RateLimiter implements a per-IP token bucket rate limiter.
// Configurable via environment variables:
//
//	RATE_LIMIT_RPS  — requests per second per IP (0 = disabled)
//	RATE_LIMIT_BURST — burst size per IP (default = RPS * 2)
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rps     float64
	burst   int
}

type bucket struct {
	tokens   float64
	lastTime time.Time
}

// NewRateLimiter creates a rate limiter. rps=0 disables it.
func NewRateLimiter(rps float64, burst int) *RateLimiter {
	if rps <= 0 {
		return nil
	}
	if burst <= 0 {
		burst = int(rps * 2)
	}
	rl := &RateLimiter{
		buckets: make(map[string]*bucket),
		rps:     rps,
		burst:   burst,
	}
	// Cleanup stale buckets every 60s.
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) cleanup() {
	for {
		time.Sleep(60 * time.Second)
		rl.mu.Lock()
		cutoff := time.Now().Add(-5 * time.Minute)
		for ip, b := range rl.buckets {
			if b.lastTime.Before(cutoff) {
				delete(rl.buckets, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.buckets[ip]
	if !ok {
		b = &bucket{tokens: float64(rl.burst), lastTime: now}
		rl.buckets[ip] = b
	}

	// Refill tokens.
	elapsed := now.Sub(b.lastTime).Seconds()
	b.tokens += elapsed * rl.rps
	if b.tokens > float64(rl.burst) {
		b.tokens = float64(rl.burst)
	}
	b.lastTime = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// Middleware returns a Gin middleware that enforces rate limiting.
// Returns nil if rate limiting is disabled (rps=0).
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !rl.allow(ip) {
			c.Header("Retry-After", "1")
			c.Header("X-RateLimit-Limit", strconv.Itoa(rl.burst))
			httpresp.Error(c, http.StatusTooManyRequests, httpresp.CodeRateLimited,
				"too many requests, please retry later", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}

// RateLimitFromConfig creates a rate limiter from config values.
// Returns nil if disabled.
func RateLimitFromConfig(rpsStr, burstStr string) *RateLimiter {
	rps, _ := strconv.ParseFloat(rpsStr, 64)
	burst, _ := strconv.Atoi(burstStr)
	return NewRateLimiter(rps, burst)
}
