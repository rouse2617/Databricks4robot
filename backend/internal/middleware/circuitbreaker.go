package middleware

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
)

// CircuitBreaker implements a simple sliding-window circuit breaker.
// When the error rate exceeds the threshold within the window, the circuit
// opens and rejects requests for a cooldown period.
//
// States:
//
//	Closed  → normal operation, counting errors
//	Open    → rejecting all requests (503)
//	HalfOpen → allowing a single probe request to test recovery
//
// Configurable via environment variables:
//
//	CB_ENABLED       — "true" to enable (default: disabled)
//	CB_WINDOW_SEC    — sliding window in seconds (default: 60)
//	CB_THRESHOLD     — error count to trip (default: 10)
//	CB_COOLDOWN_SEC  — cooldown before half-open (default: 30)
type CircuitBreaker struct {
	mu        sync.Mutex
	state     cbState
	errors    []time.Time // timestamps of 5xx errors within window
	window    time.Duration
	threshold int
	cooldown  time.Duration
	openedAt  time.Time
}

type cbState int

const (
	cbClosed cbState = iota
	cbOpen
	cbHalfOpen
)

// NewCircuitBreaker creates a circuit breaker. Returns nil if disabled.
func NewCircuitBreaker(enabled bool, windowSec, threshold, cooldownSec int) *CircuitBreaker {
	if !enabled {
		return nil
	}
	if windowSec <= 0 {
		windowSec = 60
	}
	if threshold <= 0 {
		threshold = 10
	}
	if cooldownSec <= 0 {
		cooldownSec = 30
	}
	return &CircuitBreaker{
		state:     cbClosed,
		window:    time.Duration(windowSec) * time.Second,
		threshold: threshold,
		cooldown:  time.Duration(cooldownSec) * time.Second,
	}
}

func (cb *CircuitBreaker) recordError() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	cb.errors = append(cb.errors, now)
	cb.pruneOld(now)

	if cb.state == cbClosed && len(cb.errors) >= cb.threshold {
		cb.state = cbOpen
		cb.openedAt = now
		slog.Warn("circuit breaker OPEN", "errors", len(cb.errors), "window", cb.window)
	}
}

func (cb *CircuitBreaker) pruneOld(now time.Time) {
	cutoff := now.Add(-cb.window)
	i := 0
	for i < len(cb.errors) && cb.errors[i].Before(cutoff) {
		i++
	}
	cb.errors = cb.errors[i:]
}

func (cb *CircuitBreaker) shouldAllow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case cbClosed:
		return true
	case cbOpen:
		if time.Since(cb.openedAt) > cb.cooldown {
			cb.state = cbHalfOpen
			slog.Info("circuit breaker HALF-OPEN, allowing probe request")
			return true
		}
		return false
	case cbHalfOpen:
		// Only one probe at a time — block others.
		return false
	}
	return true
}

func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == cbHalfOpen {
		cb.state = cbClosed
		cb.errors = nil
		slog.Info("circuit breaker CLOSED (recovered)")
	}
}

// Middleware returns a Gin middleware that enforces circuit breaking.
func (cb *CircuitBreaker) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cb.shouldAllow() {
			c.Header("Retry-After", "30")
			httpresp.Error(c, http.StatusServiceUnavailable, httpresp.CodeServiceUnavailable,
				"service temporarily unavailable, circuit breaker is open", nil)
			c.Abort()
			return
		}

		c.Next()

		// After handler: check if response was 5xx.
		if c.Writer.Status() >= 500 {
			cb.recordError()
		} else {
			cb.recordSuccess()
		}
	}
}
