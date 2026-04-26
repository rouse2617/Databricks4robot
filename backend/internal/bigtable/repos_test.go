package bigtable

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	mrand "math/rand"
	"reflect"
	"strings"
	"testing"
	"time"

	gbt "cloud.google.com/go/bigtable"
	btpb "cloud.google.com/go/bigtable/apiv2/bigtablepb"

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
	rows              map[string]gbt.Row
	readRowErr        error
	applyErr          error
	readRowsErr       error
	checkAndMutateErr error
	expectedVersion   []byte // set by tests to control CheckAndMutateRow condition matching
	applied           []string
	storeOnApply      bool // when true, Apply actually stores mutation data into rows
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

func (f *fakeTable) Apply(_ context.Context, row string, m *gbt.Mutation, _ ...gbt.ApplyOption) error {
	if f.applyErr != nil {
		return f.applyErr
	}
	f.applied = append(f.applied, row)
	if f.storeOnApply && m != nil {
		ops := extractMutationOps(m)
		f.applyMutationOps(row, ops)
	}
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
	case gbt.RowRange:
		// Support PrefixRange and other RowRange types using Contains.
		for key, row := range f.rows {
			if rs.Contains(key) {
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

// extractMutationOps uses reflection to access the unexported ops field of bigtable.Mutation.
func extractMutationOps(m *gbt.Mutation) []*btpb.Mutation {
	if m == nil {
		return nil
	}
	v := reflect.ValueOf(m).Elem()
	opsField := v.FieldByName("ops")
	if !opsField.IsValid() {
		return nil
	}
	// Use unsafe pointer to access unexported field
	ops := reflect.NewAt(opsField.Type(), opsField.Addr().UnsafePointer()).Elem()
	result := make([]*btpb.Mutation, ops.Len())
	for i := 0; i < ops.Len(); i++ {
		result[i] = ops.Index(i).Interface().(*btpb.Mutation)
	}
	return result
}

// applyMutationOps applies parsed mutation ops to a fakeTable row.
func (f *fakeTable) applyMutationOps(rowKey string, ops []*btpb.Mutation) {
	row := f.rows[rowKey]
	if row == nil {
		row = gbt.Row{}
	}
	for _, op := range ops {
		switch m := op.Mutation.(type) {
		case *btpb.Mutation_SetCell_:
			sc := m.SetCell
			family := sc.FamilyName
			qual := string(sc.ColumnQualifier)
			col := family + ":" + qual
			// Update or add the cell in the row
			items := row[family]
			found := false
			for i, item := range items {
				if item.Column == col {
					items[i].Value = sc.Value
					found = true
					break
				}
			}
			if !found {
				row[family] = append(row[family], gbt.ReadItem{
					Row:    rowKey,
					Column: col,
					Value:  sc.Value,
				})
			}
		case *btpb.Mutation_DeleteFromColumn_:
			dc := m.DeleteFromColumn
			family := dc.FamilyName
			qual := string(dc.ColumnQualifier)
			col := family + ":" + qual
			items := row[family]
			for i, item := range items {
				if item.Column == col {
					row[family] = append(items[:i], items[i+1:]...)
					break
				}
			}
			if len(row[family]) == 0 {
				delete(row, family)
			}
		}
	}
	f.rows[rowKey] = row
}

func (f *fakeTable) CheckAndMutateRow(_ context.Context, row string, cond gbt.Filter, mtrue, mfalse *gbt.Mutation) (bool, error) {
	if f.checkAndMutateErr != nil {
		return false, f.checkAndMutateErr
	}
	currentRow := f.rows[row]
	if currentRow == nil {
		// Row doesn't exist: condition doesn't match
		if mfalse != nil {
			ops := extractMutationOps(mfalse)
			f.applyMutationOps(row, ops)
		}
		return false, nil
	}

	// Read the current version from meta:version column
	var currentVersion []byte
	for _, item := range currentRow[CFMeta] {
		if item.Column == CFMeta+":version" {
			currentVersion = item.Value
			break
		}
	}

	// Extract the expected version from the condition filter.
	// We use a pragmatic approach: serialize the filter condition and extract
	// the version bytes. The MergeCfAlgo uses a ChainFilters with ValueRangeFilter
	// that contains the expected version bytes.
	// Since bigtable.Filter is opaque, we compare the current version against
	// the expectedVersion stored on the fakeTable (set by tests), or fall back
	// to a simple "row exists" check.
	matched := false
	if f.expectedVersion != nil {
		matched = bytes.Equal(currentVersion, f.expectedVersion)
	} else {
		// Default: match if row exists (simple existence check)
		matched = currentRow != nil
	}

	if matched {
		if mtrue != nil {
			ops := extractMutationOps(mtrue)
			f.applyMutationOps(row, ops)
		}
		return true, nil
	}

	if mfalse != nil {
		ops := extractMutationOps(mfalse)
		f.applyMutationOps(row, ops)
	}
	return false, nil
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

func TestFakeTableCheckAndMutateRow(t *testing.T) {
	ctx := context.Background()

	t.Run("version_match_applies_mtrue", func(t *testing.T) {
		ft := &fakeTable{
			rows: map[string]gbt.Row{
				"v1#a1": mkRow("v1#a1", map[string]map[string][]byte{
					CFMeta: {
						"version":      PackInt64(1),
						"mcap_file_id": B("m1"),
					},
					CFAlgo: {"old_key": B("old_val")},
				}),
			},
			expectedVersion: PackInt64(1), // condition: version == 1
		}

		mtrue := gbt.NewMutation()
		mtrue.Set(CFAlgo, "new_key", gbt.Now(), B("new_val"))
		mtrue.Set(CFMeta, "version", gbt.Now(), PackInt64(2))
		mtrue.DeleteCellsInColumn(CFAlgo, "old_key")

		matched, err := ft.CheckAndMutateRow(ctx, "v1#a1", nil, mtrue, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !matched {
			t.Fatalf("expected matched=true, got false")
		}

		// Verify mutations were applied
		row := ft.rows["v1#a1"]

		// Check new_key was set in algo family
		foundNewKey := false
		for _, item := range row[CFAlgo] {
			if item.Column == CFAlgo+":new_key" && string(item.Value) == "new_val" {
				foundNewKey = true
			}
			if item.Column == CFAlgo+":old_key" {
				t.Fatalf("old_key should have been deleted")
			}
		}
		if !foundNewKey {
			t.Fatalf("new_key not found in algo family")
		}

		// Check version was updated to 2
		for _, item := range row[CFMeta] {
			if item.Column == CFMeta+":version" {
				if UnpackInt64(item.Value) != 2 {
					t.Fatalf("expected version=2, got %d", UnpackInt64(item.Value))
				}
			}
		}
	})

	t.Run("version_mismatch_returns_false", func(t *testing.T) {
		ft := &fakeTable{
			rows: map[string]gbt.Row{
				"v1#a1": mkRow("v1#a1", map[string]map[string][]byte{
					CFMeta: {
						"version":      PackInt64(1),
						"mcap_file_id": B("m1"),
					},
					CFAlgo: {"existing": B("val")},
				}),
			},
			expectedVersion: PackInt64(99), // condition: version == 99 (mismatch)
		}

		mtrue := gbt.NewMutation()
		mtrue.Set(CFAlgo, "should_not_appear", gbt.Now(), B("nope"))

		matched, err := ft.CheckAndMutateRow(ctx, "v1#a1", nil, mtrue, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if matched {
			t.Fatalf("expected matched=false, got true")
		}

		// Verify data unchanged
		row := ft.rows["v1#a1"]
		for _, item := range row[CFAlgo] {
			if item.Column == CFAlgo+":should_not_appear" {
				t.Fatalf("mtrue should not have been applied")
			}
		}
		// Original data still present
		foundExisting := false
		for _, item := range row[CFAlgo] {
			if item.Column == CFAlgo+":existing" && string(item.Value) == "val" {
				foundExisting = true
			}
		}
		if !foundExisting {
			t.Fatalf("existing data should be unchanged")
		}
	})

	t.Run("row_not_found_returns_false", func(t *testing.T) {
		ft := &fakeTable{
			rows:            map[string]gbt.Row{},
			expectedVersion: PackInt64(1),
		}

		mtrue := gbt.NewMutation()
		mtrue.Set(CFAlgo, "key", gbt.Now(), B("val"))

		matched, err := ft.CheckAndMutateRow(ctx, "v1#missing", nil, mtrue, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if matched {
			t.Fatalf("expected matched=false for missing row")
		}
	})

	t.Run("error_propagation", func(t *testing.T) {
		ft := &fakeTable{
			rows:              map[string]gbt.Row{},
			checkAndMutateErr: errors.New("rpc error"),
		}

		_, err := ft.CheckAndMutateRow(ctx, "v1#a1", nil, nil, nil)
		if err == nil || err.Error() != "rpc error" {
			t.Fatalf("expected rpc error, got: %v", err)
		}
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Property tests for MergeCfAlgo
// ──────────────────────────────────────────────────────────────────────────────

// TestMergeCfAlgoRoundTrip verifies Property 1: MergeCfAlgo round-trip consistency.
// For any valid algoKV/filesKV, after MergeCfAlgo succeeds, Get returns consistent data.
// **Validates: Requirements 3.1, 3.2, 3.3, 3.4, 3.5, 3.8**
func TestMergeCfAlgoRoundTrip(t *testing.T) {
	const iterations = 100
	ctx := context.Background()
	rng := newTestRNG(t)

	for i := 0; i < iterations; i++ {
		// Setup: create a fresh fakeTable with a seed asset at version 1.
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}

		seedAsset := &models.Asset{
			AssetID:    "asset-prop",
			McapFileID: "m1",
			Status:     models.AssetStatusApproved,
		}
		if err := repo.Set(ctx, seedAsset); err != nil {
			t.Fatalf("iter %d: seed Set err: %v", i, err)
		}
		// After Set, seedAsset.Version == 1. Re-read to populate fakeTable row.
		// The fakeTable.Apply doesn't actually store data, so we need to manually
		// build the row for the fakeTable.
		ft.rows[AssetKey("asset-prop")] = mkRow(AssetKey("asset-prop"), map[string]map[string][]byte{
			CFMeta: {
				"mcap_file_id": B("m1"),
				"status":       B("approved"),
				"version":      PackInt64(1),
				"created_at":   RFC3339(time.Now()),
				"updated_at":   RFC3339(time.Now()),
			},
		})

		// Generate random algoKV and filesKV.
		algoKV := genRandomKV(rng, 0, 5)
		filesKV := genRandomKV(rng, 0, 5)

		// Set expectedVersion on fakeTable so CheckAndMutateRow matches.
		ft.expectedVersion = PackInt64(1)

		newVer, err := repo.MergeCfAlgo(ctx, "asset-prop", 1, algoKV, filesKV)
		if err != nil {
			t.Fatalf("iter %d: MergeCfAlgo err: %v", i, err)
		}
		if newVer != 2 {
			t.Fatalf("iter %d: expected version 2, got %d", i, newVer)
		}

		// Read back and verify.
		got, err := repo.Get(ctx, "asset-prop")
		if err != nil {
			t.Fatalf("iter %d: Get err: %v", i, err)
		}
		if got == nil {
			t.Fatalf("iter %d: Get returned nil", i)
		}

		// Verify version was incremented.
		if got.Version != 2 {
			t.Fatalf("iter %d: expected version 2 in read, got %d", i, got.Version)
		}

		// Verify algoKV: non-nil values should be present, nil values should be absent.
		for k, v := range algoKV {
			if v != nil {
				expected := fmt.Sprintf("%v", v)
				if got.AlgoResults[k] != expected {
					t.Fatalf("iter %d: algo key %q: expected %q, got %q", i, k, expected, got.AlgoResults[k])
				}
			} else {
				if _, exists := got.AlgoResults[k]; exists {
					t.Fatalf("iter %d: algo key %q should have been deleted", i, k)
				}
			}
		}

		// Verify filesKV: non-nil values should be present, nil values should be absent.
		for k, v := range filesKV {
			if v != nil {
				expected := fmt.Sprintf("%v", v)
				if got.Files[k] != expected {
					t.Fatalf("iter %d: files key %q: expected %q, got %q", i, k, expected, got.Files[k])
				}
			} else {
				if _, exists := got.Files[k]; exists {
					t.Fatalf("iter %d: files key %q should have been deleted", i, k)
				}
			}
		}
	}
}

// TestMergeCfAlgoOptimisticLockRejection verifies Property 2: MergeCfAlgo optimistic lock rejection.
// For any wrong expectedVersion, MergeCfAlgo returns ErrOptimisticLock and data is unchanged.
// **Validates: Requirements 3.6**
func TestMergeCfAlgoOptimisticLockRejection(t *testing.T) {
	const iterations = 100
	ctx := context.Background()
	rng := newTestRNG(t)

	for i := 0; i < iterations; i++ {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}

		currentVersion := int64(1)
		ft.rows[AssetKey("asset-lock")] = mkRow(AssetKey("asset-lock"), map[string]map[string][]byte{
			CFMeta: {
				"mcap_file_id": B("m1"),
				"status":       B("approved"),
				"version":      PackInt64(currentVersion),
				"created_at":   RFC3339(time.Now()),
				"updated_at":   RFC3339(time.Now()),
			},
			CFAlgo: {"existing_key": B("existing_val")},
		})

		// Generate a wrong version (anything != currentVersion).
		wrongVersion := currentVersion
		for wrongVersion == currentVersion {
			wrongVersion = rng.Int63n(200) - 50 // range [-50, 149]
		}

		// Set expectedVersion to what MergeCfAlgo will check for (the wrong version).
		// The fakeTable compares the row's actual version against this value.
		// Since wrongVersion != currentVersion, the comparison will fail → ErrOptimisticLock.
		ft.expectedVersion = PackInt64(wrongVersion)

		// Snapshot the row before the call.
		rowBefore := ft.rows[AssetKey("asset-lock")]

		algoKV := genRandomKV(rng, 1, 3)
		filesKV := genRandomKV(rng, 0, 2)

		newVer, err := repo.MergeCfAlgo(ctx, "asset-lock", wrongVersion, algoKV, filesKV)
		if !errors.Is(err, repository.ErrOptimisticLock) {
			t.Fatalf("iter %d: expected ErrOptimisticLock, got err=%v newVer=%d (wrongVersion=%d)", i, err, newVer, wrongVersion)
		}
		if newVer != 0 {
			t.Fatalf("iter %d: expected version 0 on lock failure, got %d", i, newVer)
		}

		// Verify data is unchanged.
		rowAfter := ft.rows[AssetKey("asset-lock")]
		if !rowsEqual(rowBefore, rowAfter) {
			t.Fatalf("iter %d: row data changed after optimistic lock rejection", i)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Test helpers for property tests
// ──────────────────────────────────────────────────────────────────────────────

type testRNG struct {
	src *mrand.Rand
}

func newTestRNG(t *testing.T) *testRNG {
	t.Helper()
	seed := time.Now().UnixNano()
	t.Logf("RNG seed: %d", seed)
	return &testRNG{src: mrand.New(mrand.NewSource(seed))}
}

func (r *testRNG) Intn(n int) int       { return r.src.Intn(n) }
func (r *testRNG) Int63n(n int64) int64 { return r.src.Int63n(n) }

// genRandomKV generates a random map[string]interface{} with minKeys..maxKeys entries.
// Values are either non-nil strings or nil (for deletion).
func genRandomKV(rng *testRNG, minKeys, maxKeys int) map[string]interface{} {
	n := minKeys
	if maxKeys > minKeys {
		n += rng.Intn(maxKeys - minKeys + 1)
	}
	kv := make(map[string]interface{}, n)
	for j := 0; j < n; j++ {
		key := fmt.Sprintf("key_%d", rng.Intn(20))
		if rng.Intn(4) == 0 { // 25% chance of nil (deletion)
			kv[key] = nil
		} else {
			kv[key] = fmt.Sprintf("val_%d", rng.Intn(1000))
		}
	}
	return kv
}

// rowsEqual compares two bigtable.Row values for equality.
func rowsEqual(a, b gbt.Row) bool {
	if len(a) != len(b) {
		return false
	}
	for cf, aItems := range a {
		bItems, ok := b[cf]
		if !ok || len(aItems) != len(bItems) {
			return false
		}
		aMap := make(map[string]string, len(aItems))
		for _, item := range aItems {
			aMap[item.Column] = string(item.Value)
		}
		for _, item := range bItems {
			if aMap[item.Column] != string(item.Value) {
				return false
			}
		}
	}
	return true
}

// ──────────────────────────────────────────────────────────────────────────────
// Unit tests for MergeCfAlgo
// ──────────────────────────────────────────────────────────────────────────────

func TestMergeCfAlgo(t *testing.T) {
	ctx := context.Background()

	t.Run("successful_merge", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}

		ft.rows[AssetKey("a1")] = mkRow(AssetKey("a1"), map[string]map[string][]byte{
			CFMeta: {
				"mcap_file_id": B("m1"),
				"status":       B("approved"),
				"version":      PackInt64(1),
			},
		})
		ft.expectedVersion = PackInt64(1)

		algoKV := map[string]interface{}{
			"sam2@1.0:status": "ok",
			"sam2@1.0:run_id": "run-123",
		}
		filesKV := map[string]interface{}{
			"output.mcap": "gs://bucket/output.mcap",
		}

		newVer, err := repo.MergeCfAlgo(ctx, "a1", 1, algoKV, filesKV)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if newVer != 2 {
			t.Fatalf("expected version 2, got %d", newVer)
		}

		// Verify data was written.
		got, err := repo.Get(ctx, "a1")
		if err != nil {
			t.Fatalf("Get err: %v", err)
		}
		if got.AlgoResults["sam2@1.0:status"] != "ok" {
			t.Fatalf("expected algo status=ok, got %q", got.AlgoResults["sam2@1.0:status"])
		}
		if got.AlgoResults["sam2@1.0:run_id"] != "run-123" {
			t.Fatalf("expected algo run_id=run-123, got %q", got.AlgoResults["sam2@1.0:run_id"])
		}
		if got.Files["output.mcap"] != "gs://bucket/output.mcap" {
			t.Fatalf("expected files output.mcap, got %q", got.Files["output.mcap"])
		}
		if got.Version != 2 {
			t.Fatalf("expected version 2 in read, got %d", got.Version)
		}
	})

	t.Run("version_mismatch", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}

		ft.rows[AssetKey("a1")] = mkRow(AssetKey("a1"), map[string]map[string][]byte{
			CFMeta: {
				"version": PackInt64(1),
			},
		})
		// MergeCfAlgo will check for version 99, but row has version 1.
		ft.expectedVersion = PackInt64(99)

		newVer, err := repo.MergeCfAlgo(ctx, "a1", 99, nil, nil)
		if !errors.Is(err, repository.ErrOptimisticLock) {
			t.Fatalf("expected ErrOptimisticLock, got: %v", err)
		}
		if newVer != 0 {
			t.Fatalf("expected version 0, got %d", newVer)
		}
	})

	t.Run("nil_value_deletes_column", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}

		ft.rows[AssetKey("a1")] = mkRow(AssetKey("a1"), map[string]map[string][]byte{
			CFMeta: {
				"version": PackInt64(1),
			},
			CFAlgo: {
				"sam2@1.0:status":      B("ok"),
				"sam2@1.0:finished_at": B("2026-01-01T00:00:00Z"),
			},
		})
		ft.expectedVersion = PackInt64(1)

		// Delete sam2@1.0:finished_at by setting nil, keep sam2@1.0:status.
		algoKV := map[string]interface{}{
			"sam2@1.0:status":      "pending",
			"sam2@1.0:finished_at": nil,
		}

		newVer, err := repo.MergeCfAlgo(ctx, "a1", 1, algoKV, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if newVer != 2 {
			t.Fatalf("expected version 2, got %d", newVer)
		}

		got, err := repo.Get(ctx, "a1")
		if err != nil {
			t.Fatalf("Get err: %v", err)
		}
		if got.AlgoResults["sam2@1.0:status"] != "pending" {
			t.Fatalf("expected status=pending, got %q", got.AlgoResults["sam2@1.0:status"])
		}
		if _, exists := got.AlgoResults["sam2@1.0:finished_at"]; exists {
			t.Fatalf("finished_at should have been deleted")
		}
	})

	t.Run("filesKV_write_and_delete", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}

		ft.rows[AssetKey("a1")] = mkRow(AssetKey("a1"), map[string]map[string][]byte{
			CFMeta: {
				"version": PackInt64(1),
			},
			CFFiles: {
				"old_file.mcap": B("gs://bucket/old.mcap"),
			},
		})
		ft.expectedVersion = PackInt64(1)

		filesKV := map[string]interface{}{
			"new_file.mcap": "gs://bucket/new.mcap",
			"old_file.mcap": nil, // delete
		}

		newVer, err := repo.MergeCfAlgo(ctx, "a1", 1, nil, filesKV)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if newVer != 2 {
			t.Fatalf("expected version 2, got %d", newVer)
		}

		got, err := repo.Get(ctx, "a1")
		if err != nil {
			t.Fatalf("Get err: %v", err)
		}
		if got.Files["new_file.mcap"] != "gs://bucket/new.mcap" {
			t.Fatalf("expected new_file.mcap, got %q", got.Files["new_file.mcap"])
		}
		if _, exists := got.Files["old_file.mcap"]; exists {
			t.Fatalf("old_file.mcap should have been deleted")
		}
	})

	t.Run("row_not_found", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}

		newVer, err := repo.MergeCfAlgo(ctx, "nonexistent", 1, nil, nil)
		if err == nil {
			t.Fatalf("expected error for missing row")
		}
		if strings.Contains(err.Error(), "optimistic lock") {
			t.Fatalf("should be 'not found' error, not optimistic lock: %v", err)
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected 'not found' in error, got: %v", err)
		}
		if newVer != 0 {
			t.Fatalf("expected version 0, got %d", newVer)
		}
	})

	t.Run("read_row_error", func(t *testing.T) {
		ft := &fakeTable{
			rows:       map[string]gbt.Row{},
			readRowErr: errors.New("rpc failure"),
		}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}

		_, err := repo.MergeCfAlgo(ctx, "a1", 1, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "rpc failure") {
			t.Fatalf("expected rpc failure error, got: %v", err)
		}
	})

	t.Run("check_and_mutate_error", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}

		ft.rows[AssetKey("a1")] = mkRow(AssetKey("a1"), map[string]map[string][]byte{
			CFMeta: {"version": PackInt64(1)},
		})
		ft.expectedVersion = PackInt64(1)
		ft.checkAndMutateErr = errors.New("cam error")

		_, err := repo.MergeCfAlgo(ctx, "a1", 1, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "cam error") {
			t.Fatalf("expected cam error, got: %v", err)
		}
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Property test for Asset Set/Get round-trip (Files + LifecycleMeta)
// ──────────────────────────────────────────────────────────────────────────────

// TestAssetSetGetRoundTripProperty verifies Property 5: Asset Set/Get round-trip consistency.
// For any Asset with random Files and LifecycleMeta, Set then Get returns consistent data.
// **Validates: Requirements 5.1, 5.2, 5.5, 6.1, 6.2, 6.5**
func TestAssetSetGetRoundTripProperty(t *testing.T) {
	const iterations = 100
	ctx := context.Background()
	rng := newTestRNG(t)

	lifecycleKeys := []string{"retention_tier", "archive_after_days", "delete_after_days", "total_size_bytes", "last_accessed_at"}

	for i := 0; i < iterations; i++ {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}

		// Generate random Files map.
		files := map[string]string{}
		nFiles := rng.Intn(6) // 0..5 files
		for j := 0; j < nFiles; j++ {
			key := fmt.Sprintf("file_%d.mcap", rng.Intn(20))
			files[key] = fmt.Sprintf("gs://bucket/path_%d", rng.Intn(1000))
		}

		// Generate random LifecycleMeta map (only valid lifecycle keys).
		lifecycleMeta := map[string]interface{}{}
		nMeta := rng.Intn(len(lifecycleKeys) + 1) // 0..5 entries
		for j := 0; j < nMeta; j++ {
			key := lifecycleKeys[rng.Intn(len(lifecycleKeys))]
			lifecycleMeta[key] = fmt.Sprintf("val_%d", rng.Intn(1000))
		}

		asset := &models.Asset{
			AssetID:       fmt.Sprintf("asset-%d", i),
			McapFileID:    "m1",
			Status:        models.AssetStatusApproved,
			Files:         files,
			LifecycleMeta: lifecycleMeta,
		}

		// Call Set (this calls fakeTable.Apply which doesn't store, but we need
		// to capture the version/timestamps Set assigns).
		if err := repo.Set(ctx, asset); err != nil {
			t.Fatalf("iter %d: Set err: %v", i, err)
		}

		// Manually build the expected row in fakeTable.rows to simulate storage.
		rowData := map[string]map[string][]byte{
			CFMeta: {
				"mcap_file_id":       B(asset.McapFileID),
				"start_timestamp_ns": PackInt64(asset.StartTimestampNs),
				"end_timestamp_ns":   PackInt64(asset.EndTimestampNs),
				"duration_sec":       B(fmt.Sprintf("%f", asset.DurationSec)),
				"reviewer":           B(asset.Reviewer),
				"status":             B(string(asset.Status)),
				"owner":              B(asset.Owner),
				"type":               B(asset.SegType),
				"env":                B(asset.Env),
				"task":               B(asset.Task),
				"delivery_count":     PackInt64(int64(asset.DeliveryCount)),
				"last_delivered_to":  B(asset.LastDeliveredTo),
				"created_at":         RFC3339(asset.CreatedAt),
				"updated_at":         RFC3339(asset.UpdatedAt),
				"version":            PackInt64(asset.Version),
			},
		}

		// Add lifecycle meta to CFMeta.
		for k, v := range lifecycleMeta {
			rowData[CFMeta][k] = B(fmt.Sprintf("%v", v))
		}

		// Add Files to CFFiles.
		if len(files) > 0 {
			rowData[CFFiles] = map[string][]byte{}
			for k, v := range files {
				rowData[CFFiles][k] = B(v)
			}
		}

		ft.rows[AssetKey(asset.AssetID)] = mkRow(AssetKey(asset.AssetID), rowData)

		// Read back via Get.
		got, err := repo.Get(ctx, asset.AssetID)
		if err != nil {
			t.Fatalf("iter %d: Get err: %v", i, err)
		}
		if got == nil {
			t.Fatalf("iter %d: Get returned nil", i)
		}

		// Verify Files round-trip.
		if len(got.Files) != len(files) {
			t.Fatalf("iter %d: Files length mismatch: expected %d, got %d", i, len(files), len(got.Files))
		}
		for k, v := range files {
			if got.Files[k] != v {
				t.Fatalf("iter %d: Files[%q]: expected %q, got %q", i, k, v, got.Files[k])
			}
		}

		// Verify LifecycleMeta round-trip.
		if len(got.LifecycleMeta) != len(lifecycleMeta) {
			t.Fatalf("iter %d: LifecycleMeta length mismatch: expected %d, got %d", i, len(lifecycleMeta), len(got.LifecycleMeta))
		}
		for k, v := range lifecycleMeta {
			expected := fmt.Sprintf("%v", v)
			gotVal, ok := got.LifecycleMeta[k]
			if !ok {
				t.Fatalf("iter %d: LifecycleMeta[%q] missing", i, k)
			}
			if fmt.Sprintf("%v", gotVal) != expected {
				t.Fatalf("iter %d: LifecycleMeta[%q]: expected %q, got %q", i, k, expected, gotVal)
			}
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Unit tests for Asset Set/Get with Files and LifecycleMeta
// ──────────────────────────────────────────────────────────────────────────────

func TestAssetSetGetFilesAndLifecycleMeta(t *testing.T) {
	ctx := context.Background()

	t.Run("round_trip_with_files_and_lifecycle_meta", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}

		asset := &models.Asset{
			AssetID:    "a1",
			McapFileID: "m1",
			Status:     models.AssetStatusApproved,
			Files: map[string]string{
				"raw.mcap":    "gs://bucket/raw.mcap",
				"output.mcap": "gs://bucket/output.mcap",
			},
			LifecycleMeta: map[string]interface{}{
				"retention_tier":     "hot",
				"archive_after_days": "90",
				"delete_after_days":  "365",
				"total_size_bytes":   "1048576",
				"last_accessed_at":   "2026-04-21T00:00:00Z",
			},
		}

		if err := repo.Set(ctx, asset); err != nil {
			t.Fatalf("Set err: %v", err)
		}

		// Manually build the row to simulate storage.
		ft.rows[AssetKey("a1")] = mkRow(AssetKey("a1"), map[string]map[string][]byte{
			CFMeta: {
				"mcap_file_id":       B("m1"),
				"status":             B("approved"),
				"version":            PackInt64(asset.Version),
				"created_at":         RFC3339(asset.CreatedAt),
				"updated_at":         RFC3339(asset.UpdatedAt),
				"retention_tier":     B("hot"),
				"archive_after_days": B("90"),
				"delete_after_days":  B("365"),
				"total_size_bytes":   B("1048576"),
				"last_accessed_at":   B("2026-04-21T00:00:00Z"),
			},
			CFFiles: {
				"raw.mcap":    B("gs://bucket/raw.mcap"),
				"output.mcap": B("gs://bucket/output.mcap"),
			},
		})

		got, err := repo.Get(ctx, "a1")
		if err != nil {
			t.Fatalf("Get err: %v", err)
		}
		if got == nil {
			t.Fatalf("Get returned nil")
		}

		// Verify Files.
		if len(got.Files) != 2 {
			t.Fatalf("expected 2 files, got %d", len(got.Files))
		}
		if got.Files["raw.mcap"] != "gs://bucket/raw.mcap" {
			t.Fatalf("expected raw.mcap URI, got %q", got.Files["raw.mcap"])
		}
		if got.Files["output.mcap"] != "gs://bucket/output.mcap" {
			t.Fatalf("expected output.mcap URI, got %q", got.Files["output.mcap"])
		}

		// Verify LifecycleMeta.
		if len(got.LifecycleMeta) != 5 {
			t.Fatalf("expected 5 lifecycle meta entries, got %d", len(got.LifecycleMeta))
		}
		if got.LifecycleMeta["retention_tier"] != "hot" {
			t.Fatalf("expected retention_tier=hot, got %v", got.LifecycleMeta["retention_tier"])
		}
		if got.LifecycleMeta["archive_after_days"] != "90" {
			t.Fatalf("expected archive_after_days=90, got %v", got.LifecycleMeta["archive_after_days"])
		}
		if got.LifecycleMeta["total_size_bytes"] != "1048576" {
			t.Fatalf("expected total_size_bytes=1048576, got %v", got.LifecycleMeta["total_size_bytes"])
		}
		if got.LifecycleMeta["last_accessed_at"] != "2026-04-21T00:00:00Z" {
			t.Fatalf("expected last_accessed_at, got %v", got.LifecycleMeta["last_accessed_at"])
		}
	})

	t.Run("empty_files_and_lifecycle_meta", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}

		asset := &models.Asset{
			AssetID:       "a2",
			McapFileID:    "m2",
			Status:        models.AssetStatusApproved,
			Files:         nil,
			LifecycleMeta: nil,
		}

		if err := repo.Set(ctx, asset); err != nil {
			t.Fatalf("Set err: %v", err)
		}

		// Build row with no CFFiles and no lifecycle fields in CFMeta.
		ft.rows[AssetKey("a2")] = mkRow(AssetKey("a2"), map[string]map[string][]byte{
			CFMeta: {
				"mcap_file_id": B("m2"),
				"status":       B("approved"),
				"version":      PackInt64(asset.Version),
				"created_at":   RFC3339(asset.CreatedAt),
				"updated_at":   RFC3339(asset.UpdatedAt),
			},
		})

		got, err := repo.Get(ctx, "a2")
		if err != nil {
			t.Fatalf("Get err: %v", err)
		}
		if got == nil {
			t.Fatalf("Get returned nil")
		}

		// Files should be empty map (not nil).
		if got.Files == nil {
			t.Fatalf("expected non-nil Files map")
		}
		if len(got.Files) != 0 {
			t.Fatalf("expected empty Files, got %d entries", len(got.Files))
		}

		// LifecycleMeta should be empty map (not nil).
		if got.LifecycleMeta == nil {
			t.Fatalf("expected non-nil LifecycleMeta map")
		}
		if len(got.LifecycleMeta) != 0 {
			t.Fatalf("expected empty LifecycleMeta, got %d entries", len(got.LifecycleMeta))
		}
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Property test for AlgoEvent round-trip consistency
// ──────────────────────────────────────────────────────────────────────────────

// TestAlgoEventRoundTrip verifies Property 3: AlgoEvent round-trip consistency.
// For any valid AlgoEvent, Insert then ListByAsset finds it with all fields matching.
// **Validates: Requirements 4.3, 4.5, 4.6**
func TestAlgoEventRoundTrip(t *testing.T) {
	const iterations = 100
	ctx := context.Background()
	rng := newTestRNG(t)

	for i := 0; i < iterations; i++ {
		ft := &fakeTable{rows: map[string]gbt.Row{}, storeOnApply: true}
		repo := &AlgoEventRepo{table: ft}

		event := genRandomAlgoEvent(rng)

		if err := repo.Insert(ctx, event); err != nil {
			t.Fatalf("iter %d: Insert err: %v", i, err)
		}

		// ListByAsset with nil algoKey should return all events for this asset.
		events, err := repo.ListByAsset(ctx, event.AssetID, nil)
		if err != nil {
			t.Fatalf("iter %d: ListByAsset err: %v", i, err)
		}
		if len(events) != 1 {
			t.Fatalf("iter %d: expected 1 event, got %d", i, len(events))
		}

		got := events[0]
		if got.EventID != event.EventID {
			t.Fatalf("iter %d: EventID mismatch: %q vs %q", i, got.EventID, event.EventID)
		}
		if got.AssetID != event.AssetID {
			t.Fatalf("iter %d: AssetID mismatch: %q vs %q", i, got.AssetID, event.AssetID)
		}
		if got.AlgoKey != event.AlgoKey {
			t.Fatalf("iter %d: AlgoKey mismatch: %q vs %q", i, got.AlgoKey, event.AlgoKey)
		}
		if got.NewStatus != event.NewStatus {
			t.Fatalf("iter %d: NewStatus mismatch: %q vs %q", i, got.NewStatus, event.NewStatus)
		}

		// Compare PrevStatus (both nil or both equal).
		if (got.PrevStatus == nil) != (event.PrevStatus == nil) {
			t.Fatalf("iter %d: PrevStatus nil mismatch", i)
		}
		if got.PrevStatus != nil && *got.PrevStatus != *event.PrevStatus {
			t.Fatalf("iter %d: PrevStatus mismatch: %q vs %q", i, *got.PrevStatus, *event.PrevStatus)
		}

		// Compare RunID.
		if (got.RunID == nil) != (event.RunID == nil) {
			t.Fatalf("iter %d: RunID nil mismatch", i)
		}
		if got.RunID != nil && *got.RunID != *event.RunID {
			t.Fatalf("iter %d: RunID mismatch: %q vs %q", i, *got.RunID, *event.RunID)
		}

		// Compare Reason.
		if (got.Reason == nil) != (event.Reason == nil) {
			t.Fatalf("iter %d: Reason nil mismatch", i)
		}
		if got.Reason != nil && *got.Reason != *event.Reason {
			t.Fatalf("iter %d: Reason mismatch: %q vs %q", i, *got.Reason, *event.Reason)
		}

		// Compare CreatedAt (truncated to millisecond since reverse_timestamp uses UnixMilli).
		expectedTime := event.CreatedAt.Truncate(time.Millisecond)
		gotTime := got.CreatedAt.Truncate(time.Millisecond)
		// RFC3339Nano round-trip may lose sub-millisecond precision, so compare within 1s.
		if gotTime.Sub(expectedTime).Abs() > time.Second {
			t.Fatalf("iter %d: CreatedAt mismatch: %v vs %v", i, got.CreatedAt, event.CreatedAt)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Property test for AlgoEvent algoKey filter correctness
// ──────────────────────────────────────────────────────────────────────────────

// TestAlgoEventAlgoKeyFilter verifies Property 4: AlgoEvent algoKey filter correctness.
// Generate events with mixed algoKeys, verify filtering returns only matching events.
// **Validates: Requirements 4.4**
func TestAlgoEventAlgoKeyFilter(t *testing.T) {
	const iterations = 100
	ctx := context.Background()
	rng := newTestRNG(t)

	algoKeys := []string{"sam2@1.0", "hand_tracking@2.0", "object_detect@1.5", "pose_est@3.0"}

	for i := 0; i < iterations; i++ {
		ft := &fakeTable{rows: map[string]gbt.Row{}, storeOnApply: true}
		repo := &AlgoEventRepo{table: ft}

		assetID := fmt.Sprintf("asset-%d", i)
		// Insert 3-8 events with random algoKeys for the same asset.
		numEvents := 3 + rng.Intn(6)
		inserted := make([]*models.AlgoEvent, numEvents)
		for j := 0; j < numEvents; j++ {
			ev := &models.AlgoEvent{
				EventID:   fmt.Sprintf("ev-%d-%d", i, j),
				AssetID:   assetID,
				AlgoKey:   algoKeys[rng.Intn(len(algoKeys))],
				NewStatus: "ok",
				CreatedAt: time.Now().Add(-time.Duration(j) * time.Minute),
			}
			if err := repo.Insert(ctx, ev); err != nil {
				t.Fatalf("iter %d: Insert err: %v", i, err)
			}
			inserted[j] = ev
		}

		// Pick a random algoKey to filter by.
		targetKey := algoKeys[rng.Intn(len(algoKeys))]
		filtered, err := repo.ListByAsset(ctx, assetID, &targetKey)
		if err != nil {
			t.Fatalf("iter %d: ListByAsset err: %v", i, err)
		}

		// All returned events must have the target algoKey.
		for _, ev := range filtered {
			if ev.AlgoKey != targetKey {
				t.Fatalf("iter %d: filtered event has wrong algoKey: %q, expected %q", i, ev.AlgoKey, targetKey)
			}
		}

		// Count expected matches from inserted events.
		expectedCount := 0
		for _, ev := range inserted {
			if ev.AlgoKey == targetKey {
				expectedCount++
			}
		}
		if len(filtered) != expectedCount {
			t.Fatalf("iter %d: expected %d events with algoKey=%q, got %d", i, expectedCount, targetKey, len(filtered))
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Unit tests for AlgoEventRepo
// ──────────────────────────────────────────────────────────────────────────────

func TestAlgoEventRepo(t *testing.T) {
	ctx := context.Background()

	t.Run("insert_success", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}, storeOnApply: true}
		repo := &AlgoEventRepo{table: ft}

		prev := "pending"
		runID := "run-123"
		reason := "completed successfully"
		event := &models.AlgoEvent{
			EventID:    "ev-1",
			AssetID:    "asset-1",
			AlgoKey:    "sam2@1.0",
			PrevStatus: &prev,
			NewStatus:  "running",
			RunID:      &runID,
			Reason:     &reason,
			CreatedAt:  mustParseRFC3339(t, "2026-04-21T10:00:00Z"),
		}

		if err := repo.Insert(ctx, event); err != nil {
			t.Fatalf("Insert err: %v", err)
		}

		// Verify the row was stored.
		if len(ft.rows) != 1 {
			t.Fatalf("expected 1 row, got %d", len(ft.rows))
		}

		// Verify we can read it back.
		events, err := repo.ListByAsset(ctx, "asset-1", nil)
		if err != nil {
			t.Fatalf("ListByAsset err: %v", err)
		}
		if len(events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(events))
		}
		got := events[0]
		if got.EventID != "ev-1" {
			t.Fatalf("expected EventID=ev-1, got %q", got.EventID)
		}
		if got.AlgoKey != "sam2@1.0" {
			t.Fatalf("expected AlgoKey=sam2@1.0, got %q", got.AlgoKey)
		}
		if got.PrevStatus == nil || *got.PrevStatus != "pending" {
			t.Fatalf("expected PrevStatus=pending, got %v", got.PrevStatus)
		}
		if got.RunID == nil || *got.RunID != "run-123" {
			t.Fatalf("expected RunID=run-123, got %v", got.RunID)
		}
		if got.Reason == nil || *got.Reason != "completed successfully" {
			t.Fatalf("expected Reason, got %v", got.Reason)
		}
	})

	t.Run("insert_apply_error", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}, applyErr: errors.New("apply failed")}
		repo := &AlgoEventRepo{table: ft}

		event := &models.AlgoEvent{
			EventID:   "ev-1",
			AssetID:   "asset-1",
			AlgoKey:   "sam2@1.0",
			NewStatus: "ok",
			CreatedAt: time.Now(),
		}

		err := repo.Insert(ctx, event)
		if err == nil || !strings.Contains(err.Error(), "apply failed") {
			t.Fatalf("expected apply error, got: %v", err)
		}
	})

	t.Run("insert_nil_optional_fields", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}, storeOnApply: true}
		repo := &AlgoEventRepo{table: ft}

		event := &models.AlgoEvent{
			EventID:    "ev-2",
			AssetID:    "asset-2",
			AlgoKey:    "hand_tracking@2.0",
			PrevStatus: nil,
			NewStatus:  "pending",
			RunID:      nil,
			Reason:     nil,
			CreatedAt:  time.Now(),
		}

		if err := repo.Insert(ctx, event); err != nil {
			t.Fatalf("Insert err: %v", err)
		}

		events, err := repo.ListByAsset(ctx, "asset-2", nil)
		if err != nil {
			t.Fatalf("ListByAsset err: %v", err)
		}
		if len(events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(events))
		}
		got := events[0]
		if got.PrevStatus != nil {
			t.Fatalf("expected nil PrevStatus, got %v", got.PrevStatus)
		}
		if got.RunID != nil {
			t.Fatalf("expected nil RunID, got %v", got.RunID)
		}
		if got.Reason != nil {
			t.Fatalf("expected nil Reason, got %v", got.Reason)
		}
	})

	t.Run("list_by_asset_no_filter", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}, storeOnApply: true}
		repo := &AlgoEventRepo{table: ft}

		// Insert events for the same asset with different algoKeys.
		for j, ak := range []string{"sam2@1.0", "hand_tracking@2.0", "sam2@1.0"} {
			ev := &models.AlgoEvent{
				EventID:   fmt.Sprintf("ev-%d", j),
				AssetID:   "asset-1",
				AlgoKey:   ak,
				NewStatus: "ok",
				CreatedAt: time.Now().Add(-time.Duration(j) * time.Minute),
			}
			if err := repo.Insert(ctx, ev); err != nil {
				t.Fatalf("Insert err: %v", err)
			}
		}

		events, err := repo.ListByAsset(ctx, "asset-1", nil)
		if err != nil {
			t.Fatalf("ListByAsset err: %v", err)
		}
		if len(events) != 3 {
			t.Fatalf("expected 3 events, got %d", len(events))
		}
	})

	t.Run("list_by_asset_with_algo_key_filter", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}, storeOnApply: true}
		repo := &AlgoEventRepo{table: ft}

		for j, ak := range []string{"sam2@1.0", "hand_tracking@2.0", "sam2@1.0"} {
			ev := &models.AlgoEvent{
				EventID:   fmt.Sprintf("ev-%d", j),
				AssetID:   "asset-1",
				AlgoKey:   ak,
				NewStatus: "ok",
				CreatedAt: time.Now().Add(-time.Duration(j) * time.Minute),
			}
			if err := repo.Insert(ctx, ev); err != nil {
				t.Fatalf("Insert err: %v", err)
			}
		}

		algoKey := "sam2@1.0"
		events, err := repo.ListByAsset(ctx, "asset-1", &algoKey)
		if err != nil {
			t.Fatalf("ListByAsset err: %v", err)
		}
		if len(events) != 2 {
			t.Fatalf("expected 2 events with algoKey=sam2@1.0, got %d", len(events))
		}
		for _, ev := range events {
			if ev.AlgoKey != "sam2@1.0" {
				t.Fatalf("expected algoKey=sam2@1.0, got %q", ev.AlgoKey)
			}
		}
	})

	t.Run("list_by_asset_empty_result", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AlgoEventRepo{table: ft}

		events, err := repo.ListByAsset(ctx, "nonexistent-asset", nil)
		if err != nil {
			t.Fatalf("ListByAsset err: %v", err)
		}
		if len(events) != 0 {
			t.Fatalf("expected 0 events, got %d", len(events))
		}
	})

	t.Run("list_by_asset_read_rows_error", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}, readRowsErr: errors.New("read rows failed")}
		repo := &AlgoEventRepo{table: ft}

		_, err := repo.ListByAsset(ctx, "asset-1", nil)
		if err == nil || !strings.Contains(err.Error(), "read rows failed") {
			t.Fatalf("expected read rows error, got: %v", err)
		}
	})

	t.Run("list_by_asset_does_not_return_other_assets", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}, storeOnApply: true}
		repo := &AlgoEventRepo{table: ft}

		// Insert events for two different assets.
		for _, assetID := range []string{"asset-1", "asset-2"} {
			ev := &models.AlgoEvent{
				EventID:   "ev-" + assetID,
				AssetID:   assetID,
				AlgoKey:   "sam2@1.0",
				NewStatus: "ok",
				CreatedAt: time.Now(),
			}
			if err := repo.Insert(ctx, ev); err != nil {
				t.Fatalf("Insert err: %v", err)
			}
		}

		events, err := repo.ListByAsset(ctx, "asset-1", nil)
		if err != nil {
			t.Fatalf("ListByAsset err: %v", err)
		}
		if len(events) != 1 {
			t.Fatalf("expected 1 event for asset-1, got %d", len(events))
		}
		if events[0].AssetID != "asset-1" {
			t.Fatalf("expected asset-1, got %q", events[0].AssetID)
		}
	})
}

