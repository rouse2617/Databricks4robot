package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	"reflect"
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"
)

type fakeDB struct {
	queryRow         rowScanner
	queryErr         error
	rows             rowsScanner
	execErr          error
	execRowsAffected int64
	pingErr          error
	closed           bool
	execSQLs         []string
	execArgs         [][]any
}

func (f *fakeDB) QueryRow(_ context.Context, _ string, _ ...any) rowScanner {
	if f.queryRow == nil {
		return &fakeRow{err: errNoRows}
	}
	return f.queryRow
}
func (f *fakeDB) Query(_ context.Context, _ string, _ ...any) (rowsScanner, error) {
	if f.queryErr != nil {
		return nil, f.queryErr
	}
	if f.rows == nil {
		return &fakeRows{}, nil
	}
	return f.rows, nil
}
func (f *fakeDB) Exec(_ context.Context, q string, args ...any) error {
	f.execSQLs = append(f.execSQLs, q)
	f.execArgs = append(f.execArgs, args)
	return f.execErr
}
func (f *fakeDB) ExecResult(_ context.Context, _ string, _ ...any) (int64, error) {
	if f.execErr != nil {
		return 0, f.execErr
	}
	return f.execRowsAffected, nil
}
func (f *fakeDB) Ping(_ context.Context) error { return f.pingErr }
func (f *fakeDB) Close()                       { f.closed = true }

type fakeRow struct {
	values []any
	err    error
}

func (r *fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return errors.New("scan length mismatch")
	}
	for i := range dest {
		if err := assign(dest[i], r.values[i]); err != nil {
			return err
		}
	}
	return nil
}

type fakeRows struct {
	data   [][]any
	idx    int
	closed bool
}

func (r *fakeRows) Next() bool {
	if r.idx >= len(r.data) {
		return false
	}
	r.idx++
	return true
}

func (r *fakeRows) Scan(dest ...any) error {
	if r.idx == 0 || r.idx > len(r.data) {
		return errors.New("scan called before next")
	}
	cur := r.data[r.idx-1]
	if len(dest) != len(cur) {
		return errors.New("scan length mismatch")
	}
	for i := range dest {
		if err := assign(dest[i], cur[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *fakeRows) Close() { r.closed = true }

func assign(dst any, src any) error {
	dv := reflect.ValueOf(dst)
	if dv.Kind() != reflect.Ptr || dv.IsNil() {
		return errors.New("dest must be pointer")
	}
	elem := dv.Elem()
	if src == nil {
		elem.Set(reflect.Zero(elem.Type()))
		return nil
	}
	sv := reflect.ValueOf(src)
	if sv.Type().AssignableTo(elem.Type()) {
		elem.Set(sv)
		return nil
	}
	if elem.Kind() == reflect.Ptr {
		target := elem.Type().Elem()
		switch {
		case sv.Type().AssignableTo(target):
			ptr := reflect.New(target)
			ptr.Elem().Set(sv)
			elem.Set(ptr)
			return nil
		case sv.Type().ConvertibleTo(target):
			ptr := reflect.New(target)
			ptr.Elem().Set(sv.Convert(target))
			elem.Set(ptr)
			return nil
		}
	}
	if sv.Type().ConvertibleTo(elem.Type()) {
		elem.Set(sv.Convert(elem.Type()))
		return nil
	}
	return errors.New("incompatible assign")
}

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t.Fatalf("parse time: %v", err)
	}
	return ts
}

func TestClientNewAndClose(t *testing.T) {
	old := newPool
	t.Cleanup(func() { newPool = old })

	newPool = func(_ context.Context, _ string) (pgDB, error) { return &fakeDB{}, nil }
	c, err := New(context.Background(), &config.Config{DBUser: "u", DBPassword: "p", DBHost: "h", DBPort: "5432", DBName: "d"})
	if err != nil {
		t.Fatalf("new err: %v", err)
	}
	c.Close()

	newPool = func(_ context.Context, _ string) (pgDB, error) { return nil, errors.New("boom") }
	if _, err := New(context.Background(), &config.Config{}); err == nil || !strings.Contains(err.Error(), "postgres connect") {
		t.Fatalf("expected connect error")
	}

	newPool = func(_ context.Context, _ string) (pgDB, error) { return &fakeDB{pingErr: errors.New("ping")}, nil }
	if _, err := New(context.Background(), &config.Config{}); err == nil || !strings.Contains(err.Error(), "postgres ping") {
		t.Fatalf("expected ping error")
	}

	var nilClient *Client
	nilClient.Close()
}

func TestRealDBPanicPaths(t *testing.T) {
	assertPanic := func(t *testing.T, fn func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Fatalf("expected panic")
			}
		}()
		fn()
	}
	r := &realDB{}
	assertPanic(t, func() { _ = r.QueryRow(context.Background(), "q") })
	assertPanic(t, func() { _, _ = r.Query(context.Background(), "q") })
	assertPanic(t, func() { _ = r.Exec(context.Background(), "q") })
	assertPanic(t, func() { _, _ = r.ExecResult(context.Background(), "q") })
	assertPanic(t, func() { _ = r.Ping(context.Background()) })
	assertPanic(t, func() { r.Close() })
}

func TestAssetRepo(t *testing.T) {
	ctx := context.Background()
	db := &fakeDB{execRowsAffected: 1}
	repo := &AssetRepo{c: &Client{db: db}}

	if got, err := repo.Get(ctx, "a1"); err != nil || got != nil {
		t.Fatalf("expected nil,nil not found")
	}
	db.queryRow = &fakeRow{err: errors.New("q")}
	if _, err := repo.Get(ctx, "a1"); err == nil {
		t.Fatalf("expected get error")
	}
	// Get now scans 39 columns (including algo_inputs_uris/annot_inputs_uris JSONB)
	db.queryRow = &fakeRow{values: []any{
		"a1", "m1", int64(10), int64(20), (*string)(nil),
		"ready", "segment", int64(1200),
		"o", "r", int(0), (*time.Time)(nil), "",
		"", (*time.Time)(nil), "", "", int(0),
		(*string)(nil), (*string)(nil),
		"", "", "", "", "",
		(*int)(nil), (*int64)(nil), (*int64)(nil),
		(*string)(nil), (*string)(nil),
		[]byte(`{}`), []byte(`{}`), []byte(`{}`), []byte(`{}`),
		(*string)(nil), (*int64)(nil), (*bool)(nil),
		mustTime(t, "2026-04-20T00:00:00Z"), mustTime(t, "2026-04-21T00:00:00Z"), int64(1),
	}}
	got, err := repo.Get(ctx, "a1")
	if err != nil || got == nil || got.AssetID != "a1" {
		t.Fatalf("get success failed: %v %+v", err, got)
	}
	if got.LifecycleState != "ready" || got.AssetType != "segment" || got.DurationMs != 1200 {
		t.Fatalf("new columns not read: lifecycle=%s type=%s durationMs=%d", got.LifecycleState, got.AssetType, got.DurationMs)
	}

	a := &models.Asset{AssetID: "a2", McapFileID: "m1", Status: models.AssetStatusApproved}
	if err := repo.Set(ctx, a); err != nil {
		t.Fatalf("set err: %v", err)
	}
	if a.Version != 1 || a.CreatedAt.IsZero() {
		t.Fatalf("set metadata missing")
	}
	db.execErr = errors.New("e")
	if err := repo.Set(ctx, &models.Asset{AssetID: "a3"}); err == nil {
		t.Fatalf("expected set error")
	}
	db.execErr = nil

	// Optimistic lock: zero rows affected (CAS mismatch) maps to ErrOptimisticLock.
	db.execRowsAffected = 0
	if err := repo.Set(ctx, &models.Asset{AssetID: "a4"}); !errors.Is(err, repository.ErrOptimisticLock) {
		t.Fatalf("expected ErrOptimisticLock on row CAS miss, got %v", err)
	}
	db.execRowsAffected = 1

	if err := repo.SoftDelete(ctx, "a1"); err != nil {
		t.Fatalf("soft delete err: %v", err)
	}
	db.execErr = errors.New("e")
	if err := repo.SoftDelete(ctx, "a1"); err == nil {
		t.Fatalf("expected soft delete error")
	}
	db.execErr = nil

	rows := &fakeRows{data: [][]any{
		{"a1", "m1", int64(10), int64(20), (*string)(nil),
			"ready", "segment", int64(0),
			"", "", int(0), (*time.Time)(nil), "",
			"", (*time.Time)(nil), "", "", int(0),
			(*string)(nil), (*string)(nil), (*string)(nil), (*string)(nil),
			[]byte(`{}`), []byte(`{}`), []byte(`{}`), []byte(`{}`),
			(*string)(nil), (*int64)(nil), (*bool)(nil),
			mustTime(t, "2026-04-20T00:00:00Z"), mustTime(t, "2026-04-21T00:00:00Z"), int64(1)},
	}}
	db.rows = rows
	list, err := repo.ListByMcapFile(ctx, "m1")
	if err != nil || len(list) != 1 {
		t.Fatalf("list by mcap failed: err=%v len=%d", err, len(list))
	}
	db.queryErr = errors.New("q")
	if _, err := repo.ListByMcapFile(ctx, "m1"); err == nil {
		t.Fatalf("expected list query error")
	}
	db.queryErr = nil
	db.rows = &fakeRows{data: [][]any{{"a1"}}}
	if _, err := repo.ListByMcapFile(ctx, "m1"); err == nil {
		t.Fatalf("expected scan error")
	}

	if err := repo.WriteSegmentIndex(ctx, &models.Asset{}); err != nil {
		t.Fatalf("write segment index no-op should succeed")
	}
}

func TestMcapRepo(t *testing.T) {
	ctx := context.Background()
	db := &fakeDB{}
	repo := &McapFileRepo{c: &Client{db: db}}

	if got, err := repo.Get(ctx, "m1"); err != nil || got != nil {
		t.Fatalf("expected nil,nil not found")
	}
	db.queryRow = &fakeRow{err: errors.New("q")}
	if _, err := repo.Get(ctx, "m1"); err == nil {
		t.Fatalf("expected get error")
	}
	// Get now scans 31 columns (all real columns including provenance/retention/metadata)
	db.queryRow = &fakeRow{values: []any{
		"m1", "md5", (*string)(nil), // raw_hash_sha256
		"gs://x", int64(10), (*int64)(nil), // file_duration_ms
		int64(1), int64(2),
		int(3), int(4), "pending", "o",
		"", "", "", "", // vendor_id, collector_id, task_id, device_id
		"", "", "", "", "", "", // camera_model, data_source, location_id, scene_id, environment_id, collection_method
		(*string)(nil), (*time.Time)(nil), (*string)(nil), (*string)(nil), // retention_tier, expire_at, tenant_id, project_id
		[]byte(`{}`), []byte(`{"p":"done"}`), // metadata, process_state
		mustTime(t, "2026-04-20T00:00:00Z"), mustTime(t, "2026-04-21T00:00:00Z"), int64(1),
	}}
	got, err := repo.Get(ctx, "m1")
	if err != nil || got == nil || got.McapFileID != "m1" {
		t.Fatalf("get success failed")
	}
	if got.GCSPath != "gs://x" || got.SizeBytes != 10 || got.IngestState != models.IngestStatePending {
		t.Fatalf("real columns not read: gcs=%s size=%d ingest=%s", got.GCSPath, got.SizeBytes, got.IngestState)
	}
	if got.Owner != "o" || got.ChannelCount != 3 || got.ChunkCount != 4 {
		t.Fatalf("real columns not read: owner=%s ch=%d chunk=%d", got.Owner, got.ChannelCount, got.ChunkCount)
	}
	if got.ProcessState["p"] != "done" {
		t.Fatalf("process_state not read: %v", got.ProcessState)
	}

	f := &models.McapFile{McapFileID: "m2", IngestState: models.IngestStatePending}
	if err := repo.Set(ctx, f); err != nil {
		t.Fatalf("set err: %v", err)
	}
	if f.Version != 1 || f.CreatedAt.IsZero() {
		t.Fatalf("set metadata missing")
	}
	db.execErr = errors.New("e")
	if err := repo.Set(ctx, &models.McapFile{McapFileID: "m3"}); err == nil {
		t.Fatalf("expected set error")
	}
	db.execErr = nil

	if err := repo.UpdateIngestState(ctx, "m2", models.IngestStateSummarized); err != nil {
		t.Fatalf("update ingest err: %v", err)
	}
	db.execErr = errors.New("e")
	if err := repo.UpdateIngestState(ctx, "m2", models.IngestStateFailed); err == nil {
		t.Fatalf("expected update error")
	}
}

