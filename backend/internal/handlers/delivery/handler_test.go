package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/handlers"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

type mockDeliveryRepo struct {
	withTxCalled     bool
	setFn            func(ctx context.Context, d *models.Delivery) error
	getFn            func(ctx context.Context, deliveryID string) (*models.Delivery, error)
	addItemsFn       func(ctx context.Context, deliveryID string, assetIDs []string) error
	refreshAssetFn   func(ctx context.Context, assetID string) error
	updateFn         func(ctx context.Context, d *models.Delivery, expectedRowVersion int64) error
	listByCustomerFn func(ctx context.Context, customerID string) ([]string, error)
	listItemsFn      func(ctx context.Context, deliveryID string) ([]*models.DeliveryItem, error)
}

func (m *mockDeliveryRepo) Set(ctx context.Context, d *models.Delivery) error {
	if m.setFn != nil {
		return m.setFn(ctx, d)
	}
	return nil
}
func (m *mockDeliveryRepo) Get(ctx context.Context, deliveryID string) (*models.Delivery, error) {
	if m.getFn != nil {
		return m.getFn(ctx, deliveryID)
	}
	return nil, nil
}
func (m *mockDeliveryRepo) AddItems(ctx context.Context, deliveryID string, assetIDs []string) error {
	if m.addItemsFn != nil {
		return m.addItemsFn(ctx, deliveryID, assetIDs)
	}
	return nil
}
func (m *mockDeliveryRepo) RefreshAssetDeliveryIndex(ctx context.Context, assetID string) error {
	if m.refreshAssetFn != nil {
		return m.refreshAssetFn(ctx, assetID)
	}
	return nil
}
func (m *mockDeliveryRepo) ListByCustomer(ctx context.Context, customerID string) ([]string, error) {
	if m.listByCustomerFn != nil {
		return m.listByCustomerFn(ctx, customerID)
	}
	return nil, nil
}
func (m *mockDeliveryRepo) ListByAsset(_ context.Context, _ string) ([]string, error) {
	return []string{}, nil
}
func (m *mockDeliveryRepo) ListItems(ctx context.Context, deliveryID string) ([]*models.DeliveryItem, error) {
	if m.listItemsFn != nil {
		return m.listItemsFn(ctx, deliveryID)
	}
	return []*models.DeliveryItem{}, nil
}
func (m *mockDeliveryRepo) List(_ context.Context, _, _ int, _, _ string) ([]*models.Delivery, int64, error) {
	return []*models.Delivery{}, 0, nil
}

func (m *mockDeliveryRepo) WithTx(ctx context.Context, fn func(context.Context) error) error {
	m.withTxCalled = true
	return fn(ctx)
}
func (m *mockDeliveryRepo) Update(ctx context.Context, d *models.Delivery, expectedRowVersion int64) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, d, expectedRowVersion)
	}
	return nil
}

type mockIdemRepo struct {
	lockFn func(ctx context.Context, scope, key string) error
	getFn  func(ctx context.Context, scope, key string) (*repository.IdempotencyRecord, error)
	saveFn func(ctx context.Context, rec *repository.IdempotencyRecord) error
}

type mockAssetEventRepo struct {
	appendFn func(ctx context.Context, in repository.AssetEventAppendInput) error
}

func (m *mockAssetEventRepo) Append(ctx context.Context, in repository.AssetEventAppendInput) error {
	if m.appendFn != nil {
		return m.appendFn(ctx, in)
	}
	return nil
}
func (m *mockAssetEventRepo) ListPending(context.Context, int) ([]*models.AssetEvent, error) {
	return nil, nil
}
func (m *mockAssetEventRepo) ListPendingSafe(context.Context, time.Duration, int) ([]*models.AssetEvent, error) {
	return nil, nil
}
func (m *mockAssetEventRepo) ListByAsset(context.Context, string, repository.AssetEventListOptions) ([]*models.AssetEvent, error) {
	return nil, nil
}
func (m *mockAssetEventRepo) ListVersionPromotedByLogical(context.Context, string) ([]*models.AssetEvent, error) {
	return nil, nil
}

