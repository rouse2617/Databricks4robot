package bigtable

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	gbt "cloud.google.com/go/bigtable"

	"data-platform/internal/models"
	"data-platform/internal/repository"
)

type fakeDataClient struct {
	tables      map[string]*fakeTable
	closeCalled bool
	closeErr    error
}

func (f *fakeDataClient) Open(name string) btTable {
	if f.tables == nil {
		f.tables = map[string]*fakeTable{}
	}
	if _, ok := f.tables[name]; !ok {
		f.tables[name] = &fakeTable{rows: map[string]gbt.Row{}}
	}
	return f.tables[name]
}

func (f *fakeDataClient) Close() error {
	f.closeCalled = true
	return f.closeErr
}

type fakeTable struct {
	rows        map[string]gbt.Row
	readRowErr  error
	applyErr    error
	readRowsErr error
	applied     []string
}

func (f *fakeTable) ReadRow(_ context.Context, row string, _ ...gbt.ReadOption) (gbt.Row, error) {
	if f.readRowErr != nil {
		return nil, f.readRowErr
	}
	if v, ok := f.rows[row]; ok {
		return v, nil
	}
	return nil, nil
}

func (f *fakeTable) Apply(_ context.Context, row string, _ *gbt.Mutation, _ ...gbt.ApplyOption) error {
	if f.applyErr != nil {
		return f.applyErr
	}
	f.applied = append(f.applied, row)
	return nil
}

func (f *fakeTable) ReadRows(_ context.Context, arg gbt.RowSet, fn func(gbt.Row) bool, _ ...gbt.ReadOption) error {
	if f.readRowsErr != nil {
		return f.readRowsErr
	}
	switch rs := arg.(type) {
	case gbt.RowList:
		for _, key := range rs {
			if row, ok := f.rows[key]; ok {
				if !fn(row) {
					return nil
				}
			}
		}
	default:
		for _, row := range f.rows {
			if !fn(row) {
				return nil
			}
		}
	}
	return nil
}

func mkRow(key string, cfs map[string]map[string][]byte) gbt.Row {
	row := gbt.Row{}
	for cf, quals := range cfs {
		items := make([]gbt.ReadItem, 0, len(quals))
		for qual, val := range quals {
			items = append(items, gbt.ReadItem{
				Row:    key,
				Column: cf + ":" + qual,
				Value:  val,
			})
		}
		row[cf] = items
	}
	return row
}

func mustParseRFC3339(t *testing.T, s string) time.Time {
	t.Helper()
	got, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t.Fatalf("parse time: %v", err)
	}
	return got
}

func TestClientNewTableClose(t *testing.T) {
	old := newDataClient
	t.Cleanup(func() { newDataClient = old })

	t.Run("success", func(t *testing.T) {
		fd := &fakeDataClient{}
		newDataClient = func(_ context.Context, _, _ string) (btDataClient, error) { return fd, nil }
		c, err := New(context.Background(), "p", "i")
		if err != nil {
			t.Fatalf("New err: %v", err)
		}
		_ = c.Table("assets")
		if err := c.Close(); err != nil {
			t.Fatalf("Close err: %v", err)
		}
		if !fd.closeCalled {
			t.Fatalf("expected close called")
		}
	})

	t.Run("error", func(t *testing.T) {
		newDataClient = func(_ context.Context, _, _ string) (btDataClient, error) {
			return nil, errors.New("boom")
		}
		_, err := New(context.Background(), "p", "i")
		if err == nil || !strings.Contains(err.Error(), "bigtable.New") {
			t.Fatalf("expected wrapped error, got: %v", err)
		}
	})
}

func TestRealWrappersPanicWithNil(t *testing.T) {
	assertPanic := func(t *testing.T, fn func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Fatalf("expected panic")
			}
		}()
		fn()
	}

	assertPanic(t, func() { (&realDataClient{}).Open("x") })
	assertPanic(t, func() { _ = (&realDataClient{}).Close() })
	rt := &realTable{}
	assertPanic(t, func() { _, _ = rt.ReadRow(context.Background(), "k") })
	assertPanic(t, func() { _ = rt.Apply(context.Background(), "k", gbt.NewMutation()) })
	assertPanic(t, func() { _ = rt.ReadRows(context.Background(), gbt.RowList{"k"}, func(gbt.Row) bool { return true }) })
}

