package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// execRecorderDB captures Exec calls; all other pgDB methods are inherited
// from the embedded nil interface and must not be reached.
type execRecorderDB struct {
	pgDB
	sqls []string
	args [][]any
	err  error
}

func (f *execRecorderDB) Exec(_ context.Context, sql string, args ...any) error {
	f.sqls = append(f.sqls, sql)
	f.args = append(f.args, args)
	return f.err
}

func TestRuntimeConfigBlobUpsert(t *testing.T) {
	ctx := context.Background()

	t.Run("requires hash", func(t *testing.T) {
		r := NewRuntimeConfigBlobRepo(&Client{db: &execRecorderDB{}})
		if err := r.Upsert(ctx, "", map[string]string{"f": "x"}); err == nil {
			t.Fatal("want error for empty hash")
		}
	})
	t.Run("upserts json files keyed by hash", func(t *testing.T) {
		db := &execRecorderDB{}
		r := NewRuntimeConfigBlobRepo(&Client{db: db})
		if err := r.Upsert(ctx, "h1", map[string]string{"cfg.yaml": "a: 1\n"}); err != nil {
			t.Fatalf("upsert: %v", err)
		}
		if len(db.sqls) != 1 || !strings.Contains(db.sqls[0], "ON CONFLICT (hash) DO UPDATE") {
			t.Fatalf("sql = %v, want idempotent upsert", db.sqls)
		}
		if len(db.args[0]) != 2 || db.args[0][0] != "h1" {
			t.Fatalf("args = %#v, want [h1 <json>]", db.args[0])
		}
		if payload, ok := db.args[0][1].([]byte); !ok || !strings.Contains(string(payload), "cfg.yaml") {
			t.Fatalf("payload = %#v, want json containing cfg.yaml", db.args[0][1])
		}
	})
	t.Run("wraps exec error", func(t *testing.T) {
		db := &execRecorderDB{err: errors.New("db down")}
		r := NewRuntimeConfigBlobRepo(&Client{db: db})
		if err := r.Upsert(ctx, "h1", map[string]string{"f": "x"}); err == nil {
			t.Fatal("want wrapped exec error")
		}
	})
}
