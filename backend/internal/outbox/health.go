package outbox

import (
	"net/http"
	"sync"

	"data-platform/internal/metrics"
)

// HealthStatus tracks the health of the outbox worker. It implements the
// HealthMarker interface defined in worker.go and provides an HTTP handler
// for the /healthz/outbox endpoint.
//
// A newly created HealthStatus is healthy (healthy=true, reason="").
type HealthStatus struct {
	mu      sync.RWMutex
	healthy bool
	reason  string
}

// NewHealthStatus returns a HealthStatus that starts in the healthy state.
func NewHealthStatus() *HealthStatus {
	metrics.OutboxWorkerHealth.Set(1)
	return &HealthStatus{healthy: true}
}

// MarkUnhealthy sets the health status to unhealthy with the given reason.
// It satisfies the HealthMarker interface.
func (h *HealthStatus) MarkUnhealthy(reason string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.healthy = false
	h.reason = reason
	metrics.OutboxWorkerHealth.Set(0)
}

// IsHealthy returns true when the worker is considered healthy.
func (h *HealthStatus) IsHealthy() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.healthy
}

// Reason returns the reason the worker was marked unhealthy.
// Returns an empty string when healthy.
func (h *HealthStatus) Reason() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.reason
}

// Handler returns an http.HandlerFunc suitable for registration on
// /healthz/outbox. It returns 200 OK when healthy and 503 Service
// Unavailable when unhealthy, with a plain-text body.
func (h *HealthStatus) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.IsHealthy() {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("unhealthy: " + h.Reason()))
	}
}