func TestHelpers(t *testing.T) {
	if got := AssetKey("a"); got != "v1#a" {
		t.Fatalf("AssetKey: %s", got)
	}
	if got := McapFileKey("m"); got != "v1#m" {
		t.Fatalf("McapFileKey: %s", got)
	}
	if got := DeliveryKey("d"); got != "v1#d" {
		t.Fatalf("DeliveryKey: %s", got)
	}
	if got := IdxSegmentKey("m", 12, "a"); got != "m#00000000000000000012#a" {
		t.Fatalf("IdxSegmentKey: %s", got)
	}
	now := mustParseRFC3339(t, "2026-04-21T00:00:00Z")
	if !strings.HasPrefix(IdxAssetDeliveryKey("a", now, "d"), "a#") {
		t.Fatalf("IdxAssetDeliveryKey prefix mismatch")
	}
	if !strings.HasPrefix(IdxCustomerDeliveryKey("c", now, "d"), "c#") {
		t.Fatalf("IdxCustomerDeliveryKey prefix mismatch")
	}

	if got := ParseIdxSegmentAssetID("m#1#a"); got != "a" {
		t.Fatalf("ParseIdxSegmentAssetID: %s", got)
	}
	if got := ParseIdxSegmentAssetID("no-delimiter"); got != "" {
		t.Fatalf("expected empty")
	}
	if got := ParseIdxSegmentAssetID("bad#"); got != "" {
		t.Fatalf("expected empty")
	}

	v := int64(123456789)
	if got := UnpackInt64(PackInt64(v)); got != v {
		t.Fatalf("pack/unpack mismatch: %d", got)
	}
	if got := S(B("x")); got != "x" {
		t.Fatalf("B/S mismatch: %s", got)
	}
	ts := mustParseRFC3339(t, "2026-01-01T01:02:03Z")
	if got := ParseRFC3339(RFC3339(ts)); !got.Equal(ts.UTC()) {
		t.Fatalf("time mismatch: %v", got)
	}
	if got := ParseRFC3339([]byte("invalid")); !got.IsZero() {
		t.Fatalf("expected zero time")
	}

	if got := rowKeyID("v1#abc"); got != "abc" {
		t.Fatalf("rowKeyID mismatch: %s", got)
	}
	if got := rowKeyID("v1#"); got != "v1#" {
		t.Fatalf("rowKeyID short mismatch: %s", got)
	}
}

func TestRowConverters(t *testing.T) {
	row := mkRow("v1#a1", map[string]map[string][]byte{
		CFMeta: {
			"mcap_file_id":       B("m1"),
			"start_timestamp_ns": PackInt64(10),
			"end_timestamp_ns":   PackInt64(20),
			"duration_sec":       B("1.500000"),
			"reviewer":           B("r"),
			"status":             B("approved"),
			"owner":              B("o"),
			"type":               B("task_demo"),
			"env":                B("kitchen"),
			"task":               B("cook"),
			"delivery_count":     PackInt64(2),
			"last_delivered_to":  B("urn:c"),
			"last_delivered_at":  B("2026-04-21T00:00:00Z"),
			"created_at":         B("2026-04-20T00:00:00Z"),
			"updated_at":         B("2026-04-21T00:00:00Z"),
			"version":            PackInt64(3),
		},
		CFAlgo: {"sam2@1.0:status": B("ok")},
		CFTag:  {"priority": B("A")},
	})
	a := rowToAsset(row)
	if a.AssetID != "a1" || a.McapFileID != "m1" || a.DeliveryCount != 2 || a.AlgoResults["sam2@1.0:status"] != "ok" || a.Tags["priority"] != "A" {
		t.Fatalf("unexpected asset conversion: %+v", a)
	}

	rowM := mkRow("v1#m1", map[string]map[string][]byte{
		CFMeta: {
			"mcap_uri":           B("gs://x"),
			"gcs_path":           B("gs://legacy"),
			"size_bytes":         PackInt64(100),
			"raw_hash_md5":       B("abc"),
			"ingest_state":       B("pending"),
			"start_timestamp_ns": PackInt64(1),
			"end_timestamp_ns":   PackInt64(2),
			"channel_count":      PackInt64(3),
			"chunk_count":        PackInt64(4),
			"owner":              B("owner"),
			"created_at":         B("2026-04-20T00:00:00Z"),
			"updated_at":         B("2026-04-21T00:00:00Z"),
			"version":            PackInt64(5),
		},
		CFProcess: {"step": B("done")},
	})
	m := rowToMcapFile(rowM)
	if m.McapFileID != "m1" || m.GCSPath != "gs://x" || m.ProcessState["step"] == "" {
		t.Fatalf("unexpected mcap conversion: %+v", m)
	}
	rowMFallback := mkRow("v1#m2", map[string]map[string][]byte{
		CFMeta: {"gcs_path": B("gs://legacy-only")},
	})
	m2 := rowToMcapFile(rowMFallback)
	if m2.GCSPath != "gs://legacy-only" {
		t.Fatalf("expected fallback gcs_path, got: %s", m2.GCSPath)
	}

	rowD := mkRow("v1#d1", map[string]map[string][]byte{
		CFMeta: {
			"customer_id":  B("c1"),
			"status":       B("delivered"),
			"manifest_uri": B("gs://manifest"),
			"contract_id":  B("ct"),
			"note":         B("n"),
			"asset_count":  PackInt64(7),
			"owner":        B("owner"),
			"delivered_at": B("2026-04-21T00:00:00Z"),
			"created_at":   B("2026-04-20T00:00:00Z"),
			"updated_at":   B("2026-04-21T00:00:00Z"),
			"version":      PackInt64(8),
		},
	})
	d := rowToDelivery(rowD)
	if d.DeliveryID != "d1" || d.CustomerID != "c1" || d.AssetCount != 7 {
		t.Fatalf("unexpected delivery conversion: %+v", d)
	}
}