func TestDeliveryRepo(t *testing.T) {
	ctx := context.Background()
	db := &fakeDB{}
	repo := &DeliveryRepo{c: &Client{db: db}}

	d := &models.Delivery{DeliveryID: "d1", CustomerID: "c1", Status: models.DeliveryStatusPending}
	if err := repo.Set(ctx, d); err != nil {
		t.Fatalf("set err: %v", err)
	}
	if d.Version != 1 || d.CreatedAt.IsZero() {
		t.Fatalf("set metadata missing")
	}
	db.execErr = errors.New("e")
	if err := repo.Set(ctx, &models.Delivery{DeliveryID: "d2"}); err == nil {
		t.Fatalf("expected set error")
	}
	db.execErr = nil

	if got, err := repo.Get(ctx, "d1"); err != nil || got != nil {
		t.Fatalf("expected nil,nil not found")
	}
	db.queryRow = &fakeRow{err: errors.New("q")}
	if _, err := repo.Get(ctx, "d1"); err == nil {
		t.Fatalf("expected get error")
	}
	at := mustTime(t, "2026-04-21T00:00:00Z")
	db.queryRow = &fakeRow{values: buildDeliveryRow(
		"d1", "c1", "delivered", &at,
		"ct-1", "asset_set", "", "", "urn:owner",
		"gs://m", "", int64(1), (*int64)(nil),
		(*time.Time)(nil), []byte(`{"note":"ok"}`), (*string)(nil), (*string)(nil),
	)}
	got, err := repo.Get(ctx, "d1")
	if err != nil || got == nil || got.DeliveryID != "d1" || got.ContractID != "ct-1" || got.Note != "ok" || got.Owner != "urn:owner" {
		t.Fatalf("get success failed: %v %+v", err, got)
	}
	if got.ManifestURI != "gs://m" {
		t.Fatalf("manifest_uri not read from real column: %s", got.ManifestURI)
	}
	if got.DeliveryType != "asset_set" {
		t.Fatalf("delivery_type not read: %s", got.DeliveryType)
	}
	if got.ItemCount != 1 {
		t.Fatalf("item_count not read: %d", got.ItemCount)
	}
	if got.DeliveredBy != "urn:owner" {
		t.Fatalf("delivered_by not read: %s", got.DeliveredBy)
	}

	if err := repo.AddItems(ctx, "d1", []string{"a1"}); err != nil {
		t.Fatalf("add items err: %v", err)
	}
	db.execErr = errors.New("e")
	if err := repo.AddItems(ctx, "d1", []string{"a1"}); err == nil {
		t.Fatalf("expected add items error")
	}
	db.execErr = nil

	db.rows = &fakeRows{data: [][]any{{"d1"}}}
	ids, err := repo.ListByCustomer(ctx, "c1")
	if err != nil || len(ids) != 1 {
		t.Fatalf("list by customer failed")
	}
	db.queryErr = errors.New("q")
	if _, err := repo.ListByCustomer(ctx, "c1"); err == nil {
		t.Fatalf("expected list query error")
	}
	db.queryErr = nil
	db.rows = &fakeRows{data: [][]any{{struct{}{}}}}
	if _, err := repo.ListByCustomer(ctx, "c1"); err == nil {
		t.Fatalf("expected list scan error")
	}
}

func TestIdempotencyRepo(t *testing.T) {
	ctx := context.Background()
	db := &fakeDB{}
	repo := &IdempotencyRepo{c: &Client{db: db}}

	if err := repo.Lock(ctx, "s", "k"); err != nil {
		t.Fatalf("lock err: %v", err)
	}
	if len(db.execSQLs) != 1 || !strings.Contains(db.execSQLs[0], "pg_advisory_xact_lock") {
		t.Fatalf("expected advisory lock SQL, got %#v", db.execSQLs)
	}

	if got, err := repo.Get(ctx, "s", "k"); err != nil || got != nil {
		t.Fatalf("expected nil,nil not found")
	}
	db.queryRow = &fakeRow{err: errors.New("q")}
	if _, err := repo.Get(ctx, "s", "k"); err == nil {
		t.Fatalf("expected get error")
	}
	db.queryRow = &fakeRow{values: []any{
		"s", "k", "h", 200, []byte(`{}`), mustTime(t, "2026-04-21T00:00:00Z"),
	}}
	rec, err := repo.Get(ctx, "s", "k")
	if err != nil || rec == nil || rec.Scope != "s" || rec.Key != "k" {
		t.Fatalf("get success failed")
	}

	in := &repository.IdempotencyRecord{Scope: "s", Key: "k", RequestHash: "h", StatusCode: 201, Response: []byte(`{}`)}
	if err := repo.Save(ctx, in); err != nil {
		t.Fatalf("save err: %v", err)
	}
	if in.CreatedAt.IsZero() {
		t.Fatalf("created_at should be set")
	}
	db.execErr = errors.New("e")
	if err := repo.Save(ctx, in); err == nil {
		t.Fatalf("expected save error")
	}
}

func TestRepoConstructors(t *testing.T) {
	c := &Client{db: &fakeDB{}}
	if NewAssetRepo(c) == nil {
		t.Fatalf("NewAssetRepo nil")
	}
	if NewMcapFileRepo(c) == nil {
		t.Fatalf("NewMcapFileRepo nil")
	}
	if NewDeliveryRepo(c) == nil {
		t.Fatalf("NewDeliveryRepo nil")
	}
	if NewIdempotencyRepo(c) == nil {
		t.Fatalf("NewIdempotencyRepo nil")
	}
}

func TestMergeCfAlgo_Success(t *testing.T) {
	ctx := context.Background()
	db := &fakeDB{execRowsAffected: 1}
	repo := &AssetRepo{c: &Client{db: db}}

	algoKV := map[string]interface{}{"hand_tracking@1.2.0:status": "running"}
	filesKV := map[string]interface{}{}

	newVer, err := repo.MergeCfAlgo(ctx, "a1", 5, algoKV, filesKV)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if newVer != 6 {
		t.Fatalf("expected version 6, got %d", newVer)
	}
}

func TestMergeCfAlgo_OptimisticLockConflict(t *testing.T) {
	ctx := context.Background()
	db := &fakeDB{execRowsAffected: 0} // 0 rows affected = version mismatch
	repo := &AssetRepo{c: &Client{db: db}}

	algoKV := map[string]interface{}{"hand_tracking@1.2.0:status": "running"}
	filesKV := map[string]interface{}{}

	_, err := repo.MergeCfAlgo(ctx, "a1", 5, algoKV, filesKV)
	if err == nil {
		t.Fatalf("expected optimistic lock error")
	}
	if !errors.Is(err, repository.ErrOptimisticLock) {
		t.Fatalf("expected ErrOptimisticLock, got %v", err)
	}
}

func TestMergeCfAlgo_DBError(t *testing.T) {
	ctx := context.Background()
	db := &fakeDB{execErr: errors.New("db down")}
	repo := &AssetRepo{c: &Client{db: db}}

	_, err := repo.MergeCfAlgo(ctx, "a1", 5, map[string]interface{}{}, map[string]interface{}{})
	if err == nil || !strings.Contains(err.Error(), "MergeCfAlgo") {
		t.Fatalf("expected db error, got %v", err)
	}
}

