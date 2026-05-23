package audit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type fakeAuditQuerier struct {
	querySQL   string
	queryArgs  []any
	querySQLs  []string
	queryArgss [][]any
	rows       auditRows
	err        error
	results    []fakeAuditQueryResult
	called     bool
}

type fakeAuditQueryResult struct {
	rows auditRows
	err  error
}

func (q *fakeAuditQuerier) Query(_ context.Context, sql string, args ...any) (auditRows, error) {
	q.called = true
	q.querySQL = sql
	q.queryArgs = args
	q.querySQLs = append(q.querySQLs, sql)
	q.queryArgss = append(q.queryArgss, args)
	if len(q.results) > 0 {
		idx := len(q.querySQLs) - 1
		if idx >= len(q.results) {
			return &fakeAuditRows{}, nil
		}
		result := q.results[idx]
		if result.err != nil {
			return nil, result.err
		}
		if result.rows == nil {
			return &fakeAuditRows{}, nil
		}
		return result.rows, nil
	}
	if q.err != nil {
		return nil, q.err
	}
	if q.rows == nil {
		return &fakeAuditRows{}, nil
	}
	return q.rows, nil
}

type fakeAuditRows struct {
	data   [][]any
	idx    int
	closed bool
	err    error
}

func (r *fakeAuditRows) Next() bool {
	if r.idx >= len(r.data) {
		return false
	}
	r.idx++
	return true
}

func (r *fakeAuditRows) Scan(dest ...any) error {
	if r.idx == 0 || r.idx > len(r.data) {
		return errors.New("scan called before next")
	}
	cur := r.data[r.idx-1]
	if len(dest) != len(cur) {
		return errors.New("scan length mismatch")
	}
	for i := range dest {
		if err := assignAuditValue(dest[i], cur[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *fakeAuditRows) Close() { r.closed = true }

func (r *fakeAuditRows) Err() error { return r.err }

func assignAuditValue(dst any, src any) error {
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
	return errors.New("type mismatch")
}

func setupAuditRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/audit/search", h.HandleAuditSearch)
	r.GET("/audit/lineage-search", h.HandleLineageSearch)
	return r
}

func doAuditSearchReq(r *gin.Engine, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHandleAuditSearchValidation(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "invalid limit", path: "/audit/search?limit=0"},
		{name: "invalid cursor", path: "/audit/search?cursor=abc"},
		{name: "invalid time_from", path: "/audit/search?time_from=not-time"},
		{name: "invalid time_to", path: "/audit/search?time_to=not-time"},
		{name: "inverted time range", path: "/audit/search?time_from=2026-05-24T00:00:00Z&time_to=2026-05-23T00:00:00Z"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &fakeAuditQuerier{}
			r := setupAuditRouter(&Handler{db: q})

			w := doAuditSearchReq(r, tt.path)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
			}
			var body struct {
				Code string `json:"code"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.Code != "INVALID_ARGUMENT" {
				t.Fatalf("expected INVALID_ARGUMENT, got %q", body.Code)
			}
			if q.called {
				t.Fatal("did not expect database query for invalid request")
			}
		})
	}
}

func TestHandleAuditSearchSuccessWithFiltersAndCursor(t *testing.T) {
	occurred := time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)
	created := occurred.Add(time.Second)
	rows := &fakeAuditRows{data: [][]any{
		{"evt-100", int64(100), "algo_finished", "asset", "asset001", "mcap0001", "tenant1", "project1", "backend", "user", "alice", "run0000000000001", occurred, created},
		{"evt-099", int64(99), "algo_finished", "asset", "asset002", "mcap0001", "tenant1", "project1", "backend", "user", "alice", "run0000000000001", occurred.Add(-time.Minute), created},
		{"evt-098", int64(98), "algo_finished", "asset", "asset003", "mcap0001", "tenant1", "project1", "backend", "user", "alice", "run0000000000001", occurred.Add(-2 * time.Minute), created},
	}}
	q := &fakeAuditQuerier{rows: rows}
	r := setupAuditRouter(&Handler{db: q})

	w := doAuditSearchReq(r, "/audit/search?actor=alice&event_type=algo_finished&run_id=run0000000000001&time_from=2026-05-23T00:00:00Z&time_to=2026-05-24T00:00:00Z&cursor=101&limit=2")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !rows.closed {
		t.Fatal("expected rows to be closed")
	}
	for _, want := range []string{
		"actor_id ILIKE $1",
		"event_type = $2",
		"run_id = $3",
		"occurred_at >= $4",
		"occurred_at <= $5",
		"event_seq < $6",
		"ORDER BY event_seq DESC",
		"LIMIT $7",
	} {
		if !strings.Contains(q.querySQL, want) {
			t.Fatalf("query missing %q:\n%s", want, q.querySQL)
		}
	}
	if got, want := len(q.queryArgs), 7; got != want {
		t.Fatalf("expected %d query args, got %d", want, got)
	}
	if got := q.queryArgs[0]; got != "%alice%" {
		t.Fatalf("expected actor arg %%alice%%, got %#v", got)
	}
	if got := q.queryArgs[len(q.queryArgs)-1]; got != 3 {
		t.Fatalf("expected query limit 3, got %#v", got)
	}

	var body struct {
		Items      []auditEventRow `json:"items"`
		Limit      int             `json:"limit"`
		NextCursor int64           `json:"next_cursor"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Limit != 2 || body.NextCursor != 99 {
		t.Fatalf("expected limit=2 next_cursor=99, got limit=%d next_cursor=%d", body.Limit, body.NextCursor)
	}
	if len(body.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(body.Items))
	}
	if body.Items[0].EventSeq != 100 || body.Items[1].EventSeq != 99 {
		t.Fatalf("unexpected item order: %+v", body.Items)
	}
}

func TestHandleAuditSearchEmptyItemsArray(t *testing.T) {
	q := &fakeAuditQuerier{}
	r := setupAuditRouter(&Handler{db: q})

	w := doAuditSearchReq(r, "/audit/search")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Items []auditEventRow `json:"items"`
		Limit int             `json:"limit"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Items == nil {
		t.Fatal("expected items to be an empty array, got null")
	}
	if len(body.Items) != 0 || body.Limit != 50 {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestHandleAuditSearchNotConfigured(t *testing.T) {
	r := setupAuditRouter(&Handler{})

	w := doAuditSearchReq(r, "/audit/search")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestHandleLineageSearchValidation(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "missing asset_id", path: "/audit/lineage-search"},
		{name: "invalid direction", path: "/audit/lineage-search?asset_id=asset001&direction=sideways"},
		{name: "invalid depth integer", path: "/audit/lineage-search?asset_id=asset001&depth=abc"},
		{name: "invalid depth zero", path: "/audit/lineage-search?asset_id=asset001&depth=0"},
		{name: "unsupported relation type", path: "/audit/lineage-search?asset_id=asset001&relation_types=unknown"},
		{name: "empty relation type", path: "/audit/lineage-search?asset_id=asset001&relation_types=derived_from,"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &fakeAuditQuerier{}
			r := setupAuditRouter(&Handler{db: q})

			w := doAuditSearchReq(r, tt.path)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
			}
			var body struct {
				Code string `json:"code"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.Code != "INVALID_ARGUMENT" {
				t.Fatalf("expected INVALID_ARGUMENT, got %q", body.Code)
			}
			if q.called {
				t.Fatal("did not expect database query for invalid request")
			}
		})
	}
}

