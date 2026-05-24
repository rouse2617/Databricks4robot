package lakehouse

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	lakehouseport "github.com/CyberOrigin2077/cyber-databrew/internal/lakehouse"
)

type fakeLakehouseQuerier struct {
	rowsByTable map[string][]lakehouseport.Row
	errByTable  map[string]error
	queries     []string
}

func (f *fakeLakehouseQuerier) Status(context.Context) lakehouseport.Status {
	return lakehouseport.Status{Enabled: true, Healthy: true, Backend: "fake"}
}

func (f *fakeLakehouseQuerier) Query(_ context.Context, sql string) ([]lakehouseport.Row, error) {
	f.queries = append(f.queries, sql)
	tableName := ""
	switch {
	case strings.Contains(sql, "silver_asset_events_current"):
		tableName = "silver_asset_events_current"
	case strings.Contains(sql, "bronze_asset_events"):
		tableName = "bronze_asset_events"
	}
	if err := f.errByTable[tableName]; err != nil {
		return nil, err
	}
	return f.rowsByTable[tableName], nil
}

func (f *fakeLakehouseQuerier) Close() {}

func TestTablesIncludesSilverWhenAvailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	lake := &fakeLakehouseQuerier{
		rowsByTable: map[string][]lakehouseport.Row{
			"bronze_asset_events": {
				{"table_name": "bronze_asset_events", "row_count": int64(10)},
			},
			"silver_asset_events_current": {
				{"table_name": "silver_asset_events_current", "row_count": int64(7)},
			},
		},
	}
	h := New("", lake, nil)
	r := gin.New()
	r.GET("/lakehouse/tables", h.Tables)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/lakehouse/tables", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Items []struct {
			TableName string `json:"table_name"`
			RowCount  int64  `json:"row_count"`
		} `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected bronze and silver rows, got %+v", resp.Items)
	}
	if resp.Items[0].TableName != "bronze_asset_events" || resp.Items[0].RowCount != 10 {
		t.Fatalf("unexpected bronze row: %+v", resp.Items[0])
	}
	if resp.Items[1].TableName != "silver_asset_events_current" || resp.Items[1].RowCount != 7 {
		t.Fatalf("unexpected silver row: %+v", resp.Items[1])
	}
}

func TestTablesToleratesMissingSilver(t *testing.T) {
	gin.SetMode(gin.TestMode)
	lake := &fakeLakehouseQuerier{
		rowsByTable: map[string][]lakehouseport.Row{
			"bronze_asset_events": {
				{"table_name": "bronze_asset_events", "row_count": int64(10)},
			},
		},
		errByTable: map[string]error{
			"silver_asset_events_current": errors.New("not found: silver_asset_events_current"),
		},
	}
	h := New("", lake, nil)
	r := gin.New()
	r.GET("/lakehouse/tables", h.Tables)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/lakehouse/tables", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Items []struct {
			TableName string `json:"table_name"`
			RowCount  int64  `json:"row_count"`
		} `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].TableName != "bronze_asset_events" {
		t.Fatalf("expected only bronze row, got %+v", resp.Items)
	}
}

func TestTablesFailsWhenBronzeUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	lake := &fakeLakehouseQuerier{
		errByTable: map[string]error{
			"bronze_asset_events": lakehouseport.ErrLakehouseDisabled,
		},
	}
	h := New("", lake, nil)
	r := gin.New()
	r.GET("/lakehouse/tables", h.Tables)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/lakehouse/tables", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestLakehouseTableCountShapesRows(t *testing.T) {
	lake := &fakeLakehouseQuerier{
		rowsByTable: map[string][]lakehouseport.Row{
			"silver_asset_events_current": {
				{"table_name": "silver_asset_events_current", "row_count": float64(12)},
			},
		},
	}
	h := New("", lake, nil)
	items, err := h.lakehouseTableCount(context.Background(), "silver_asset_events_current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 || items[0]["table_name"] != "silver_asset_events_current" || items[0]["row_count"] != int64(12) {
		t.Fatalf("unexpected shaped rows: %+v", items)
	}
}

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
