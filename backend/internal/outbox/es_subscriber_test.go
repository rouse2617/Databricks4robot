package outbox

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/elasticsearch"
	"github.com/CyberOrigin2077/cyber-databrew/internal/metrics"
)

type stubSearchBuilder struct {
	doc map[string]any
	ok  bool
	err error
}

func (s stubSearchBuilder) Build(context.Context, string) (map[string]any, bool, error) {
	return s.doc, s.ok, s.err
}

// recordingBuilder tracks Build calls per assetID and returns per-asset
// configured results so multi-asset / dedupe / delete tests can drive the
// handler precisely.
type recordingBuilder struct {
	mu      sync.Mutex
	calls   []string
	results map[string]struct {
		doc map[string]any
		ok  bool
		err error
	}
}

func newRecordingBuilder() *recordingBuilder {
	return &recordingBuilder{
		results: map[string]struct {
			doc map[string]any
			ok  bool
			err error
		}{},
	}
}

func (b *recordingBuilder) setOK(assetID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.results[assetID] = struct {
		doc map[string]any
		ok  bool
		err error
	}{
		doc: map[string]any{"asset_id": assetID},
		ok:  true,
	}
}

func (b *recordingBuilder) setMissing(assetID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.results[assetID] = struct {
		doc map[string]any
		ok  bool
		err error
	}{ok: false}
}

func (b *recordingBuilder) Build(_ context.Context, assetID string) (map[string]any, bool, error) {
	b.mu.Lock()
	b.calls = append(b.calls, assetID)
	r, ok := b.results[assetID]
	b.mu.Unlock()
	if !ok {
		return nil, false, nil
	}
	return r.doc, r.ok, r.err
}

func (b *recordingBuilder) buildCalls() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]string, len(b.calls))
	copy(out, b.calls)
	return out
}

// bulkRequest models a single observed _bulk POST. ItemIDs preserves the
// per-doc order written into the ndjson body; Versions is parallel to ItemIDs.
type bulkRequest struct {
	ItemIDs  []string
	Versions []int64
}

// bulkRecorder is a fake Elasticsearch server that records every _bulk and
// DELETE call. Each _bulk response is generated from a per-test responder.
type bulkRecorder struct {
	mu      sync.Mutex
	bulks   []bulkRequest
	deletes []string
}

func (r *bulkRecorder) snapshotBulks() []bulkRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]bulkRequest, len(r.bulks))
	copy(out, r.bulks)
	return out
}

func (r *bulkRecorder) snapshotDeletes() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.deletes))
	copy(out, r.deletes)
	return out
}

// parseBulkBody reads the ndjson sent to /_bulk and returns the per-line
// action metadata in submission order.
func parseBulkBody(t *testing.T, body io.Reader) bulkRequest {
	t.Helper()
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	br := bulkRequest{}
	expectAction := true
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if expectAction {
			var meta struct {
				Index struct {
					ID      string `json:"_id"`
					Version int64  `json:"version"`
				} `json:"index"`
			}
			if err := json.Unmarshal([]byte(line), &meta); err != nil {
				t.Fatalf("parse bulk meta: %v: %s", err, line)
			}
			br.ItemIDs = append(br.ItemIDs, meta.Index.ID)
			br.Versions = append(br.Versions, meta.Index.Version)
			expectAction = false
		} else {
			expectAction = true
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan bulk body: %v", err)
	}
	return br
}

// successBulkResponse renders a bulk response where every item succeeds.
func successBulkResponse(itemIDs []string) []byte {
	items := make([]map[string]any, 0, len(itemIDs))
	for _, id := range itemIDs {
		items = append(items, map[string]any{
			"index": map[string]any{"_id": id, "status": 200},
		})
	}
	body, _ := json.Marshal(map[string]any{"errors": false, "items": items})
	return body
}