func TestListWithFilters_NoFilters(t *testing.T) {
	ctx := context.Background()
	db := &fakeDB{}
	repo := &AssetRepo{c: &Client{db: db}}

	// COUNT returns 2
	db.queryRow = &fakeRow{values: []any{int64(2)}}
	// DATA returns 2 rows (31 columns each — includes metadata/files/algo/annot JSONB)
	db.rows = &fakeRows{data: [][]any{
		{"a1", "m1", int64(10), int64(20), (*string)(nil),
			"ready", "segment", int64(0),
			"", "", int(0), (*time.Time)(nil), "",
			"", (*time.Time)(nil), "", "", int(0),
			(*string)(nil), (*string)(nil), (*string)(nil), (*string)(nil),
			[]byte(`{}`), []byte(`{}`), []byte(`{}`), []byte(`{}`),
			(*string)(nil), (*int64)(nil), (*bool)(nil),
			mustTime(t, "2026-04-20T00:00:00Z"), mustTime(t, "2026-04-21T00:00:00Z"), int64(1)},
		{"a2", "m1", int64(20), int64(30), (*string)(nil),
			"ready", "segment", int64(0),
			"", "", int(0), (*time.Time)(nil), "",
			"", (*time.Time)(nil), "", "", int(0),
			(*string)(nil), (*string)(nil), (*string)(nil), (*string)(nil),
			[]byte(`{}`), []byte(`{}`), []byte(`{}`), []byte(`{}`),
			(*string)(nil), (*int64)(nil), (*bool)(nil),
			mustTime(t, "2026-04-20T00:00:00Z"), mustTime(t, "2026-04-21T00:00:00Z"), int64(2)},
	}}

	assets, total, err := repo.ListWithFilters(ctx, "", nil, 1, 20, filter.OrderByClause{SQL: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2, got %d", total)
	}
	if len(assets) != 2 {
		t.Fatalf("expected 2 assets, got %d", len(assets))
	}
	if assets[0].AssetID != "a1" || assets[1].AssetID != "a2" {
		t.Fatalf("unexpected asset IDs: %s, %s", assets[0].AssetID, assets[1].AssetID)
	}
}

func TestListWithFilters_WithWhereSQL(t *testing.T) {
	ctx := context.Background()
	db := &fakeDB{}
	repo := &AssetRepo{c: &Client{db: db}}

	db.queryRow = &fakeRow{values: []any{int64(1)}}
	db.rows = &fakeRows{data: [][]any{
		{"a1", "m1", int64(10), int64(20), (*string)(nil),
			"ready", "segment", int64(0),
			"", "", int(0), (*time.Time)(nil), "",
			"", (*time.Time)(nil), "", "", int(0),
			(*string)(nil), (*string)(nil), (*string)(nil), (*string)(nil),
			[]byte(`{}`), []byte(`{}`), []byte(`{}`), []byte(`{}`),
			(*string)(nil), (*int64)(nil), (*bool)(nil),
			mustTime(t, "2026-04-20T00:00:00Z"), mustTime(t, "2026-04-21T00:00:00Z"), int64(1)},
	}}

	assets, total, err := repo.ListWithFilters(ctx, "lifecycle_state = $1", []interface{}{"ready"}, 1, 10, filter.OrderByClause{SQL: "created_at DESC"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(assets) != 1 || assets[0].AssetID != "a1" {
		t.Fatalf("unexpected result")
	}
}

func TestListWithFilters_CountError(t *testing.T) {
	ctx := context.Background()
	db := &fakeDB{}
	repo := &AssetRepo{c: &Client{db: db}}

	db.queryRow = &fakeRow{err: errors.New("count fail")}
	_, _, err := repo.ListWithFilters(ctx, "", nil, 1, 20, filter.OrderByClause{SQL: ""})
	if err == nil || !strings.Contains(err.Error(), "count") {
		t.Fatalf("expected count error, got %v", err)
	}
}

func TestListWithFilters_QueryError(t *testing.T) {
	ctx := context.Background()
	db := &fakeDB{}
	repo := &AssetRepo{c: &Client{db: db}}

	db.queryRow = &fakeRow{values: []any{int64(1)}}
	db.queryErr = errors.New("query fail")
	_, _, err := repo.ListWithFilters(ctx, "", nil, 1, 20, filter.OrderByClause{SQL: ""})
	if err == nil || !strings.Contains(err.Error(), "query") {
		t.Fatalf("expected query error, got %v", err)
	}
}

func TestListWithFilters_ScanError(t *testing.T) {
	ctx := context.Background()
	db := &fakeDB{}
	repo := &AssetRepo{c: &Client{db: db}}

	db.queryRow = &fakeRow{values: []any{int64(1)}}
	db.rows = &fakeRows{data: [][]any{{"a1"}}} // too few columns → scan error
	_, _, err := repo.ListWithFilters(ctx, "", nil, 1, 20, filter.OrderByClause{SQL: ""})
	if err == nil || !strings.Contains(err.Error(), "scan") {
		t.Fatalf("expected scan error, got %v", err)
	}
}

// Property 20: Optimistic lock version increment
func TestProperty20_OptimisticLockVersionIncrement(t *testing.T) {
	ctx := context.Background()

	for _, ver := range []int64{0, 1, 5, 100, 999} {
		db := &fakeDB{execRowsAffected: 1}
		repo := &AssetRepo{c: &Client{db: db}}

		newVer, err := repo.MergeCfAlgo(ctx, "a1", ver,
			map[string]interface{}{"k": "v"}, map[string]interface{}{})
		if err != nil {
			t.Fatalf("version %d: unexpected error: %v", ver, err)
		}
		if newVer != ver+1 {
			t.Fatalf("version %d: expected new version %d, got %d", ver, ver+1, newVer)
		}
	}

	for _, ver := range []int64{0, 1, 5, 100, 999} {
		db := &fakeDB{execRowsAffected: 0}
		repo := &AssetRepo{c: &Client{db: db}}

		_, err := repo.MergeCfAlgo(ctx, "a1", ver,
			map[string]interface{}{"k": "v"}, map[string]interface{}{})
		if !errors.Is(err, repository.ErrOptimisticLock) {
			t.Fatalf("version %d: expected ErrOptimisticLock, got %v", ver, err)
		}
	}
}

func TestProperty6_RealColumnConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		owner := rapid.StringMatching(`[a-z]{0,10}`).Draw(t, "owner")
		reviewer := rapid.StringMatching(`[a-z]{0,10}`).Draw(t, "reviewer")
		durationMs := rapid.Int64Range(0, 3_600_000).Draw(t, "duration_ms")
		deliveryCount := rapid.IntRange(0, 100).Draw(t, "delivery_count")
		lastDeliveredTo := rapid.StringMatching(`[a-z]{0,10}`).Draw(t, "last_delivered_to")
		retentionTier := rapid.SampledFrom([]string{"", "hot", "warm", "cold"}).Draw(t, "retention_tier")
		lifecycleState := rapid.SampledFrom([]string{"created", "processing", "ready", "rejected", "delivered", "archived", "superseded", "failed"}).Draw(t, "lifecycle_state")
		assetType := rapid.SampledFrom([]string{"segment", "clip", "frame_set", "derived_asset"}).Draw(t, "asset_type")

		a := &models.Asset{
			AssetID:         "prop6-" + rapid.StringMatching(`[a-f0-9]{8}`).Draw(t, "id"),
			McapFileID:      "m1",
			Owner:           owner,
			Reviewer:        reviewer,
			DurationMs:      durationMs,
			DurationSec:     float64(durationMs) / 1000.0,
			DeliveryCount:   deliveryCount,
			LastDeliveredTo: lastDeliveredTo,
			RetentionTier:   retentionTier,
			LifecycleState:  lifecycleState,
			AssetType:       assetType,
			SegType:         assetType,
			Status:          models.AssetStatus(LifecycleStateToStatus(lifecycleState)),
		}

		capDB := &capturingDB{fakeDB: &fakeDB{execRowsAffected: 1}}
		repo := &AssetRepo{c: &Client{db: capDB}}

		err := repo.Set(context.Background(), a)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
		capturedArgs := capDB.lastArgs

		// Set() now passes 29 positional args (no assets.status column).
		if len(capturedArgs) < 29 {
			t.Fatalf("expected 29 args, got %d", len(capturedArgs))
		}

		// lifecycle_state (arg index 5)
		if got := capturedArgs[5]; got != a.LifecycleState {
			t.Fatalf("lifecycle_state mismatch: real=%v want=%v", got, a.LifecycleState)
		}

		// asset_type (arg index 6)
		if got := capturedArgs[6]; got != a.AssetType {
			t.Fatalf("asset_type mismatch: real=%v want=%v", got, a.AssetType)
		}

		// owner (arg index 8)
		if got := capturedArgs[8]; got != a.Owner {
			t.Fatalf("owner mismatch: real=%v want=%v", got, a.Owner)
		}

		// reviewer (arg index 9)
		if got := capturedArgs[9]; got != a.Reviewer {
			t.Fatalf("reviewer mismatch: real=%v want=%v", got, a.Reviewer)
		}

		// delivery_count (arg index 10)
		if got := capturedArgs[10]; got != a.DeliveryCount {
			t.Fatalf("delivery_count mismatch: real=%v want=%v", got, a.DeliveryCount)
		}

		// last_delivered_to (arg index 12)
		if got := capturedArgs[12]; got != a.LastDeliveredTo {
			t.Fatalf("last_delivered_to mismatch: real=%v want=%v", got, a.LastDeliveredTo)
		}

		expectedStatus := LifecycleStateToStatus(a.LifecycleState)
		if string(a.Status) != expectedStatus {
			t.Fatalf("status/lifecycle mismatch: status=%v expected=%v for lifecycle=%v", a.Status, expectedStatus, a.LifecycleState)
		}
	})
}

// capturingDB wraps fakeDB to capture the last ExecResult call args.
type capturingDB struct {
	*fakeDB
	lastArgs []any
}

func (c *capturingDB) ExecResult(ctx context.Context, sql string, args ...any) (int64, error) {
	c.lastArgs = args
	return c.fakeDB.ExecResult(ctx, sql, args...)
}

func (c *capturingDB) QueryRow(ctx context.Context, sql string, args ...any) rowScanner {
	return c.fakeDB.QueryRow(ctx, sql, args...)
}

func (c *capturingDB) Query(ctx context.Context, sql string, args ...any) (rowsScanner, error) {
	return c.fakeDB.Query(ctx, sql, args...)
}

func (c *capturingDB) Exec(ctx context.Context, sql string, args ...any) error {
	return c.fakeDB.Exec(ctx, sql, args...)
}

func (c *capturingDB) Ping(ctx context.Context) error { return c.fakeDB.Ping(ctx) }
func (c *capturingDB) Close()                         { c.fakeDB.Close() }

// ---------------------------------------------------------------------------
// TestDualWrite_OptimisticLockConflict verifies that version mismatch during
// Set() returns repository.ErrOptimisticLock.
// ---------------------------------------------------------------------------
func TestDualWrite_OptimisticLockConflict(t *testing.T) {
	ctx := context.Background()

	db := &fakeDB{execRowsAffected: 0}
	repo := &AssetRepo{c: &Client{db: db}}

	a := &models.Asset{
		AssetID:        "conflict-1",
		McapFileID:     "m1",
		Status:         models.AssetStatusApproved,
		LifecycleState: "ready",
		AssetType:      "segment",
		Owner:          "alice",
		Version:        5,
	}

	err := repo.Set(ctx, a)
	if err == nil {
		t.Fatalf("expected error on version conflict, got nil")
	}
	if !errors.Is(err, repository.ErrOptimisticLock) {
		t.Fatalf("expected ErrOptimisticLock, got %v", err)
	}
	if a.Version != 6 {
		t.Fatalf("expected version to be bumped to 6, got %d", a.Version)
	}

	db.execRowsAffected = 1
	b := &models.Asset{
		AssetID:        "success-1",
		McapFileID:     "m1",
		Status:         models.AssetStatusApproved,
		LifecycleState: "ready",
		AssetType:      "segment",
		Owner:          "bob",
		Version:        0,
	}
	if err := repo.Set(ctx, b); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if b.Version != 1 {
		t.Fatalf("expected version 1, got %d", b.Version)
	}
}

// ---------------------------------------------------------------------------
// buildAssetRow builds a fake row slice matching the SELECT in
// Get/ListWithFilters/ListByMcapFile.
// ---------------------------------------------------------------------------
func buildAssetRow(
	assetID string,
	lifecycleState string,
	assetType string,
	durationMs int64,
	owner string,
	reviewer string,
	deliveryCount int,
	lastDeliveredAt *time.Time,
	lastDeliveredTo string,
	retentionTier string,
	expireAt *time.Time,
	storageURI string,
	thumbURI string,
	assetLevel int,
) []any {
	return []any{
		assetID, "m1", int64(100), int64(200), (*string)(nil),
		lifecycleState, assetType, durationMs,
		owner, reviewer, deliveryCount, lastDeliveredAt, lastDeliveredTo,
		retentionTier, expireAt, storageURI, thumbURI, assetLevel,
		(*string)(nil), (*string)(nil),
		"", "", "", "", "",
		(*int)(nil), (*int64)(nil), (*int64)(nil),
		(*string)(nil), (*string)(nil),
		[]byte(`{}`), []byte(`{}`), []byte(`{}`), []byte(`{}`),
		(*string)(nil), (*int64)(nil), (*bool)(nil), // logical_asset_id, revision, is_current
		time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC),
		int64(1),
	}
}

