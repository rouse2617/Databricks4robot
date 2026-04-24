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

	"data-platform/internal/models"
	"data-platform/internal/repository"
)

type mockDeliveryRepo struct {
	setFn            func(ctx context.Context, d *models.Delivery) error
	getFn            func(ctx context.Context, deliveryID string) (*models.Delivery, error)
	writeIndexesFn   func(ctx context.Context, assetID string, d *models.Delivery) error
	listByCustomerFn func(ctx context.Context, customerID string) ([]string, error)
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
func (m *mockDeliveryRepo) WriteIndexes(ctx context.Context, assetID string, d *models.Delivery) error {
	if m.writeIndexesFn != nil {
		return m.writeIndexesFn(ctx, assetID, d)
	}
	return nil
}
func (m *mockDeliveryRepo) ListByCustomer(ctx context.Context, customerID string) ([]string, error) {
	if m.listByCustomerFn != nil {
		return m.listByCustomerFn(ctx, customerID)
	}
	return nil, nil
}

type mockIdemRepo struct {
	getFn  func(ctx context.Context, scope, key string) (*repository.IdempotencyRecord, error)
	saveFn func(ctx context.Context, rec *repository.IdempotencyRecord) error
}

func (m *mockIdemRepo) Get(ctx context.Context, scope, key string) (*repository.IdempotencyRecord, error) {
	if m.getFn != nil {
		return m.getFn(ctx, scope, key)
	}
	return nil, nil
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
	repo := &mockDeliveryRepo{}
	idem := &mockIdemRepo{}
	h := New(repo, idem)
	r := setupDeliveryRouter(http.MethodPost, "/deliveries", h.Commit)

	w := doDeliveryReq(t, r, http.MethodPost, "/deliveries", map[string]any{"x": 1}, nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	w = doDeliveryReq(t, r, http.MethodPost, "/deliveries", map[string]any{
		"asset_ids":   []string{"a1"},
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
		"asset_ids":   []string{"a1"},
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
		}{AssetIDs: []string{"a1"}, CustomerID: "c1"}
		h := hashDeliveryRequest(req)
		return &repository.IdempotencyRecord{
			RequestHash: h,
			StatusCode:  201,
			Response:    json.RawMessage(`{"delivery_id":"d1"}`),
		}, nil
	}
	w = doDeliveryReq(t, r, http.MethodPost, "/deliveries", map[string]any{
		"asset_ids":   []string{"a1"},
		"customer_id": "c1",
	}, map[string]string{"Idempotency-Key": "k1"})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 replay, got %d", w.Code)
	}

	// Fresh commit error on set
	idem.getFn = func(context.Context, string, string) (*repository.IdempotencyRecord, error) { return nil, nil }
	repo.setFn = func(context.Context, *models.Delivery) error { return errors.New("boom") }
	w = doDeliveryReq(t, r, http.MethodPost, "/deliveries", map[string]any{
		"asset_ids":   []string{"a1"},
		"customer_id": "c1",
	}, map[string]string{"Idempotency-Key": "k2"})
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	// Fresh commit success
	saved := false
	repo.setFn = nil
	idem.saveFn = func(context.Context, *repository.IdempotencyRecord) error { saved = true; return nil }
	repo.writeIndexesFn = func(context.Context, string, *models.Delivery) error { return errors.New("idx partial") }
	w = doDeliveryReq(t, r, http.MethodPost, "/deliveries", map[string]any{
		"asset_ids":   []string{"a1", "a2"},
		"customer_id": "c1",
	}, map[string]string{"Idempotency-Key": "k3"})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	if !saved {
		t.Fatalf("expected idempotency save")
	}
}

func TestGetAndListByCustomer(t *testing.T) {
	repo := &mockDeliveryRepo{
		getFn: func(context.Context, string) (*models.Delivery, error) { return nil, nil },
	}
	idem := &mockIdemRepo{}
	h := New(repo, idem)

	r := setupDeliveryRouter(http.MethodGet, "/deliveries/:id", h.Get)
	w := doDeliveryReq(t, r, http.MethodGet, "/deliveries/d1", nil, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	repo.getFn = func(context.Context, string) (*models.Delivery, error) { return nil, errors.New("boom") }
	w = doDeliveryReq(t, r, http.MethodGet, "/deliveries/d1", nil, nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	now := time.Now()
	repo.getFn = func(context.Context, string) (*models.Delivery, error) {
		return &models.Delivery{DeliveryID: "d1", DeliveredAt: &now}, nil
	}
	w = doDeliveryReq(t, r, http.MethodGet, "/deliveries/d1", nil, nil)
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

func TestDeliveryHelpers(t *testing.T) {
	p, s := parsePageParams("2", "10")
	if p != 2 || s != 10 {
		t.Fatalf("unexpected parse: %d %d", p, s)
	}
	p, s = parsePageParams("-1", "1000")
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
