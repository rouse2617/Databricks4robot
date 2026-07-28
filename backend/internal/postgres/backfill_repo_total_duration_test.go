package postgres

import (
	"context"
	"strings"
	"testing"
)

func TestBackfillRepoTotalDurationByBatchIDs_EmptyIDsShortCircuits(t *testing.T) {
	db := &fakeDB{}
	repo := &BackfillRepo{c: &Client{db: db}}

	got, err := repo.TotalDurationByBatchIDs(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %v", got)
	}
	if len(db.querySQLs) != 0 {
		t.Errorf("expected no SQL to be issued, got %d queries", len(db.querySQLs))
	}
}

func TestBackfillRepoTotalDurationByBatchIDs_ResultMapping(t *testing.T) {
	db := &fakeDB{
		rows: &fakeRows{
			data: [][]any{
				{"batch-a", int64(5000)},
				{"batch-b", int64(90_500)},
			},
		},
	}
	repo := &BackfillRepo{c: &Client{db: db}}

	got, err := repo.TotalDurationByBatchIDs(context.Background(), []string{"batch-a", "batch-b", "batch-c"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["batch-a"] != 5000 {
		t.Errorf("batch-a = %d, want 5000", got["batch-a"])
	}
	if got["batch-b"] != 90_500 {
		t.Errorf("batch-b = %d, want 90_500", got["batch-b"])
	}
	if _, present := got["batch-c"]; present {
		t.Errorf("batch-c must be absent (implicit-0 at caller), got %d", got["batch-c"])
	}
	if len(db.querySQLs) != 1 {
		t.Fatalf("expected 1 SQL, got %d", len(db.querySQLs))
	}
	// The query joins pipeline_runs × unnested asset_ids × assets and filters
	// soft-deleted assets — sanity-check the essential clauses so a schema
	// rename doesn't silently drift the semantics.
	q := db.querySQLs[0]
	for _, want := range []string{
		"pipeline_runs",
		"unnest(pr.asset_ids)",
		"is_deleted = FALSE",
		"batch_job_id = ANY($1)",
		"GROUP BY pr.batch_job_id",
	} {
		if !strings.Contains(q, want) {
			t.Errorf("SQL missing %q; got:\n%s", want, q)
		}
	}
}