func TestAssetRepo(t *testing.T) {
	ctx := context.Background()
	mainT := &fakeTable{rows: map[string]gbt.Row{}}
	idxT := &fakeTable{rows: map[string]gbt.Row{}}
	repo := &AssetRepo{table: mainT, idxTable: idxT}

	_, err := repo.Get(ctx, "x")
	if err != nil {
		t.Fatalf("unexpected get err: %v", err)
	}
	mainT.readRowErr = errors.New("read")
	if _, err = repo.Get(ctx, "x"); err == nil {
		t.Fatalf("expected get err")
	}
	mainT.readRowErr = nil
	mainT.rows[AssetKey("a1")] = mkRow(AssetKey("a1"), map[string]map[string][]byte{CFMeta: {"mcap_file_id": B("m1")}})
	if got, err := repo.Get(ctx, "a1"); err != nil || got == nil {
		t.Fatalf("get success failed: %v", err)
	}

	lastDelivered := mustParseRFC3339(t, "2026-04-21T00:00:00Z")
	a := &models.Asset{
		AssetID:         "a1",
		McapFileID:      "m1",
		Status:          models.AssetStatusApproved,
		LastDeliveredAt: &lastDelivered,
		AlgoResults:     map[string]string{"sam2@1.2.0:status": "ok"},
		Tags:            map[string]string{"priority": "A"},
	}
	if err := repo.Set(ctx, a); err != nil {
		t.Fatalf("set err: %v", err)
	}
	if a.Version != 1 || a.CreatedAt.IsZero() || a.UpdatedAt.IsZero() {
		t.Fatalf("set metadata not updated")
	}
	mainT.applyErr = errors.New("apply")
	if err := repo.Set(ctx, &models.Asset{AssetID: "a2"}); err == nil {
		t.Fatalf("expected set err")
	}
	mainT.applyErr = nil

	if err := repo.WriteSegmentIndex(ctx, &models.Asset{AssetID: "a1", McapFileID: "m1", StartTimestampNs: 10}); err != nil {
		t.Fatalf("write idx err: %v", err)
	}
	idxT.applyErr = errors.New("idx")
	if err := repo.WriteSegmentIndex(ctx, &models.Asset{AssetID: "a1", McapFileID: "m1", StartTimestampNs: 10}); err == nil {
		t.Fatalf("expected write idx err")
	}
	idxT.applyErr = nil

	if err := repo.SoftDelete(ctx, "a1"); err != nil {
		t.Fatalf("soft delete err: %v", err)
	}
	mainT.applyErr = errors.New("delete")
	if err := repo.SoftDelete(ctx, "a1"); err == nil {
		t.Fatalf("expected soft delete err")
	}
	mainT.applyErr = nil

	idxT.rows["m1#00000000000000000010#a1"] = mkRow("m1#00000000000000000010#a1", map[string]map[string][]byte{CFRef: {"asset_id": B("a1")}})
	mainT.rows[AssetKey("a1")] = mkRow(AssetKey("a1"), map[string]map[string][]byte{CFMeta: {"mcap_file_id": B("m1")}})
	items, err := repo.ListByMcapFile(ctx, "m1")
	if err != nil || len(items) != 1 {
		t.Fatalf("list by mcap failed: err=%v len=%d", err, len(items))
	}
	idxT.readRowsErr = errors.New("idx read")
	if _, err := repo.ListByMcapFile(ctx, "m1"); err == nil {
		t.Fatalf("expected list by mcap err")
	}
	idxT.readRowsErr = nil

	if got, err := repo.GetBatch(ctx, nil); err != nil || got != nil {
		t.Fatalf("get batch empty mismatch")
	}
	mainT.readRowsErr = errors.New("batch read")
	if _, err := repo.GetBatch(ctx, []string{"a1"}); err == nil {
		t.Fatalf("expected get batch err")
	}
}

