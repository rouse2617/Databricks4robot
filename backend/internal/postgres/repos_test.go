package postgres

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"data-platform/internal/config"
	"data-platform/internal/models"
	"data-platform/internal/repository"
)

type fakeDB struct {
	queryRow rowScanner
	queryErr error
	rows     rowsScanner
	execErr  error
	pingErr  error
	closed   bool
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
func (f *fakeDB) Exec(_ context.Context, _ string, _ ...any) error { return f.execErr }
func (f *fakeDB) Ping(_ context.Context) error                     { return f.pingErr }
func (f *fakeDB) Close()                                           { f.closed = true }

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
	assertPanic(t, func() { _ = r.Ping(context.Background()) })
	assertPanic(t, func() { r.Close() })
}

func TestAssetJSONHelpers(t *testing.T) {
	now := mustTime(t, "2026-04-21T00:00:00Z")
	a := &models.Asset{
		EndTimestampNs:  20,
		DurationSec:     1.5,
		Reviewer:        "r",
		Owner:           "o",
		SegType:         "task",
		Env:             "k",
		Task:            "cook",
		DeliveryCount:   2,
		LastDeliveredTo: "c",
		LastDeliveredAt: &now,
		AlgoResults:     map[string]string{"k": "v"},
		Tags:            map[string]string{"t": "x"},
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	meta, algo, tag := assetJSON(a)
	got := &models.Asset{}
	applyAssetJSON(got, meta, algo, tag)
	if got.EndTimestampNs != 20 || got.DurationSec != 1.5 || got.Reviewer != "r" || got.Owner != "o" || got.AlgoResults["k"] != "v" || got.Tags["t"] != "x" {
		t.Fatalf("applyAssetJSON mismatch: %+v", got)
	}
	applyAssetJSON(&models.Asset{}, []byte(`{}`), []byte(`bad`), []byte(`bad`))
}

func TestMcapJSONHelpers(t *testing.T) {
	f := &models.McapFile{
		GCSPath:          "gs://x",
		SizeBytes:        10,
		IngestState:      models.IngestStatePending,
		StartTimestampNs: 1,
		EndTimestampNs:   2,
		ChannelCount:     3,
		ChunkCount:       4,
		Owner:            "o",
		ProcessState:     map[string]string{"p": "done"},
	}
	meta, proc := mcapJSON(f)
	got := &models.McapFile{}
	applyMcapJSON(got, meta, proc)
	if got.GCSPath != "gs://x" || got.SizeBytes != 10 || got.ProcessState["p"] != "done" {
		t.Fatalf("applyMcapJSON mismatch: %+v", got)
	}
	applyMcapJSON(&models.McapFile{}, []byte(`{}`), []byte(`bad`))
}

func TestAssetRepo(t *testing.T) {
	ctx := context.Background()
	db := &fakeDB{}
	repo := &AssetRepo{c: &Client{db: db}}

	if got, err := repo.Get(ctx, "a1"); err != nil || got != nil {
		t.Fatalf("expected nil,nil not found")
	}
	db.queryRow = &fakeRow{err: errors.New("q")}
	if _, err := repo.Get(ctx, "a1"); err == nil {
		t.Fatalf("expected get error")
	}
	db.queryRow = &fakeRow{values: []any{
		"a1", "m1", int64(10), "approved",
		[]byte(`{"end_timestamp_ns":20,"duration_sec":1.2}`), []byte(`{"algo":"ok"}`), []byte(`{"tag":"v"}`),
		mustTime(t, "2026-04-20T00:00:00Z"), mustTime(t, "2026-04-21T00:00:00Z"), int64(1),
	}}
	got, err := repo.Get(ctx, "a1")
	if err != nil || got == nil || got.AssetID != "a1" {
		t.Fatalf("get success failed: %v %+v", err, got)
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

	if err := repo.SoftDelete(ctx, "a1"); err != nil {
		t.Fatalf("soft delete err: %v", err)
	}
	db.execErr = errors.New("e")
	if err := repo.SoftDelete(ctx, "a1"); err == nil {
		t.Fatalf("expected soft delete error")
	}
	db.execErr = nil

	rows := &fakeRows{data: [][]any{
		{"a1", "m1", int64(10), "approved", []byte(`{}`), []byte(`{}`), []byte(`{}`), mustTime(t, "2026-04-20T00:00:00Z"), mustTime(t, "2026-04-21T00:00:00Z"), int64(1)},
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
	db.queryRow = &fakeRow{values: []any{
		"m1", "md5", []byte(`{"gcs_path":"gs://x","size_bytes":10}`), []byte(`{"p":"done"}`),
		mustTime(t, "2026-04-20T00:00:00Z"), mustTime(t, "2026-04-21T00:00:00Z"), int64(1),
	}}
	got, err := repo.Get(ctx, "m1")
	if err != nil || got == nil || got.McapFileID != "m1" {
		t.Fatalf("get success failed")
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
	db.queryRow = &fakeRow{values: []any{
		"d1", "c1", "delivered", &at, []byte(`{"manifest_uri":"gs://m","contract_id":"ct-1","note":"ok","owner":"urn:owner","asset_count":1}`),
		mustTime(t, "2026-04-20T00:00:00Z"), mustTime(t, "2026-04-21T00:00:00Z"), int64(1),
	}}
	got, err := repo.Get(ctx, "d1")
	if err != nil || got == nil || got.DeliveryID != "d1" || got.ContractID != "ct-1" || got.Note != "ok" || got.Owner != "urn:owner" {
		t.Fatalf("get success failed")
	}

	if err := repo.WriteIndexes(ctx, "a1", &models.Delivery{DeliveryID: "d1"}); err != nil {
		t.Fatalf("write indexes err: %v", err)
	}
	db.execErr = errors.New("e")
	if err := repo.WriteIndexes(ctx, "a1", &models.Delivery{DeliveryID: "d1"}); err == nil {
		t.Fatalf("expected write indexes error")
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