func TestHandleLineageSearchDownstreamSuccess(t *testing.T) {
	created := time.Date(2026, 5, 23, 13, 0, 0, 0, time.UTC)
	rows := &fakeAuditRows{data: [][]any{
		{"asset-root", "asset-child", "derived_from", "algo", "hand_track", "2.0.0", "run0000000000001", created, "asset-child", 1},
		{"asset-child", "asset-grandchild", "split_from", "", "", "", "", created.Add(time.Second), "asset-grandchild", 2},
	}}
	q := &fakeAuditQuerier{rows: rows}
	r := setupAuditRouter(&Handler{db: q})

	w := doAuditSearchReq(r, "/audit/lineage-search?asset_id=asset-root&direction=downstream&depth=2&relation_types=derived_from,split_from,split_from")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !rows.closed {
		t.Fatal("expected rows to be closed")
	}
	for _, want := range []string{
		"WITH RECURSIVE rec AS",
		"ar.parent_asset_id = $1",
		"ar.relation_type = ANY($3::text[])",
		"NOT ar.child_asset_id = ANY(r.path)",
		"ORDER BY depth ASC, related_asset_id ASC",
	} {
		if !strings.Contains(q.querySQL, want) {
			t.Fatalf("query missing %q:\n%s", want, q.querySQL)
		}
	}
	if got, want := q.queryArgs[0], "asset-root"; got != want {
		t.Fatalf("asset arg: got %#v want %#v", got, want)
	}
	if got, want := q.queryArgs[1], 2; got != want {
		t.Fatalf("depth arg: got %#v want %#v", got, want)
	}
	if got, want := q.queryArgs[2], []string{"derived_from", "split_from"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("relation_types arg: got %#v want %#v", got, want)
	}

	var body struct {
		AssetID       string        `json:"asset_id"`
		Direction     string        `json:"direction"`
		Depth         int           `json:"depth"`
		RelationTypes []string      `json:"relation_types"`
		Nodes         []lineageNode `json:"nodes"`
		Count         int           `json:"count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.AssetID != "asset-root" || body.Direction != "downstream" || body.Depth != 2 {
		t.Fatalf("unexpected envelope: %+v", body)
	}
	if !reflect.DeepEqual(body.RelationTypes, []string{"derived_from", "split_from"}) {
		t.Fatalf("unexpected relation_types: %#v", body.RelationTypes)
	}
	if body.Count != 2 || len(body.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got count=%d len=%d", body.Count, len(body.Nodes))
	}
	if body.Nodes[0].AssetID != "asset-child" || body.Nodes[0].RelationType != "derived_from" || body.Nodes[0].Direction != "downstream" || body.Nodes[0].Depth != 1 {
		t.Fatalf("unexpected first node: %+v", body.Nodes[0])
	}
	if body.Nodes[0].ParentAssetID != "asset-root" || body.Nodes[0].ChildAssetID != "asset-child" || body.Nodes[0].RunID != "run0000000000001" {
		t.Fatalf("unexpected first edge metadata: %+v", body.Nodes[0])
	}
}

func TestHandleLineageSearchBothDirectionsAndDefaults(t *testing.T) {
	created := time.Date(2026, 5, 23, 13, 30, 0, 0, time.UTC)
	upstreamRows := &fakeAuditRows{data: [][]any{
		{"asset-parent", "asset-mid", "contains", "", "", "", "", created, "asset-parent", 1},
	}}
	downstreamRows := &fakeAuditRows{data: [][]any{
		{"asset-mid", "asset-child", "split_from", "", "", "", "", created, "asset-child", 1},
	}}
	q := &fakeAuditQuerier{results: []fakeAuditQueryResult{
		{rows: upstreamRows},
		{rows: downstreamRows},
	}}
	r := setupAuditRouter(&Handler{db: q})

	w := doAuditSearchReq(r, "/audit/lineage-search?asset_id=asset-mid")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if got, want := len(q.querySQLs), 2; got != want {
		t.Fatalf("expected %d queries, got %d", want, got)
	}
	if !strings.Contains(q.querySQLs[0], "ar.child_asset_id = $1") {
		t.Fatalf("expected upstream query to start from child_asset_id:\n%s", q.querySQLs[0])
	}
	if !strings.Contains(q.querySQLs[1], "ar.parent_asset_id = $1") {
		t.Fatalf("expected downstream query to start from parent_asset_id:\n%s", q.querySQLs[1])
	}
	if got, want := q.queryArgss[0][2], defaultLineageRelationTypes; !reflect.DeepEqual(got, want) {
		t.Fatalf("default relation types: got %#v want %#v", got, want)
	}

	var body struct {
		Direction     string        `json:"direction"`
		Depth         int           `json:"depth"`
		RelationTypes []string      `json:"relation_types"`
		Nodes         []lineageNode `json:"nodes"`
		Count         int           `json:"count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Direction != "both" || body.Depth != 10 {
		t.Fatalf("unexpected defaults: %+v", body)
	}
	if !reflect.DeepEqual(body.RelationTypes, defaultLineageRelationTypes) {
		t.Fatalf("unexpected relation_types: %#v", body.RelationTypes)
	}
	if body.Count != 2 || body.Nodes[0].Direction != "upstream" || body.Nodes[1].Direction != "downstream" {
		t.Fatalf("unexpected nodes: %+v", body.Nodes)
	}
}

