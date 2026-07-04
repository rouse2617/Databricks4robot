package backfill

import (
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
)

// These tests cover the pure (no repo, no pipelineUC) facets of the
// dispatcher:
//   * Default config hydration
//   * BatchSize clamping under WorkerCount
//   * Backoff schedule (exponential with cap and jitter)
//   * Permanent-error classification
//   * Deterministic wfName fallback hygiene
//
// End-to-end tests (claim -> submit -> mark dispatched) need to drive
// pipelineUC.DeployByTemplateID and require a real Usecase. Those live
// in dispatcher_e2e_test.go (Commit B scope, when entry points are
// wired). The current Commit A dispatcher is intentionally free of
// side-effecting tests beyond the unit checks here.

func TestDispatcherConfig_normalized_fillsDefaults(t *testing.T) {
	c := DispatcherConfig{}.normalized()
	if c.Tick != 5*time.Second {
		t.Fatalf("tick default = %v, want 5s", c.Tick)
	}
	if c.LeaseSec != 60 {
		t.Fatalf("lease default = %d, want 60", c.LeaseSec)
	}
	if c.MaxAttempts != 3 {
		t.Fatalf("max attempts default = %d, want 3", c.MaxAttempts)
	}
	if c.WorkerCount != 5 {
		t.Fatalf("worker count default = %d, want 5", c.WorkerCount)
	}
	if c.JobBufferSize != 64 {
		t.Fatalf("buffer default = %d, want 64", c.JobBufferSize)
	}
	if c.BackoffBase != 1*time.Second {
		t.Fatalf("backoff base default = %v, want 1s", c.BackoffBase)
	}
	if c.BackoffMax != 30*time.Second {
		t.Fatalf("backoff max default = %v, want 30s", c.BackoffMax)
	}
	if c.InstanceID == "" {
		t.Fatal("instance id should be auto-populated")
	}
}

func TestDispatcherConfig_normalized_clampsBatchToWorkerCount(t *testing.T) {
	c := DispatcherConfig{WorkerCount: 2, BatchSize: 10}.normalized()
	if c.BatchSize != 2 {
		t.Fatalf("batch should clamp to workerCount, got %d", c.BatchSize)
	}
}

func TestDispatcher_backoffForAttempt_exponential(t *testing.T) {
	d := &Dispatcher{cfg: DispatcherConfig{
		BackoffBase: 1 * time.Second,
		BackoffMax:  60 * time.Second,
	}.normalized()}
	// Loosen tolerance to absorb jitter without flake-looping on CI.
	type tc struct {
		attempt int
		min     time.Duration
		max     time.Duration
	}
	cases := []tc{
		{1, 1 * time.Second, 2 * time.Second}, // base
		{2, 2 * time.Second, 3 * time.Second},
		{3, 3 * time.Second, 5 * time.Second},
		{4, 7 * time.Second, 9 * time.Second},
		{5, 14 * time.Second, 18 * time.Second},
		{6, 28 * time.Second, 36 * time.Second},
		{7, 50 * time.Second, 70 * time.Second}, // capped at max(60s)
	}
	for _, c := range cases {
		got := d.backoffForAttempt(c.attempt)
		if got < c.min || got > c.max {
			t.Errorf("attempt %d: backoff = %v, want in [%v, %v]", c.attempt, got, c.min, c.max)
		}
	}
}

func TestDispatcher_backoffForAttempt_cappedAtMax(t *testing.T) {
	d := &Dispatcher{cfg: DispatcherConfig{
		BackoffBase: 1 * time.Second,
		BackoffMax:  10 * time.Second,
	}.normalized()}
	got := d.backoffForAttempt(20)
	if got > 13*time.Second { // cap + ~10% jitter, generous ceiling
		t.Fatalf("backoff should cap near %v, got %v", d.cfg.BackoffMax, got)
	}
}

func TestDispatcher_isPermanentDeployError(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("generic error"), false},
		{pipelineUC.ErrTemplateNotFound, true},
		{pipelineUC.ErrInvalidArgument, true},
		{pipelineUC.ErrWorkflowUnavailable, true},
	}
	for _, c := range cases {
		got := isPermanentDeployError(c.err)
		if got != c.want {
			t.Errorf("isPermanentDeployError(%v) = %v, want %v", c.err, got, c.want)
		}
	}
}

func TestDispatcher_deriveWfNameFallback_dns1123ish(t *testing.T) {
	item := &models.BackfillItem{
		JobID:              "Job_ABC-123!",
		ID:                 "abcdef1234567890",
		DispatchGeneration: 7,
	}
	wf := deriveWfNameFallback(item)
	if wf == "" {
		t.Fatal("name should not be empty")
	}
	for _, r := range wf {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		t.Fatalf("name %q contains illegal char %q", wf, r)
	}
	// Deterministic.
	again := deriveWfNameFallback(item)
	if wf != again {
		t.Fatalf("non-deterministic: %q vs %q", wf, again)
	}
}

// Compile-time anchors: forces rand dependency + sync import paths to
// stay associated with this file (we use rand/atomic elsewhere under
// different names). They aren't expected to run anything — they exist
// so a downstream refactor that drops the rand import also drops our
// reference here, intentionally.
var _ = rand.New