func TestGet_PopulatesBothOldAndNewFields(t *testing.T) {
	ctx := context.Background()

	row := buildAssetRow(
		"get-1",
		"ready",      // lifecycle_state
		"clip",       // asset_type
		int64(12500), // duration_ms
		"alice",      // owner
		"bob",        // reviewer
		3,            // delivery_count
		nil,          // last_delivered_at
		"acme",       // last_delivered_to
		"hot",        // retention_tier
		nil,          // expire_at
		"gs://store", // storage_uri
		"gs://thumb", // thumb_uri
		0,            // asset_level
	)

	db := &fakeDB{queryRow: &fakeRow{values: row}}
	repo := &AssetRepo{c: &Client{db: db}}

	got, err := repo.Get(ctx, "get-1")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got == nil {
		t.Fatalf("Get() returned nil")
	}

	// New typed fields from real columns
	if got.LifecycleState != "ready" {
		t.Errorf("LifecycleState: got %q, want %q", got.LifecycleState, "ready")
	}
	if got.AssetType != "clip" {
		t.Errorf("AssetType: got %q, want %q", got.AssetType, "clip")
	}
	if got.DurationMs != 12500 {
		t.Errorf("DurationMs: got %d, want %d", got.DurationMs, 12500)
	}
	if got.RetentionTier != "hot" {
		t.Errorf("RetentionTier: got %q, want %q", got.RetentionTier, "hot")
	}
	if got.StorageURI != "gs://store" {
		t.Errorf("StorageURI: got %q, want %q", got.StorageURI, "gs://store")
	}
	if got.ThumbURI != "gs://thumb" {
		t.Errorf("ThumbURI: got %q, want %q", got.ThumbURI, "gs://thumb")
	}

	// Real columns read directly
	if got.Status != "approved" {
		t.Errorf("Status: got %q, want %q", got.Status, "approved")
	}
	if got.Owner != "alice" {
		t.Errorf("Owner: got %q, want %q", got.Owner, "alice")
	}
	if got.Reviewer != "bob" {
		t.Errorf("Reviewer: got %q, want %q", got.Reviewer, "bob")
	}
	if got.DeliveryCount != 3 {
		t.Errorf("DeliveryCount: got %d, want %d", got.DeliveryCount, 3)
	}
	if got.LastDeliveredTo != "acme" {
		t.Errorf("LastDeliveredTo: got %q, want %q", got.LastDeliveredTo, "acme")
	}

	// SyncLegacyFields: DurationSec computed from DurationMs
	wantDurationSec := float64(12500) / 1000.0
	if got.DurationSec != wantDurationSec {
		t.Errorf("DurationSec: got %f, want %f", got.DurationSec, wantDurationSec)
	}

	// SyncLegacyFields: SegType mirrors AssetType
	if got.SegType != got.AssetType {
		t.Errorf("SegType should mirror AssetType: SegType=%q, AssetType=%q", got.SegType, got.AssetType)
	}
}

func TestGet_DurationSecComputedFromDurationMs(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name       string
		durationMs int64
		wantSec    float64
	}{
		{"zero", 0, 0.0},
		{"one_second", 1000, 1.0},
		{"fractional", 1500, 1.5},
		{"large", 3600000, 3600.0},
		{"small", 1, 0.001},
		{"exact_12500", 12500, 12.5},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			row := buildAssetRow(
				"dur-"+tc.name, "ready", "segment", tc.durationMs,
				"", "", 0, nil, "", "", nil, "", "", 0,
			)
			db := &fakeDB{queryRow: &fakeRow{values: row}}
			repo := &AssetRepo{c: &Client{db: db}}

			got, err := repo.Get(ctx, "dur-"+tc.name)
			if err != nil {
				t.Fatalf("Get() error: %v", err)
			}
			if got.DurationMs != tc.durationMs {
				t.Errorf("DurationMs: got %d, want %d", got.DurationMs, tc.durationMs)
			}
			if got.DurationSec != tc.wantSec {
				t.Errorf("DurationSec: got %f, want %f", got.DurationSec, tc.wantSec)
			}
		})
	}
}

func TestGet_SegTypeMirrorsAssetType(t *testing.T) {
	ctx := context.Background()

	assetTypes := []string{"segment", "clip", "frame_set", "derived_asset"}
	for _, at := range assetTypes {
		t.Run(at, func(t *testing.T) {
			row := buildAssetRow(
				"seg-"+at, "ready", at, int64(1000),
				"", "", 0, nil, "", "", nil, "", "", 0,
			)
			db := &fakeDB{queryRow: &fakeRow{values: row}}
			repo := &AssetRepo{c: &Client{db: db}}

			got, err := repo.Get(ctx, "seg-"+at)
			if err != nil {
				t.Fatalf("Get() error: %v", err)
			}
			if got.SegType != at {
				t.Errorf("SegType: got %q, want %q", got.SegType, at)
			}
			if got.AssetType != at {
				t.Errorf("AssetType: got %q, want %q", got.AssetType, at)
			}
		})
	}
}

func TestListWithFilters_PopulatesBothOldAndNewFields(t *testing.T) {
	ctx := context.Background()

	// List queries return 31 columns (includes metadata/files/algo/annot JSONB)
	row := []any{
		"lf-1", "m1", int64(100), int64(200), (*string)(nil),
		"delivered", "frame_set", int64(5000),
		"dave", "carol", int(1), (*time.Time)(nil), "partner",
		"warm", (*time.Time)(nil), "", "", int(0),
		(*string)(nil), (*string)(nil), (*string)(nil), (*string)(nil),
		[]byte(`{}`), []byte(`{}`), []byte(`{}`), []byte(`{}`),
		(*string)(nil), (*int64)(nil), (*bool)(nil),
		time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC),
		int64(1),
	}

	db := &fakeDB{
		queryRow: &fakeRow{values: []any{int64(1)}}, // COUNT
		rows:     &fakeRows{data: [][]any{row}},
	}
	repo := &AssetRepo{c: &Client{db: db}}

	assets, total, err := repo.ListWithFilters(ctx, "", nil, 1, 20, filter.OrderByClause{SQL: ""})
	if err != nil {
		t.Fatalf("ListWithFilters() error: %v", err)
	}
	if total != 1 || len(assets) != 1 {
		t.Fatalf("expected 1 asset, got total=%d len=%d", total, len(assets))
	}

	a := assets[0]
	if a.LifecycleState != "delivered" {
		t.Errorf("LifecycleState: got %q, want %q", a.LifecycleState, "delivered")
	}
	if a.AssetType != "frame_set" {
		t.Errorf("AssetType: got %q, want %q", a.AssetType, "frame_set")
	}
	if a.DurationMs != 5000 {
		t.Errorf("DurationMs: got %d, want %d", a.DurationMs, 5000)
	}
	if a.DurationSec != 5.0 {
		t.Errorf("DurationSec: got %f, want %f", a.DurationSec, 5.0)
	}
	if a.SegType != "frame_set" {
		t.Errorf("SegType: got %q, want %q", a.SegType, "frame_set")
	}
	if a.Status != "approved" {
		t.Errorf("Status: got %q, want %q", a.Status, "approved")
	}
	if a.Owner != "dave" {
		t.Errorf("Owner: got %q, want %q", a.Owner, "dave")
	}
	if a.Reviewer != "carol" {
		t.Errorf("Reviewer: got %q, want %q", a.Reviewer, "carol")
	}
}

func TestListByMcapFile_PopulatesBothOldAndNewFields(t *testing.T) {
	ctx := context.Background()

	// List queries return 31 columns (includes metadata/files/algo/annot JSONB)
	row := []any{
		"lm-1", "m1", int64(100), int64(200), (*string)(nil),
		"rejected", "segment", int64(7500),
		"frank", "eve", int(0), (*time.Time)(nil), "",
		"cold", (*time.Time)(nil), "gs://s/data", "gs://s/thumb", int(1),
		(*string)(nil), (*string)(nil), (*string)(nil), (*string)(nil),
		[]byte(`{}`), []byte(`{}`), []byte(`{}`), []byte(`{}`),
		(*string)(nil), (*int64)(nil), (*bool)(nil),
		time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC),
		int64(1),
	}

	db := &fakeDB{rows: &fakeRows{data: [][]any{row}}}
	repo := &AssetRepo{c: &Client{db: db}}

	assets, err := repo.ListByMcapFile(ctx, "m1")
	if err != nil {
		t.Fatalf("ListByMcapFile() error: %v", err)
	}
	if len(assets) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(assets))
	}

	a := assets[0]
	if a.LifecycleState != "rejected" {
		t.Errorf("LifecycleState: got %q, want %q", a.LifecycleState, "rejected")
	}
	if a.AssetType != "segment" {
		t.Errorf("AssetType: got %q, want %q", a.AssetType, "segment")
	}
	if a.DurationMs != 7500 {
		t.Errorf("DurationMs: got %d, want %d", a.DurationMs, 7500)
	}
	if a.RetentionTier != "cold" {
		t.Errorf("RetentionTier: got %q, want %q", a.RetentionTier, "cold")
	}
	if a.StorageURI != "gs://s/data" {
		t.Errorf("StorageURI: got %q, want %q", a.StorageURI, "gs://s/data")
	}
	if a.ThumbURI != "gs://s/thumb" {
		t.Errorf("ThumbURI: got %q, want %q", a.ThumbURI, "gs://s/thumb")
	}
	if a.AssetLevel != 1 {
		t.Errorf("AssetLevel: got %d, want %d", a.AssetLevel, 1)
	}
	if a.DurationSec != 7.5 {
		t.Errorf("DurationSec: got %f, want %f", a.DurationSec, 7.5)
	}
	if a.SegType != "segment" {
		t.Errorf("SegType: got %q, want %q", a.SegType, "segment")
	}
	if a.Status != "rejected" {
		t.Errorf("Status: got %q, want %q", a.Status, "rejected")
	}
	if a.Owner != "frank" {
		t.Errorf("Owner: got %q, want %q", a.Owner, "frank")
	}
	if a.Reviewer != "eve" {
		t.Errorf("Reviewer: got %q, want %q", a.Reviewer, "eve")
	}
}

// ---------------------------------------------------------------------------
// AssetTagRepo tests
// ---------------------------------------------------------------------------

func TestAssetTagRepoConstructor(t *testing.T) {
	c := &Client{db: &fakeDB{}}
	if NewAssetTagRepo(c) == nil {
		t.Fatalf("NewAssetTagRepo nil")
	}
}

func TestAssetTagRepo_Upsert_Success(t *testing.T) {
	ctx := context.Background()
	db := &fakeDB{}
	repo := &AssetTagRepo{c: &Client{db: db}}

	err := repo.Upsert(ctx, repository.AssetTagUpsertInput{AssetID: "a1", TagKey: "quality", TagValue: "good", TagType: "string", SourceType: "human", SourceName: "tester"})
	if err != nil {
		t.Fatalf("Upsert err: %v", err)
	}
}

func TestAssetTagRepo_Upsert_TagInsertError(t *testing.T) {
	ctx := context.Background()
	db := &fakeDB{execErr: errors.New("tag insert fail")}
	repo := &AssetTagRepo{c: &Client{db: db}}

	err := repo.Upsert(ctx, repository.AssetTagUpsertInput{AssetID: "a1", TagKey: "quality", TagValue: "good", TagType: "string", SourceType: "human", SourceName: "tester"})
	if err == nil || !strings.Contains(err.Error(), "asset_tags") {
		t.Fatalf("expected asset_tags error, got %v", err)
	}
}

