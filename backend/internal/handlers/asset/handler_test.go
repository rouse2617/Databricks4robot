package asset

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/handlers"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	assetUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/asset"
)

type mockAssetRepo struct {
	insertNewFn       func(ctx context.Context, a *models.Asset) error
	getFn             func(ctx context.Context, assetID string) (*models.Asset, error)
	setFn             func(ctx context.Context, a *models.Asset) error
	softDeleteFn      func(ctx context.Context, assetID string) error
	listByMcapFileFn  func(ctx context.Context, mcapFileID string) ([]*models.Asset, error)
	writeSegIndexFn   func(ctx context.Context, a *models.Asset) error
	listWithFiltersFn func(ctx context.Context, whereSQL string, args []interface{}, page, pageSize int, orderBy filter.OrderByClause) ([]*models.Asset, int64, error)
	// CYB-4306: cost-lookup mock. Nil (default) returns empty rows so tests
	// that don't set it exercise the missing/filtered_out paths.
	lookupCostsFn func(ctx context.Context, assetIDs []string, startAt, endAt time.Time, byAlgo bool) ([]repository.AssetCostRow, error)
}

type fakeAssetSQLQuerier struct {
	querySQL  string
	queryArgs []any
	allArgs   [][]any
	rows      assetSQLRows
	err       error
	called    bool
	calls     int
	queries   []assetSQLQueryResult
}

type assetSQLQueryResult struct {
	rows assetSQLRows
	err  error
}

func (q *fakeAssetSQLQuerier) Query(_ context.Context, sql string, args ...any) (assetSQLRows, error) {
	q.called = true
	q.calls++
	q.querySQL = sql
	q.queryArgs = args
	q.allArgs = append(q.allArgs, append([]any(nil), args...))
	if len(q.queries) > 0 {
		res := q.queries[0]
		q.queries = q.queries[1:]
		if res.err != nil {
			return nil, res.err
		}
		if res.rows == nil {
			return &fakeAssetSQLRows{}, nil
		}
		return res.rows, nil
	}
	if q.err != nil {
		return nil, q.err
	}
	if q.rows == nil {
		return &fakeAssetSQLRows{}, nil
	}
	return q.rows, nil
}

type fakeAssetSQLRows struct {
	data   [][]any
	idx    int
	closed bool
	err    error
}

func (r *fakeAssetSQLRows) Next() bool {
	if r.idx >= len(r.data) {
		return false
	}
	r.idx++
	return true
}