func (m *mockAssetEventRepo) ListGlobal(_ context.Context, opts repository.AssetEventListOptions) ([]*models.AssetEvent, error) {
	return nil, nil
}

func (m *mockAssetEventRepo) MarkPublished(context.Context, []int64) error { return nil }
func (m *mockAssetEventRepo) MarkFailed(context.Context, int64, string) error {
	return nil
}
func (m *mockAssetEventRepo) CountPending(context.Context) (int64, error) { return 0, nil }
func (m *mockAssetEventRepo) CountPendingClaimable(context.Context, time.Duration) (int64, error) {
	return 0, nil
}
func (m *mockAssetEventRepo) CountProcessing(context.Context) (int64, error) { return 0, nil }
func (m *mockAssetEventRepo) OldestPendingAge(context.Context) (float64, error) {
	return 0, nil
}
func (m *mockAssetEventRepo) ListBetweenSeq(context.Context, int64, int64, int) ([]*models.AssetEvent, error) {
	return nil, nil
}

func (m *mockAssetEventRepo) PublishStateCounts(context.Context) (map[string]int64, error) {
	return map[string]int64{}, nil
}

func (m *mockIdemRepo) Get(ctx context.Context, scope, key string) (*repository.IdempotencyRecord, error) {
	if m.getFn != nil {
		return m.getFn(ctx, scope, key)
	}
	return nil, nil
}
func (m *mockIdemRepo) Lock(ctx context.Context, scope, key string) error {
	if m.lockFn != nil {
		return m.lockFn(ctx, scope, key)
	}
	return nil
}
func (m *mockIdemRepo) Save(ctx context.Context, rec *repository.IdempotencyRecord) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, rec)
	}
	return nil
}

func setupDeliveryRouter(method, path string, fn gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Handle(method, path, fn)
	return r
}

