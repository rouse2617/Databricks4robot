package postgres

import (
	"context"
	"errors"
	"testing"
)

// plainFakeDB satisfies pgDB but NOT advisoryLocker.
type plainFakeDB struct{ pgDB }

// lockerFakeDB scripts the advisoryLocker surface.
type lockerFakeDB struct {
	pgDB
	acquired bool
	err      error
	keys     []int64
}

func (f *lockerFakeDB) WithAdvisoryLock(ctx context.Context, key int64, fn func(context.Context) error) (bool, error) {
	f.keys = append(f.keys, key)
	if f.err != nil {
		return false, f.err
	}
	if !f.acquired {
		return false, nil
	}
	return true, fn(ctx)
}

// A pgDB without session-lock support (tx-bound DBs, fakes) runs the cycle
// unguarded and reports acquired — the lock is an optimization, never a
// dependency.
func TestWithSubmitterCycleLock_NoLockerRunsUnguarded(t *testing.T) {
	r := &BackfillRepo{c: &Client{db: &plainFakeDB{}}}
	ran := false
	acquired, err := r.WithSubmitterCycleLock(context.Background(), func(context.Context) error {
		ran = true
		return nil
	})
	if err != nil || !acquired || !ran {
		t.Fatalf("acquired=%v err=%v ran=%v, want true/nil/true", acquired, err, ran)
	}
}

// The pool-backed pgDB delegates to the session advisory lock with the fixed
// submitter key; a held lock skips fn.
func TestWithSubmitterCycleLock_DelegatesToAdvisoryLock(t *testing.T) {
	fake := &lockerFakeDB{acquired: false}
	r := &BackfillRepo{c: &Client{db: fake}}
	ran := false
	acquired, err := r.WithSubmitterCycleLock(context.Background(), func(context.Context) error {
		ran = true
		return nil
	})
	if err != nil || acquired || ran {
		t.Fatalf("acquired=%v err=%v ran=%v, want false/nil/false (lock held elsewhere)", acquired, err, ran)
	}
	if len(fake.keys) != 1 || fake.keys[0] != submitterCycleLockKey {
		t.Fatalf("lock keys = %v, want [%d]", fake.keys, submitterCycleLockKey)
	}

	fake.acquired = true
	acquired, err = r.WithSubmitterCycleLock(context.Background(), func(context.Context) error {
		ran = true
		return nil
	})
	if err != nil || !acquired || !ran {
		t.Fatalf("acquired=%v err=%v ran=%v, want true/nil/true", acquired, err, ran)
	}
}

// Lock-layer errors surface to the caller (the submitter degrades to
// unguarded on its side).
func TestWithSubmitterCycleLock_PropagatesError(t *testing.T) {
	fake := &lockerFakeDB{err: errors.New("conn refused")}
	r := &BackfillRepo{c: &Client{db: fake}}
	acquired, err := r.WithSubmitterCycleLock(context.Background(), func(context.Context) error { return nil })
	if acquired || err == nil {
		t.Fatalf("acquired=%v err=%v, want false/non-nil", acquired, err)
	}
}