// Property 8: Tag upsert writes to asset_tags table
func TestProperty8_TagUpsertConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		assetID := "prop8-" + rapid.StringMatching(`[a-f0-9]{8}`).Draw(t, "asset_id")
		tagKey := rapid.StringMatching(`[a-z_]{1,20}`).Draw(t, "tag_key")
		tagValue := rapid.StringMatching(`[a-zA-Z0-9_ ]{0,50}`).Draw(t, "tag_value")
		tagType := rapid.SampledFrom([]string{"string", "number", "bool"}).Draw(t, "tag_type")
		sourceType := rapid.SampledFrom([]string{"human", "system", "algo"}).Draw(t, "source_type")

		tracker := &execTracker{fakeDB: &fakeDB{}}
		repo := &AssetTagRepo{c: &Client{db: tracker}}

		err := repo.Upsert(context.Background(), repository.AssetTagUpsertInput{AssetID: assetID, TagKey: tagKey, TagValue: tagValue, TagType: tagType, SourceType: sourceType, SourceName: "tester"})
		if err != nil {
			t.Fatalf("Upsert failed: %v", err)
		}

		// Verify exactly 1 Exec call (asset_tags only).
		if len(tracker.calls) != 1 {
			t.Fatalf("expected 1 Exec call, got %d", len(tracker.calls))
		}

		call := tracker.calls[0]
		if len(call) < 5 {
			t.Fatalf("asset_tags call: expected ≥5 args, got %d", len(call))
		}
		if call[0] != assetID {
			t.Fatalf("asset_tags asset_id: got %v, want %v", call[0], assetID)
		}
		if call[1] != tagKey {
			t.Fatalf("asset_tags tag_key: got %v, want %v", call[1], tagKey)
		}
		if call[2] != tagValue {
			t.Fatalf("asset_tags tag_value: got %v, want %v", call[2], tagValue)
		}
	})
}

// execTracker wraps fakeDB and records all Exec call arguments.
type execTracker struct {
	*fakeDB
	calls [][]any
}

func (e *execTracker) Exec(ctx context.Context, sql string, args ...any) error {
	e.calls = append(e.calls, args)
	return e.fakeDB.Exec(ctx, sql, args...)
}

func (e *execTracker) ExecResult(ctx context.Context, sql string, args ...any) (int64, error) {
	return e.fakeDB.ExecResult(ctx, sql, args...)
}

func (e *execTracker) QueryRow(ctx context.Context, sql string, args ...any) rowScanner {
	return e.fakeDB.QueryRow(ctx, sql, args...)
}

func (e *execTracker) Query(ctx context.Context, sql string, args ...any) (rowsScanner, error) {
	return e.fakeDB.Query(ctx, sql, args...)
}

func (e *execTracker) Ping(ctx context.Context) error { return e.fakeDB.Ping(ctx) }
func (e *execTracker) Close()                         { e.fakeDB.Close() }

// ---------------------------------------------------------------------------
// AssetAlgoLatestRepo tests
// ---------------------------------------------------------------------------

func TestAssetAlgoLatestRepoConstructor(t *testing.T) {
	c := &Client{db: &fakeDB{}}
	if NewAssetAlgoLatestRepo(c) == nil {
		t.Fatalf("NewAssetAlgoLatestRepo nil")
	}
}

func TestAssetAlgoLatestRepo_Upsert_Success(t *testing.T) {
	ctx := context.Background()
	db := &fakeDB{}
	repo := &AssetAlgoLatestRepo{c: &Client{db: db}}

	err := repo.Upsert(ctx, &models.AssetAlgoLatest{
		AssetID: "a1", AlgoName: "hand_tracking", AlgoVersion: "1.2.0", Status: "running",
	})
	if err != nil {
		t.Fatalf("Upsert err: %v", err)
	}
}

func TestAssetAlgoLatestRepo_Upsert_AlgoInsertError(t *testing.T) {
	ctx := context.Background()
	db := &fakeDB{execErr: errors.New("algo insert fail")}
	repo := &AssetAlgoLatestRepo{c: &Client{db: db}}

	err := repo.Upsert(ctx, &models.AssetAlgoLatest{
		AssetID: "a1", AlgoName: "hand_tracking", AlgoVersion: "1.2.0", Status: "running",
	})
	if err == nil || !strings.Contains(err.Error(), "AssetAlgoLatestRepo.Upsert") {
		t.Fatalf("expected AssetAlgoLatestRepo.Upsert error, got %v", err)
	}
}

func TestAssetAlgoLatestRepo_Upsert_NilRow(t *testing.T) {
	repo := &AssetAlgoLatestRepo{c: &Client{db: &fakeDB{}}}
	if err := repo.Upsert(context.Background(), nil); err == nil {
		t.Fatal("expected error for nil row")
	}
}

// Property 9: Upsert writes the canonical positional arguments to
// asset_algo_latest, in column order.
func TestProperty9_AlgoUpsertConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		assetID := "prop9-" + rapid.StringMatching(`[a-f0-9]{8}`).Draw(t, "asset_id")
		algoName := rapid.StringMatching(`[a-z_]{1,15}`).Draw(t, "algo_name")
		algoVersion := rapid.StringMatching(`[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}`).Draw(t, "algo_version")
		status := rapid.SampledFrom([]string{"pending", "running", "ok", "failed", "blocked"}).Draw(t, "status")

		tracker := &execTracker{fakeDB: &fakeDB{}}
		repo := &AssetAlgoLatestRepo{c: &Client{db: tracker}}

		err := repo.Upsert(context.Background(), &models.AssetAlgoLatest{
			AssetID: assetID, AlgoName: algoName, AlgoVersion: algoVersion, Status: status,
		})
		if err != nil {
			t.Fatalf("Upsert failed: %v", err)
		}

		if len(tracker.calls) != 1 {
			t.Fatalf("expected 1 Exec call, got %d", len(tracker.calls))
		}

		call := tracker.calls[0]
		// args 1..4 are asset_id, algo_name, algo_version, status (1-indexed in SQL).
		if call[0] != assetID {
			t.Fatalf("asset_id: got %v, want %v", call[0], assetID)
		}
		if call[1] != algoName {
			t.Fatalf("algo_name: got %v, want %v", call[1], algoName)
		}
		if call[2] != algoVersion {
			t.Fatalf("algo_version: got %v, want %v", call[2], algoVersion)
		}
		if call[3] != status {
			t.Fatalf("status: got %v, want %v", call[3], status)
		}
	})
}

// ---------------------------------------------------------------------------
// MergeCfAlgo dual-write to asset_algo_latest
// ---------------------------------------------------------------------------

func TestMergeCfAlgo_DualWriteToAlgoLatest(t *testing.T) {
	ctx := context.Background()
	tracker := &execTracker{fakeDB: &fakeDB{execRowsAffected: 1}}
	repo := &AssetRepo{c: &Client{db: tracker}}

	algoKV := map[string]interface{}{
		"hand_tracking@1.2.0:status":     "running",
		"hand_tracking@1.2.0:started_at": "2026-04-21T00:00:00Z",
		"hand_tracking@1.2.0:method":     "gpu",
	}
	filesKV := map[string]interface{}{}

	newVer, err := repo.MergeCfAlgo(ctx, "a1", 5, algoKV, filesKV)
	if err != nil {
		t.Fatalf("MergeCfAlgo err: %v", err)
	}
	if newVer != 6 {
		t.Fatalf("expected version 6, got %d", newVer)
	}

	// Should have 1 Exec call for asset_algo_latest upsert (for :status key).
	if len(tracker.calls) != 1 {
		t.Fatalf("expected 1 Exec call for asset_algo_latest, got %d", len(tracker.calls))
	}

	call := tracker.calls[0]
	if len(call) < 4 {
		t.Fatalf("expected ≥4 args, got %d", len(call))
	}
	if call[0] != "a1" {
		t.Fatalf("asset_id: got %v, want a1", call[0])
	}
	if call[1] != "hand_tracking" {
		t.Fatalf("algo_name: got %v, want hand_tracking", call[1])
	}
	if call[2] != "1.2.0" {
		t.Fatalf("algo_version: got %v, want 1.2.0", call[2])
	}
	if call[3] != "running" {
		t.Fatalf("status: got %v, want running", call[3])
	}
}

func TestMergeCfAlgo_NoStatusKey_NoAlgoLatestWrite(t *testing.T) {
	ctx := context.Background()
	tracker := &execTracker{fakeDB: &fakeDB{execRowsAffected: 1}}
	repo := &AssetRepo{c: &Client{db: tracker}}

	algoKV := map[string]interface{}{
		"hand_tracking@1.2.0:started_at": "2026-04-21T00:00:00Z",
		"hand_tracking@1.2.0:method":     "gpu",
	}
	filesKV := map[string]interface{}{}

	_, err := repo.MergeCfAlgo(ctx, "a1", 5, algoKV, filesKV)
	if err != nil {
		t.Fatalf("MergeCfAlgo err: %v", err)
	}

	if len(tracker.calls) != 0 {
		t.Fatalf("expected 0 Exec calls (no :status key), got %d", len(tracker.calls))
	}
}

// ---------------------------------------------------------------------------
// parseAlgoKVKey tests
// ---------------------------------------------------------------------------

