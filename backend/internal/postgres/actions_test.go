package postgres

import (
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// fakeScanRow assigns a fixed value slice into Scan's destinations by reflection.
// It errors when the destination count differs from the value count — exactly
// the pgx failure mode Bug 1 (CYB-3267) hit (22 columns vs 21 destinations).
type fakeScanRow struct{ vals []any }

func (f *fakeScanRow) Scan(dest ...any) error {
	if len(dest) != len(f.vals) {
		return &pgconn.PgError{
			Message: "number of field descriptions must equal number of destinations",
		}
	}
	for i, d := range dest {
		reflect.ValueOf(d).Elem().Set(reflect.ValueOf(f.vals[i]))
	}
	return nil
}

// CYB-3267 Bug 1 regression: scanAction must have exactly one destination per
// actionSelectCols column, in order, so task_id is read and tenant_id/project_id
// are not shifted. The value slice below matches actionSelectCols order.
func TestScanAction_ColumnDestinationAlignment(t *testing.T) {
	now := time.Now()
	row := &fakeScanRow{vals: []any{
		"act-1", "asset-1", // action_id, asset_id
		int64(1000), int64(3000), (*int)(nil), // start_ns, end_ns, action_index
		"label-1", []string{"l1"}, // primary_label, labels
		"desc-1", []byte("{}"), // description, attrs
		"human", "sn-1", "sv-1", // source_type, source_name, source_version
		"run-1", (*float64)(nil), "ext-1", // run_id, confidence, external_id
		"task-1",             // task_id  ← the one Bug 1 dropped
		"tenant-1", "proj-1", // tenant_id, project_id
		false, int64(2), now, now, // is_deleted, version, created_at, updated_at
	}}

	a, err := scanAction(row)
	if err != nil {
		t.Fatalf("scanAction failed (column/destination mismatch?): %v", err)
	}
	if a.TaskID != "task-1" {
		t.Errorf("TaskID = %q, want %q", a.TaskID, "task-1")
	}
	// If task_id were skipped, these would be shifted by one column.
	if a.TenantID != "tenant-1" {
		t.Errorf("TenantID = %q, want %q (shifted?)", a.TenantID, "tenant-1")
	}
	if a.ProjectID != "proj-1" {
		t.Errorf("ProjectID = %q, want %q (shifted?)", a.ProjectID, "proj-1")
	}
	if a.ExternalID != "ext-1" {
		t.Errorf("ExternalID = %q, want %q", a.ExternalID, "ext-1")
	}
}

func TestIsActionIDSchemaMismatch(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		if isActionIDSchemaMismatch(nil) {
			t.Fatal("expected false for nil pgErr")
		}
	})

	t.Run("uuid parse error", func(t *testing.T) {
		pgErr := &pgconn.PgError{
			Code:    "22P02",
			Message: `invalid input syntax for type uuid: "Z4PZDKBQ"`,
		}
		if !isActionIDSchemaMismatch(pgErr) {
			t.Fatal("expected true for 22P02 type uuid parse error")
		}
	})

	t.Run("other parse error", func(t *testing.T) {
		pgErr := &pgconn.PgError{
			Code:    "22P02",
			Message: `invalid input syntax for type integer: "abc"`,
		}
		if isActionIDSchemaMismatch(pgErr) {
			t.Fatal("expected false for non-uuid parse error")
		}
	})

	t.Run("different code", func(t *testing.T) {
		pgErr := &pgconn.PgError{
			Code:    "23505",
			Message: "duplicate key value violates unique constraint",
		}
		if isActionIDSchemaMismatch(pgErr) {
			t.Fatal("expected false for non-22P02 error code")
		}
	})
}