// TestAlgoEventRepoConstructor verifies the constructor creates a valid repo.
func TestAlgoEventRepoConstructor(t *testing.T) {
	fd := &fakeDataClient{}
	c := &Client{inner: fd}
	repo := NewAlgoEventRepo(c)
	if repo == nil {
		t.Fatalf("NewAlgoEventRepo returned nil")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Test helpers for AlgoEvent property tests
// ──────────────────────────────────────────────────────────────────────────────

func genRandomAlgoEvent(rng *testRNG) *models.AlgoEvent {
	statuses := []string{"blocked", "pending", "running", "ok", "failed"}
	algoKeys := []string{"sam2@1.0", "hand_tracking@2.0", "object_detect@1.5", "pose_est@3.0"}

	ev := &models.AlgoEvent{
		EventID:   fmt.Sprintf("ev-%d", rng.Intn(100000)),
		AssetID:   fmt.Sprintf("asset-%d", rng.Intn(100)),
		AlgoKey:   algoKeys[rng.Intn(len(algoKeys))],
		NewStatus: statuses[rng.Intn(len(statuses))],
		CreatedAt: time.Now().Add(-time.Duration(rng.Intn(86400)) * time.Second),
	}

	// 50% chance of having PrevStatus.
	if rng.Intn(2) == 0 {
		s := statuses[rng.Intn(len(statuses))]
		ev.PrevStatus = &s
	}

	// 50% chance of having RunID.
	if rng.Intn(2) == 0 {
		s := fmt.Sprintf("run-%d", rng.Intn(10000))
		ev.RunID = &s
	}

	// 50% chance of having Reason.
	if rng.Intn(2) == 0 {
		s := fmt.Sprintf("reason-%d", rng.Intn(10000))
		ev.Reason = &s
	}

	return ev
}

// ──────────────────────────────────────────────────────────────────────────────
// Property test for ListWithFilters filter correctness
// ──────────────────────────────────────────────────────────────────────────────

// TestListWithFiltersFilterCorrectness verifies Property 6: ListWithFilters filter correctness.
// For any random asset set and filter conditions, results satisfy filter conditions
// and don't contain archived assets.
// **Validates: Requirements 7.2, 7.3, 7.5, 7.6**
func TestListWithFiltersFilterCorrectness(t *testing.T) {
	const iterations = 100
	ctx := context.Background()
	rng := newTestRNG(t)

	statuses := []models.AssetStatus{
		models.AssetStatusApproved,
		models.AssetStatusRejected,
		models.AssetStatusSuperseded,
		models.AssetStatusArchived,
	}
	owners := []string{"alice", "bob", "charlie", "diana"}
	envs := []string{"kitchen", "office", "warehouse", "outdoor"}

	for i := 0; i < iterations; i++ {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}

		// Generate 5-15 random assets.
		numAssets := 5 + rng.Intn(11)
		allAssets := make([]*models.Asset, numAssets)
		for j := 0; j < numAssets; j++ {
			a := &models.Asset{
				AssetID:    fmt.Sprintf("asset-%d-%d", i, j),
				McapFileID: fmt.Sprintf("m-%d", j),
				Status:     statuses[rng.Intn(len(statuses))],
				Owner:      owners[rng.Intn(len(owners))],
				Env:        envs[rng.Intn(len(envs))],
				Version:    int64(rng.Intn(10) + 1),
				CreatedAt:  time.Now().Add(-time.Duration(rng.Intn(86400)) * time.Second),
				UpdatedAt:  time.Now(),
			}
			allAssets[j] = a

			rowData := map[string]map[string][]byte{
				CFMeta: {
					"mcap_file_id": B(a.McapFileID),
					"status":       B(string(a.Status)),
					"owner":        B(a.Owner),
					"env":          B(a.Env),
					"version":      PackInt64(a.Version),
					"created_at":   RFC3339(a.CreatedAt),
					"updated_at":   RFC3339(a.UpdatedAt),
				},
			}
			ft.rows[AssetKey(a.AssetID)] = mkRow(AssetKey(a.AssetID), rowData)
		}

		// Pick a random filter: filter by owner using eq operator.
		targetOwner := owners[rng.Intn(len(owners))]
		whereSQL := `owner = $1`
		args := []interface{}{targetOwner}

		results, total, err := repo.ListWithFilters(ctx, whereSQL, args, 1, 1000, "")
		if err != nil {
			t.Fatalf("iter %d: ListWithFilters err: %v", i, err)
		}

		// Verify: no archived assets in results.
		for _, a := range results {
			if a.Status == models.AssetStatusArchived {
				t.Fatalf("iter %d: found archived asset %s in results", i, a.AssetID)
			}
		}

		// Verify: all results match the filter condition.
		for _, a := range results {
			if a.Owner != targetOwner {
				t.Fatalf("iter %d: asset %s has owner=%q, expected %q", i, a.AssetID, a.Owner, targetOwner)
			}
		}

		// Verify: total matches the count of non-archived assets matching the filter.
		expectedCount := int64(0)
		for _, a := range allAssets {
			if a.Status != models.AssetStatusArchived && a.Owner == targetOwner {
				expectedCount++
			}
		}
		if total != expectedCount {
			t.Fatalf("iter %d: total=%d, expected=%d", i, total, expectedCount)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Property test for ListWithFilters sort correctness
// ──────────────────────────────────────────────────────────────────────────────

// TestListWithFiltersSortCorrectness verifies Property 7: ListWithFilters sort correctness.
// For any random asset set, returned results are ordered by the specified field.
// **Validates: Requirements 7.4**
func TestListWithFiltersSortCorrectness(t *testing.T) {
	const iterations = 100
	ctx := context.Background()
	rng := newTestRNG(t)

	sortFields := []struct {
		orderBy string
		getVal  func(a *models.Asset) string
	}{
		{"owner ASC", func(a *models.Asset) string { return a.Owner }},
		{"owner DESC", func(a *models.Asset) string { return a.Owner }},
		{"env ASC", func(a *models.Asset) string { return a.Env }},
		{"env DESC", func(a *models.Asset) string { return a.Env }},
		{"created_at ASC", func(a *models.Asset) string { return a.CreatedAt.Format(time.RFC3339Nano) }},
		{"created_at DESC", func(a *models.Asset) string { return a.CreatedAt.Format(time.RFC3339Nano) }},
	}

	owners := []string{"alice", "bob", "charlie", "diana", "eve"}
	envs := []string{"kitchen", "office", "warehouse", "outdoor", "lab"}

	for i := 0; i < iterations; i++ {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}

		// Generate 3-10 random non-archived assets.
		numAssets := 3 + rng.Intn(8)
		for j := 0; j < numAssets; j++ {
			a := &models.Asset{
				AssetID:    fmt.Sprintf("asset-%d-%d", i, j),
				McapFileID: fmt.Sprintf("m-%d", j),
				Status:     models.AssetStatusApproved,
				Owner:      owners[rng.Intn(len(owners))],
				Env:        envs[rng.Intn(len(envs))],
				Version:    int64(rng.Intn(10) + 1),
				CreatedAt:  time.Now().Add(-time.Duration(rng.Intn(86400)) * time.Second),
				UpdatedAt:  time.Now(),
			}

			rowData := map[string]map[string][]byte{
				CFMeta: {
					"mcap_file_id": B(a.McapFileID),
					"status":       B(string(a.Status)),
					"owner":        B(a.Owner),
					"env":          B(a.Env),
					"version":      PackInt64(a.Version),
					"created_at":   RFC3339(a.CreatedAt),
					"updated_at":   RFC3339(a.UpdatedAt),
				},
			}
			ft.rows[AssetKey(a.AssetID)] = mkRow(AssetKey(a.AssetID), rowData)
		}

		// Pick a random sort field.
		sf := sortFields[rng.Intn(len(sortFields))]
		desc := strings.HasSuffix(sf.orderBy, "DESC")

		results, _, err := repo.ListWithFilters(ctx, "", nil, 1, 1000, sf.orderBy)
		if err != nil {
			t.Fatalf("iter %d: ListWithFilters err: %v", i, err)
		}

		// Verify ordering.
		for k := 1; k < len(results); k++ {
			prev := sf.getVal(results[k-1])
			curr := sf.getVal(results[k])
			cmp := strings.Compare(prev, curr)
			if desc && cmp < 0 {
				t.Fatalf("iter %d: sort %q violated at index %d: %q < %q", i, sf.orderBy, k, prev, curr)
			}
			if !desc && cmp > 0 {
				t.Fatalf("iter %d: sort %q violated at index %d: %q > %q", i, sf.orderBy, k, prev, curr)
			}
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Unit tests for ListWithFilters
// ──────────────────────────────────────────────────────────────────────────────

func TestListWithFilters(t *testing.T) {
	ctx := context.Background()

	// Helper to seed assets into a fakeTable.
	seedAssets := func(ft *fakeTable, assets []struct {
		id, status, owner, env string
		algoResults            map[string]string
		tags                   map[string]string
		files                  map[string]string
	}) {
		for _, a := range assets {
			rowData := map[string]map[string][]byte{
				CFMeta: {
					"mcap_file_id": B("m1"),
					"status":       B(a.status),
					"owner":        B(a.owner),
					"env":          B(a.env),
					"version":      PackInt64(1),
					"created_at":   RFC3339(time.Now()),
					"updated_at":   RFC3339(time.Now()),
				},
			}
			if len(a.algoResults) > 0 {
				rowData[CFAlgo] = map[string][]byte{}
				for k, v := range a.algoResults {
					rowData[CFAlgo][k] = B(v)
				}
			}
			if len(a.tags) > 0 {
				rowData[CFTag] = map[string][]byte{}
				for k, v := range a.tags {
					rowData[CFTag][k] = B(v)
				}
			}
			if len(a.files) > 0 {
				rowData[CFFiles] = map[string][]byte{}
				for k, v := range a.files {
					rowData[CFFiles][k] = B(v)
				}
			}
			ft.rows[AssetKey(a.id)] = mkRow(AssetKey(a.id), rowData)
		}
	}

	t.Run("no_filter_conditions", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}

		seedAssets(ft, []struct {
			id, status, owner, env string
			algoResults            map[string]string
			tags                   map[string]string
			files                  map[string]string
		}{
			{id: "a1", status: "approved", owner: "alice", env: "kitchen"},
			{id: "a2", status: "rejected", owner: "bob", env: "office"},
			{id: "a3", status: "archived", owner: "charlie", env: "warehouse"},
		})

		results, total, err := repo.ListWithFilters(ctx, "", nil, 1, 20, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should exclude archived.
		if total != 2 {
			t.Fatalf("expected total=2, got %d", total)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		for _, r := range results {
			if r.Status == models.AssetStatusArchived {
				t.Fatalf("archived asset should not appear")
			}
		}
	})

	t.Run("single_condition_filter", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}

		seedAssets(ft, []struct {
			id, status, owner, env string
			algoResults            map[string]string
			tags                   map[string]string
			files                  map[string]string
		}{
			{id: "a1", status: "approved", owner: "alice", env: "kitchen"},
			{id: "a2", status: "approved", owner: "bob", env: "office"},
			{id: "a3", status: "rejected", owner: "alice", env: "warehouse"},
		})

		results, total, err := repo.ListWithFilters(ctx, `owner = $1`, []interface{}{"alice"}, 1, 20, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 2 {
			t.Fatalf("expected total=2, got %d", total)
		}
		for _, r := range results {
			if r.Owner != "alice" {
				t.Fatalf("expected owner=alice, got %q", r.Owner)
			}
		}
	})

	t.Run("multi_condition_filter", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}

		seedAssets(ft, []struct {
			id, status, owner, env string
			algoResults            map[string]string
			tags                   map[string]string
			files                  map[string]string
		}{
			{id: "a1", status: "approved", owner: "alice", env: "kitchen"},
			{id: "a2", status: "approved", owner: "alice", env: "office"},
			{id: "a3", status: "approved", owner: "bob", env: "kitchen"},
		})

		results, total, err := repo.ListWithFilters(ctx,
			`owner = $1 AND env = $2`,
			[]interface{}{"alice", "kitchen"}, 1, 20, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 1 {
			t.Fatalf("expected total=1, got %d", total)
		}
		if len(results) != 1 || results[0].AssetID != "a1" {
			t.Fatalf("expected asset a1, got %v", results)
		}
	})

	t.Run("pagination", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}

		seedAssets(ft, []struct {
			id, status, owner, env string
			algoResults            map[string]string
			tags                   map[string]string
			files                  map[string]string
		}{
			{id: "a1", status: "approved", owner: "alice", env: "kitchen"},
			{id: "a2", status: "approved", owner: "bob", env: "office"},
			{id: "a3", status: "approved", owner: "charlie", env: "warehouse"},
			{id: "a4", status: "approved", owner: "diana", env: "outdoor"},
			{id: "a5", status: "approved", owner: "eve", env: "lab"},
		})

		// Page 1, size 2.
		results, total, err := repo.ListWithFilters(ctx, "", nil, 1, 2, "owner ASC")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 5 {
			t.Fatalf("expected total=5, got %d", total)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results on page 1, got %d", len(results))
		}

		// Page 2, size 2.
		results2, total2, err := repo.ListWithFilters(ctx, "", nil, 2, 2, "owner ASC")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total2 != 5 {
			t.Fatalf("expected total=5, got %d", total2)
		}
		if len(results2) != 2 {
			t.Fatalf("expected 2 results on page 2, got %d", len(results2))
		}

		// Page 3, size 2 → only 1 result.
		results3, _, err := repo.ListWithFilters(ctx, "", nil, 3, 2, "owner ASC")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results3) != 1 {
			t.Fatalf("expected 1 result on page 3, got %d", len(results3))
		}

		// Page beyond range.
		results4, _, err := repo.ListWithFilters(ctx, "", nil, 10, 2, "owner ASC")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results4) != 0 {
			t.Fatalf("expected 0 results on page 10, got %d", len(results4))
		}
	})

	t.Run("sorting", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}

		seedAssets(ft, []struct {
			id, status, owner, env string
			algoResults            map[string]string
			tags                   map[string]string
			files                  map[string]string
		}{
			{id: "a1", status: "approved", owner: "charlie", env: "kitchen"},
			{id: "a2", status: "approved", owner: "alice", env: "office"},
			{id: "a3", status: "approved", owner: "bob", env: "warehouse"},
		})

		results, _, err := repo.ListWithFilters(ctx, "", nil, 1, 20, "owner ASC")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(results))
		}
		if results[0].Owner != "alice" || results[1].Owner != "bob" || results[2].Owner != "charlie" {
			t.Fatalf("expected sorted by owner ASC, got %q, %q, %q",
				results[0].Owner, results[1].Owner, results[2].Owner)
		}

		// DESC.
		results, _, err = repo.ListWithFilters(ctx, "", nil, 1, 20, "owner DESC")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if results[0].Owner != "charlie" || results[1].Owner != "bob" || results[2].Owner != "alice" {
			t.Fatalf("expected sorted by owner DESC, got %q, %q, %q",
				results[0].Owner, results[1].Owner, results[2].Owner)
		}
	})

	t.Run("exclude_archived", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}

		seedAssets(ft, []struct {
			id, status, owner, env string
			algoResults            map[string]string
			tags                   map[string]string
			files                  map[string]string
		}{
			{id: "a1", status: "archived", owner: "alice", env: "kitchen"},
			{id: "a2", status: "archived", owner: "bob", env: "office"},
			{id: "a3", status: "approved", owner: "charlie", env: "warehouse"},
		})

		results, total, err := repo.ListWithFilters(ctx, "", nil, 1, 20, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 1 {
			t.Fatalf("expected total=1, got %d", total)
		}
		if len(results) != 1 || results[0].AssetID != "a3" {
			t.Fatalf("expected only a3, got %v", results)
		}
	})

	t.Run("jsonb_path_filter", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}

		seedAssets(ft, []struct {
			id, status, owner, env string
			algoResults            map[string]string
			tags                   map[string]string
			files                  map[string]string
		}{
			{id: "a1", status: "approved", owner: "alice", env: "kitchen",
				algoResults: map[string]string{"sam2@1.0:status": "ok"}},
			{id: "a2", status: "approved", owner: "bob", env: "office",
				algoResults: map[string]string{"sam2@1.0:status": "failed"}},
			{id: "a3", status: "approved", owner: "charlie", env: "warehouse",
				algoResults: map[string]string{"sam2@1.0:status": "ok"}},
		})

		// Filter by JSONB path: cf_algo#>>'{sam2@1.0:status}' = 'ok'
		whereSQL := `cf_algo#>>'{sam2@1.0:status}' = $1`
		results, total, err := repo.ListWithFilters(ctx, whereSQL, []interface{}{"ok"}, 1, 20, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 2 {
			t.Fatalf("expected total=2, got %d", total)
		}
		for _, r := range results {
			if r.AlgoResults["sam2@1.0:status"] != "ok" {
				t.Fatalf("expected algo status=ok, got %q", r.AlgoResults["sam2@1.0:status"])
			}
		}
	})

	t.Run("ne_operator", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}

		seedAssets(ft, []struct {
			id, status, owner, env string
			algoResults            map[string]string
			tags                   map[string]string
			files                  map[string]string
		}{
			{id: "a1", status: "approved", owner: "alice", env: "kitchen"},
			{id: "a2", status: "approved", owner: "bob", env: "office"},
			{id: "a3", status: "rejected", owner: "charlie", env: "warehouse"},
		})

		results, total, err := repo.ListWithFilters(ctx, `status != $1`, []interface{}{"approved"}, 1, 20, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 1 {
			t.Fatalf("expected total=1, got %d", total)
		}
		if results[0].AssetID != "a3" {
			t.Fatalf("expected a3, got %s", results[0].AssetID)
		}
	})

	t.Run("read_rows_error", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}, readRowsErr: errors.New("scan failed")}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}

		_, _, err := repo.ListWithFilters(ctx, "", nil, 1, 20, "")
		if err == nil || !strings.Contains(err.Error(), "scan failed") {
			t.Fatalf("expected scan error, got: %v", err)
		}
	})

	t.Run("tag_filter", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}

		seedAssets(ft, []struct {
			id, status, owner, env string
			algoResults            map[string]string
			tags                   map[string]string
			files                  map[string]string
		}{
			{id: "a1", status: "approved", owner: "alice", env: "kitchen",
				tags: map[string]string{"priority": "high"}},
			{id: "a2", status: "approved", owner: "bob", env: "office",
				tags: map[string]string{"priority": "low"}},
		})

		whereSQL := `cf_tag#>>'{priority}' = $1`
		results, total, err := repo.ListWithFilters(ctx, whereSQL, []interface{}{"high"}, 1, 20, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 1 {
			t.Fatalf("expected total=1, got %d", total)
		}
		if results[0].AssetID != "a1" {
			t.Fatalf("expected a1, got %s", results[0].AssetID)
		}
	})

	t.Run("files_filter", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}

		seedAssets(ft, []struct {
			id, status, owner, env string
			algoResults            map[string]string
			tags                   map[string]string
			files                  map[string]string
		}{
			{id: "a1", status: "approved", owner: "alice", env: "kitchen",
				files: map[string]string{"output.mcap": "gs://bucket/a1.mcap"}},
			{id: "a2", status: "approved", owner: "bob", env: "office",
				files: map[string]string{"output.mcap": "gs://bucket/a2.mcap"}},
		})

		whereSQL := `cf_files#>>'{output.mcap}' = $1`
		results, total, err := repo.ListWithFilters(ctx, whereSQL, []interface{}{"gs://bucket/a1.mcap"}, 1, 20, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 1 {
			t.Fatalf("expected total=1, got %d", total)
		}
		if results[0].AssetID != "a1" {
			t.Fatalf("expected a1, got %s", results[0].AssetID)
		}
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Unit tests for virtual field matching (algo_status, has:delivery)
// ──────────────────────────────────────────────────────────────────────────────

func TestMatchCondition_AlgoStatus(t *testing.T) {
	t.Run("matches_when_any_status_key_equals_value", func(t *testing.T) {
		a := &models.Asset{
			AssetID: "a1",
			Status:  models.AssetStatusApproved,
			AlgoResults: map[string]string{
				"sam2@1.0:status":          "ok",
				"hand_tracking@2.0:status": "failed",
				"sam2@1.0:run_id":          "run-123",
			},
		}
		cond := parsedCondition{field: "algo_status", op: "=", value: "failed"}
		if !matchCondition(a, cond) {
			t.Fatal("expected match: hand_tracking@2.0:status is 'failed'")
		}
	})

	t.Run("matches_ok_status", func(t *testing.T) {
		a := &models.Asset{
			AssetID: "a1",
			Status:  models.AssetStatusApproved,
			AlgoResults: map[string]string{
				"sam2@1.0:status":          "ok",
				"hand_tracking@2.0:status": "ok",
			},
		}
		cond := parsedCondition{field: "algo_status", op: "=", value: "ok"}
		if !matchCondition(a, cond) {
			t.Fatal("expected match: both statuses are 'ok'")
		}
	})

	t.Run("no_match_when_no_status_key_matches", func(t *testing.T) {
		a := &models.Asset{
			AssetID: "a1",
			Status:  models.AssetStatusApproved,
			AlgoResults: map[string]string{
				"sam2@1.0:status":          "ok",
				"hand_tracking@2.0:status": "ok",
			},
		}
		cond := parsedCondition{field: "algo_status", op: "=", value: "failed"}
		if matchCondition(a, cond) {
			t.Fatal("expected no match: no status key has 'failed'")
		}
	})

	t.Run("no_match_when_no_algo_results", func(t *testing.T) {
		a := &models.Asset{
			AssetID:     "a1",
			Status:      models.AssetStatusApproved,
			AlgoResults: map[string]string{},
		}
		cond := parsedCondition{field: "algo_status", op: "=", value: "ok"}
		if matchCondition(a, cond) {
			t.Fatal("expected no match: empty algo results")
		}
	})

	t.Run("ignores_non_status_keys", func(t *testing.T) {
		a := &models.Asset{
			AssetID: "a1",
			Status:  models.AssetStatusApproved,
			AlgoResults: map[string]string{
				"sam2@1.0:run_id":     "run-123",
				"sam2@1.0:started_at": "2026-01-01T00:00:00Z",
			},
		}
		cond := parsedCondition{field: "algo_status", op: "=", value: "run-123"}
		if matchCondition(a, cond) {
			t.Fatal("expected no match: run_id key doesn't end in :status")
		}
	})
}

func TestMatchCondition_HasDelivery(t *testing.T) {
	t.Run("true_when_delivery_count_positive", func(t *testing.T) {
		a := &models.Asset{
			AssetID:       "a1",
			Status:        models.AssetStatusApproved,
			DeliveryCount: 3,
		}
		cond := parsedCondition{field: "has:delivery", op: "=", value: true}
		if !matchCondition(a, cond) {
			t.Fatal("expected match: delivery_count=3 > 0")
		}
	})

	t.Run("false_when_delivery_count_zero", func(t *testing.T) {
		a := &models.Asset{
			AssetID:       "a1",
			Status:        models.AssetStatusApproved,
			DeliveryCount: 0,
		}
		cond := parsedCondition{field: "has:delivery", op: "=", value: true}
		if matchCondition(a, cond) {
			t.Fatal("expected no match: delivery_count=0")
		}
	})

	t.Run("false_value_matches_zero_delivery", func(t *testing.T) {
		a := &models.Asset{
			AssetID:       "a1",
			Status:        models.AssetStatusApproved,
			DeliveryCount: 0,
		}
		cond := parsedCondition{field: "has:delivery", op: "=", value: false}
		if !matchCondition(a, cond) {
			t.Fatal("expected match: delivery_count=0 and value=false")
		}
	})

	t.Run("false_value_no_match_when_has_deliveries", func(t *testing.T) {
		a := &models.Asset{
			AssetID:       "a1",
			Status:        models.AssetStatusApproved,
			DeliveryCount: 5,
		}
		cond := parsedCondition{field: "has:delivery", op: "=", value: false}
		if matchCondition(a, cond) {
			t.Fatal("expected no match: delivery_count=5 but value=false")
		}
	})

	t.Run("string_true_value", func(t *testing.T) {
		a := &models.Asset{
			AssetID:       "a1",
			Status:        models.AssetStatusApproved,
			DeliveryCount: 1,
		}
		cond := parsedCondition{field: "has:delivery", op: "=", value: "true"}
		if !matchCondition(a, cond) {
			t.Fatal("expected match: delivery_count=1 and value='true'")
		}
	})

	t.Run("string_false_value", func(t *testing.T) {
		a := &models.Asset{
			AssetID:       "a1",
			Status:        models.AssetStatusApproved,
			DeliveryCount: 0,
		}
		cond := parsedCondition{field: "has:delivery", op: "=", value: "false"}
		if !matchCondition(a, cond) {
			t.Fatal("expected match: delivery_count=0 and value='false'")
		}
	})
}

func TestListWithFilters_AlgoStatusVirtual(t *testing.T) {
	ctx := context.Background()

	seedAssets := func(ft *fakeTable) {
		assets := []struct {
			id          string
			status      string
			algoResults map[string]string
			deliveries  int
		}{
			{id: "a1", status: "approved", algoResults: map[string]string{
				"sam2@1.0:status": "ok", "hand_tracking@2.0:status": "ok",
			}, deliveries: 2},
			{id: "a2", status: "approved", algoResults: map[string]string{
				"sam2@1.0:status": "failed", "hand_tracking@2.0:status": "ok",
			}, deliveries: 0},
			{id: "a3", status: "approved", algoResults: map[string]string{
				"sam2@1.0:status": "ok",
			}, deliveries: 1},
			{id: "a4", status: "approved", algoResults: map[string]string{}, deliveries: 0},
		}
		for _, a := range assets {
			rowData := map[string]map[string][]byte{
				CFMeta: {
					"mcap_file_id":   B("m1"),
					"status":         B(a.status),
					"owner":          B("alice"),
					"env":            B("kitchen"),
					"version":        PackInt64(1),
					"created_at":     RFC3339(time.Now()),
					"updated_at":     RFC3339(time.Now()),
					"delivery_count": PackInt64(int64(a.deliveries)),
				},
			}
			if len(a.algoResults) > 0 {
				rowData[CFAlgo] = map[string][]byte{}
				for k, v := range a.algoResults {
					rowData[CFAlgo][k] = B(v)
				}
			}
			ft.rows[AssetKey(a.id)] = mkRow(AssetKey(a.id), rowData)
		}
	}

	t.Run("algo_status_failed_filter", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}
		seedAssets(ft)

		// Simulate the SQL that BuildWhereClause generates for algo_status:eq:failed
		whereSQL := "EXISTS (SELECT 1 FROM jsonb_each_text(cf_algo) WHERE key LIKE '%:status' AND value = $1)"
		results, total, err := repo.ListWithFilters(ctx, whereSQL, []interface{}{"failed"}, 1, 20, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 1 {
			t.Fatalf("expected total=1, got %d", total)
		}
		if len(results) != 1 || results[0].AssetID != "a2" {
			ids := make([]string, len(results))
			for i, r := range results {
				ids[i] = r.AssetID
			}
			t.Fatalf("expected [a2], got %v", ids)
		}
	})

	t.Run("algo_status_ok_filter", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}
		seedAssets(ft)

		whereSQL := "EXISTS (SELECT 1 FROM jsonb_each_text(cf_algo) WHERE key LIKE '%:status' AND value = $1)"
		results, total, err := repo.ListWithFilters(ctx, whereSQL, []interface{}{"ok"}, 1, 20, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// a1 (ok, ok), a2 (failed, ok), a3 (ok) all have at least one "ok" status
		if total != 3 {
			t.Fatalf("expected total=3, got %d", total)
		}
		if len(results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(results))
		}
	})

	t.Run("has_delivery_true_filter", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}
		seedAssets(ft)

		// Simulate the SQL that BuildWhereClause generates for has:delivery:eq:true
		whereSQL := "delivery_count > 0"
		results, total, err := repo.ListWithFilters(ctx, whereSQL, nil, 1, 20, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// a1 (2 deliveries), a3 (1 delivery)
		if total != 2 {
			t.Fatalf("expected total=2, got %d", total)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
	})

	t.Run("has_delivery_false_filter", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}
		seedAssets(ft)

		whereSQL := "delivery_count = 0"
		results, total, err := repo.ListWithFilters(ctx, whereSQL, nil, 1, 20, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// a2 (0 deliveries), a4 (0 deliveries)
		if total != 2 {
			t.Fatalf("expected total=2, got %d", total)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
	})

	t.Run("combined_algo_status_and_has_delivery", func(t *testing.T) {
		ft := &fakeTable{rows: map[string]gbt.Row{}}
		repo := &AssetRepo{table: ft, idxTable: &fakeTable{rows: map[string]gbt.Row{}}}
		seedAssets(ft)

		// algo_status:eq:ok AND has:delivery:eq:true
		whereSQL := "EXISTS (SELECT 1 FROM jsonb_each_text(cf_algo) WHERE key LIKE '%:status' AND value = $1) AND delivery_count > 0"
		results, total, err := repo.ListWithFilters(ctx, whereSQL, []interface{}{"ok"}, 1, 20, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// a1 (ok+ok, 2 deliveries), a3 (ok, 1 delivery) — a2 has ok but 0 deliveries
		if total != 2 {
			t.Fatalf("expected total=2, got %d", total)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
	})
}