// newRecordingESServer spins up an httptest.Server that captures every _bulk
// and DELETE call and lets the test inject the response per request.
func newRecordingESServer(t *testing.T, recorder *bulkRecorder, bulkRespond func(req bulkRequest) (status int, body []byte)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/_bulk") {
			req := parseBulkBody(t, r.Body)
			recorder.mu.Lock()
			recorder.bulks = append(recorder.bulks, req)
			recorder.mu.Unlock()

			status, body := http.StatusOK, successBulkResponse(req.ItemIDs)
			if bulkRespond != nil {
				if cs, cb := bulkRespond(req); cb != nil {
					status, body = cs, cb
				}
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = w.Write(body)
			return
		}
		if r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/_doc/") {
			parts := strings.Split(r.URL.Path, "/_doc/")
			id := parts[len(parts)-1]
			recorder.mu.Lock()
			recorder.deletes = append(recorder.deletes, id)
			recorder.mu.Unlock()
			w.WriteHeader(http.StatusOK)
			return
		}
		t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
	}))
}

// ---------- Single-event handler tests (legacy path) ----------

func TestESSubscriberHandleMessage_EmptyAssetNoop(t *testing.T) {
	s := &ESSubscriber{}
	if err := s.handleData(context.Background(), []byte(`{"event_seq":1}`)); err != nil {
		t.Fatalf("expected noop nil, got %v", err)
	}
}