func TestParseAlgoKVKey(t *testing.T) {
	cases := []struct {
		key         string
		wantName    string
		wantVersion string
		wantField   string
	}{
		{"hand_tracking@1.2.0:status", "hand_tracking", "1.2.0", "status"},
		{"deface@2.0:started_at", "deface", "2.0", "started_at"},
		{"algo@v1:output_uri", "algo", "v1", "output_uri"},
		{"bad_key", "", "", ""},
		{"no_at:field", "", "", ""},
		{"@version:field", "", "", ""},
		{"name@:field", "", "", ""},
		{"name@ver:", "", "", ""},
		{":field", "", "", ""},
		{"", "", "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			name, ver, field := parseAlgoKVKey(tc.key)
			if name != tc.wantName || ver != tc.wantVersion || field != tc.wantField {
				t.Errorf("parseAlgoKVKey(%q) = (%q, %q, %q), want (%q, %q, %q)",
					tc.key, name, ver, field, tc.wantName, tc.wantVersion, tc.wantField)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AssetEventRepo tests
// ---------------------------------------------------------------------------

func TestAssetEventRepoConstructor(t *testing.T) {
	c := &Client{db: &fakeDB{}}
	if NewAssetEventRepo(c) == nil {
		t.Fatalf("NewAssetEventRepo nil")
	}
}

func TestAssetEventRepo_ListBetweenSeq_Scan(t *testing.T) {
	tm := mustTime(t, "2024-01-01T00:00:00Z")
	payload := []byte(`{}`)
	rows := &fakeRows{data: [][]any{{
		"evt-1", int64(2), "x", "asset", "v1",
		"a1", "m1", "t1", "p1",
		"backend", "published", payload,
		0, "",
		tm, tm, nil,
	}}}
	repo := &AssetEventRepo{c: &Client{db: &fakeDB{rows: rows}}}
	out, err := repo.ListBetweenSeq(context.Background(), 1, 10, 100)
	if err != nil {
		t.Fatalf("ListBetweenSeq: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len=%d want 1", len(out))
	}
	if out[0].EventSeq != 2 || out[0].EventID != "evt-1" {
		t.Fatalf("unexpected row: %+v", out[0])
	}
}

func TestAssetEventRepo_ListBetweenSeq_QueryError(t *testing.T) {
	repo := &AssetEventRepo{c: &Client{db: &fakeDB{queryErr: errors.New("boom")}}}
	_, err := repo.ListBetweenSeq(context.Background(), 0, 10, 10)
	if err == nil || !strings.Contains(err.Error(), "ListBetweenSeq") {
		t.Fatalf("expected ListBetweenSeq error, got %v", err)
	}
}

func TestAssetEventRepo_Append_Success(t *testing.T) {
	ctx := context.Background()
	db := &fakeDB{}
	repo := &AssetEventRepo{c: &Client{db: db}}

	err := repo.Append(ctx, repository.AssetEventAppendInput{
		EventType: "asset_created", AssetID: "a1",
		EventPayload: []byte(`{"action":"create"}`),
	})
	if err != nil {
		t.Fatalf("Append err: %v", err)
	}
}

func TestAssetEventRepo_Append_RequiresEventType(t *testing.T) {
	repo := &AssetEventRepo{c: &Client{db: &fakeDB{}}}
	if err := repo.Append(context.Background(), repository.AssetEventAppendInput{AssetID: "a1"}); err == nil {
		t.Fatal("expected error when event_type is empty")
	}
}

func TestAssetEventRepo_Append_NilPayload(t *testing.T) {
	ctx := context.Background()
	tracker := &execTracker{fakeDB: &fakeDB{}}
	repo := &AssetEventRepo{c: &Client{db: tracker}}

	err := repo.Append(ctx, repository.AssetEventAppendInput{
		EventType: "asset_updated", AssetID: "a1", McapFileID: "m1",
		EventPayload: nil,
	})
	if err != nil {
		t.Fatalf("Append err: %v", err)
	}
	if len(tracker.calls) != 1 {
		t.Fatalf("expected 1 Exec call, got %d", len(tracker.calls))
	}
	// SQL positional: $1=event_type, $2=aggregate_type, $3=schema,
	// $4=asset_id, $5=mcap_file_id, $6=tenant_id, $7=project_id,
	// $8=event_source, $9=actor_type, $10=actor_id, $11=request_id,
	// $12=idempotency_key, $13=run_id, $14=event_payload.
	payloadArg, ok := tracker.calls[0][13].([]byte)
	if !ok {
		t.Fatalf("payload arg is not []byte: %T", tracker.calls[0][13])
	}
	if string(payloadArg) != "{}" {
		t.Fatalf("expected '{}' payload for nil input, got %q", string(payloadArg))
	}
}

func TestAssetEventRepo_Append_NullableIDs(t *testing.T) {
	ctx := context.Background()
	tracker := &execTracker{fakeDB: &fakeDB{}}
	repo := &AssetEventRepo{c: &Client{db: tracker}}

	err := repo.Append(ctx, repository.AssetEventAppendInput{
		EventType: "delivery_created", EventPayload: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("Append err: %v", err)
	}
	if len(tracker.calls) != 1 {
		t.Fatalf("expected 1 Exec call, got %d", len(tracker.calls))
	}
	// $4=asset_id, $5=mcap_file_id (after $1=event_type, $2=aggregate_type, $3=schema).
	if tracker.calls[0][3] != nil {
		t.Fatalf("expected nil assetID for empty string, got %v", tracker.calls[0][3])
	}
	if tracker.calls[0][4] != nil {
		t.Fatalf("expected nil mcapFileID for empty string, got %v", tracker.calls[0][4])
	}

	tracker.calls = nil
	err = repo.Append(ctx, repository.AssetEventAppendInput{
		EventType: "asset_updated", AssetID: "a1", McapFileID: "m1",
		EventPayload: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("Append err: %v", err)
	}
	if tracker.calls[0][3] != "a1" {
		t.Fatalf("expected assetID 'a1', got %v", tracker.calls[0][3])
	}
	if tracker.calls[0][4] != "m1" {
		t.Fatalf("expected mcapFileID 'm1', got %v", tracker.calls[0][4])
	}
}

func TestAssetEventRepo_Append_DBError(t *testing.T) {
	ctx := context.Background()
	db := &fakeDB{execErr: errors.New("insert fail")}
	repo := &AssetEventRepo{c: &Client{db: db}}

	err := repo.Append(ctx, repository.AssetEventAppendInput{
		EventType: "asset_created", AssetID: "a1", EventPayload: []byte(`{}`),
	})
	if err == nil || !strings.Contains(err.Error(), "AssetEventRepo.Append") {
		t.Fatalf("expected Append error, got %v", err)
	}
}

// Property 10: Mutation event invariant
func TestProperty10_MutationEventInvariant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		eventType := rapid.SampledFrom([]string{
			"asset_created", "asset_updated",
			"tag_upserted", "algo_updated",
			"mcap_file_created", "mcap_file_updated",
			"delivery_created", "delivery_updated",
			"asset_lifecycle_changed",
		}).Draw(t, "event_type")

		assetID := rapid.SampledFrom([]string{
			"",
			"a-" + rapid.StringMatching(`[a-f0-9]{8}`).Draw(t, "asset_id_suffix"),
		}).Draw(t, "asset_id")

		mcapFileID := rapid.SampledFrom([]string{
			"",
			"m-" + rapid.StringMatching(`[a-f0-9]{8}`).Draw(t, "mcap_id_suffix"),
		}).Draw(t, "mcap_file_id")

		payloadKey := rapid.StringMatching(`[a-z_]{1,10}`).Draw(t, "payload_key")
		payloadVal := rapid.StringMatching(`[a-zA-Z0-9]{0,20}`).Draw(t, "payload_val")
		payload, _ := json.Marshal(map[string]string{payloadKey: payloadVal})

		tracker := &execTracker{fakeDB: &fakeDB{}}
		repo := &AssetEventRepo{c: &Client{db: tracker}}

		err := repo.Append(context.Background(), repository.AssetEventAppendInput{
			EventType: eventType, AssetID: assetID, McapFileID: mcapFileID,
			EventPayload: payload,
		})
		if err != nil {
			t.Fatalf("Append failed: %v", err)
		}

		if len(tracker.calls) != 1 {
			t.Fatalf("expected 1 Exec call, got %d", len(tracker.calls))
		}

		call := tracker.calls[0]
		// $1=event_type, $2=aggregate_type, $3=schema_version,
		// $4=asset_id, $5=mcap_file_id, $6=tenant, $7=project,
		// $8=event_source, $9..$13 actor/run, $14=payload.
		if len(call) < 14 {
			t.Fatalf("expected 14 args, got %d", len(call))
		}
		if call[0] != eventType {
			t.Fatalf("event_type mismatch: got %v, want %v", call[0], eventType)
		}
		if call[1] != "asset" {
			t.Fatalf("aggregate_type: got %v, want asset", call[1])
		}
		if call[2] != "v1" {
			t.Fatalf("schema_version: got %v, want v1", call[2])
		}
		if assetID == "" {
			if call[3] != nil {
				t.Fatalf("expected nil assetID for empty input, got %v", call[3])
			}
		} else if call[3] != assetID {
			t.Fatalf("assetID mismatch: got %v, want %v", call[3], assetID)
		}
		if mcapFileID == "" {
			if call[4] != nil {
				t.Fatalf("expected nil mcapFileID for empty input, got %v", call[4])
			}
		} else if call[4] != mcapFileID {
			t.Fatalf("mcapFileID mismatch: got %v, want %v", call[4], mcapFileID)
		}
		if call[7] != "backend" {
			t.Fatalf("event_source: got %v, want backend", call[7])
		}
		payloadArg, ok := call[13].([]byte)
		if !ok {
			t.Fatalf("payload arg is not []byte: %T", call[13])
		}
		var parsed map[string]interface{}
		if err := json.Unmarshal(payloadArg, &parsed); err != nil {
			t.Fatalf("payload is not valid JSON: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// McapFileRepo tests
// ---------------------------------------------------------------------------

// buildMcapRow builds a fake row slice (31 columns) matching the SELECT in
// McapFileRepo.Get() and List().
func buildMcapRow(
	mcapFileID string,
	rawHashMD5 string,
	mcapURI string,
	sizeBytes int64,
	startNs int64,
	endNs int64,
	channelCount int,
	chunkCount int,
	ingestState string,
	owner string,
	processState []byte,
) []any {
	return []any{
		mcapFileID, rawHashMD5, (*string)(nil), // raw_hash_sha256
		mcapURI, sizeBytes, (*int64)(nil), // file_duration_ms
		startNs, endNs,
		channelCount, chunkCount, ingestState, owner,
		"", "", "", "", // vendor_id, collector_id, task_id, device_id
		"", "", "", "", "", "", // camera_model, data_source, location_id, scene_id, environment_id, collection_method
		(*string)(nil), (*time.Time)(nil), (*string)(nil), (*string)(nil), // retention_tier, expire_at, tenant_id, project_id
		[]byte(`{}`), processState, // metadata, process_state
		time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC),
		int64(1),
	}
}

func TestMcapFileRepo_Get_PopulatesRealColumns(t *testing.T) {
	ctx := context.Background()

	row := buildMcapRow(
		"m-get-1", "md5hash",
		"gs://bucket/file.mcap", int64(1024), int64(100), int64(200),
		5, 10, "summarized", "alice",
		[]byte(`{"tracker":"done","deface":"running"}`),
	)

	db := &fakeDB{queryRow: &fakeRow{values: row}}
	repo := &McapFileRepo{c: &Client{db: db}}

	got, err := repo.Get(ctx, "m-get-1")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got == nil {
		t.Fatalf("Get() returned nil")
	}

	if got.McapFileID != "m-get-1" {
		t.Errorf("McapFileID: got %q, want %q", got.McapFileID, "m-get-1")
	}
	if got.GCSPath != "gs://bucket/file.mcap" {
		t.Errorf("GCSPath: got %q, want %q", got.GCSPath, "gs://bucket/file.mcap")
	}
	if got.SizeBytes != 1024 {
		t.Errorf("SizeBytes: got %d, want %d", got.SizeBytes, 1024)
	}
	if got.StartTimestampNs != 100 {
		t.Errorf("StartTimestampNs: got %d, want %d", got.StartTimestampNs, 100)
	}
	if got.EndTimestampNs != 200 {
		t.Errorf("EndTimestampNs: got %d, want %d", got.EndTimestampNs, 200)
	}
	if got.ChannelCount != 5 {
		t.Errorf("ChannelCount: got %d, want %d", got.ChannelCount, 5)
	}
	if got.ChunkCount != 10 {
		t.Errorf("ChunkCount: got %d, want %d", got.ChunkCount, 10)
	}
	if got.IngestState != models.IngestStateSummarized {
		t.Errorf("IngestState: got %q, want %q", got.IngestState, models.IngestStateSummarized)
	}
	if got.Owner != "alice" {
		t.Errorf("Owner: got %q, want %q", got.Owner, "alice")
	}
	if got.ProcessState["tracker"] != "done" || got.ProcessState["deface"] != "running" {
		t.Errorf("ProcessState: got %v, want tracker=done,deface=running", got.ProcessState)
	}
}

func TestMcapFileRepo_Set_WritesRealColumns(t *testing.T) {
	ctx := context.Background()

	tracker := &execTracker{fakeDB: &fakeDB{}}
	repo := &McapFileRepo{c: &Client{db: tracker}}

	f := &models.McapFile{
		McapFileID:       "m-dw-1",
		GCSPath:          "gs://bucket/test.mcap",
		SizeBytes:        2048,
		RawHashMD5:       "abc123",
		IngestState:      models.IngestStatePending,
		StartTimestampNs: 1000,
		EndTimestampNs:   2000,
		ChannelCount:     8,
		ChunkCount:       16,
		Owner:            "bob",
		ProcessState:     map[string]string{"algo1": "done", "algo2": "pending"},
	}

	err := repo.Set(ctx, f)
	if err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	if f.Version != 1 {
		t.Errorf("Version: got %d, want 1", f.Version)
	}
	if f.CreatedAt.IsZero() {
		t.Errorf("CreatedAt should be set")
	}

	if len(tracker.calls) != 1 {
		t.Fatalf("expected 1 Exec call, got %d", len(tracker.calls))
	}

	args := tracker.calls[0]
	// Set() passes 31 args (all real columns including provenance/retention/metadata)
	if len(args) != 31 {
		t.Fatalf("expected 31 args, got %d", len(args))
	}

	if args[0] != "m-dw-1" {
		t.Errorf("mcap_file_id: got %v, want m-dw-1", args[0])
	}
	if args[3] != "gs://bucket/test.mcap" {
		t.Errorf("mcap_uri: got %v, want gs://bucket/test.mcap", args[3])
	}
	if args[4] != int64(2048) {
		t.Errorf("size_bytes: got %v, want 2048", args[4])
	}
	if args[6] != int64(1000) {
		t.Errorf("start_timestamp_ns: got %v, want 1000", args[6])
	}
	if args[7] != int64(2000) {
		t.Errorf("end_timestamp_ns: got %v, want 2000", args[7])
	}
	if args[8] != int(8) {
		t.Errorf("channel_count: got %v, want 8", args[8])
	}
	if args[9] != int(16) {
		t.Errorf("chunk_count: got %v, want 16", args[9])
	}
	if args[10] != "pending" {
		t.Errorf("ingest_state: got %v, want pending", args[10])
	}
	if args[11] != "bob" {
		t.Errorf("owner: got %v, want bob", args[11])
	}

	// Verify process_state JSONB (arg index 27 in the new layout).
	psRaw, ok := args[27].([]byte)
	if !ok {
		t.Fatalf("process_state arg is not []byte: %T", args[27])
	}
	var ps map[string]string
	if err := json.Unmarshal(psRaw, &ps); err != nil {
		t.Fatalf("unmarshal process_state: %v", err)
	}
	if ps["algo1"] != "done" || ps["algo2"] != "pending" {
		t.Errorf("process_state: got %v, want algo1=done,algo2=pending", ps)
	}
}

func TestMcapFileRepo_List_ReadsRealColumns(t *testing.T) {
	ctx := context.Background()

	row := buildMcapRow(
		"m-list-1", "md5a",
		"gs://b/a.mcap", int64(512), int64(50), int64(150),
		3, 7, "pending", "carol",
		[]byte(`{"algo_x":"ok"}`),
	)

	db := &fakeDB{
		queryRow: &fakeRow{values: []any{int64(1)}}, // COUNT
		rows:     &fakeRows{data: [][]any{row}},
	}
	repo := &McapFileRepo{c: &Client{db: db}}

	files, total, err := repo.List(ctx, 1, 20, "", "")
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 1 || len(files) != 1 {
		t.Fatalf("expected 1 file, got total=%d len=%d", total, len(files))
	}

	f := files[0]
	if f.McapFileID != "m-list-1" {
		t.Errorf("McapFileID: got %q, want %q", f.McapFileID, "m-list-1")
	}
	if f.GCSPath != "gs://b/a.mcap" {
		t.Errorf("GCSPath: got %q, want %q", f.GCSPath, "gs://b/a.mcap")
	}
	if f.SizeBytes != 512 {
		t.Errorf("SizeBytes: got %d, want %d", f.SizeBytes, 512)
	}
	if f.IngestState != models.IngestStatePending {
		t.Errorf("IngestState: got %q, want %q", f.IngestState, models.IngestStatePending)
	}
	if f.Owner != "carol" {
		t.Errorf("Owner: got %q, want %q", f.Owner, "carol")
	}
	if f.ProcessState["algo_x"] != "ok" {
		t.Errorf("ProcessState: got %v, want algo_x=ok", f.ProcessState)
	}
}

func TestMcapFileRepo_List_FiltersByRealColumns(t *testing.T) {
	ctx := context.Background()

	row := buildMcapRow(
		"m-filt-1", "md5b",
		"gs://b/b.mcap", int64(256), int64(0), int64(0),
		1, 2, "summarized", "dave",
		[]byte(`{}`),
	)

	db := &fakeDB{
		queryRow: &fakeRow{values: []any{int64(1)}},
		rows:     &fakeRows{data: [][]any{row}},
	}
	repo := &McapFileRepo{c: &Client{db: db}}

	files, total, err := repo.List(ctx, 1, 20, "summarized", "dave")
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 1 || len(files) != 1 {
		t.Fatalf("expected 1 file, got total=%d len=%d", total, len(files))
	}
	if files[0].IngestState != models.IngestStateSummarized {
		t.Errorf("IngestState: got %q, want %q", files[0].IngestState, models.IngestStateSummarized)
	}
}

func TestMcapFileRepo_List_CountError(t *testing.T) {
	ctx := context.Background()
	db := &fakeDB{queryRow: &fakeRow{err: errors.New("count fail")}}
	repo := &McapFileRepo{c: &Client{db: db}}

	_, _, err := repo.List(ctx, 1, 20, "", "")
	if err == nil || !strings.Contains(err.Error(), "count") {
		t.Fatalf("expected count error, got %v", err)
	}
}

func TestMcapFileRepo_List_QueryError(t *testing.T) {
	ctx := context.Background()
	db := &fakeDB{
		queryRow: &fakeRow{values: []any{int64(1)}},
		queryErr: errors.New("query fail"),
	}
	repo := &McapFileRepo{c: &Client{db: db}}

	_, _, err := repo.List(ctx, 1, 20, "", "")
	if err == nil || !strings.Contains(err.Error(), "query") {
		t.Fatalf("expected query error, got %v", err)
	}
}

func TestMcapFileRepo_List_ScanError(t *testing.T) {
	ctx := context.Background()
	db := &fakeDB{
		queryRow: &fakeRow{values: []any{int64(1)}},
		rows:     &fakeRows{data: [][]any{{"m1"}}}, // too few columns
	}
	repo := &McapFileRepo{c: &Client{db: db}}

	_, _, err := repo.List(ctx, 1, 20, "", "")
	if err == nil || !strings.Contains(err.Error(), "scan") {
		t.Fatalf("expected scan error, got %v", err)
	}
}

func TestMcapFileRepo_UpdateIngestState(t *testing.T) {
	ctx := context.Background()
	tracker := &execTracker{fakeDB: &fakeDB{}}
	repo := &McapFileRepo{c: &Client{db: tracker}}

	err := repo.UpdateIngestState(ctx, "m-uis-1", models.IngestStateSummarized)
	if err != nil {
		t.Fatalf("UpdateIngestState() error: %v", err)
	}

	if len(tracker.calls) != 1 {
		t.Fatalf("expected 1 Exec call, got %d", len(tracker.calls))
	}

	args := tracker.calls[0]
	if len(args) < 2 {
		t.Fatalf("expected ≥2 args, got %d", len(args))
	}
	if args[0] != "m-uis-1" {
		t.Errorf("mcap_file_id: got %v, want m-uis-1", args[0])
	}
	if args[1] != "summarized" {
		t.Errorf("ingest_state: got %v, want summarized", args[1])
	}
}

// Property: McapFile real column consistency
func TestProperty_McapFileRealColumnConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		gcsPath := "gs://" + rapid.StringMatching(`[a-z]{1,10}/[a-z]{1,10}\.mcap`).Draw(t, "gcs_path")
		sizeBytes := rapid.Int64Range(0, 10_000_000).Draw(t, "size_bytes")
		ingestState := rapid.SampledFrom([]models.IngestState{
			models.IngestStatePending, models.IngestStateSummarized, models.IngestStateFailed,
		}).Draw(t, "ingest_state")
		startNs := rapid.Int64Range(0, 1_000_000_000).Draw(t, "start_ns")
		endNs := rapid.Int64Range(startNs, startNs+1_000_000_000).Draw(t, "end_ns")
		channelCount := rapid.IntRange(0, 100).Draw(t, "channel_count")
		chunkCount := rapid.IntRange(0, 1000).Draw(t, "chunk_count")
		owner := rapid.StringMatching(`[a-z]{0,10}`).Draw(t, "owner")

		f := &models.McapFile{
			McapFileID:       "prop-mcap-" + rapid.StringMatching(`[a-f0-9]{8}`).Draw(t, "id"),
			GCSPath:          gcsPath,
			SizeBytes:        sizeBytes,
			RawHashMD5:       "hash-" + rapid.StringMatching(`[a-f0-9]{8}`).Draw(t, "hash"),
			IngestState:      ingestState,
			StartTimestampNs: startNs,
			EndTimestampNs:   endNs,
			ChannelCount:     channelCount,
			ChunkCount:       chunkCount,
			Owner:            owner,
			ProcessState:     map[string]string{"algo": "ok"},
		}

		tracker := &execTracker{fakeDB: &fakeDB{}}
		repo := &McapFileRepo{c: &Client{db: tracker}}

		err := repo.Set(context.Background(), f)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		if len(tracker.calls) != 1 {
			t.Fatalf("expected 1 Exec call, got %d", len(tracker.calls))
		}

		args := tracker.calls[0]
		if len(args) != 31 {
			t.Fatalf("expected 31 args, got %d", len(args))
		}

		// Verify real column values.
		if args[3] != gcsPath {
			t.Fatalf("mcap_uri mismatch: real=%v want=%v", args[3], gcsPath)
		}
		if args[4] != sizeBytes {
			t.Fatalf("size_bytes mismatch: real=%v want=%v", args[4], sizeBytes)
		}
		if args[10] != string(ingestState) {
			t.Fatalf("ingest_state mismatch: real=%v want=%v", args[10], ingestState)
		}
		if args[11] != owner {
			t.Fatalf("owner mismatch: real=%v want=%v", args[11], owner)
		}
		if args[6] != startNs {
			t.Fatalf("start_timestamp_ns mismatch: real=%v want=%v", args[6], startNs)
		}
		if args[7] != endNs {
			t.Fatalf("end_timestamp_ns mismatch: real=%v want=%v", args[7], endNs)
		}
		if args[8] != channelCount {
			t.Fatalf("channel_count mismatch: real=%v want=%v", args[8], channelCount)
		}
		if args[9] != chunkCount {
			t.Fatalf("chunk_count mismatch: real=%v want=%v", args[9], chunkCount)
		}
	})
}

// ---------------------------------------------------------------------------
// DeliveryRepo tests
// ---------------------------------------------------------------------------

// buildDeliveryRow builds a fake row slice (20 columns) matching the SELECT in
// DeliveryRepo.Get() and List().
func buildDeliveryRow(
	deliveryID string,
	customerID string,
	status string,
	deliveredAt *time.Time,
	contractID string,
	deliveryType string,
	requestedBy string,
	approvedBy string,
	deliveredBy string,
	manifestURI string,
	replayManifestURI string,
	itemCount int64,
	totalSizeBytes *int64,
	completedAt *time.Time,
	metadataJSON []byte,
	tenantID *string,
	projectID *string,
) []any {
	return []any{
		deliveryID, customerID, status, deliveredAt,
		contractID, deliveryType, requestedBy, approvedBy, deliveredBy,
		manifestURI, replayManifestURI, itemCount, totalSizeBytes,
		completedAt, metadataJSON, tenantID, projectID,
		time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC),
		int64(1),
	}
}

func TestDeliveryRepo_Get_PopulatesNewColumns(t *testing.T) {
	ctx := context.Background()

	at := mustTime(t, "2026-04-21T00:00:00Z")
	completedAt := mustTime(t, "2026-04-22T00:00:00Z")
	totalSize := int64(1024000)
	tenant := "tenant-1"
	project := "proj-1"

	row := buildDeliveryRow(
		"d-get-1", "c1", "delivered", &at,
		"ct-100", "replay", "alice", "bob", "carol",
		"gs://manifest", "gs://replay-manifest", int64(42), &totalSize,
		&completedAt, []byte(`{"key":"val","note":"test note"}`), &tenant, &project,
	)

	db := &fakeDB{queryRow: &fakeRow{values: row}}
	repo := &DeliveryRepo{c: &Client{db: db}}

	got, err := repo.Get(ctx, "d-get-1")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got == nil {
		t.Fatalf("Get() returned nil")
	}

	if got.ContractID != "ct-100" {
		t.Errorf("ContractID: got %q, want %q", got.ContractID, "ct-100")
	}
	if got.DeliveryType != "replay" {
		t.Errorf("DeliveryType: got %q, want %q", got.DeliveryType, "replay")
	}
	if got.RequestedBy != "alice" {
		t.Errorf("RequestedBy: got %q, want %q", got.RequestedBy, "alice")
	}
	if got.ApprovedBy != "bob" {
		t.Errorf("ApprovedBy: got %q, want %q", got.ApprovedBy, "bob")
	}
	if got.DeliveredBy != "carol" {
		t.Errorf("DeliveredBy: got %q, want %q", got.DeliveredBy, "carol")
	}
	if got.ManifestURI != "gs://manifest" {
		t.Errorf("ManifestURI: got %q, want %q", got.ManifestURI, "gs://manifest")
	}
	if got.ReplayManifestURI != "gs://replay-manifest" {
		t.Errorf("ReplayManifestURI: got %q, want %q", got.ReplayManifestURI, "gs://replay-manifest")
	}
	if got.ItemCount != 42 {
		t.Errorf("ItemCount: got %d, want %d", got.ItemCount, 42)
	}
	if got.TotalSizeBytes == nil || *got.TotalSizeBytes != 1024000 {
		t.Errorf("TotalSizeBytes: got %v, want 1024000", got.TotalSizeBytes)
	}
	if got.CompletedAt == nil || !got.CompletedAt.Equal(completedAt) {
		t.Errorf("CompletedAt: got %v, want %v", got.CompletedAt, completedAt)
	}
	if got.TenantID != "tenant-1" {
		t.Errorf("TenantID: got %q, want %q", got.TenantID, "tenant-1")
	}
	if got.ProjectID != "proj-1" {
		t.Errorf("ProjectID: got %q, want %q", got.ProjectID, "proj-1")
	}
	if got.Note != "test note" {
		t.Errorf("Note: got %q, want %q", got.Note, "test note")
	}
	if got.Owner != "carol" {
		t.Errorf("Owner: got %q, want %q (synced from DeliveredBy)", got.Owner, "carol")
	}
	if got.AssetCount != 42 {
		t.Errorf("AssetCount: got %d, want %d (synced from ItemCount)", got.AssetCount, 42)
	}
}

func TestDeliveryRepo_Set_WritesRealColumns(t *testing.T) {
	ctx := context.Background()

	tracker := &execTracker{fakeDB: &fakeDB{}}
	repo := &DeliveryRepo{c: &Client{db: tracker}}

	d := &models.Delivery{
		DeliveryID:        "d-dw-1",
		CustomerID:        "c1",
		Status:            models.DeliveryStatusDelivered,
		ContractID:        "ct-200",
		DeliveryType:      "asset_set",
		RequestedBy:       "alice",
		ApprovedBy:        "bob",
		Owner:             "carol",
		ManifestURI:       "gs://manifest/batch1",
		ReplayManifestURI: "gs://replay/batch1",
		ItemCount:         10,
		Note:              "batch delivery",
		TenantID:          "t1",
		ProjectID:         "p1",
	}

	err := repo.Set(ctx, d)
	if err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	if d.Version != 1 {
		t.Errorf("Version: got %d, want 1", d.Version)
	}
	if d.CreatedAt.IsZero() {
		t.Errorf("CreatedAt should be set")
	}
	if d.DeliveredBy != "carol" {
		t.Errorf("DeliveredBy should be synced from Owner: got %q, want %q", d.DeliveredBy, "carol")
	}
	if d.AssetCount != 10 {
		t.Errorf("AssetCount should be synced from ItemCount: got %d, want %d", d.AssetCount, 10)
	}

	if len(tracker.calls) != 1 {
		t.Fatalf("expected 1 Exec call, got %d", len(tracker.calls))
	}

	args := tracker.calls[0]
	// Set() passes 20 args.
	if len(args) != 20 {
		t.Fatalf("expected 20 args, got %d", len(args))
	}

	if args[0] != "d-dw-1" {
		t.Errorf("delivery_id: got %v, want d-dw-1", args[0])
	}
	if args[4] != "ct-200" {
		t.Errorf("contract_id: got %v, want ct-200", args[4])
	}
	if args[5] != "asset_set" {
		t.Errorf("delivery_type: got %v, want asset_set", args[5])
	}
	if args[6] != "alice" {
		t.Errorf("requested_by: got %v, want alice", args[6])
	}
	if args[7] != "bob" {
		t.Errorf("approved_by: got %v, want bob", args[7])
	}
	if args[8] != "carol" {
		t.Errorf("delivered_by: got %v, want carol", args[8])
	}
	if args[9] != "gs://manifest/batch1" {
		t.Errorf("manifest_uri: got %v, want gs://manifest/batch1", args[9])
	}
	if args[11] != int64(10) {
		t.Errorf("item_count: got %v, want 10", args[11])
	}
	metadata, ok := args[14].([]byte)
	if !ok {
		t.Fatalf("metadata arg type: got %T want []byte", args[14])
	}
	var meta map[string]any
	if err := json.Unmarshal(metadata, &meta); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if got := meta["note"]; got != "batch delivery" {
		t.Fatalf("metadata.note: got %v want %q", got, "batch delivery")
	}
}

func TestDeliveryRepo_List_ReadsNewColumns(t *testing.T) {
	ctx := context.Background()

	at := mustTime(t, "2026-04-21T00:00:00Z")
	row := buildDeliveryRow(
		"d-list-1", "c1", "pending", &at,
		"ct-300", "asset_set", "req-user", "appr-user", "del-user",
		"gs://m/list", "", int64(5), (*int64)(nil),
		(*time.Time)(nil), []byte(`{"note":"list note"}`), (*string)(nil), (*string)(nil),
	)

	db := &fakeDB{
		queryRow: &fakeRow{values: []any{int64(1)}}, // COUNT
		rows:     &fakeRows{data: [][]any{row}},
	}
	repo := &DeliveryRepo{c: &Client{db: db}}

	deliveries, total, err := repo.List(ctx, 1, 20, "", "")
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 1 || len(deliveries) != 1 {
		t.Fatalf("expected 1 delivery, got total=%d len=%d", total, len(deliveries))
	}

	d := deliveries[0]
	if d.DeliveryID != "d-list-1" {
		t.Errorf("DeliveryID: got %q, want %q", d.DeliveryID, "d-list-1")
	}
	if d.ContractID != "ct-300" {
		t.Errorf("ContractID: got %q, want %q", d.ContractID, "ct-300")
	}
	if d.DeliveryType != "asset_set" {
		t.Errorf("DeliveryType: got %q, want %q", d.DeliveryType, "asset_set")
	}
	if d.ManifestURI != "gs://m/list" {
		t.Errorf("ManifestURI: got %q, want %q", d.ManifestURI, "gs://m/list")
	}
	if d.ItemCount != 5 {
		t.Errorf("ItemCount: got %d, want %d", d.ItemCount, 5)
	}
	if d.Note != "list note" {
		t.Errorf("Note: got %q, want %q", d.Note, "list note")
	}
	if d.Owner != "del-user" {
		t.Errorf("Owner: got %q, want %q (synced from DeliveredBy)", d.Owner, "del-user")
	}
}

func TestDeliveryRepo_AddItems_InsertsDeliveryItems(t *testing.T) {
	ctx := context.Background()
	db := &fakeDB{}
	repo := &DeliveryRepo{c: &Client{db: db}}

	if err := repo.AddItems(ctx, "d1", []string{"a1b2c3d4", "b2c3d4e5"}); err != nil {
		t.Fatalf("AddItems() error: %v", err)
	}
	if len(db.execSQLs) != 2 {
		t.Fatalf("expected two execs, got %d", len(db.execSQLs))
	}
	sql := db.execSQLs[0]
	if !strings.Contains(sql, "INSERT INTO delivery_items") {
		t.Fatalf("expected AddItems SQL to insert into delivery_items")
	}
	if len(db.execArgs) != 2 || len(db.execArgs[0]) != 2 {
		t.Fatalf("expected 2 args per insert, got %v", db.execArgs)
	}
	if got := db.execArgs[0][0]; got != "d1" {
		t.Fatalf("expected delivery_id arg d1, got %v", got)
	}
	if got := db.execArgs[0][1]; got != "a1b2c3d4" {
		t.Fatalf("expected asset_id arg a1b2c3d4, got %v", got)
	}
}

func TestDeliveryRepo_RefreshAssetDeliveryIndex_RecomputesCounters(t *testing.T) {
	ctx := context.Background()
	db := &fakeDB{}
	repo := &DeliveryRepo{c: &Client{db: db}}

	if err := repo.RefreshAssetDeliveryIndex(ctx, "a1b2c3d4"); err != nil {
		t.Fatalf("RefreshAssetDeliveryIndex() error: %v", err)
	}
	if len(db.execSQLs) != 1 {
		t.Fatalf("expected exactly one exec, got %d", len(db.execSQLs))
	}
	sql := db.execSQLs[0]
	if !strings.Contains(sql, "FROM delivery_items di") || !strings.Contains(sql, "delivery_count = COALESCE") {
		t.Fatalf("expected refresh SQL to recompute delivery counters, got %s", sql)
	}
	if len(db.execArgs) != 1 || len(db.execArgs[0]) != 1 {
		t.Fatalf("expected one arg, got %v", db.execArgs)
	}
	if got := db.execArgs[0][0]; got != "a1b2c3d4" {
		t.Fatalf("expected asset_id arg a1b2c3d4, got %v", got)
	}
}
