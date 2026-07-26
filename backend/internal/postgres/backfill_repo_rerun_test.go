package postgres

import (
	"context"
	"strings"
	"testing"
)

// PrepareItemsForRerun must reset submit_attempts alongside the other fields so
// a human rerun earns a fresh transient-retry budget (parity with
// ResetFailedItems on the DLQ path — CYB-3678 / review P1-5). Without the
// reset, a poison item flipped back to pending is re-poisoned on its first
// transient submit error.
func TestPrepareItemsForRerunResetsSubmitAttempts(t *testing.T) {
	db := &fakeDB{}
	r := &BackfillRepo{c: &Client{db: db}}

	if err := r.PrepareItemsForRerun(context.Background(), []string{"item-1", "item-2"}); err != nil {
		t.Fatalf("PrepareItemsForRerun: %v", err)
	}
	if len(db.execSQLs) != 1 {
		t.Fatalf("exec calls = %d, want 1", len(db.execSQLs))
	}
	sql := db.execSQLs[0]
	for _, want := range []string{
		"status = 'pending'",
		"error_message = NULL",
		"started_at = NULL",
		"finished_at = NULL",
		"submit_attempts = 0",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("PrepareItemsForRerun SQL missing %q:\n%s", want, sql)
		}
	}
}

// An empty item set is a no-op: no UPDATE is issued.
func TestPrepareItemsForRerunEmptyIsNoop(t *testing.T) {
	db := &fakeDB{}
	r := &BackfillRepo{c: &Client{db: db}}

	if err := r.PrepareItemsForRerun(context.Background(), nil); err != nil {
		t.Fatalf("PrepareItemsForRerun(nil): %v", err)
	}
	if len(db.execSQLs) != 0 {
		t.Fatalf("exec calls = %d, want 0 (empty is a no-op)", len(db.execSQLs))
	}
}