func TestESSubscriberHandleMessage_DeleteOnMissingDoc(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/assets/_doc/a1") {
			w.WriteHeader(http.StatusOK)
			return
		}
		t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	s := &ESSubscriber{
		ES:      elasticsearch.New(server.URL, "assets", "", ""),
		Builder: stubSearchBuilder{ok: false},
	}
	if err := s.handleData(context.Background(), []byte(`{"event_seq":2,"asset_id":"a1"}`)); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestESSubscriberHandleMessage_ConflictAccepted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/_bulk") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
  "errors": true,
  "items": [
    {
      "index": {
        "_id": "a1",
        "status": 409,
        "error": { "type": "version_conflict_engine_exception", "reason": "conflict" }
      }
    }
  ]
}`))
			return
		}
		t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	s := &ESSubscriber{
		ES:      elasticsearch.New(server.URL, "assets", "", ""),
		Builder: stubSearchBuilder{ok: true, doc: map[string]any{"asset_id": "a1"}},
	}
	if err := s.handleData(context.Background(), []byte(`{"event_seq":3,"asset_id":"a1"}`)); err != nil {
		t.Fatalf("expected nil on conflict, got %v", err)
	}
}

// ---------- Batch handler tests ----------

func TestESSubscriberHandleBatch_MultiAssetSingleBulk(t *testing.T) {
	rec := &bulkRecorder{}
	server := newRecordingESServer(t, rec, nil)
	defer server.Close()

	builder := newRecordingBuilder()
	builder.setOK("a1")
	builder.setOK("a2")
	builder.setOK("a3")

	s := &ESSubscriber{
		ES:      elasticsearch.New(server.URL, "assets", "", ""),
		Builder: builder,
	}

	batch := [][]byte{
		[]byte(`{"event_seq":1,"asset_id":"a1"}`),
		[]byte(`{"event_seq":2,"asset_id":"a2"}`),
		[]byte(`{"event_seq":3,"asset_id":"a3"}`),
	}
	beforeBatchCount := readHistogramCount(t, metrics.OutboxSubscriberBatchSize)
	if err := s.handleBatch(context.Background(), batch); err != nil {
		t.Fatalf("handleBatch returned error: %v", err)
	}
	afterBatchCount := readHistogramCount(t, metrics.OutboxSubscriberBatchSize)
	if afterBatchCount-beforeBatchCount != 1 {
		t.Fatalf("expected batch histogram count +1, got before=%v after=%v", beforeBatchCount, afterBatchCount)
	}

	bulks := rec.snapshotBulks()
	if len(bulks) != 1 {
		t.Fatalf("expected exactly 1 bulk request, got %d", len(bulks))
	}
	if got := bulks[0].ItemIDs; !equalUnordered(got, []string{"a1", "a2", "a3"}) {
		t.Fatalf("bulk item ids mismatch: %v", got)
	}
	for i, v := range bulks[0].Versions {
		if v == 0 {
			t.Fatalf("bulk item %d missing external version: %#v", i, bulks[0])
		}
	}
	if len(rec.snapshotDeletes()) != 0 {
		t.Fatalf("expected no deletes, got %v", rec.snapshotDeletes())
	}
}

func TestESSubscriberHandleBatch_DedupeSameAssetKeepsLatestSeq(t *testing.T) {
	rec := &bulkRecorder{}
	server := newRecordingESServer(t, rec, nil)
	defer server.Close()

	builder := newRecordingBuilder()
	builder.setOK("a1")
	builder.setOK("a2")

	s := &ESSubscriber{
		ES:      elasticsearch.New(server.URL, "assets", "", ""),
		Builder: builder,
	}

	// Three events for a1 (seq 5, 7, 6) — only the latest seq=7 should be the
	// version sent. One event for a2 to ensure cross-asset coexistence.
	batch := [][]byte{
		[]byte(`{"event_seq":5,"asset_id":"a1"}`),
		[]byte(`{"event_seq":7,"asset_id":"a1"}`),
		[]byte(`{"event_seq":6,"asset_id":"a1"}`),
		[]byte(`{"event_seq":11,"asset_id":"a2"}`),
	}
	if err := s.handleBatch(context.Background(), batch); err != nil {
		t.Fatalf("handleBatch returned error: %v", err)
	}

	calls := builder.buildCalls()
	a1Calls := 0
	for _, c := range calls {
		if c == "a1" {
			a1Calls++
		}
	}
	if a1Calls != 1 {
		t.Fatalf("expected exactly 1 Build call for a1 (dedupe), got %d (all calls=%v)", a1Calls, calls)
	}

	bulks := rec.snapshotBulks()
	if len(bulks) != 1 {
		t.Fatalf("expected exactly 1 bulk request, got %d", len(bulks))
	}
	for i, id := range bulks[0].ItemIDs {
		if id == "a1" && bulks[0].Versions[i] != 7 {
			t.Fatalf("expected a1 version=7, got %d", bulks[0].Versions[i])
		}
		if id == "a2" && bulks[0].Versions[i] != 11 {
			t.Fatalf("expected a2 version=11, got %d", bulks[0].Versions[i])
		}
	}
}

func TestESSubscriberHandleBatch_ConflictTolerated(t *testing.T) {
	rec := &bulkRecorder{}
	server := newRecordingESServer(t, rec, func(req bulkRequest) (int, []byte) {
		// Return 409 for a1, 200 for the rest.
		items := make([]map[string]any, 0, len(req.ItemIDs))
		for _, id := range req.ItemIDs {
			if id == "a1" {
				items = append(items, map[string]any{
					"index": map[string]any{
						"_id":    id,
						"status": 409,
						"error": map[string]any{
							"type": "version_conflict_engine_exception", "reason": "conflict",
						},
					},
				})
				continue
			}
			items = append(items, map[string]any{
				"index": map[string]any{"_id": id, "status": 200},
			})
		}
		body, _ := json.Marshal(map[string]any{"errors": true, "items": items})
		return http.StatusOK, body
	})
	defer server.Close()

	builder := newRecordingBuilder()
	builder.setOK("a1")
	builder.setOK("a2")

	s := &ESSubscriber{
		ES:      elasticsearch.New(server.URL, "assets", "", ""),
		Builder: builder,
	}

	batch := [][]byte{
		[]byte(`{"event_seq":1,"asset_id":"a1"}`),
		[]byte(`{"event_seq":2,"asset_id":"a2"}`),
	}
	beforeConflict := readCounterValue(t, metrics.OutboxSubscriberESErrorsTotal.WithLabelValues("conflict"))
	if err := s.handleBatch(context.Background(), batch); err != nil {
		t.Fatalf("expected nil on 409, got %v", err)
	}
	afterConflict := readCounterValue(t, metrics.OutboxSubscriberESErrorsTotal.WithLabelValues("conflict"))
	if afterConflict-beforeConflict != 1 {
		t.Fatalf("expected conflict counter +1, got before=%v after=%v", beforeConflict, afterConflict)
	}
}

func TestESSubscriberHandleBatch_NonConflictFailureReturnsError(t *testing.T) {
	rec := &bulkRecorder{}
	server := newRecordingESServer(t, rec, func(req bulkRequest) (int, []byte) {
		items := make([]map[string]any, 0, len(req.ItemIDs))
		for _, id := range req.ItemIDs {
			items = append(items, map[string]any{
				"index": map[string]any{
					"_id":    id,
					"status": 500,
					"error": map[string]any{
						"type": "internal", "reason": "boom",
					},
				},
			})
		}
		body, _ := json.Marshal(map[string]any{"errors": true, "items": items})
		return http.StatusOK, body
	})
	defer server.Close()

	builder := newRecordingBuilder()
	builder.setOK("a1")

	s := &ESSubscriber{
		ES:      elasticsearch.New(server.URL, "assets", "", ""),
		Builder: builder,
	}

	batch := [][]byte{[]byte(`{"event_seq":1,"asset_id":"a1"}`)}
	if err := s.handleBatch(context.Background(), batch); err == nil {
		t.Fatalf("expected error on non-409 failure")
	}
}

func TestESSubscriberHandleBatch_MissingDocDeletePathHybrid(t *testing.T) {
	rec := &bulkRecorder{}
	server := newRecordingESServer(t, rec, nil)
	defer server.Close()

	builder := newRecordingBuilder()
	builder.setOK("a1")
	builder.setMissing("a2") // -> delete
	builder.setOK("a3")

	s := &ESSubscriber{
		ES:      elasticsearch.New(server.URL, "assets", "", ""),
		Builder: builder,
	}

	batch := [][]byte{
		[]byte(`{"event_seq":1,"asset_id":"a1"}`),
		[]byte(`{"event_seq":2,"asset_id":"a2"}`),
		[]byte(`{"event_seq":3,"asset_id":"a3"}`),
	}
	if err := s.handleBatch(context.Background(), batch); err != nil {
		t.Fatalf("handleBatch returned error: %v", err)
	}

	bulks := rec.snapshotBulks()
	if len(bulks) != 1 {
		t.Fatalf("expected exactly 1 bulk request, got %d", len(bulks))
	}
	if !equalUnordered(bulks[0].ItemIDs, []string{"a1", "a3"}) {
		t.Fatalf("bulk should only contain upserts a1, a3, got %v", bulks[0].ItemIDs)
	}

	dels := rec.snapshotDeletes()
	if !equalUnordered(dels, []string{"a2"}) {
		t.Fatalf("expected delete for a2, got %v", dels)
	}
}

func TestESSubscriberHandleBatch_SkipsEmptyAssetIDEntries(t *testing.T) {
	rec := &bulkRecorder{}
	server := newRecordingESServer(t, rec, nil)
	defer server.Close()

	builder := newRecordingBuilder()
	builder.setOK("a1")

	s := &ESSubscriber{
		ES:      elasticsearch.New(server.URL, "assets", "", ""),
		Builder: builder,
	}

	batch := [][]byte{
		[]byte(`{"event_seq":1}`),
		[]byte(`{"event_seq":2,"asset_id":"a1"}`),
	}
	if err := s.handleBatch(context.Background(), batch); err != nil {
		t.Fatalf("handleBatch returned error: %v", err)
	}

	bulks := rec.snapshotBulks()
	if len(bulks) != 1 || len(bulks[0].ItemIDs) != 1 || bulks[0].ItemIDs[0] != "a1" {
		t.Fatalf("expected single bulk with a1 only, got %#v", bulks)
	}
}

// ---------- ESSubscriber.Run dispatch / backward-compat ----------

// fakeBatchSubscriber implements both EventSubscriber AND BatchEventSubscriber
// so we can verify ESSubscriber feature-detects the batch path.
type fakeBatchSubscriber struct {
	mu              sync.Mutex
	receiveCalled   bool
	batchCalled     bool
	deliveredBatch  [][]byte
	deliveredSingle []byte
}

func (f *fakeBatchSubscriber) Receive(ctx context.Context, handler func(context.Context, []byte) error) error {
	f.mu.Lock()
	f.receiveCalled = true
	f.mu.Unlock()
	if f.deliveredSingle != nil {
		return handler(ctx, f.deliveredSingle)
	}
	return nil
}

func (f *fakeBatchSubscriber) ReceiveBatch(
	ctx context.Context,
	batchSize int,
	wait time.Duration,
	handler func(context.Context, [][]byte) error,
) error {
	f.mu.Lock()
	f.batchCalled = true
	f.mu.Unlock()
	if f.deliveredBatch != nil {
		return handler(ctx, f.deliveredBatch)
	}
	return nil
}

func (f *fakeBatchSubscriber) Close() error { return nil }

func TestESSubscriberRun_BatchSizeOneUsesLegacyReceive(t *testing.T) {
	rec := &bulkRecorder{}
	server := newRecordingESServer(t, rec, nil)
	defer server.Close()

	builder := newRecordingBuilder()
	builder.setOK("a1")

	fake := &fakeBatchSubscriber{deliveredSingle: []byte(`{"event_seq":1,"asset_id":"a1"}`)}

	s := &ESSubscriber{
		Subscriber: fake,
		ES:         elasticsearch.New(server.URL, "assets", "", ""),
		Builder:    builder,
		BatchSize:  1, // legacy
	}
	if err := s.Run(context.Background()); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if fake.batchCalled {
		t.Fatalf("expected legacy Receive path with BatchSize=1, but ReceiveBatch was called")
	}
	if !fake.receiveCalled {
		t.Fatalf("expected legacy Receive path with BatchSize=1")
	}

	bulks := rec.snapshotBulks()
	if len(bulks) != 1 || len(bulks[0].ItemIDs) != 1 {
		t.Fatalf("expected 1 bulk with 1 doc (single-message path), got %#v", bulks)
	}
}

func TestESSubscriberRun_BatchPathWhenSubscriberSupportsIt(t *testing.T) {
	rec := &bulkRecorder{}
	server := newRecordingESServer(t, rec, nil)
	defer server.Close()

	builder := newRecordingBuilder()
	builder.setOK("a1")
	builder.setOK("a2")

	fake := &fakeBatchSubscriber{
		deliveredBatch: [][]byte{
			[]byte(`{"event_seq":1,"asset_id":"a1"}`),
			[]byte(`{"event_seq":2,"asset_id":"a2"}`),
		},
	}

	s := &ESSubscriber{
		Subscriber:  fake,
		ES:          elasticsearch.New(server.URL, "assets", "", ""),
		Builder:     builder,
		BatchSize:   10,
		BatchWaitMs: 0,
	}
	if err := s.Run(context.Background()); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !fake.batchCalled {
		t.Fatalf("expected ReceiveBatch path when BatchSize>1 and subscriber supports it")
	}
	if fake.receiveCalled {
		t.Fatalf("expected legacy Receive NOT to be called in batch path")
	}

	bulks := rec.snapshotBulks()
	if len(bulks) != 1 {
		t.Fatalf("expected 1 bulk request, got %d", len(bulks))
	}
	if !equalUnordered(bulks[0].ItemIDs, []string{"a1", "a2"}) {
		t.Fatalf("bulk should batch a1+a2, got %v", bulks[0].ItemIDs)
	}
}

// legacyOnlySubscriber implements ONLY EventSubscriber (no batch interface)
// to assert ESSubscriber falls back to single-message even when BatchSize>1.
type legacyOnlySubscriber struct {
	delivered []byte
	called    bool
}

func (l *legacyOnlySubscriber) Receive(ctx context.Context, handler func(context.Context, []byte) error) error {
	l.called = true
	if l.delivered != nil {
		return handler(ctx, l.delivered)
	}
	return nil
}

func (l *legacyOnlySubscriber) Close() error { return nil }

func TestESSubscriberRun_NonBatchSubscriberFallsBackToReceive(t *testing.T) {
	rec := &bulkRecorder{}
	server := newRecordingESServer(t, rec, nil)
	defer server.Close()

	builder := newRecordingBuilder()
	builder.setOK("a1")

	leg := &legacyOnlySubscriber{delivered: []byte(`{"event_seq":1,"asset_id":"a1"}`)}

	s := &ESSubscriber{
		Subscriber: leg,
		ES:         elasticsearch.New(server.URL, "assets", "", ""),
		Builder:    builder,
		BatchSize:  10, // requested but subscriber doesn't implement BatchEventSubscriber
	}
	if err := s.Run(context.Background()); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !leg.called {
		t.Fatalf("expected legacy Receive path when subscriber does not implement BatchEventSubscriber")
	}
}

// ---------- helpers ----------

func equalUnordered(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	ac := append([]string(nil), a...)
	bc := append([]string(nil), b...)
	sortStrings(ac)
	sortStrings(bc)
	for i := range ac {
		if ac[i] != bc[i] {
			return false
		}
	}
	return true
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