func TestMcapRepo(t *testing.T) {
	ctx := context.Background()
	tb := &fakeTable{rows: map[string]gbt.Row{}}
	repo := &McapFileRepo{table: tb}

	if _, err := repo.Get(ctx, "x"); err != nil {
		t.Fatalf("unexpected get err: %v", err)
	}
	tb.readRowErr = errors.New("read")
	if _, err := repo.Get(ctx, "x"); err == nil {
		t.Fatalf("expected get err")
	}
	tb.readRowErr = nil
	tb.rows[McapFileKey("m1")] = mkRow(McapFileKey("m1"), map[string]map[string][]byte{CFMeta: {"mcap_uri": B("gs://x")}})
	if got, err := repo.Get(ctx, "m1"); err != nil || got == nil {
		t.Fatalf("get success failed: %v", err)
	}

	m := &models.McapFile{
		McapFileID:   "m1",
		IngestState:  models.IngestStatePending,
		ProcessState: map[string]string{"hand_tracking": "completed"},
	}
	if err := repo.Set(ctx, m); err != nil {
		t.Fatalf("set err: %v", err)
	}
	if m.Version != 1 || m.CreatedAt.IsZero() || m.UpdatedAt.IsZero() {
		t.Fatalf("set metadata not updated")
	}
	tb.applyErr = errors.New("apply")
	if err := repo.Set(ctx, &models.McapFile{McapFileID: "m2"}); err == nil {
		t.Fatalf("expected set err")
	}
	tb.applyErr = nil

	if err := repo.UpdateIngestState(ctx, "m1", models.IngestStateSummarized); err != nil {
		t.Fatalf("update ingest err: %v", err)
	}
	tb.applyErr = errors.New("update")
	if err := repo.UpdateIngestState(ctx, "m1", models.IngestStateFailed); err == nil {
		t.Fatalf("expected update ingest err")
	}
}