func TestHandleLineageSearchEmptyNodesArray(t *testing.T) {
	q := &fakeAuditQuerier{}
	r := setupAuditRouter(&Handler{db: q})

	w := doAuditSearchReq(r, "/audit/lineage-search?asset_id=asset-empty&direction=upstream")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Nodes []lineageNode `json:"nodes"`
		Count int           `json:"count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Nodes == nil {
		t.Fatal("expected nodes to be an empty array, got null")
	}
	if body.Count != 0 || len(body.Nodes) != 0 {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestHandleLineageSearchDatabaseErrors(t *testing.T) {
	t.Run("query error", func(t *testing.T) {
		q := &fakeAuditQuerier{err: errors.New("boom")}
		r := setupAuditRouter(&Handler{db: q})

		w := doAuditSearchReq(r, "/audit/lineage-search?asset_id=asset001")
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("scan error", func(t *testing.T) {
		q := &fakeAuditQuerier{rows: &fakeAuditRows{data: [][]any{{"too-few-columns"}}}}
		r := setupAuditRouter(&Handler{db: q})

		w := doAuditSearchReq(r, "/audit/lineage-search?asset_id=asset001&direction=downstream")
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("row iteration error", func(t *testing.T) {
		q := &fakeAuditQuerier{rows: &fakeAuditRows{err: errors.New("rows failed")}}
		r := setupAuditRouter(&Handler{db: q})

		w := doAuditSearchReq(r, "/audit/lineage-search?asset_id=asset001&direction=downstream")
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
		}
	})
}

func TestHandleLineageSearchNotConfigured(t *testing.T) {
	r := setupAuditRouter(&Handler{})

	w := doAuditSearchReq(r, "/audit/lineage-search?asset_id=asset001")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", w.Code, w.Body.String())
	}
}
