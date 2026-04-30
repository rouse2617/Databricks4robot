package outbox

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// Compile-time check: HealthStatus must satisfy HealthMarker.
var _ HealthMarker = (*HealthStatus)(nil)

func TestNewHealthStatus_StartsHealthy(t *testing.T) {
	h := NewHealthStatus()
	if !h.IsHealthy() {
		t.Fatal("expected new HealthStatus to be healthy")
	}
	if h.Reason() != "" {
		t.Fatalf("expected empty reason, got %q", h.Reason())
	}
}

func TestMarkUnhealthy_SetsState(t *testing.T) {
	h := NewHealthStatus()
	h.MarkUnhealthy("panic: nil pointer")

	if h.IsHealthy() {
		t.Fatal("expected unhealthy after MarkUnhealthy")
	}
	if h.Reason() != "panic: nil pointer" {
		t.Fatalf("expected reason %q, got %q", "panic: nil pointer", h.Reason())
	}
}

func TestMarkUnhealthy_OverwritesReason(t *testing.T) {
	h := NewHealthStatus()
	h.MarkUnhealthy("first failure")
	h.MarkUnhealthy("second failure")

	if h.Reason() != "second failure" {
		t.Fatalf("expected reason %q, got %q", "second failure", h.Reason())
	}
}

func TestHandler_Healthy_Returns200(t *testing.T) {
	h := NewHealthStatus()
	handler := h.Handler()

	req := httptest.NewRequest(http.MethodGet, "/healthz/outbox", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("expected body %q, got %q", "ok", rec.Body.String())
	}
}

func TestHandler_Unhealthy_Returns503(t *testing.T) {
	h := NewHealthStatus()
	h.MarkUnhealthy("ES bulk timeout")
	handler := h.Handler()

	req := httptest.NewRequest(http.MethodGet, "/healthz/outbox", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rec.Code)
	}
	body := rec.Body.String()
	if body != "unhealthy: ES bulk timeout" {
		t.Fatalf("expected body %q, got %q", "unhealthy: ES bulk timeout", body)
	}
}

func TestHealthStatus_ConcurrentAccess(t *testing.T) {
	h := NewHealthStatus()
	var wg sync.WaitGroup

	// Concurrent writers.
	for i := range 50 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			h.MarkUnhealthy("reason")
		}(i)
	}

	// Concurrent readers.
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = h.IsHealthy()
			_ = h.Reason()
		}()
	}

	wg.Wait()
	// No race detector failures = pass.
}
