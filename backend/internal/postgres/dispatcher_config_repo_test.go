package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// ── CYB-3679: dispatcher config SQL surface ──────────────────────────────────

type dispatcherRows struct {
	rows    [][]any
	i       int
	err     error
	scanErr error
}

func (r *dispatcherRows) Next() bool { return r.i < len(r.rows) }
func (r *dispatcherRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	row := r.rows[r.i]
	r.i++
	*(dest[0].(*string)) = row[0].(string)
	*(dest[1].(*int)) = row[1].(int)
	*(dest[2].(*int)) = row[2].(int)
	*(dest[3].(*float64)) = row[3].(float64)
	*(dest[4].(*bool)) = row[4].(bool)
	*(dest[5].(*string)) = row[5].(string)
	*(dest[6].(*time.Time)) = row[6].(time.Time)
	return nil
}
func (r *dispatcherRows) Err() error { return r.err }
func (r *dispatcherRows) Close()     {}

type dispatcherFakeDB struct {
	pgDB
	rows     *dispatcherRows
	queryErr error
	execErr  error
	lastSQL  string
	lastArgs []any
}

func (f *dispatcherFakeDB) Query(_ context.Context, sql string, args ...any) (rowsScanner, error) {
	f.lastSQL, f.lastArgs = sql, args
	if f.queryErr != nil {
		return nil, f.queryErr
	}
	return f.rows, nil
}
func (f *dispatcherFakeDB) ExecResult(_ context.Context, sql string, args ...any) (int64, error) {
	f.lastSQL, f.lastArgs = sql, args
	return 1, f.execErr
}

func TestDispatcherConfigRepo_List(t *testing.T) {
	now := time.Now()
	db := &dispatcherFakeDB{rows: &dispatcherRows{rows: [][]any{
		{"clu-a", 16, 25, 5.0, true, "op@x", now},
	}}}
	r := NewDispatcherConfigRepo(&Client{db: db})
	out, err := r.List(context.Background())
	if err != nil || len(out) != 1 {
		t.Fatalf("out=%v err=%v", out, err)
	}
	c := out[0]
	if c.ClusterID != "clu-a" || c.MaxConcurrency != 16 || c.SubmitBatch != 25 || c.RatePerSec != 5 || !c.Paused || c.UpdatedBy != "op@x" {
		t.Fatalf("row = %+v", c)
	}

	db.queryErr = errors.New("db down")
	if _, err := r.List(context.Background()); err == nil {
		t.Fatal("want query error")
	}

	// Scan error surfaces wrapped.
	db.queryErr = nil
	db.rows = &dispatcherRows{rows: [][]any{{"clu-a", 16, 25, 5.0, true, "op@x", now}}, scanErr: errors.New("bad row")}
	if _, err := r.List(context.Background()); err == nil || !strings.Contains(err.Error(), "scan") {
		t.Fatalf("err = %v, want scan error", err)
	}
}

func TestDispatcherConfigRepo_Upsert(t *testing.T) {
	db := &dispatcherFakeDB{}
	r := NewDispatcherConfigRepo(&Client{db: db})
	cfg := &models.DispatcherConfig{ClusterID: "clu-a", MaxConcurrency: 16, SubmitBatch: 25, RatePerSec: 5, Paused: true, UpdatedBy: "op@x"}
	if err := r.Upsert(context.Background(), cfg); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if !strings.Contains(db.lastSQL, "ON CONFLICT (cluster_id)") || db.lastArgs[0] != "clu-a" {
		t.Fatalf("sql=%q args=%v", db.lastSQL, db.lastArgs)
	}

	if err := r.Upsert(context.Background(), nil); err == nil {
		t.Fatal("nil cfg must error")
	}
	if err := r.Upsert(context.Background(), &models.DispatcherConfig{}); err == nil {
		t.Fatal("empty cluster must error")
	}
	db.execErr = errors.New("db down")
	if err := r.Upsert(context.Background(), cfg); err == nil {
		t.Fatal("want exec error")
	}
}

func TestDispatcherConfigRepo_Delete(t *testing.T) {
	db := &dispatcherFakeDB{}
	r := NewDispatcherConfigRepo(&Client{db: db})
	if err := r.Delete(context.Background(), "clu-a"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !strings.Contains(db.lastSQL, "DELETE FROM dispatcher_configs") {
		t.Fatalf("sql=%q", db.lastSQL)
	}
	if err := r.Delete(context.Background(), ""); err == nil {
		t.Fatal("empty cluster must error")
	}
	db.execErr = errors.New("db down")
	if err := r.Delete(context.Background(), "clu-a"); err == nil {
		t.Fatal("want exec error")
	}
}