func (r *fakeAssetSQLRows) Scan(dest ...any) error {
	if r.idx == 0 || r.idx > len(r.data) {
		return errors.New("scan called before next")
	}
	cur := r.data[r.idx-1]
	if len(dest) != len(cur) {
		return fmt.Errorf("scan length mismatch: got %d dest, %d values", len(dest), len(cur))
	}
	for i := range dest {
		if err := assignAssetSQLValue(dest[i], cur[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *fakeAssetSQLRows) Close() { r.closed = true }

func (r *fakeAssetSQLRows) Err() error { return r.err }

func assignAssetSQLValue(dst any, src any) error {
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
	if elem.Kind() == reflect.Ptr && sv.Type().AssignableTo(elem.Type().Elem()) {
		ptr := reflect.New(elem.Type().Elem())
		ptr.Elem().Set(sv)
		elem.Set(ptr)
		return nil
	}
	return fmt.Errorf("type mismatch: cannot assign %T to %s", src, elem.Type())
}

// mockDeliveryRepoForAsset implements repository.DeliveryRepository for asset handler tests.
type mockDeliveryRepoForAsset struct {
	listByAssetFn func(ctx context.Context, assetID string) ([]string, error)
}

func (m *mockDeliveryRepoForAsset) Set(context.Context, *models.Delivery) error { return nil }
func (m *mockDeliveryRepoForAsset) Get(context.Context, string) (*models.Delivery, error) {
	return nil, nil
}
func (m *mockDeliveryRepoForAsset) AddItems(context.Context, string, []string) error {
	return nil
}
func (m *mockDeliveryRepoForAsset) RefreshAssetDeliveryIndex(context.Context, string) error {
	return nil
}
func (m *mockDeliveryRepoForAsset) ListByCustomer(context.Context, string) ([]string, error) {
	return nil, nil
}
func (m *mockDeliveryRepoForAsset) ListByAsset(_ context.Context, _ string) ([]string, error) {
	if m.listByAssetFn != nil {
		return m.listByAssetFn(context.Background(), "")
	}
	return []string{}, nil
}
func (m *mockDeliveryRepoForAsset) ListItems(context.Context, string) ([]*models.DeliveryItem, error) {
	return []*models.DeliveryItem{}, nil
}
func (m *mockDeliveryRepoForAsset) List(_ context.Context, _, _ int, _ string, _ string) ([]*models.Delivery, int64, error) {
	return []*models.Delivery{}, 0, nil
}
func (m *mockDeliveryRepoForAsset) Update(_ context.Context, _ *models.Delivery, _ int64) error {
	return nil
}

func (m *mockAssetRepo) InsertNew(ctx context.Context, a *models.Asset) error {
	if m.insertNewFn != nil {
		return m.insertNewFn(ctx, a)
	}
	return m.Set(ctx, a)
}
func (m *mockAssetRepo) Get(ctx context.Context, assetID string) (*models.Asset, error) {
	return m.GetAll(ctx, assetID)
}
func (m *mockAssetRepo) GetAll(ctx context.Context, assetID string) (*models.Asset, error) {
	if m.getFn != nil {
		return m.getFn(ctx, assetID)
	}
	return nil, nil
}
func (m *mockAssetRepo) FindExistingIDs(ctx context.Context, assetIDs []string) (map[string]struct{}, error) {
	out := make(map[string]struct{})
	for _, assetID := range assetIDs {
		a, err := m.Get(ctx, assetID)
		if err != nil {
			return nil, err
		}
		if a != nil {
			out[assetID] = struct{}{}
		}
	}
	return out, nil
}
func (m *mockAssetRepo) Set(ctx context.Context, a *models.Asset) error {
	if m.setFn != nil {
		return m.setFn(ctx, a)
	}
	return nil
}
func (m *mockAssetRepo) SoftDelete(ctx context.Context, assetID string) error {
	if m.softDeleteFn != nil {
		return m.softDeleteFn(ctx, assetID)
	}
	return nil
}
func (m *mockAssetRepo) ListByMcapFile(ctx context.Context, mcapFileID string) ([]*models.Asset, error) {
	if m.listByMcapFileFn != nil {
		return m.listByMcapFileFn(ctx, mcapFileID)
	}
	return nil, nil
}
func (m *mockAssetRepo) ListByLogicalAssetID(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}
func (m *mockAssetRepo) WriteSegmentIndex(ctx context.Context, a *models.Asset) error {
	if m.writeSegIndexFn != nil {
		return m.writeSegIndexFn(ctx, a)
	}
	return nil
}
func (m *mockAssetRepo) ListWithFilters(ctx context.Context, whereSQL string, args []interface{}, page, pageSize int, orderBy filter.OrderByClause) ([]*models.Asset, int64, error) {
	if m.listWithFiltersFn != nil {
		return m.listWithFiltersFn(ctx, whereSQL, args, page, pageSize, orderBy)
	}
	return []*models.Asset{}, 0, nil
}
func (m *mockAssetRepo) ListDescendants(_ context.Context, _ string) ([]*models.Asset, error) {
	return nil, nil
}
func (m *mockAssetRepo) MergeCfAlgo(_ context.Context, _ string, _ int64, _ map[string]interface{}, _ map[string]interface{}) (int64, error) {
	return 0, nil
}

// lookupDurationsFn lets a test inject specific rows returned by the mock
// LookupDurations. When nil, the mock resolves ids by calling Get for each
// requested id (asset_id path only). CYB-4294.
type lookupDurationsCall struct {
	ids   []string
	minMs int64
	maxMs int64
}

var _mockLookupCalls []lookupDurationsCall // shared per-test via TestMain reset; simple, tests reset before use

// LookupCosts injects the cost aggregate for the LookupCosts handler tests.
// CYB-4306.
func (m *mockAssetRepo) LookupCosts(ctx context.Context, assetIDs []string, startAt, endAt time.Time, byAlgo bool) ([]repository.AssetCostRow, error) {
	if m.lookupCostsFn != nil {
		return m.lookupCostsFn(ctx, assetIDs, startAt, endAt, byAlgo)
	}
	return nil, nil
}

func (m *mockAssetRepo) LookupDurations(ctx context.Context, ids []string, minMs, maxMs int64) ([]repository.DurationRow, error) {
	_mockLookupCalls = append(_mockLookupCalls, lookupDurationsCall{ids: ids, minMs: minMs, maxMs: maxMs})
	if len(ids) == 0 {
		return nil, nil
	}
	out := make([]repository.DurationRow, 0, len(ids))
	// Try each id as an asset_id via the getFn. This is enough for the
	// handler-layer test where we only care about wiring and validation.
	for _, id := range ids {
		a, err := m.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		if a == nil {
			continue
		}
		if minMs > 0 && a.DurationMs < minMs {
			continue
		}
		if maxMs > 0 && a.DurationMs > maxMs {
			continue
		}
		out = append(out, repository.DurationRow{
			AssetID:      a.AssetID,
			GraceVideoID: a.GraceVideoID,
			DurationMs:   a.DurationMs,
		})
	}
	return out, nil
}

func setupAssetRouter(method, path string, fn gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Handle(method, path, fn)
	return r
}

func doReq(t *testing.T, r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestGet(t *testing.T) {
	repo := &mockAssetRepo{}
	h := New(assetUC.New(repo), &mockDeliveryRepoForAsset{})
	r := setupAssetRouter(http.MethodGet, "/assets/:id", h.Get)

	repo.getFn = func(context.Context, string) (*models.Asset, error) { return nil, nil }
	w := doReq(t, r, http.MethodGet, "/assets/aaaaaaaa", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	repo.getFn = func(context.Context, string) (*models.Asset, error) { return nil, errors.New("boom") }
	w = doReq(t, r, http.MethodGet, "/assets/aaaaaaaa", nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	repo.getFn = func(context.Context, string) (*models.Asset, error) { return &models.Asset{AssetID: "aaaaaaaa"}, nil }
	w = doReq(t, r, http.MethodGet, "/assets/aaaaaaaa", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestList(t *testing.T) {
	repo := &mockAssetRepo{
		listByMcapFileFn: func(context.Context, string) ([]*models.Asset, error) {
			return []*models.Asset{{AssetID: "aaaaaaaa"}, {AssetID: "bbbbbbbb"}}, nil
		},
		listWithFiltersFn: func(_ context.Context, whereSQL string, args []interface{}, page, pageSize int, orderBy filter.OrderByClause) ([]*models.Asset, int64, error) {
			if page < 1 || pageSize < 1 {
				t.Fatalf("unexpected pagination: %d %d", page, pageSize)
			}
			return []*models.Asset{}, 0, nil
		},
	}
	h := New(assetUC.New(repo), &mockDeliveryRepoForAsset{})
	r := setupAssetRouter(http.MethodGet, "/assets", h.List)

	// No filters, no mcap_file_id → returns empty list via ListWithFilters (200 OK).
	w := doReq(t, r, http.MethodGet, "/assets", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for no filters, got %d", w.Code)
	}

	// Invalid filter → 400.
	w = doReq(t, r, http.MethodGet, "/assets?filter=badformat", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid filter, got %d", w.Code)
	}

	// mcap_file_id backward compat → 200.
	w = doReq(t, r, http.MethodGet, "/assets?mcap_file_id=m1&page=1&page_size=1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Promoted field resolves to real column; tag sort now resolves via projection subquery.
	repo.listWithFiltersFn = func(_ context.Context, whereSQL string, args []interface{}, page, pageSize int, orderBy filter.OrderByClause) ([]*models.Asset, int64, error) {
		if whereSQL != "owner = $1" {
			t.Fatalf("unexpected whereSQL: %s", whereSQL)
		}
		if len(args) != 1 || args[0] != "alice" {
			t.Fatalf("unexpected args: %#v", args)
		}
		if orderBy.SQL != "(SELECT t.tag_value FROM asset_tags t WHERE t.asset_id = assets.asset_id AND t.tag_key = $1 LIMIT 1) DESC" {
			t.Fatalf("unexpected orderBy: %s", orderBy.SQL)
		}
		return []*models.Asset{}, 0, nil
	}
	w = doReq(t, r, http.MethodGet, "/assets?filter=owner:eq:alice&sort_by=-tags.notes", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for resolved whitelist fields, got %d", w.Code)
	}

	// Invalid sort field → 400.
	w = doReq(t, r, http.MethodGet, "/assets?sort_by=-drop_table", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid sort field, got %d", w.Code)
	}
}

func TestCreate(t *testing.T) {
	repo := &mockAssetRepo{}
	h := New(assetUC.New(repo), &mockDeliveryRepoForAsset{})
	r := setupAssetRouter(http.MethodPost, "/assets", h.Create)

	w := doReq(t, r, http.MethodPost, "/assets", map[string]any{"bad": 1})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	w = doReq(t, r, http.MethodPost, "/assets", map[string]any{
		"mcap_file_id":       "m1",
		"start_timestamp_ns": 10,
		"end_timestamp_ns":   10,
		"reviewer":           "r1",
	})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", w.Code)
	}

	w = doReq(t, r, http.MethodPost, "/assets", map[string]any{
		"mcap_file_id":       "m1",
		"start_timestamp_ns": 10,
		"end_timestamp_ns":   1000010,
		"reviewer":           "r1",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var created map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal create response failed: %v", err)
	}
	if got, _ := created["asset_type"].(string); got != "segment" {
		t.Fatalf("expected asset_type=segment by default, got %v", created["asset_type"])
	}

	repo.insertNewFn = func(context.Context, *models.Asset) error {
		return fmt.Errorf("postgres AssetRepo.InsertNew: %w", &pgconn.PgError{
			Code:           "23514",
			ConstraintName: "assets_mcap_file_id_check",
		})
	}
	w = doReq(t, r, http.MethodPost, "/assets", map[string]any{
		"mcap_file_id":       "bad-format-mcap-id",
		"start_timestamp_ns": 10,
		"end_timestamp_ns":   1000010,
		"reviewer":           "r1",
	})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for mcap_file_id check violation, got %d", w.Code)
	}

	repo.insertNewFn = func(context.Context, *models.Asset) error {
		return fmt.Errorf("postgres AssetRepo.InsertNew: %w", &pgconn.PgError{
			Code:           "23503",
			ConstraintName: "fk_assets_mcap",
		})
	}
	w = doReq(t, r, http.MethodPost, "/assets", map[string]any{
		"mcap_file_id":       "A1B2C3D4",
		"start_timestamp_ns": 10,
		"end_timestamp_ns":   1000010,
		"reviewer":           "r1",
	})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for missing mcap_file FK violation, got %d", w.Code)
	}

	// Validation-error body must surface English json tag names, not the
	// Chinese `label:"..."` tags that historically lived on the Create
	// request struct — they leaked into external integrator error bodies.
	// Send an empty body → missing required fields → 400 with a binding
	// error referencing the field names.
	w = doReq(t, r, http.MethodPost, "/assets", map[string]any{})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing required fields, got %d body=%s", w.Code, w.Body.String())
	}
	for _, chinese := range []string{"起始时间戳", "结束时间戳", "审核人"} {
		if bytes.Contains(w.Body.Bytes(), []byte(chinese)) {
			t.Fatalf("validation error must not leak Chinese label %q: %s", chinese, w.Body.String())
		}
	}
}

func TestGetAssetTypeSchema(t *testing.T) {
	h := New(assetUC.New(&mockAssetRepo{}), &mockDeliveryRepoForAsset{})
	r := setupAssetRouter(http.MethodGet, "/asset-types/:type/schema", h.GetAssetTypeSchema)

	w := doReq(t, r, http.MethodGet, "/asset-types/dataset/schema", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var schema map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &schema); err != nil {
		t.Fatalf("schema response is not JSON: %v", err)
	}
	if got, _ := schema["title"].(string); got == "" {
		t.Fatalf("expected schema title, got %#v", schema)
	}

	w = doReq(t, r, http.MethodGet, "/asset-types/unknown/schema", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown schema, got %d", w.Code)
	}
}

func TestHandleRatingsHistoryValidationAndNotConfigured(t *testing.T) {
	q := &fakeAssetSQLQuerier{}
	h := &Handler{pgq: q}
	r := setupAssetRouter(http.MethodGet, "/logical-assets/:id/ratings-history", h.HandleRatingsHistory)

	w := doReq(t, r, http.MethodGet, "/logical-assets/not-valid/ratings-history", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
	if q.called {
		t.Fatal("did not expect query for invalid id")
	}

	r = setupAssetRouter(http.MethodGet, "/logical-assets/:id/ratings-history", (&Handler{}).HandleRatingsHistory)
	w = doReq(t, r, http.MethodGet, "/logical-assets/aaaaaaaa/ratings-history", nil)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestHandleRatingsHistoryMissingLogicalAsset(t *testing.T) {
	q := &fakeAssetSQLQuerier{}
	h := &Handler{pgq: q}
	r := setupAssetRouter(http.MethodGet, "/logical-assets/:id/ratings-history", h.HandleRatingsHistory)

	w := doReq(t, r, http.MethodGet, "/logical-assets/aaaaaaaa/ratings-history", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(q.querySQL, "asset_metrics") || strings.Contains(q.querySQL, "asset_algo_latest") {
		t.Fatalf("unexpected ratings-history query:\n%s", q.querySQL)
	}
	if got, want := q.queryArgs[0], "aaaaaaaa"; got != want {
		t.Fatalf("logical id arg: got %#v want %#v", got, want)
	}
}

func TestHandleRatingsHistoryNoRatings(t *testing.T) {
	created := time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC)
	rows := &fakeAssetSQLRows{data: [][]any{
		ratingsHistoryRow("asset001", int64(1), true, "ready", created, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil),
	}}
	q := &fakeAssetSQLQuerier{rows: rows}
	h := &Handler{pgq: q}
	r := setupAssetRouter(http.MethodGet, "/logical-assets/:id/ratings-history", h.HandleRatingsHistory)

	w := doReq(t, r, http.MethodGet, "/logical-assets/aaaaaaaa/ratings-history", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !rows.closed {
		t.Fatal("expected rows to be closed")
	}
	var body struct {
		LogicalAssetID string                   `json:"logical_asset_id"`
		Items          []ratingsHistoryRevision `json:"items"`
		Count          int                      `json:"count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.LogicalAssetID != "aaaaaaaa" || body.Count != 1 || len(body.Items) != 1 {
		t.Fatalf("unexpected envelope: %+v", body)
	}
	if body.Items[0].AssetID != "asset001" || !body.Items[0].IsCurrent || body.Items[0].Revision != 1 {
		t.Fatalf("unexpected revision: %+v", body.Items[0])
	}
	if body.Items[0].Ratings == nil || len(body.Items[0].Ratings) != 0 {
		t.Fatalf("expected empty ratings array, got %#v", body.Items[0].Ratings)
	}
}

func TestHandleRatingsHistoryGroupsAndSortsMetricRows(t *testing.T) {
	t1 := time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(time.Hour)
	score45 := 4.5
	score40 := 4.0
	conf := 0.99
	runID := "run0000000000001"
	sourceName := "reviewer_a"
	unit := "stars"
	rows := &fakeAssetSQLRows{data: [][]any{
		ratingsHistoryRow("asset001", int64(1), false, "ready", t1,
			"rating.quality_score", "float", unit, score45, nil, nil, nil,
			"asset", "", "manual_rating", "v1", nil, runID, "human", sourceName, conf, t1.Add(10*time.Minute), t1.Add(10*time.Minute)),
		ratingsHistoryRow("asset002", int64(2), true, "ready", t2,
			"rating.action_completeness", "float", nil, score40, nil, nil, nil,
			"asset", "", "manual_rating", "v1", nil, nil, "human", "reviewer_b", nil, t2.Add(10*time.Minute), t2.Add(10*time.Minute)),
		ratingsHistoryRow("asset002", int64(2), true, "ready", t2,
			"rating.label_correctness", "float", nil, score45, nil, nil, nil,
			"asset", "", "manual_rating", "v1", nil, nil, "human", "reviewer_a", nil, t2.Add(20*time.Minute), t2.Add(20*time.Minute)),
	}}
	q := &fakeAssetSQLQuerier{rows: rows}
	h := &Handler{pgq: q}
	r := setupAssetRouter(http.MethodGet, "/logical-assets/:id/ratings-history", h.HandleRatingsHistory)

	w := doReq(t, r, http.MethodGet, "/logical-assets/aaaaaaaa/ratings-history", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	for _, want := range []string{
		"m.metric_key LIKE 'rating.%'",
		"ORDER BY a.revision ASC NULLS LAST",
		"m.metric_key ASC NULLS LAST",
		"m.source_type ASC NULLS LAST",
		"m.source_name ASC NULLS LAST",
		"m.recorded_at ASC NULLS LAST",
	} {
		if !strings.Contains(q.querySQL, want) {
			t.Fatalf("query missing %q:\n%s", want, q.querySQL)
		}
	}

	var body struct {
		Items []ratingsHistoryRevision `json:"items"`
		Count int                      `json:"count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Count != 2 || len(body.Items) != 2 {
		t.Fatalf("expected 2 revision items, got %+v", body)
	}
	if body.Items[0].AssetID != "asset001" || body.Items[0].Revision != 1 || len(body.Items[0].Ratings) != 1 {
		t.Fatalf("unexpected first revision: %+v", body.Items[0])
	}
	if body.Items[1].AssetID != "asset002" || body.Items[1].Revision != 2 || !body.Items[1].IsCurrent || len(body.Items[1].Ratings) != 2 {
		t.Fatalf("unexpected second revision: %+v", body.Items[1])
	}
	firstRating := body.Items[0].Ratings[0]
	if firstRating.MetricKey != "rating.quality_score" || firstRating.MetricValue == nil || *firstRating.MetricValue != score45 {
		t.Fatalf("unexpected first rating: %+v", firstRating)
	}
	if firstRating.MetricUnit == nil || *firstRating.MetricUnit != unit || firstRating.RunID == nil || *firstRating.RunID != runID {
		t.Fatalf("unexpected optional fields: %+v", firstRating)
	}
	if body.Items[1].Ratings[0].MetricKey != "rating.action_completeness" || body.Items[1].Ratings[1].MetricKey != "rating.label_correctness" {
		t.Fatalf("unexpected rating order/grouping: %+v", body.Items[1].Ratings)
	}
}

func TestHandleRatingsHistoryDatabaseErrors(t *testing.T) {
	t.Run("query error", func(t *testing.T) {
		q := &fakeAssetSQLQuerier{err: errors.New("boom")}
		h := &Handler{pgq: q}
		r := setupAssetRouter(http.MethodGet, "/logical-assets/:id/ratings-history", h.HandleRatingsHistory)

		w := doReq(t, r, http.MethodGet, "/logical-assets/aaaaaaaa/ratings-history", nil)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("scan error", func(t *testing.T) {
		q := &fakeAssetSQLQuerier{rows: &fakeAssetSQLRows{data: [][]any{{"too-few-columns"}}}}
		h := &Handler{pgq: q}
		r := setupAssetRouter(http.MethodGet, "/logical-assets/:id/ratings-history", h.HandleRatingsHistory)

		w := doReq(t, r, http.MethodGet, "/logical-assets/aaaaaaaa/ratings-history", nil)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("row iteration error", func(t *testing.T) {
		q := &fakeAssetSQLQuerier{rows: &fakeAssetSQLRows{err: errors.New("rows failed")}}
		h := &Handler{pgq: q}
		r := setupAssetRouter(http.MethodGet, "/logical-assets/:id/ratings-history", h.HandleRatingsHistory)

		w := doReq(t, r, http.MethodGet, "/logical-assets/aaaaaaaa/ratings-history", nil)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
		}
	})
}

func ratingsHistoryRow(values ...any) []any {
	return values
}

func TestUpdateDeleteAndCommitSegments(t *testing.T) {
	repo := &mockAssetRepo{
		getFn: func(context.Context, string) (*models.Asset, error) {
			return &models.Asset{AssetID: "aaaaaaaa", Tags: map[string]string{}}, nil
		},
	}
	h := New(assetUC.New(repo), &mockDeliveryRepoForAsset{})

	// Update not found
	repo.getFn = func(context.Context, string) (*models.Asset, error) { return nil, nil }
	r := setupAssetRouter(http.MethodPatch, "/assets/:id", h.Update)
	w := doReq(t, r, http.MethodPatch, "/assets/aaaaaaaa", map[string]any{"status": "approved"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	// Update success
	repo.getFn = func(context.Context, string) (*models.Asset, error) {
		return &models.Asset{AssetID: "aaaaaaaa", Tags: map[string]string{}}, nil
	}
	w = doReq(t, r, http.MethodPatch, "/assets/aaaaaaaa", map[string]any{"status": "approved"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Delete
	r = setupAssetRouter(http.MethodDelete, "/assets/:id", h.Delete)
	w = doReq(t, r, http.MethodDelete, "/assets/aaaaaaaa", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Commit segments bad body
	r = setupAssetRouter(http.MethodPost, "/internal/commit-segments", h.CommitSegments)
	w = doReq(t, r, http.MethodPost, "/internal/commit-segments", map[string]any{"x": 1})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	// Commit segments invalid range
	w = doReq(t, r, http.MethodPost, "/internal/commit-segments", map[string]any{
		"mcap_file_id": "m1",
		"reviewer":     "r1",
		"ranges":       [][2]int64{{10, 10}},
	})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", w.Code)
	}
}

func TestUpdate_WithLifecycleStateOnly(t *testing.T) {
	repo := &mockAssetRepo{
		getFn: func(context.Context, string) (*models.Asset, error) {
			return &models.Asset{
				AssetID:        "aaaaaaaa",
				Status:         models.AssetStatusApproved,
				LifecycleState: "ready",
				Tags:           map[string]string{},
			}, nil
		},
	}
	h := New(assetUC.New(repo), &mockDeliveryRepoForAsset{})
	r := setupAssetRouter(http.MethodPatch, "/assets/:id", h.Update)

	w := doReq(t, r, http.MethodPatch, "/assets/aaaaaaaa", map[string]any{
		"lifecycle_state": "rejected",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if got, _ := resp["lifecycle_state"].(string); got != "rejected" {
		t.Fatalf("expected lifecycle_state=rejected, got %v", resp["lifecycle_state"])
	}
	if got, _ := resp["status"].(string); got != "rejected" {
		t.Fatalf("expected status=rejected, got %v", resp["status"])
	}
}

func TestListDeliveriesAndHelpers(t *testing.T) {
	h := New(assetUC.New(&mockAssetRepo{}), &mockDeliveryRepoForAsset{})
	r := setupAssetRouter(http.MethodGet, "/assets/:id/deliveries", h.ListDeliveries)
	w := doReq(t, r, http.MethodGet, "/assets/aaaaaaaa/deliveries", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	p, s := handlers.ParsePageParams("2", "50")
	if p != 2 || s != 50 {
		t.Fatalf("unexpected pagination parse: %d %d", p, s)
	}
	p, s = handlers.ParsePageParams("-1", "1000")
	if p != 1 || s != 20 {
		t.Fatalf("expected default pagination")
	}

	items := []*models.Asset{{AssetID: "aaaaaaaa"}, {AssetID: "bbbbbbbb"}}
	got := paginateAssets(items, 1, 1)
	if len(got) != 1 || got[0].AssetID != "aaaaaaaa" {
		t.Fatalf("paginate first page failed")
	}
	if len(paginateAssets(items, 3, 1)) != 0 {
		t.Fatalf("expected empty page")
	}
	if len(paginateAssets(nil, 1, 20)) != 0 {
		t.Fatalf("expected empty list")
	}
}

// ─── Filter Param Parsing and Error Response Tests ──────────────────────────

func TestListFilterParsing(t *testing.T) {
	repo := &mockAssetRepo{
		listWithFiltersFn: func(_ context.Context, whereSQL string, args []interface{}, page, pageSize int, orderBy filter.OrderByClause) ([]*models.Asset, int64, error) {
			return []*models.Asset{{AssetID: "aaaaaaaa"}}, 1, nil
		},
	}
	h := New(assetUC.New(repo), &mockDeliveryRepoForAsset{})
	r := setupAssetRouter(http.MethodGet, "/assets", h.List)

	// Valid filter → 200.
	w := doReq(t, r, http.MethodGet, "/assets?filter=status:eq:approved", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid filter, got %d", w.Code)
	}

	// Multiple valid filters → 200.
	w = doReq(t, r, http.MethodGet, "/assets?filter=status:eq:approved&filter=owner:eq:alice", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for multiple valid filters, got %d", w.Code)
	}

	// Invalid operator → 400 INVALID_FILTER.
	w = doReq(t, r, http.MethodGet, "/assets?filter=status:badop:value", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid operator, got %d", w.Code)
	}
	var errBody map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errBody)
	if errBody["code"] != "INVALID_FILTER" {
		t.Fatalf("expected INVALID_FILTER code, got %v", errBody["code"])
	}

	// Blacklisted field → 400 INVALID_FILTER.
	w = doReq(t, r, http.MethodGet, "/assets?filter=is_deleted:eq:true", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for blacklisted field, got %d", w.Code)
	}
	json.Unmarshal(w.Body.Bytes(), &errBody)
	if errBody["code"] != "INVALID_FILTER" {
		t.Fatalf("expected INVALID_FILTER code, got %v", errBody["code"])
	}

	// sort_by parameter → 200.
	w = doReq(t, r, http.MethodGet, "/assets?sort_by=-created_at", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for sort_by, got %d", w.Code)
	}
}

func TestListResponseFormat(t *testing.T) {
	repo := &mockAssetRepo{
		listWithFiltersFn: func(_ context.Context, _ string, _ []interface{}, _, _ int, _ filter.OrderByClause) ([]*models.Asset, int64, error) {
			return []*models.Asset{{AssetID: "aaaaaaaa"}, {AssetID: "bbbbbbbb"}}, 5, nil
		},
	}
	h := New(assetUC.New(repo), &mockDeliveryRepoForAsset{})
	r := setupAssetRouter(http.MethodGet, "/assets", h.List)

	w := doReq(t, r, http.MethodGet, "/assets?page=2&page_size=2", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["total"] != float64(5) {
		t.Fatalf("expected total=5, got %v", resp["total"])
	}
	if resp["page"] != float64(2) {
		t.Fatalf("expected page=2, got %v", resp["page"])
	}
	if resp["page_size"] != float64(2) {
		t.Fatalf("expected page_size=2, got %v", resp["page_size"])
	}
	items, ok := resp["items"].([]interface{})
	if !ok || len(items) != 2 {
		t.Fatalf("expected 2 items, got %v", resp["items"])
	}
}

// ─── Algo Handler Tests ─────────────────────────────────────────────────────

func setupAlgoRouter(h *AlgoHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/assets/:id/algo", h.ListCurrent)
	r.POST("/assets/:id/algo/:algo_key/start", h.Start)
	r.POST("/assets/:id/algo/:algo_key/finish", h.Finish)
	r.POST("/assets/:id/algo/:algo_key/reset", h.Reset)
	return r
}

// algoTestEnv bundles the dependencies an AlgoHandler test needs.
type algoTestEnv struct {
	asset      *mockAssetRepo
	algoLatest *handlerAlgoLatestRepo
	events     *handlerAssetEventRepo
}

// newAlgoEnv constructs an environment with optional pre-seeded algo state.
// presence controls whether the asset exists; statuses keys are algo_key strings.
func newAlgoEnv(t *testing.T, presence bool, statuses map[string]string) (*assetH_AlgoHandler, *algoTestEnv) {
	t.Helper()
	env := &algoTestEnv{
		asset:      &mockAssetRepo{},
		algoLatest: newHandlerAlgoLatestRepo(),
		events:     &handlerAssetEventRepo{},
	}
	if presence {
		env.asset.getFn = func(context.Context, string) (*models.Asset, error) {
			return &models.Asset{AssetID: "aaaaaaaa", Version: 1}, nil
		}
	} else {
		env.asset.getFn = func(context.Context, string) (*models.Asset, error) { return nil, nil }
	}
	for algoKey, status := range statuses {
		env.algoLatest.seed(algoKey, status, "")
	}
	uc := assetUC.NewAlgoUsecase(
		handlerTxRunner{},
		env.asset,
		env.algoLatest,
		env.events,
		buildTestAlgoRegistry(t),
	)
	h := NewAlgoHandler(uc)
	return h, env
}

// assetH_AlgoHandler is a tiny alias to keep handler tests local-imports clean.
type assetH_AlgoHandler = AlgoHandler

func TestAlgoStart(t *testing.T) {
	// Missing method field → 400.
	h, _ := newAlgoEnv(t, true, nil)
	r := setupAlgoRouter(h)
	w := doReq(t, r, http.MethodPost, "/assets/aaaaaaaa/algo/hand_tracking@1.2.0/start", map[string]any{})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing method, got %d", w.Code)
	}

	// Invalid algo key → 400.
	w = doReq(t, r, http.MethodPost, "/assets/aaaaaaaa/algo/invalid_key/start", map[string]any{"method": "test"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid algo key, got %d", w.Code)
	}

	// Asset not found → 404.
	h2, _ := newAlgoEnv(t, false, nil)
	r2 := setupAlgoRouter(h2)
	w = doReq(t, r2, http.MethodPost, "/assets/99999999/algo/hand_tracking@1.2.0/start", map[string]any{"method": "test"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for nonexistent asset, got %d", w.Code)
	}

	// Already running → 409.
	h3, _ := newAlgoEnv(t, true, map[string]string{"hand_tracking@1.2.0": "running"})
	r3 := setupAlgoRouter(h3)
	w = doReq(t, r3, http.MethodPost, "/assets/aaaaaaaa/algo/hand_tracking@1.2.0/start", map[string]any{"method": "test"})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for already running, got %d", w.Code)
	}
}

func TestAlgoFinish(t *testing.T) {
	// Missing status → 400.
	h, _ := newAlgoEnv(t, true, map[string]string{"env_analysis@1.0.0": "running"})
	r := setupAlgoRouter(h)
	w := doReq(t, r, http.MethodPost, "/assets/aaaaaaaa/algo/env_analysis@1.0.0/finish", map[string]any{})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing status, got %d", w.Code)
	}

	// Failed without reason → 422 MISSING_REASON.
	w = doReq(t, r, http.MethodPost, "/assets/aaaaaaaa/algo/env_analysis@1.0.0/finish", map[string]any{"status": "failed"})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for missing reason, got %d", w.Code)
	}
	var errBody map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errBody)
	if errBody["code"] != "MISSING_REASON" {
		t.Fatalf("expected MISSING_REASON, got %v", errBody["code"])
	}

	// hand_tracking ok with no output_uri → 422.
	h2, _ := newAlgoEnv(t, true, map[string]string{"hand_tracking@1.2.0": "running"})
	r2 := setupAlgoRouter(h2)
	w = doReq(t, r2, http.MethodPost, "/assets/aaaaaaaa/algo/hand_tracking@1.2.0/finish", map[string]any{"status": "ok"})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for missing required field, got %d", w.Code)
	}
	json.Unmarshal(w.Body.Bytes(), &errBody)
	if errBody["code"] != "MISSING_REQUIRED_FIELD" {
		t.Fatalf("expected MISSING_REQUIRED_FIELD, got %v", errBody["code"])
	}

	// Invalid state transition (status=pending) → 409.
	h3, _ := newAlgoEnv(t, true, map[string]string{"env_analysis@1.0.0": "pending"})
	r3 := setupAlgoRouter(h3)
	w = doReq(t, r3, http.MethodPost, "/assets/aaaaaaaa/algo/env_analysis@1.0.0/finish", map[string]any{
		"status": "failed", "reason": "test",
	})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for invalid state transition, got %d", w.Code)
	}
}

func TestAlgoReset(t *testing.T) {
	// Reset from pending → 409.
	h, _ := newAlgoEnv(t, true, map[string]string{"env_analysis@1.0.0": "pending"})
	r := setupAlgoRouter(h)
	w := doReq(t, r, http.MethodPost, "/assets/aaaaaaaa/algo/env_analysis@1.0.0/reset", nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for reset from pending, got %d", w.Code)
	}

	// Invalid algo key → 400.
	w = doReq(t, r, http.MethodPost, "/assets/aaaaaaaa/algo/bad_key/reset", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid algo key, got %d", w.Code)
	}
}

func TestAssetListEvents(t *testing.T) {
	const (
		testAssetID    = "11111111"
		missingAssetID = "22222222"
	)

	makeHandler := func(presence bool, seed func(*handlerAssetEventRepo)) *Handler {
		assetRepo := &mockAssetRepo{}
		if presence {
			assetRepo.getFn = func(context.Context, string) (*models.Asset, error) {
				return &models.Asset{AssetID: testAssetID, Version: 1}, nil
			}
		} else {
			assetRepo.getFn = func(context.Context, string) (*models.Asset, error) { return nil, nil }
		}
		eventRepo := &handlerAssetEventRepo{}
		if seed != nil {
			seed(eventRepo)
		}
		uc := assetUC.NewWithProjections(handlerTxRunner{}, assetRepo, nil, nil, eventRepo, nil, nil)
		return New(uc, &mockDeliveryRepoForAsset{})
	}

	// Invalid path id → 400.
	hBad := makeHandler(false, nil)
	rBad := setupAssetRouter(http.MethodGet, "/assets/:id/events", hBad.ListEvents)
	w := doReq(t, rBad, http.MethodGet, "/assets/not-a-uuid/events", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid asset id, got %d", w.Code)
	}

	// Asset not found → 404.
	h := makeHandler(false, nil)
	r := setupAssetRouter(http.MethodGet, "/assets/:id/events", h.ListEvents)
	w = doReq(t, r, http.MethodGet, "/assets/"+missingAssetID+"/events", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for nonexistent asset, got %d", w.Code)
	}

	// Success with wildcard event_type filter and cursor pagination.
	h2 := makeHandler(true, func(repo *handlerAssetEventRepo) {
		repo.seed("asset_created", 1, map[string]any{"asset_id": testAssetID})
		repo.seed("algo_started", 2, map[string]any{"asset_id": testAssetID, "algo_key": "env_analysis@1.0.0"})
		repo.seed("algo_finished", 3, map[string]any{"asset_id": testAssetID, "algo_key": "env_analysis@1.0.0"})
		repo.seed("tag_upserted", 4, map[string]any{"asset_id": testAssetID, "tag_key": "quality"})
	})
	r2 := setupAssetRouter(http.MethodGet, "/assets/:id/events", h2.ListEvents)
	w = doReq(t, r2, http.MethodGet, "/assets/"+testAssetID+"/events?event_type=algo_*&limit=1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Items      []models.AssetEvent `json:"items"`
		NextCursor *int64              `json:"next_cursor"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].EventType != "algo_finished" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.NextCursor == nil || *resp.NextCursor != 3 {
		t.Fatalf("expected next_cursor=3, got %+v", resp.NextCursor)
	}

	w = doReq(t, r2, http.MethodGet, "/assets/"+testAssetID+"/events?event_type=algo_*&algo_key=env_analysis@1.0.0&cursor=3", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on cursor follow-up, got %d", w.Code)
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal follow-up: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].EventType != "algo_started" {
		t.Fatalf("unexpected follow-up response: %+v", resp)
	}
}

func TestAssetEventsStream(t *testing.T) {
	const testAssetID = "11111111"
	occurredAt := time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC)

	makeHandler := func(assetExists bool, q *fakeAssetSQLQuerier) *Handler {
		assetRepo := &mockAssetRepo{}
		if assetExists {
			assetRepo.getFn = func(context.Context, string) (*models.Asset, error) {
				return &models.Asset{AssetID: testAssetID, Version: 1}, nil
			}
		} else {
			assetRepo.getFn = func(context.Context, string) (*models.Asset, error) { return nil, nil }
		}
		h := New(assetUC.New(assetRepo), &mockDeliveryRepoForAsset{})
		h.pgq = q
		return h
	}

	t.Run("streams first batch and resumes after Last-Event-ID", func(t *testing.T) {
		q := &fakeAssetSQLQuerier{
			queries: []assetSQLQueryResult{
				{
					rows: &fakeAssetSQLRows{data: [][]any{{
						"event-11",
						int64(11),
						"asset_updated",
						testAssetID,
						json.RawMessage(`{"field":"status"}`),
						occurredAt,
					}}},
				},
				{err: context.Canceled},
			},
		}
		h := makeHandler(true, q)
		r := setupAssetRouter(http.MethodGet, "/assets/:id/events/stream", h.HandleEventsStream)
		srv := httptest.NewServer(r)
		defer srv.Close()

		req, err := http.NewRequest(http.MethodGet, srv.URL+"/assets/"+testAssetID+"/events/stream", nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		req.Header.Set("Last-Event-ID", "10")
		resp, err := srv.Client().Do(req)
		if err != nil {
			t.Fatalf("stream request: %v", err)
		}
		defer resp.Body.Close()
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read stream: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", resp.StatusCode, string(bodyBytes))
		}
		if got := resp.Header.Get("Content-Type"); !strings.Contains(got, "text/event-stream") {
			t.Fatalf("expected text/event-stream content type, got %q", got)
		}
		body := string(bodyBytes)
		for _, want := range []string{
			"id: 11\n",
			"event: asset_updated\n",
			`"event_seq":11`,
			`"event_payload":{"field":"status"}`,
			": keepalive\n\n",
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("expected stream body to contain %q, got %s", want, body)
			}
		}
		if !q.called || len(q.allArgs) == 0 || len(q.allArgs[0]) != 2 || q.allArgs[0][0] != testAssetID || q.allArgs[0][1] != int64(10) {
			t.Fatalf("expected first SQL query with asset id and after seq, got called=%v args=%#v", q.called, q.allArgs)
		}
		if !strings.Contains(q.querySQL, "event_seq > $2") || !strings.Contains(q.querySQL, "ORDER BY event_seq ASC") || !strings.Contains(q.querySQL, "LIMIT 50") {
			t.Fatalf("unexpected stream query: %s", q.querySQL)
		}
	})

	t.Run("rejects malformed Last-Event-ID before opening stream", func(t *testing.T) {
		q := &fakeAssetSQLQuerier{}
		h := makeHandler(true, q)
		r := setupAssetRouter(http.MethodGet, "/assets/:id/events/stream", h.HandleEventsStream)

		req := httptest.NewRequest(http.MethodGet, "/assets/"+testAssetID+"/events/stream", nil)
		req.Header.Set("Last-Event-ID", "not-a-number")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
		}
		if q.called {
			t.Fatalf("expected no stream query for malformed Last-Event-ID")
		}
	})

	t.Run("rejects missing asset before opening stream", func(t *testing.T) {
		q := &fakeAssetSQLQuerier{}
		h := makeHandler(false, q)
		r := setupAssetRouter(http.MethodGet, "/assets/:id/events/stream", h.HandleEventsStream)

		w := doReq(t, r, http.MethodGet, "/assets/22222222/events/stream", nil)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
		}
		if q.called {
			t.Fatalf("expected no stream query for missing asset")
		}
	})
}

func TestUpsertTagAndDeleteTag(t *testing.T) {
	assetRepo := &mockAssetRepo{
		getFn: func(context.Context, string) (*models.Asset, error) {
			return &models.Asset{AssetID: "aaaaaaaa", McapFileID: "m1", Version: 1}, nil
		},
	}
	tagRepo := newHandlerAssetTagRepo()
	eventRepo := &handlerAssetEventRepo{}
	uc := assetUC.NewWithProjections(
		handlerTxRunner{},
		assetRepo,
		tagRepo,
		nil,
		eventRepo,
		buildTestTagRegistryForAsset(t),
		nil,
	)
	h := New(uc, &mockDeliveryRepoForAsset{})

	r := gin.New()
	r.POST("/assets/:id/tags", h.UpsertTag)
	r.DELETE("/assets/:id/tags/:key", h.DeleteTag)

	w := doReq(t, r, http.MethodPost, "/assets/aaaaaaaa/tags", map[string]any{
		"key": "quality", "value": "good",
		"source_type": "human", "source_name": "tester",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from upsert tag, got %d", w.Code)
	}
	if row, ok := tagRepo.rows[tagRowKey("aaaaaaaa", "quality")]; !ok || row.TagValue != "good" {
		t.Fatalf("expected projection row quality=good, got %#v", row)
	}

	w = doReq(t, r, http.MethodDelete, "/assets/aaaaaaaa/tags/quality", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from delete tag, got %d", w.Code)
	}
	if _, ok := tagRepo.rows[tagRowKey("aaaaaaaa", "quality")]; ok {
		t.Fatalf("expected tag projection row to be deleted")
	}
}

func TestUpsertTag_InvalidTag(t *testing.T) {
	assetRepo := &mockAssetRepo{
		getFn: func(context.Context, string) (*models.Asset, error) {
			return &models.Asset{AssetID: "aaaaaaaa", McapFileID: "m1", Version: 1}, nil
		},
	}
	uc := assetUC.NewWithProjections(
		handlerTxRunner{},
		assetRepo,
		newHandlerAssetTagRepo(),
		nil,
		&handlerAssetEventRepo{},
		buildTestTagRegistryForAsset(t),
		nil,
	)
	h := New(uc, &mockDeliveryRepoForAsset{})
	r := setupAssetRouter(http.MethodPost, "/assets/:id/tags", h.UpsertTag)

	w := doReq(t, r, http.MethodPost, "/assets/aaaaaaaa/tags", map[string]any{"key": "unknown_tag", "value": "x"})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for invalid tag, got %d", w.Code)
	}
}

func TestListTagHistory(t *testing.T) {
	const tagHistAssetID = "33333333"

	assetRepo := &mockAssetRepo{
		getFn: func(context.Context, string) (*models.Asset, error) {
			return &models.Asset{AssetID: tagHistAssetID, McapFileID: "m1", Version: 1}, nil
		},
	}
	eventRepo := &handlerAssetEventRepo{}
	eventRepo.seed("asset_created", 1, map[string]any{"asset_id": tagHistAssetID})
	eventRepo.seed("tag_upserted", 2, map[string]any{"asset_id": tagHistAssetID, "tag_key": "quality", "tag_value": "good"})
	eventRepo.seed("tag_deleted", 3, map[string]any{"asset_id": tagHistAssetID, "tag_key": "quality", "tag_value": "good"})
	eventRepo.seed("algo_started", 4, map[string]any{"asset_id": tagHistAssetID, "algo_key": "env_analysis@1.0.0"})
	uc := assetUC.NewWithProjections(handlerTxRunner{}, assetRepo, newHandlerAssetTagRepo(), nil, eventRepo, nil, nil)
	h := New(uc, &mockDeliveryRepoForAsset{})
	r := setupAssetRouter(http.MethodGet, "/assets/:id/tags/history", h.ListTagHistory)

	w := doReq(t, r, http.MethodGet, "/assets/"+tagHistAssetID+"/tags/history?limit=1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from tag history, got %d", w.Code)
	}
	var resp struct {
		Items      []models.AssetEvent `json:"items"`
		NextCursor *int64              `json:"next_cursor"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].EventType != "tag_deleted" {
		t.Fatalf("unexpected tag history response: %+v", resp)
	}
	if resp.NextCursor == nil || *resp.NextCursor != 3 {
		t.Fatalf("expected next_cursor=3, got %+v", resp.NextCursor)
	}
}

func TestAlgoListCurrent(t *testing.T) {
	h, _ := newAlgoEnv(t, true, map[string]string{"hand_tracking@1.2.0": "running"})
	r := setupAlgoRouter(h)
	w := doReq(t, r, http.MethodGet, "/assets/aaaaaaaa/algo", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	items, ok := resp["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected 1 algo row, got %v", resp["items"])
	}
}

// ─── Property 10: Soft-deleted assets never appear in query results ─────────
// **Validates: Requirements 3.6**
// This property is validated at the handler level by ensuring the ListWithFilters
// repo method is called (which always adds is_deleted=FALSE in the SQL).
// We verify the handler correctly delegates to ListWithFilters and returns
// only the items from the repo (which excludes soft-deleted assets).
func TestProperty10_SoftDeletedAssetsNeverInResults(t *testing.T) {
	// The mock repo simulates a repo that never returns soft-deleted assets.
	// We verify the handler returns exactly what the repo returns.
	repo := &mockAssetRepo{
		listWithFiltersFn: func(_ context.Context, _ string, _ []interface{}, _, _ int, _ filter.OrderByClause) ([]*models.Asset, int64, error) {
			// Only non-deleted assets returned.
			return []*models.Asset{
				{AssetID: "live-1"},
				{AssetID: "live-2"},
			}, 2, nil
		},
	}
	h := New(assetUC.New(repo), &mockDeliveryRepoForAsset{})
	r := setupAssetRouter(http.MethodGet, "/assets", h.List)

	w := doReq(t, r, http.MethodGet, "/assets?filter=status:eq:approved", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Items []struct {
			AssetID string `json:"asset_id"`
		} `json:"items"`
		Total int `json:"total"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Total != 2 {
		t.Fatalf("expected total=2, got %d", resp.Total)
	}
	for _, item := range resp.Items {
		if item.AssetID != "live-1" && item.AssetID != "live-2" {
			t.Fatalf("unexpected asset in results: %s", item.AssetID)
		}
	}
}

// ─── Test Helpers for Algo Handler ──────────────────────────────────────────

// handlerTxRunner runs fn directly. Repos in this file are not transactional
// in any meaningful sense — they hold serialised maps protected by their
// own mutex, mirroring the contract a single PG transaction provides.
type handlerTxRunner struct{}

func (handlerTxRunner) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type handlerAssetTagRepo struct {
	mu   sync.Mutex
	rows map[string]*models.AssetTag
}

func newHandlerAssetTagRepo() *handlerAssetTagRepo {
	return &handlerAssetTagRepo{rows: map[string]*models.AssetTag{}}
}

func tagRowKey(assetID, tagKey string) string { return assetID + "|" + tagKey }

func (m *handlerAssetTagRepo) Upsert(_ context.Context, in repository.AssetTagUpsertInput) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rows[tagRowKey(in.AssetID, in.TagKey)] = &models.AssetTag{
		AssetID:       in.AssetID,
		TagKey:        in.TagKey,
		TagValue:      in.TagValue,
		TagType:       in.TagType,
		SourceType:    in.SourceType,
		SourceName:    in.SourceName,
		SourceVersion: in.SourceVersion,
		RunID:         in.RunID,
		AppliedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	return nil
}

func (m *handlerAssetTagRepo) ListByAsset(_ context.Context, assetID string) ([]*models.AssetTag, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*models.AssetTag
	for _, row := range m.rows {
		if row.AssetID == assetID {
			cp := *row
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (m *handlerAssetTagRepo) Delete(_ context.Context, assetID, tagKey, sourceType string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if sourceType == "" {
		delete(m.rows, tagRowKey(assetID, tagKey))
		return nil
	}
	if row, ok := m.rows[tagRowKey(assetID, tagKey)]; ok && row.SourceType == sourceType {
		delete(m.rows, tagRowKey(assetID, tagKey))
	}
	return nil
}

// handlerAlgoLatestRepo is a tiny in-memory AssetAlgoLatestRepository used
// only by AlgoHandler tests. It enforces the (asset_id, algo_name) PK and
// monotonic algo_version guard like the production repo.
type handlerAlgoLatestRepo struct {
	mu   sync.Mutex
	rows map[string]*models.AssetAlgoLatest
}

func newHandlerAlgoLatestRepo() *handlerAlgoLatestRepo {
	return &handlerAlgoLatestRepo{rows: make(map[string]*models.AssetAlgoLatest)}
}

func handlerKey(assetID, algoName string) string { return assetID + "|" + algoName }

func (m *handlerAlgoLatestRepo) Upsert(_ context.Context, row *models.AssetAlgoLatest) error {
	if row == nil {
		return errors.New("nil row")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	k := handlerKey(row.AssetID, row.AlgoName)
	if cur, ok := m.rows[k]; ok && cur.AlgoVersion > row.AlgoVersion {
		return nil
	}
	cp := *row
	if cp.UpdatedAt.IsZero() {
		cp.UpdatedAt = time.Now().UTC()
	}
	m.rows[k] = &cp
	return nil
}

func (m *handlerAlgoLatestRepo) GetByAlgo(_ context.Context, assetID, algoName string) (*models.AssetAlgoLatest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rows[handlerKey(assetID, algoName)]
	if !ok {
		return nil, nil
	}
	cp := *r
	return &cp, nil
}

func (m *handlerAlgoLatestRepo) ListByAsset(_ context.Context, assetID string) ([]*models.AssetAlgoLatest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*models.AssetAlgoLatest
	for _, r := range m.rows {
		if r.AssetID == assetID {
			cp := *r
			out = append(out, &cp)
		}
	}
	return out, nil
}

// seed pre-populates state. algoKey is the "name@version" string used in
// the public API.
func (m *handlerAlgoLatestRepo) seed(algoKey, status, runID string) {
	parts := strings.SplitN(algoKey, "@", 2)
	if len(parts) != 2 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rows[handlerKey("aaaaaaaa", parts[0])] = &models.AssetAlgoLatest{
		AssetID: "aaaaaaaa", AlgoName: parts[0], AlgoVersion: parts[1],
		Status: status, RunID: runID, UpdatedAt: time.Now().UTC(),
	}
}

// handlerAssetEventRepo is a tiny in-memory AssetEventRepository used by handler tests.
type handlerAssetEventRepo struct {
	mu     sync.Mutex
	events []*models.AssetEvent
}

func (m *handlerAssetEventRepo) Append(_ context.Context, in repository.AssetEventAppendInput) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	payload := in.EventPayload
	if payload == nil {
		payload = []byte(`{}`)
	}
	m.events = append(m.events, &models.AssetEvent{
		EventID:      "evt-" + time.Now().UTC().Format("150405.000000000"),
		EventSeq:     int64(len(m.events) + 1),
		EventType:    in.EventType,
		AssetID:      in.AssetID,
		McapFileID:   in.McapFileID,
		EventPayload: append([]byte(nil), payload...),
		CreatedAt:    time.Now().UTC(),
		OccurredAt:   time.Now().UTC(),
	})
	return nil
}

func (m *handlerAssetEventRepo) ListPending(context.Context, int) ([]*models.AssetEvent, error) {
	return nil, nil
}

func (m *handlerAssetEventRepo) ListPendingSafe(_ context.Context, _ time.Duration, limit int) ([]*models.AssetEvent, error) {
	return m.ListPending(context.Background(), limit)
}

func (m *handlerAssetEventRepo) MarkPublished(context.Context, []int64) error { return nil }

func (m *handlerAssetEventRepo) MarkFailed(context.Context, int64, string) error { return nil }

func (m *handlerAssetEventRepo) CountPending(context.Context) (int64, error) { return 0, nil }
func (m *handlerAssetEventRepo) CountPendingClaimable(context.Context, time.Duration) (int64, error) {
	return 0, nil
}
func (m *handlerAssetEventRepo) CountProcessing(context.Context) (int64, error) { return 0, nil }

func (m *handlerAssetEventRepo) OldestPendingAge(context.Context) (float64, error) {
	return 0, nil
}

func (m *handlerAssetEventRepo) ListBetweenSeq(context.Context, int64, int64, int) ([]*models.AssetEvent, error) {
	return nil, nil
}

func (m *handlerAssetEventRepo) PublishStateCounts(context.Context) (map[string]int64, error) {
	return map[string]int64{}, nil
}

func (m *handlerAssetEventRepo) ListVersionPromotedByLogical(context.Context, string) ([]*models.AssetEvent, error) {
	return nil, nil
}

func (m *handlerAssetEventRepo) ListByAsset(_ context.Context, assetID string, opts repository.AssetEventListOptions) ([]*models.AssetEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}
	allow := map[string]struct{}{}
	for _, t := range opts.EventTypes {
		allow[t] = struct{}{}
	}
	var out []*models.AssetEvent
	for i := len(m.events) - 1; i >= 0; i-- {
		e := m.events[i]
		if e.AssetID != assetID {
			continue
		}
		if len(allow) > 0 {
			if _, ok := allow[e.EventType]; !ok {
				matched := false
				for _, pattern := range opts.EventTypePatterns {
					if strings.HasSuffix(pattern, "%") && strings.HasPrefix(e.EventType, strings.TrimSuffix(pattern, "%")) {
						matched = true
						break
					}
				}
				if !matched {
					continue
				}
			}
		} else if len(opts.EventTypePatterns) > 0 {
			matched := false
			for _, pattern := range opts.EventTypePatterns {
				if strings.HasSuffix(pattern, "%") && strings.HasPrefix(e.EventType, strings.TrimSuffix(pattern, "%")) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		if opts.AlgoKey != "" {
			var payload map[string]any
			if err := json.Unmarshal(e.EventPayload, &payload); err != nil {
				continue
			}
			if got, _ := payload["algo_key"].(string); got != opts.AlgoKey {
				continue
			}
		}
		if opts.BeforeEventSeq != nil && e.EventSeq >= *opts.BeforeEventSeq {
			continue
		}
		if opts.AfterEventSeq != nil && e.EventSeq <= *opts.AfterEventSeq {
			continue
		}
		out = append(out, e)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (m *handlerAssetEventRepo) ListGlobal(_ context.Context, opts repository.AssetEventListOptions) ([]*models.AssetEvent, error) {
	return nil, nil
}

func (m *handlerAssetEventRepo) seed(eventType string, seq int64, payload map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	raw, _ := json.Marshal(payload)
	aid := "aaaaaaaa"
	if payload != nil {
		if s, ok := payload["asset_id"].(string); ok && s != "" {
			aid = s
		}
	}
	m.events = append(m.events, &models.AssetEvent{
		EventID:      "evt-seed-" + strings.ReplaceAll(time.Now().UTC().Format(time.RFC3339Nano), ":", "-"),
		EventSeq:     seq,
		EventType:    eventType,
		AssetID:      aid,
		EventPayload: raw,
		CreatedAt:    time.Now().UTC(),
		OccurredAt:   time.Now().UTC(),
	})
}

func buildTestAlgoRegistry(t *testing.T) *config.AlgoRegistry {
	t.Helper()
	reg, err := config.LoadAlgoRegistry("../../../config/algo_registry.yaml")
	if err != nil {
		t.Fatalf("failed to load algo registry: %v", err)
	}
	return reg
}

func buildTestTagRegistryForAsset(t *testing.T) *config.TagRegistry {
	t.Helper()
	reg, err := config.LoadTagRegistry("../../../config/tag_registry.yaml")
	if err != nil {
		t.Fatalf("failed to load tag registry: %v", err)
	}
	return reg
}

func TestHandleRatingsHistoryNoPG(t *testing.T) {
	h := New(nil, nil)
	r := setupAssetRouter(http.MethodGet, "/logical-assets/:id/ratings-history", h.HandleRatingsHistory)

	w := doReq(t, r, http.MethodGet, "/logical-assets/abc12345/ratings-history", nil)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── buildLineageResponse tests ────────────────────────────────────────────

func TestBuildLineageResponse_nilDB(t *testing.T) {
	h := New(nil, nil)
	// pg and pgq both nil → early return with empty skeleton
	res, err := h.buildLineageResponse(context.Background(), "aa111111")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.AssetID != "aa111111" {
		t.Fatalf("expected asset_id aa111111, got %s", res.AssetID)
	}
	if len(res.Downstream["algo_results"].([]any)) != 0 {
		t.Fatal("expected empty algo_results")
	}
	if len(res.Downstream["deliveries"].([]any)) != 0 {
		t.Fatal("expected empty deliveries")
	}
	if len(res.Downstream["eval_results"].([]any)) != 0 {
		t.Fatal("expected empty eval_results")
	}
}

func TestBuildLineageResponse_happyPath(t *testing.T) {
	now := time.Now()
	algoRows := &fakeAssetSQLRows{
		data: [][]any{
			{"algo-a", "v1", "ok", "run-111", "s3://output/a"},
			{"algo-b", "v2", "ok", "run-222", "s3://output/b"},
		},
	}
	delRows := &fakeAssetSQLRows{
		data: [][]any{
			{"del-111", "c1", &now},
		},
	}
	evalRows := &fakeAssetSQLRows{
		data: [][]any{
			{"eval-1", "accuracy", float64(0.95)},
		},
	}
	// CYB-3281: downstream.children — (asset_id, asset_type, parent_asset_id, root_asset_id, import_batch)
	childRows := &fakeAssetSQLRows{
		data: [][]any{
			{"act00001", "action", "aa111111", "root0001", "batch-x"},
		},
	}

	q := &fakeAssetSQLQuerier{
		queries: []assetSQLQueryResult{
			{rows: algoRows},
			{rows: delRows},
			{rows: evalRows},
			{rows: childRows},
		},
	}

	h := New(nil, nil)
	h.pgq = q

	res, err := h.buildLineageResponse(context.Background(), "aa111111")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Round-trip through JSON to read response fields concretely.
	b, _ := json.Marshal(res)
	var body map[string]any
	if err := json.Unmarshal(b, &body); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}

	ds := body["downstream"].(map[string]any)

	algos := ds["algo_results"].([]any)
	if len(algos) != 2 {
		t.Fatalf("expected 2 algo results, got %d", len(algos))
	}
	a0 := algos[0].(map[string]any)
	if a0["algo_name"] != "algo-a" || a0["run_id"] != "run-111" {
		t.Fatalf("unexpected algo[0]: %v", a0)
	}

	dels := ds["deliveries"].([]any)
	if len(dels) != 1 {
		t.Fatalf("expected 1 delivery, got %d", len(dels))
	}
	d0 := dels[0].(map[string]any)
	if d0["delivery_id"] != "del-111" || d0["customer_id"] != "c1" {
		t.Fatalf("unexpected delivery[0]: %v", d0)
	}
	if _, ok := d0["delivered_at"]; !ok {
		t.Fatal("expected delivered_at for non-nil timestamp")
	}

	evals := ds["eval_results"].([]any)
	if len(evals) != 1 {
		t.Fatalf("expected 1 eval result, got %d", len(evals))
	}
	e0 := evals[0].(map[string]any)
	if e0["eval_name"] != "eval-1" || e0["metric_value"] != float64(0.95) {
		t.Fatalf("unexpected eval[0]: %v", e0)
	}

	// CYB-3281: downstream.children lists immediate child assets.
	children := ds["children"].([]any)
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	c0 := children[0].(map[string]any)
	if c0["asset_id"] != "act00001" || c0["asset_type"] != "action" || c0["parent_asset_id"] != "aa111111" {
		t.Fatalf("unexpected child[0]: %v", c0)
	}

	up := body["upstream"].(map[string]any)
	if len(up) != 0 {
		t.Fatal("expected empty upstream when pg is nil")
	}
}

func TestBuildLineageResponse_algoRowsErr(t *testing.T) {
	algoRows := &fakeAssetSQLRows{
		data: [][]any{
			{"algo-a", "v1", "ok", "run-111", "s3://output/a"},
		},
		err: fmt.Errorf("connection lost"),
	}
	delRows := &fakeAssetSQLRows{
		data: [][]any{
			{"del-111", "c1", nil},
		},
	}
	evalRows := &fakeAssetSQLRows{
		data: [][]any{
			{"eval-1", "accuracy", float64(0.92)},
		},
	}

	q := &fakeAssetSQLQuerier{
		queries: []assetSQLQueryResult{
			{rows: algoRows},
			{rows: delRows},
			{rows: evalRows},
		},
	}

	h := New(nil, nil)
	h.pgq = q

	res, err := h.buildLineageResponse(context.Background(), "aa111111")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	b, _ := json.Marshal(res)
	var body map[string]any
	if err := json.Unmarshal(b, &body); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	ds := body["downstream"].(map[string]any)

	// Algo row scanned before the error should still appear
	algos := ds["algo_results"].([]any)
	if len(algos) != 1 {
		t.Fatalf("expected 1 scanned algo result despite rows.Err, got %d", len(algos))
	}
	a0 := algos[0].(map[string]any)
	if a0["algo_name"] != "algo-a" {
		t.Fatalf("expected algo-a, got %v", a0)
	}

	dels := ds["deliveries"].([]any)
	if len(dels) != 1 {
		t.Fatalf("expected 1 delivery, got %d", len(dels))
	}
	evals := ds["eval_results"].([]any)
	if len(evals) != 1 {
		t.Fatalf("expected 1 eval result, got %d", len(evals))
	}
}

func TestBuildLineageResponse_scanErrorLogged(t *testing.T) {
	algoRows := &fakeAssetSQLRows{
		data: [][]any{
			{"algo-a", "v1", "ok", "run-111", "s3://output/a"},
			{"broken-row"},
		},
	}
	delRows := &fakeAssetSQLRows{data: [][]any{}}
	evalRows := &fakeAssetSQLRows{data: [][]any{}}

	q := &fakeAssetSQLQuerier{
		queries: []assetSQLQueryResult{
			{rows: algoRows},
			{rows: delRows},
			{rows: evalRows},
		},
	}

	h := New(nil, nil)
	h.pgq = q

	res, err := h.buildLineageResponse(context.Background(), "aa111111")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	b, _ := json.Marshal(res)
	var body map[string]any
	if err := json.Unmarshal(b, &body); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	ds := body["downstream"].(map[string]any)

	algos := ds["algo_results"].([]any)
	if len(algos) != 1 {
		t.Fatalf("expected 1 valid algo result (broken row skipped), got %d", len(algos))
	}
	a0 := algos[0].(map[string]any)
	if a0["algo_name"] != "algo-a" {
		t.Fatalf("expected algo-a, got %v", a0)
	}
}

// CYB-4294: POST /assets/durations — happy path, empty-ids 400, over-cap
// 400, invalid range 400.
func TestLookupDurationsHandler(t *testing.T) {
	repo := &mockAssetRepo{
		getFn: func(_ context.Context, id string) (*models.Asset, error) {
			switch id {
			case "aaaaaaaa":
				return &models.Asset{AssetID: "aaaaaaaa", DurationMs: 5_000}, nil
			case "bbbbbbbb":
				return &models.Asset{AssetID: "bbbbbbbb", DurationMs: 60_000}, nil
			}
			return nil, nil
		},
	}
	h := New(assetUC.New(repo), &mockDeliveryRepoForAsset{})
	r := setupAssetRouter(http.MethodPost, "/assets/durations", h.LookupDurations)

	// Happy path: two known ids, one missing.
	_mockLookupCalls = nil
	body := map[string]any{
		"ids":             []string{"aaaaaaaa", "bbbbbbbb", "zzzzzzzz"},
		"min_duration_ms": 0,
		"max_duration_ms": 0,
	}
	w := doReq(t, r, http.MethodPost, "/assets/durations", body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	items, _ := got["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d: %+v", len(items), got)
	}
	missing, _ := got["missing_ids"].([]any)
	if len(missing) != 1 || missing[0].(string) != "zzzzzzzz" {
		t.Fatalf("expected missing=[zzzzzzzz], got %+v", missing)
	}
	stats, _ := got["stats"].(map[string]any)
	if int(stats["matched_count"].(float64)) != 2 {
		t.Fatalf("expected matched_count=2, got %+v", stats)
	}
	if int(stats["missing_count"].(float64)) != 1 {
		t.Fatalf("expected missing_count=1, got %+v", stats)
	}
	// Repo was called with the 3 deduped ids and both bounds = 0.
	if len(_mockLookupCalls) != 1 || len(_mockLookupCalls[0].ids) != 3 {
		t.Fatalf("expected 1 repo call with 3 ids, got %+v", _mockLookupCalls)
	}

	// Empty ids → 400 IdListRequired.
	w = doReq(t, r, http.MethodPost, "/assets/durations", map[string]any{"ids": []string{}})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("empty ids expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "ID_LIST_REQUIRED") {
		t.Fatalf("expected ID_LIST_REQUIRED in body: %s", w.Body.String())
	}

	// Over 5000 → 400 IdListTooLarge.
	big := make([]string, 5001)
	for i := range big {
		big[i] = "id" + fmt.Sprintf("%08d", i)
	}
	w = doReq(t, r, http.MethodPost, "/assets/durations", map[string]any{"ids": big})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("over-cap expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "ID_LIST_TOO_LARGE") {
		t.Fatalf("expected ID_LIST_TOO_LARGE in body: %s", w.Body.String())
	}

	// min > max → 400 InvalidDurationRange.
	w = doReq(t, r, http.MethodPost, "/assets/durations", map[string]any{
		"ids":             []string{"aaaaaaaa"},
		"min_duration_ms": 1000,
		"max_duration_ms": 500,
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("min>max expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "INVALID_DURATION_RANGE") {
		t.Fatalf("expected INVALID_DURATION_RANGE in body: %s", w.Body.String())
	}

	// Negative bound → 400 InvalidDurationRange.
	w = doReq(t, r, http.MethodPost, "/assets/durations", map[string]any{
		"ids":             []string{"aaaaaaaa"},
		"min_duration_ms": -1,
	})
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "INVALID_DURATION_RANGE") {
		t.Fatalf("negative bound: expected 400 INVALID_DURATION_RANGE, got %d %s", w.Code, w.Body.String())
	}
}

// CYB-4306: POST /assets/costs — happy path plus every 400 branch.
func TestLookupCostsHandler(t *testing.T) {
	repo := &mockAssetRepo{
		getFn: func(_ context.Context, id string) (*models.Asset, error) {
			switch id {
			case "aaaaaaaa":
				return &models.Asset{AssetID: "aaaaaaaa"}, nil
			}
			return nil, nil
		},
		lookupCostsFn: func(_ context.Context, assetIDs []string, _, _ time.Time, _ bool) ([]repository.AssetCostRow, error) {
			out := []repository.AssetCostRow{}
			for _, a := range assetIDs {
				if a == "aaaaaaaa" {
					out = append(out, repository.AssetCostRow{
						AssetID: "aaaaaaaa", TotalCostUSD: 1.25, GPUSec: 60, CPUSec: 30, RunCount: 2,
					})
				}
			}
			return out, nil
		},
	}
	h := New(assetUC.New(repo), &mockDeliveryRepoForAsset{})
	r := setupAssetRouter(http.MethodPost, "/assets/costs", h.LookupCosts)

	start := time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	end := time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)

	// Happy path: one known id + one missing.
	body := map[string]any{
		"ids":      []string{"aaaaaaaa", "zzzzzzzz"},
		"start_at": start,
		"end_at":   end,
	}
	w := doReq(t, r, http.MethodPost, "/assets/costs", body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	items, _ := got["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d: %+v", len(items), got)
	}
	missing, _ := got["missing_ids"].([]any)
	if len(missing) != 1 || missing[0].(string) != "zzzzzzzz" {
		t.Fatalf("expected missing=[zzzzzzzz], got %+v", missing)
	}
	stats, _ := got["stats"].(map[string]any)
	if int(stats["matched_count"].(float64)) != 1 {
		t.Fatalf("expected matched_count=1, got %+v", stats)
	}

	// Empty ids → 400 IdListRequired.
	w = doReq(t, r, http.MethodPost, "/assets/costs", map[string]any{
		"ids":      []string{},
		"start_at": start,
		"end_at":   end,
	})
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "ID_LIST_REQUIRED") {
		t.Fatalf("empty ids: got %d %s", w.Code, w.Body.String())
	}

	// Over-cap → 400 IdListTooLarge.
	big := make([]string, 5001)
	for i := range big {
		big[i] = fmt.Sprintf("id%08d", i)
	}
	w = doReq(t, r, http.MethodPost, "/assets/costs", map[string]any{
		"ids":      big,
		"start_at": start,
		"end_at":   end,
	})
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "ID_LIST_TOO_LARGE") {
		t.Fatalf("over-cap: got %d %s", w.Code, w.Body.String())
	}

	// Missing start_at → 400 InvalidTimeRange.
	w = doReq(t, r, http.MethodPost, "/assets/costs", map[string]any{
		"ids":    []string{"aaaaaaaa"},
		"end_at": end,
	})
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "INVALID_TIME_RANGE") {
		t.Fatalf("missing start_at: got %d %s", w.Code, w.Body.String())
	}

	// Missing end_at → 400 InvalidTimeRange.
	w = doReq(t, r, http.MethodPost, "/assets/costs", map[string]any{
		"ids":      []string{"aaaaaaaa"},
		"start_at": start,
	})
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "INVALID_TIME_RANGE") {
		t.Fatalf("missing end_at: got %d %s", w.Code, w.Body.String())
	}

	// end < start → 400 InvalidTimeRange.
	w = doReq(t, r, http.MethodPost, "/assets/costs", map[string]any{
		"ids":      []string{"aaaaaaaa"},
		"start_at": end,
		"end_at":   start,
	})
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "INVALID_TIME_RANGE") {
		t.Fatalf("end<start: got %d %s", w.Code, w.Body.String())
	}

	// Window > 90 days → 400 WindowTooLarge.
	longEnd := time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC).Add(91 * 24 * time.Hour).Format(time.RFC3339)
	w = doReq(t, r, http.MethodPost, "/assets/costs", map[string]any{
		"ids":      []string{"aaaaaaaa"},
		"start_at": start,
		"end_at":   longEnd,
	})
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "WINDOW_TOO_LARGE") {
		t.Fatalf("window>90d: got %d %s", w.Code, w.Body.String())
	}

	// Bad group_by → 400 InvalidGroupBy.
	w = doReq(t, r, http.MethodPost, "/assets/costs", map[string]any{
		"ids":      []string{"aaaaaaaa"},
		"start_at": start,
		"end_at":   end,
		"group_by": "bogus",
	})
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "INVALID_GROUP_BY") {
		t.Fatalf("bad group_by: got %d %s", w.Code, w.Body.String())
	}

	// Valid "asset_algo" is accepted (200) even with no items — sanity check.
	w = doReq(t, r, http.MethodPost, "/assets/costs", map[string]any{
		"ids":      []string{"aaaaaaaa"},
		"start_at": start,
		"end_at":   end,
		"group_by": "asset_algo",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("asset_algo: got %d %s", w.Code, w.Body.String())
	}
}
