package lakehouse

import (
	"testing"
	"time"
)

func TestParseDaysQuery(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{name: "valid", input: "30", want: 30},
		{name: "trim spaces", input: " 7 ", want: 7},
		{name: "zero", input: "0", wantErr: true},
		{name: "negative", input: "-3", wantErr: true},
		{name: "not number", input: "abc", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDaysQuery(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (value=%d)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestParseEventTypeShareDate(t *testing.T) {
	now := time.Date(2026, 5, 18, 1, 2, 3, 0, time.UTC)
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "default empty", input: "", want: "2026-05-18"},
		{name: "latest keyword", input: "latest", want: "2026-05-18"},
		{name: "latest keyword uppercase", input: "LATEST", want: "2026-05-18"},
		{name: "valid date", input: "2026-05-15", want: "2026-05-15"},
		{name: "invalid date", input: "2026/05/15", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseEventTypeShareDate(tt.input, now)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (value=%q)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseQualityWindow(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{name: "default empty", input: "", want: 30},
		{name: "seven days", input: "7d", want: 7},
		{name: "thirty days", input: "30d", want: 30},
		{name: "sixty days", input: "60d", want: 60},
		{name: "ninety days", input: "90d", want: 90},
		{name: "trim spaces", input: " 60d ", want: 60},
		{name: "invalid value", input: "14d", wantErr: true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseQualityWindow(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (value=%d)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestBuildRealtimeSyncStatus(t *testing.T) {
	now := time.Date(2026, 5, 18, 2, 0, 0, 0, time.UTC)
	t.Run("unavailable when checkpoint missing", func(t *testing.T) {
		got := buildRealtimeSyncStatus(BronzeSyncProgress{
			OutboxPublishedMaxSeq: 100,
			BronzeMaxEventSeq:     0,
			BronzeLagEvents:       100,
			CheckedAt:             now,
		})
		if got.Available {
			t.Fatalf("expected unavailable status")
		}
		if got.Source != "realtime" {
			t.Fatalf("unexpected source: %s", got.Source)
		}
	})

	t.Run("available with computed diff", func(t *testing.T) {
		ingestedAt := now.Add(-5 * time.Minute)
		got := buildRealtimeSyncStatus(BronzeSyncProgress{
			OutboxPublishedMaxSeq: 1000,
			BronzeMaxEventSeq:     990,
			BronzeLagEvents:       10,
			BronzeLastIngestedAt:  &ingestedAt,
			CheckedAt:             now,
		})
		if !got.Available {
			t.Fatalf("expected available status")
		}
		if got.Data == nil {
			t.Fatalf("expected data payload")
		}
		if got.Data.CountDiffPct != 0.01 {
			t.Fatalf("unexpected count diff pct: %v", got.Data.CountDiffPct)
		}
		if !got.Data.IsAlert {
			t.Fatalf("expected alert when lag > 0")
		}
		if got.Data.PGTotalCount != 1000 || got.Data.IcebergTotalCount != 990 {
			t.Fatalf("unexpected counts: pg=%d iceberg=%d", got.Data.PGTotalCount, got.Data.IcebergTotalCount)
		}
	})
}
