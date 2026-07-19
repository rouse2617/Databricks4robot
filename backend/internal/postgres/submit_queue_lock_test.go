package postgres

import (
	"context"
	"errors"
	"strings"
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
	acquired, err := r.WithSubmitterClusterLock(context.Background(), "clu-a", func(context.Context) error {
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
	acquired, err := r.WithSubmitterClusterLock(context.Background(), "clu-a", func(context.Context) error {
		ran = true
		return nil
	})
	if err != nil || acquired || ran {
		t.Fatalf("acquired=%v err=%v ran=%v, want false/nil/false (lock held elsewhere)", acquired, err, ran)
	}
	if len(fake.keys) != 1 || fake.keys[0] != clusterLockKey("clu-a") {
		t.Fatalf("lock keys = %v, want [%d]", fake.keys, clusterLockKey("clu-a"))
	}

	fake.acquired = true
	acquired, err = r.WithSubmitterClusterLock(context.Background(), "clu-a", func(context.Context) error {
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
	acquired, err := r.WithSubmitterClusterLock(context.Background(), "clu-a", func(context.Context) error { return nil })
	if acquired || err == nil {
		t.Fatalf("acquired=%v err=%v, want false/non-nil", acquired, err)
	}
}

// ── CYB-3678: attempts + DLQ reset SQL surface ──────────────────────────────

type attemptsFakeDB struct {
	pgDB
	rowVal   int
	rowErr   error
	execN    int64
	execErr  error
	lastSQL  string
	lastArgs []any
}

type attemptsRow struct {
	val int
	err error
}

func (r attemptsRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*(dest[0].(*int)) = r.val
	return nil
}

func (f *attemptsFakeDB) QueryRow(_ context.Context, sql string, args ...any) rowScanner {
	f.lastSQL, f.lastArgs = sql, args
	return attemptsRow{val: f.rowVal, err: f.rowErr}
}

func (f *attemptsFakeDB) ExecResult(_ context.Context, sql string, args ...any) (int64, error) {
	f.lastSQL, f.lastArgs = sql, args
	return f.execN, f.execErr
}

func TestIncrementItemSubmitAttempts(t *testing.T) {
	db := &attemptsFakeDB{rowVal: 3}
	r := &BackfillRepo{c: &Client{db: db}}
	n, err := r.IncrementItemSubmitAttempts(context.Background(), "item-1")
	if err != nil || n != 3 {
		t.Fatalf("n=%d err=%v, want 3/nil", n, err)
	}
	if !strings.Contains(db.lastSQL, "submit_attempts + 1") || db.lastArgs[0] != "item-1" {
		t.Fatalf("sql=%q args=%v", db.lastSQL, db.lastArgs)
	}

	db.rowErr = errors.New("db down")
	if _, err := r.IncrementItemSubmitAttempts(context.Background(), "item-1"); err == nil {
		t.Fatal("want wrapped error")
	}
}

func TestResetFailedItems(t *testing.T) {
	db := &attemptsFakeDB{execN: 4}
	r := &BackfillRepo{c: &Client{db: db}}
	n, err := r.ResetFailedItems(context.Background(), "job-1")
	if err != nil || n != 4 {
		t.Fatalf("n=%d err=%v, want 4/nil", n, err)
	}
	if !strings.Contains(db.lastSQL, "status = 'failed'") || !strings.Contains(db.lastSQL, "submit_attempts = 0") {
		t.Fatalf("sql=%q", db.lastSQL)
	}

	db.execErr = errors.New("db down")
	if _, err := r.ResetFailedItems(context.Background(), "job-1"); err == nil {
		t.Fatal("want wrapped error")
	}
}
