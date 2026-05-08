package lakehouse

import (
	"testing"
	"time"
)

func TestNewRealtimeSyncStatusData_UsesRatioAndStableShape(t *testing.T) {
	checkedAt := time.Date(2026, 4, 30, 2, 3, 4, 0, time.UTC)
	got := newRealtimeSyncStatusData(checkedAt, 1000, 990, 12345)

	if got.CheckedAt != checkedAt {
		t.Fatalf("checked_at = %v, want %v", got.CheckedAt, checkedAt)
	}
	if got.CountDiffPct != 0.01 {
		t.Fatalf("count_diff_pct = %v, want 0.01", got.CountDiffPct)
	}
	if !got.IsAlert {
		t.Fatal("expected is_alert=true above the 0.1% reconciliation threshold")
	}
	if got.DagsterRunID != "" {
		t.Fatalf("dagster_run_id = %q, want empty for realtime source", got.DagsterRunID)
	}
	if got.PgStatusDist == nil || got.IcebergStatusDist == nil || got.StatusDiff == nil {
		t.Fatal("expected realtime payload maps to be non-nil")
	}
	if got.IcebergMaxSeq != 12345 {
		t.Fatalf("iceberg_max_seq = %d, want 12345", got.IcebergMaxSeq)
	}
}

func TestNewRealtimeSyncStatusData_ZeroPgWithIcebergRowsIsFullDiff(t *testing.T) {
	got := newRealtimeSyncStatusData(time.Now(), 0, 12, 12)
	if got.CountDiffPct != 1.0 {
		t.Fatalf("count_diff_pct = %v, want 1.0", got.CountDiffPct)
	}
	if !got.IsAlert {
		t.Fatal("expected full diff to trigger alert")
	}
}