func doDeliveryReq(t *testing.T, r *gin.Engine, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
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
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCommit(t *testing.T) {
	const (
		assetA = "aa111111"
		assetB = "bb222222"
	)
	repo := &mockDeliveryRepo{}
	idem := &mockIdemRepo{}
	h := New(repo, idem, nil)
	r := setupDeliveryRouter(http.MethodPost, "/deliveries", h.Commit)

	w := doDeliveryReq(t, r, http.MethodPost, "/deliveries", map[string]any{"x": 1}, nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	w = doDeliveryReq(t, r, http.MethodPost, "/deliveries", map[string]any{
		"asset_ids":   []string{"not-a-uuid"},
		"customer_id": "c1",
	}, map[string]string{"Idempotency-Key": "k-bad-uuid"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid asset uuid, got %d", w.Code)
	}

	w = doDeliveryReq(t, r, http.MethodPost, "/deliveries", map[string]any{
		"asset_ids":   []string{assetA},
		"customer_id": "c1",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 missing idem key, got %d", w.Code)
	}

	// Replay: same key, different hash -> 409
	idem.getFn = func(context.Context, string, string) (*repository.IdempotencyRecord, error) {
		return &repository.IdempotencyRecord{
			RequestHash: "different",
			StatusCode:  201,
			Response:    json.RawMessage(`{"delivery_id":"d1"}`),
		}, nil
	}
	w = doDeliveryReq(t, r, http.MethodPost, "/deliveries", map[string]any{
		"asset_ids":   []string{assetA},
		"customer_id": "c1",
	}, map[string]string{"Idempotency-Key": "k1"})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}

	// Replay success
	idem.getFn = func(context.Context, string, string) (*repository.IdempotencyRecord, error) {
		req := struct {
			AssetIDs   []string `json:"asset_ids"`
			CustomerID string   `json:"customer_id"`
			ContractID string   `json:"contract_id"`
			Note       string   `json:"note"`
			Owner      string   `json:"owner"`
		}{AssetIDs: []string{assetA}, CustomerID: "c1"}
		h := hashDeliveryRequest(req)
		return &repository.IdempotencyRecord{
			RequestHash: h,
			StatusCode:  201,
			Response:    json.RawMessage(`{"delivery_id":"d1"}`),
		}, nil
	}
	w = doDeliveryReq(t, r, http.MethodPost, "/deliveries", map[string]any{
		"asset_ids":   []string{assetA},
		"customer_id": "c1",
	}, map[string]string{"Idempotency-Key": "k1"})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 replay, got %d", w.Code)
	}

	// Fresh commit error on set
	idem.getFn = func(context.Context, string, string) (*repository.IdempotencyRecord, error) { return nil, nil }
	repo.setFn = func(context.Context, *models.Delivery) error { return errors.New("boom") }
	w = doDeliveryReq(t, r, http.MethodPost, "/deliveries", map[string]any{
		"asset_ids":   []string{assetA},
		"customer_id": "c1",
	}, map[string]string{"Idempotency-Key": "k2"})
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	// Fresh commit error on secondary index write
	saved := false
	repo.setFn = nil
	idem.saveFn = func(context.Context, *repository.IdempotencyRecord) error { saved = true; return nil }
	repo.refreshAssetFn = func(context.Context, string) error { return errors.New("idx partial") }
	w = doDeliveryReq(t, r, http.MethodPost, "/deliveries", map[string]any{
		"asset_ids":   []string{assetA, assetB},
		"customer_id": "c1",
	}, map[string]string{"Idempotency-Key": "k3"})
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	if saved {
		t.Fatalf("did not expect idempotency save on failed commit")
	}

	// Fresh commit success
	repo.refreshAssetFn = nil
	w = doDeliveryReq(t, r, http.MethodPost, "/deliveries", map[string]any{
		"asset_ids":   []string{assetA, assetB},
		"customer_id": "c1",
	}, map[string]string{"Idempotency-Key": "k4"})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	if !saved {
		t.Fatalf("expected idempotency save")
	}
}

func TestCommit_RechecksIdempotencyInsideTransaction(t *testing.T) {
	const assetID = "aa111111"
	getCalls := 0
	setCalled := false
	repo := &mockDeliveryRepo{
		setFn: func(context.Context, *models.Delivery) error {
			setCalled = true
			return nil
		},
	}
	idem := &mockIdemRepo{
		getFn: func(context.Context, string, string) (*repository.IdempotencyRecord, error) {
			getCalls++
			if getCalls == 1 {
				return nil, nil
			}
			return &repository.IdempotencyRecord{
				RequestHash: "different",
				StatusCode:  http.StatusCreated,
				Response:    json.RawMessage(`{"delivery_id":"other"}`),
			}, nil
		},
	}
	h := New(repo, idem, nil)
	r := setupDeliveryRouter(http.MethodPost, "/deliveries", h.Commit)

	w := doDeliveryReq(t, r, http.MethodPost, "/deliveries", map[string]any{
		"asset_ids":   []string{assetID},
		"customer_id": "c1",
	}, map[string]string{"Idempotency-Key": "k-race"})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 after second idempotency check, got %d body=%s", w.Code, w.Body.String())
	}
	if setCalled {
		t.Fatal("did not expect delivery side effects after idempotency conflict")
	}
	if getCalls < 2 {
		t.Fatalf("expected idempotency to be checked before and inside transaction, got %d call(s)", getCalls)
	}
}

func TestHandleAddItems_DuplicateDoesNotInflateAssetCount(t *testing.T) {
	validUUID := "66666666-6666-4666-8666-666666666666"
	var updated *models.Delivery
	repo := &mockDeliveryRepo{
		getFn: func(context.Context, string) (*models.Delivery, error) {
			return &models.Delivery{
				DeliveryID: validUUID,
				Status:     models.DeliveryStatusPending,
				AssetCount: 1,
				ItemCount:  1,
				Version:    2,
			}, nil
		},
		listItemsFn: func(context.Context, string) ([]*models.DeliveryItem, error) {
			return []*models.DeliveryItem{{DeliveryID: validUUID, AssetID: "aa111111"}}, nil
		},
		updateFn: func(_ context.Context, d *models.Delivery, expectedRowVersion int64) error {
			if expectedRowVersion != 2 {
				t.Fatalf("expected version 2, got %d", expectedRowVersion)
			}
			cp := *d
			updated = &cp
			return nil
		},
	}
	h := New(repo, &mockIdemRepo{}, nil)
	r := setupDeliveryRouter(http.MethodPost, "/deliveries/:id/items", h.HandleAddItems)

	w := doDeliveryReq(t, r, http.MethodPost, "/deliveries/"+validUUID+"/items", map[string]any{
		"asset_ids": []string{"aa111111"},
	}, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if updated == nil {
		t.Fatal("expected update")
	}
	if updated.AssetCount != 1 || updated.ItemCount != 1 {
		t.Fatalf("expected count to remain 1/1, got %d/%d", updated.AssetCount, updated.ItemCount)
	}
	var body struct {
		AssetCount int `json:"asset_count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.AssetCount != 1 {
		t.Fatalf("expected response asset_count=1, got %d", body.AssetCount)
	}
}

func TestHandleCancel_DoesNotUpdateWhenRefreshFails(t *testing.T) {
	validUUID := "77777777-7777-4777-8777-777777777777"
	repo := &mockDeliveryRepo{
		getFn: func(context.Context, string) (*models.Delivery, error) {
			return &models.Delivery{
				DeliveryID: validUUID,
				Status:     models.DeliveryStatusDelivered,
				Version:    4,
			}, nil
		},
		listItemsFn: func(context.Context, string) ([]*models.DeliveryItem, error) {
			return []*models.DeliveryItem{{DeliveryID: validUUID, AssetID: "aa111111"}}, nil
		},
		updateFn: func(context.Context, *models.Delivery, int64) error {
			return nil
		},
		refreshAssetFn: func(context.Context, string) error {
			return errors.New("refresh failed")
		},
	}
	h := New(repo, &mockIdemRepo{}, nil)
	r := setupDeliveryRouter(http.MethodPost, "/deliveries/:id/cancel", h.HandleCancel)

	w := doDeliveryReq(t, r, http.MethodPost, "/deliveries/"+validUUID+"/cancel", map[string]any{
		"cancelled_by":  "ops-user",
		"cancel_reason": "customer revoked",
	}, nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
	}
	if !repo.withTxCalled {
		t.Fatal("expected cancel update and index refresh to run in one transaction")
	}
}

func TestGetAndListByCustomer(t *testing.T) {
	validUUID := "11111111-1111-1111-1111-111111111111"
	repo := &mockDeliveryRepo{
		getFn: func(context.Context, string) (*models.Delivery, error) { return nil, nil },
	}
	idem := &mockIdemRepo{}
	h := New(repo, idem, nil)

	r := setupDeliveryRouter(http.MethodGet, "/deliveries/:id", h.Get)

	// invalid UUID → 400
	w := doDeliveryReq(t, r, http.MethodGet, "/deliveries/not-a-uuid", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid UUID, got %d", w.Code)
	}

	// valid UUID, not found → 404
	w = doDeliveryReq(t, r, http.MethodGet, "/deliveries/"+validUUID, nil, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	// repo error → 500
	repo.getFn = func(context.Context, string) (*models.Delivery, error) { return nil, errors.New("boom") }
	w = doDeliveryReq(t, r, http.MethodGet, "/deliveries/"+validUUID, nil, nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	now := time.Now()
	repo.getFn = func(context.Context, string) (*models.Delivery, error) {
		return &models.Delivery{DeliveryID: validUUID, DeliveredAt: &now}, nil
	}
	w = doDeliveryReq(t, r, http.MethodGet, "/deliveries/"+validUUID, nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	repo.listByCustomerFn = func(context.Context, string) ([]string, error) { return []string{"d1", "d2", "d3"}, nil }
	r = setupDeliveryRouter(http.MethodGet, "/customers/:customer_id/deliveries", h.ListByCustomer)
	w = doDeliveryReq(t, r, http.MethodGet, "/customers/c1/deliveries?page=1&page_size=2", nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	repo.listByCustomerFn = func(context.Context, string) ([]string, error) { return nil, errors.New("boom") }
	w = doDeliveryReq(t, r, http.MethodGet, "/customers/c1/deliveries", nil, nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestListItems(t *testing.T) {
	validUUID := "22222222-2222-2222-2222-222222222222"
	repo := &mockDeliveryRepo{
		getFn: func(context.Context, string) (*models.Delivery, error) {
			now := time.Now()
			return &models.Delivery{DeliveryID: validUUID, DeliveredAt: &now}, nil
		},
		listItemsFn: func(context.Context, string) ([]*models.DeliveryItem, error) {
			return []*models.DeliveryItem{
				{DeliveryID: validUUID, AssetID: "a1"},
				{DeliveryID: validUUID, AssetID: "a2"},
			}, nil
		},
	}
	h := New(repo, &mockIdemRepo{}, nil)
	r := setupDeliveryRouter(http.MethodGet, "/deliveries/:id/items", h.ListItems)

	// invalid UUID → 400
	w := doDeliveryReq(t, r, http.MethodGet, "/deliveries/not-a-uuid/items", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid UUID, got %d", w.Code)
	}

	w = doDeliveryReq(t, r, http.MethodGet, "/deliveries/"+validUUID+"/items", nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	repo.listItemsFn = func(context.Context, string) ([]*models.DeliveryItem, error) {
		return nil, errors.New("boom")
	}
	w = doDeliveryReq(t, r, http.MethodGet, "/deliveries/"+validUUID+"/items", nil, nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	repo.getFn = func(context.Context, string) (*models.Delivery, error) { return nil, nil }
	w = doDeliveryReq(t, r, http.MethodGet, "/deliveries/"+validUUID+"/items", nil, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestDeliveryHelpers(t *testing.T) {
	p, s := handlers.ParsePageParams("2", "10")
	if p != 2 || s != 10 {
		t.Fatalf("unexpected parse: %d %d", p, s)
	}
	p, s = handlers.ParsePageParams("-1", "1000")
	if p != 1 || s != 20 {
		t.Fatalf("expected defaults")
	}

	ids, next := paginateIDs([]string{"a", "b", "c"}, 1, 2)
	if len(ids) != 2 || next != "2" {
		t.Fatalf("unexpected first page")
	}
	ids, next = paginateIDs([]string{"a"}, 2, 1)
	if len(ids) != 0 || next != "" {
		t.Fatalf("expected empty page")
	}
	ids, next = paginateIDs(nil, 1, 20)
	if len(ids) != 0 || next != "" {
		t.Fatalf("expected empty list")
	}

	h1 := hashDeliveryRequest(map[string]any{"a": 1})
	h2 := hashDeliveryRequest(map[string]any{"a": 1})
	if h1 != h2 {
		t.Fatalf("hash should be stable")
	}
}

func TestCommit_AppendsAssetEvents(t *testing.T) {
	const assetID = "aa111111"
	repo := &mockDeliveryRepo{}
	idem := &mockIdemRepo{
		getFn: func(context.Context, string, string) (*repository.IdempotencyRecord, error) { return nil, nil },
	}
	appended := 0
	eventRepo := &mockAssetEventRepo{
		appendFn: func(_ context.Context, in repository.AssetEventAppendInput) error {
			appended++
			if in.EventType != "delivery_committed" {
				t.Fatalf("unexpected event type: %s", in.EventType)
			}
			if in.AssetID != assetID {
				t.Fatalf("unexpected asset id: %s", in.AssetID)
			}
			return nil
		},
	}
	h := New(repo, idem, nil, eventRepo)
	r := setupDeliveryRouter(http.MethodPost, "/deliveries", h.Commit)

	w := doDeliveryReq(t, r, http.MethodPost, "/deliveries", map[string]any{
		"asset_ids":   []string{assetID},
		"customer_id": "cust-1",
	}, map[string]string{"Idempotency-Key": "k-evt-1", "X-Request-ID": "req-evt"})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
	}
	if appended != 1 {
		t.Fatalf("expected 1 appended event, got %d", appended)
	}
}

func TestHandleAck_TransitionsToAccepted(t *testing.T) {
	validUUID := "33333333-3333-4333-8333-333333333333"
	var updated *models.Delivery
	repo := &mockDeliveryRepo{
		getFn: func(context.Context, string) (*models.Delivery, error) {
			return &models.Delivery{
				DeliveryID: validUUID,
				Status:     models.DeliveryStatusDelivered,
				Version:    3,
			}, nil
		},
		updateFn: func(_ context.Context, d *models.Delivery, expectedRowVersion int64) error {
			if expectedRowVersion != 3 {
				t.Fatalf("expected row version 3, got %d", expectedRowVersion)
			}
			cp := *d
			updated = &cp
			return nil
		},
	}
	h := New(repo, &mockIdemRepo{}, nil)
	r := setupDeliveryRouter(http.MethodPost, "/deliveries/:id/ack", h.HandleAck)

	w := doDeliveryReq(t, r, http.MethodPost, "/deliveries/"+validUUID+"/ack", map[string]any{
		"acknowledged_by": "ops-user",
	}, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if updated == nil {
		t.Fatal("expected update to be called")
	}
	if updated.Status != models.DeliveryStatusAccepted {
		t.Fatalf("expected status accepted, got %s", updated.Status)
	}
	if updated.AcknowledgedBy != "ops-user" {
		t.Fatalf("expected acknowledged_by to be set, got %q", updated.AcknowledgedBy)
	}
	if updated.AcknowledgedAt == nil {
		t.Fatal("expected acknowledged_at to be set")
	}
}

func TestHandleCancel_RefreshesAssetIndexesForDeliveredDelivery(t *testing.T) {
	validUUID := "44444444-4444-4444-8444-444444444444"
	refreshed := map[string]int{}
	repo := &mockDeliveryRepo{
		getFn: func(context.Context, string) (*models.Delivery, error) {
			return &models.Delivery{
				DeliveryID: validUUID,
				Status:     models.DeliveryStatusDelivered,
				Version:    4,
			}, nil
		},
		listItemsFn: func(context.Context, string) ([]*models.DeliveryItem, error) {
			return []*models.DeliveryItem{
				{DeliveryID: validUUID, AssetID: "aa111111"},
				{DeliveryID: validUUID, AssetID: "bb222222"},
			}, nil
		},
		refreshAssetFn: func(_ context.Context, assetID string) error {
			refreshed[assetID]++
			return nil
		},
	}
	h := New(repo, &mockIdemRepo{}, nil)
	r := setupDeliveryRouter(http.MethodPost, "/deliveries/:id/cancel", h.HandleCancel)

	w := doDeliveryReq(t, r, http.MethodPost, "/deliveries/"+validUUID+"/cancel", map[string]any{
		"cancelled_by":  "ops-user",
		"cancel_reason": "customer revoked",
	}, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if refreshed["aa111111"] != 1 || refreshed["bb222222"] != 1 {
		t.Fatalf("expected asset indexes refreshed once each, got %#v", refreshed)
	}
}

func TestHandleCancel_PendingDoesNotRefreshAssetIndexes(t *testing.T) {
	validUUID := "55555555-5555-4555-8555-555555555555"
	refreshed := 0
	repo := &mockDeliveryRepo{
		getFn: func(context.Context, string) (*models.Delivery, error) {
			return &models.Delivery{
				DeliveryID: validUUID,
				Status:     models.DeliveryStatusPending,
				Version:    2,
			}, nil
		},
		refreshAssetFn: func(_ context.Context, assetID string) error {
			refreshed++
			return nil
		},
	}
	h := New(repo, &mockIdemRepo{}, nil)
	r := setupDeliveryRouter(http.MethodPost, "/deliveries/:id/cancel", h.HandleCancel)

	w := doDeliveryReq(t, r, http.MethodPost, "/deliveries/"+validUUID+"/cancel", map[string]any{
		"cancelled_by":  "ops-user",
		"cancel_reason": "operator stop",
	}, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if refreshed != 0 {
		t.Fatalf("expected no asset index refresh for pending cancellation, got %d", refreshed)
	}
}
