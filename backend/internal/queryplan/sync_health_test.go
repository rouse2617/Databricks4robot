package queryplan

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestSyncHealthCache_GapDefaultsToZero(t *testing.T) {
	c := NewSyncHealthCache(nil, 0)
	if got := c.Gap(); got != 0 {
		t.Fatalf("nil fetcher must yield gap=0, got %d", got)
	}
}

func TestSyncHealthCache_RefreshStoresGap(t *testing.T) {
	c := NewSyncHealthCache(func(_ context.Context) (int64, error) { return 7, nil }, time.Minute)
	if err := c.Refresh(context.Background()); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if got := c.Gap(); got != 7 {
		t.Fatalf("gap after refresh = %d, want 7", got)
	}
	if c.UpdatedAt().IsZero() {
		t.Fatal("UpdatedAt should be set after successful refresh")
	}
}

func TestSyncHealthCache_RefreshErrorPreservesPreviousGap(t *testing.T) {
	var calls atomic.Int32
	c := NewSyncHealthCache(func(_ context.Context) (int64, error) {
		n := calls.Add(1)
		if n == 1 {
			return 5, nil
		}
		return 0, errors.New("boom")
	}, time.Minute)
	if err := c.Refresh(context.Background()); err != nil {
		t.Fatalf("first refresh: %v", err)
	}
	prevUpdated := c.UpdatedAt()
	if err := c.Refresh(context.Background()); err == nil {
		t.Fatal("expected error on second refresh")
	}
	if got := c.Gap(); got != 5 {
		t.Fatalf("gap after failed refresh = %d, want 5 (preserved)", got)
	}
	if !c.UpdatedAt().Equal(prevUpdated) {
		t.Fatal("UpdatedAt must not advance on failed refresh")
	}
}

func TestSyncHealthCache_RunTicksAndCancels(t *testing.T) {
	var calls atomic.Int32
	c := NewSyncHealthCache(func(_ context.Context) (int64, error) {
		calls.Add(1)
		return 3, nil
	}, 20*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = c.Run(ctx)
		close(done)
	}()
	// Give the initial refresh + one tick a chance to fire.
	time.Sleep(60 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Run did not exit after ctx cancel")
	}
	if got := calls.Load(); got < 2 {
		t.Fatalf("expected at least 2 refresh calls, got %d", got)
	}
	if c.Gap() != 3 {
		t.Fatalf("gap should be 3 after Run, got %d", c.Gap())
	}
}