func TestDeliveryRepo(t *testing.T) {
	ctx := context.Background()
	mainT := &fakeTable{rows: map[string]gbt.Row{}}
	idxA := &fakeTable{rows: map[string]gbt.Row{}}
	idxC := &fakeTable{rows: map[string]gbt.Row{}}
	repo := &DeliveryRepo{table: mainT, idxAsset: idxA, idxCustomer: idxC}

	if _, err := repo.Get(ctx, "x"); err != nil {
		t.Fatalf("unexpected get err: %v", err)
	}
	mainT.readRowErr = errors.New("read")
	if _, err := repo.Get(ctx, "x"); err == nil {
		t.Fatalf("expected get err")
	}
	mainT.readRowErr = nil
	mainT.rows[DeliveryKey("d1")] = mkRow(DeliveryKey("d1"), map[string]map[string][]byte{CFMeta: {"customer_id": B("c1")}})
	if got, err := repo.Get(ctx, "d1"); err != nil || got == nil {
		t.Fatalf("get success failed: %v", err)
	}

	deliveredAt := mustParseRFC3339(t, "2026-04-21T00:00:00Z")
	d := &models.Delivery{
		DeliveryID:  "d1",
		CustomerID:  "c1",
		Status:      models.DeliveryStatusPending,
		DeliveredAt: &deliveredAt,
	}
	if err := repo.Set(ctx, d); err != nil {
		t.Fatalf("set err: %v", err)
	}
	if d.Version != 1 || d.CreatedAt.IsZero() || d.UpdatedAt.IsZero() {
		t.Fatalf("set metadata not updated")
	}
	mainT.applyErr = errors.New("apply")
	if err := repo.Set(ctx, &models.Delivery{DeliveryID: "d2"}); err == nil {
		t.Fatalf("expected set err")
	}
	mainT.applyErr = nil

	if err := repo.WriteIndexes(ctx, "a1", d); err != nil {
		t.Fatalf("write indexes err: %v", err)
	}
	idxA.applyErr = errors.New("idxA")
	if err := repo.WriteIndexes(ctx, "a1", d); err == nil {
		t.Fatalf("expected idx asset err")
	}
	idxA.applyErr = nil
	idxC.applyErr = errors.New("idxC")
	if err := repo.WriteIndexes(ctx, "a1", d); err == nil {
		t.Fatalf("expected idx customer err")
	}
	idxC.applyErr = nil
	d.DeliveredAt = nil
	if err := repo.WriteIndexes(ctx, "a1", d); err != nil {
		t.Fatalf("write indexes with nil deliveredAt err: %v", err)
	}

	idxA.rows["a1#1#d1"] = mkRow("a1#1#d1", map[string]map[string][]byte{CFRef: {"delivery_id": B("d1")}})
	ids, err := repo.ListByAsset(ctx, "a1")
	if err != nil || len(ids) == 0 {
		t.Fatalf("list by asset failed: err=%v len=%d", err, len(ids))
	}
	idxA.readRowsErr = errors.New("asset read")
	if _, err := repo.ListByAsset(ctx, "a1"); err == nil {
		t.Fatalf("expected list by asset err")
	}
	idxA.readRowsErr = nil

	idxC.rows["c1#1#d1"] = mkRow("c1#1#d1", map[string]map[string][]byte{CFRef: {"delivery_id": B("d1")}})
	ids, err = repo.ListByCustomer(ctx, "c1")
	if err != nil || len(ids) == 0 {
		t.Fatalf("list by customer failed: err=%v len=%d", err, len(ids))
	}
	idxC.readRowsErr = errors.New("customer read")
	if _, err := repo.ListByCustomer(ctx, "c1"); err == nil {
		t.Fatalf("expected list by customer err")
	}
}

func TestIdempotencyRepo(t *testing.T) {
	ctx := context.Background()
	tb := &fakeTable{rows: map[string]gbt.Row{}}
	repo := &IdempotencyRepo{table: tb}

	if got, err := repo.Get(ctx, "scope", "k1"); err != nil || got != nil {
		t.Fatalf("expected nil,nil for missing row")
	}
	tb.readRowErr = errors.New("read")
	if _, err := repo.Get(ctx, "scope", "k1"); err == nil {
		t.Fatalf("expected get err")
	}
	tb.readRowErr = nil
	key := idemRowKey("scope", "k1")
	tb.rows[key] = mkRow(key, map[string]map[string][]byte{
		CFIdem: {
			"request_hash":  B("h1"),
			"status_code":   B("200"),
			"response_json": []byte(`{"ok":true}`),
			"created_at":    B("2026-04-21T00:00:00Z"),
		},
	})
	got, err := repo.Get(ctx, "scope", "k1")
	if err != nil {
		t.Fatalf("get err: %v", err)
	}
	if got == nil || got.RequestHash != "h1" || got.StatusCode != 200 || string(got.Response) != `{"ok":true}` {
		t.Fatalf("unexpected record: %+v", got)
	}

	if got := idemRowKey("a", "b"); got != "v1#a#b" {
		t.Fatalf("idem row key mismatch: %s", got)
	}

	rec := &repository.IdempotencyRecord{Scope: "s", Key: "k", RequestHash: "h", StatusCode: 201, Response: []byte(`{}`)}
	if err := repo.Save(ctx, rec); err != nil {
		t.Fatalf("save err: %v", err)
	}
	tb.applyErr = errors.New("apply")
	if err := repo.Save(ctx, rec); err == nil {
		t.Fatalf("expected save err")
	}
}

func TestRepoConstructors(t *testing.T) {
	fd := &fakeDataClient{}
	c := &Client{inner: fd}
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
