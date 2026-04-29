package bigtable

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	gbt "cloud.google.com/go/bigtable"
	"github.com/gin-gonic/gin"

	"data-platform/internal/config"
	assetH "data-platform/internal/handlers/asset"
	deliveryH "data-platform/internal/handlers/delivery"
	mcapH "data-platform/internal/handlers/mcap"
	assetUC "data-platform/internal/usecase/asset"
	"data-platform/routes"
)

// e2eEnv holds the test server and helpers for end-to-end tests.
type e2eEnv struct {
	server *httptest.Server
	token  string
}

// setupE2E creates a full Gin server backed by fakeTable/fakeDataClient.
func setupE2E(t *testing.T) *e2eEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	// Create fakeDataClient with all required tables (storeOnApply=true).
	fd := &fakeDataClient{tables: map[string]*fakeTable{
		TableAssets:                {rows: map[string]gbt.Row{}, storeOnApply: true},
		TableMcapFiles:             {rows: map[string]gbt.Row{}, storeOnApply: true},
		TableDeliveries:            {rows: map[string]gbt.Row{}, storeOnApply: true},
		TableIdxSegmentsByFile:     {rows: map[string]gbt.Row{}, storeOnApply: true},
		TableIdxAssetDeliveries:    {rows: map[string]gbt.Row{}, storeOnApply: true},
		TableIdxCustomerDeliveries: {rows: map[string]gbt.Row{}, storeOnApply: true},
		TableIdempotencyKeys:       {rows: map[string]gbt.Row{}, storeOnApply: true},
		TableAssetAlgoEvents:       {rows: map[string]gbt.Row{}, storeOnApply: true},
	}}

	btClient := &Client{inner: fd}

	// Load registries from config directory.
	algoRegistry, err := config.LoadAlgoRegistry("../../config/algo_registry.yaml")
	if err != nil {
		t.Fatalf("load algo registry: %v", err)
	}
	tagRegistry, err := config.LoadTagRegistry("../../config/tag_registry.yaml")
	if err != nil {
		t.Fatalf("load tag registry: %v", err)
	}

	// Create repos.
	assetRepo := NewAssetRepo(btClient)
	deliveryRepo := NewDeliveryRepo(btClient)
	idemRepo := NewIdempotencyRepo(btClient)
	_ = NewAlgoEventRepo(btClient) // legacy; new AlgoUsecase no longer uses it

	// Create usecases.
	uc := assetUC.NewFull(assetRepo, tagRegistry, algoRegistry)
	// AlgoUsecase requires projection / event repos that the Bigtable
	// backend never implemented. Bigtable is deprecated as a runtime target
	// (CLAUDE.md) so this end-to-end test wires it with no-op mocks just
	// for compilation; algo endpoints are not exercised in this suite.
	algoUC := assetUC.NewAlgoUsecase(
		bigtableNoopTxRunner{},
		assetRepo,
		bigtableNoopAlgoLatestRepo{},
		bigtableNoopAssetEventRepo{},
		algoRegistry,
	)

	// Create handlers.
	assetHandler := assetH.New(uc, deliveryRepo)
	algoHandler := assetH.NewAlgoHandler(algoUC)
	mcapHandler := mcapH.New(NewMcapFileRepo(btClient))
	deliveryHandler := deliveryH.New(deliveryRepo, idemRepo)

	// Minimal config.
	cfg := &config.Config{
		Env:            "test",
		Port:           "0",
		StorageBackend: "bigtable",
		GraceToken:     "test-token",
	}

	// Build Gin engine and register all routes.
	r := gin.New()
	r.Use(gin.Recovery())
	routes.RegisterAll(r, cfg, assetHandler, mcapHandler, deliveryHandler, algoHandler, nil, nil, nil)

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	return &e2eEnv{server: srv, token: cfg.GraceToken}
}

// doJSON sends a JSON request and returns the response.
func (e *e2eEnv) doJSON(t *testing.T, method, path string, body interface{}) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, e.server.URL+path, bodyReader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Grace-Token", e.token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

// readJSON reads the response body into a map.
func readJSON(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	defer resp.Body.Close()
	var m map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return m
}

// ──────────────────────────────────────────────────────────────────────────────
// Task 12.2: Asset CRUD E2E Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestE2E_AssetCRUD(t *testing.T) {
	env := setupE2E(t)

	// 1. POST /api/v1/assets → create asset.
	createBody := map[string]interface{}{
		"mcap_file_id":       "mcap-001",
		"start_timestamp_ns": 1000000000,
		"end_timestamp_ns":   2000000000,
		"reviewer":           "alice",
		"owner":              "team-a",
		"type":               "task_demo",
		"env":                "indoor",
		"task":               "pick_and_place",
		"tags":               map[string]string{"priority": "high"},
	}
	resp := env.doJSON(t, "POST", "/api/v1/assets", createBody)
	if resp.StatusCode != http.StatusCreated {
		body := readJSON(t, resp)
		t.Fatalf("POST /assets: expected 201, got %d: %v", resp.StatusCode, body)
	}
	created := readJSON(t, resp)
	assetID, ok := created["asset_id"].(string)
	if !ok || assetID == "" {
		t.Fatalf("POST /assets: missing asset_id in response: %v", created)
	}
	if created["mcap_file_id"] != "mcap-001" {
		t.Fatalf("POST /assets: mcap_file_id mismatch: %v", created["mcap_file_id"])
	}
	if created["status"] != "approved" {
		t.Fatalf("POST /assets: expected status=approved, got %v", created["status"])
	}

	// 2. GET /api/v1/assets/:id → get asset.
	resp = env.doJSON(t, "GET", "/api/v1/assets/"+assetID, nil)
	if resp.StatusCode != http.StatusOK {
		body := readJSON(t, resp)
		t.Fatalf("GET /assets/:id: expected 200, got %d: %v", resp.StatusCode, body)
	}
	got := readJSON(t, resp)
	if got["asset_id"] != assetID {
		t.Fatalf("GET /assets/:id: asset_id mismatch")
	}
	// Verify Files and LifecycleMeta are present.
	if got["files"] == nil {
		t.Fatalf("GET /assets/:id: files should not be nil")
	}
	if got["lifecycle_meta"] == nil {
		t.Fatalf("GET /assets/:id: lifecycle_meta should not be nil")
	}

	// 3. PATCH /api/v1/assets/:id → update asset.
	updateBody := map[string]interface{}{
		"reviewer": "bob",
		"tags":     map[string]string{"quality": "good"},
	}
	resp = env.doJSON(t, "PATCH", "/api/v1/assets/"+assetID, updateBody)
	if resp.StatusCode != http.StatusOK {
		body := readJSON(t, resp)
		t.Fatalf("PATCH /assets/:id: expected 200, got %d: %v", resp.StatusCode, body)
	}
	updated := readJSON(t, resp)
	if updated["reviewer"] != "bob" {
		t.Fatalf("PATCH /assets/:id: reviewer not updated, got %v", updated["reviewer"])
	}

	// 4. DELETE /api/v1/assets/:id → soft delete.
	resp = env.doJSON(t, "DELETE", "/api/v1/assets/"+assetID, nil)
	if resp.StatusCode != http.StatusOK {
		body := readJSON(t, resp)
		t.Fatalf("DELETE /assets/:id: expected 200, got %d: %v", resp.StatusCode, body)
	}
	delResp := readJSON(t, resp)
	if delResp["deleted"] != true {
		t.Fatalf("DELETE /assets/:id: expected deleted=true, got %v", delResp["deleted"])
	}

	// 5. GET /api/v1/assets → list (should exclude archived asset).
	// Create a second asset that is NOT deleted.
	createBody2 := map[string]interface{}{
		"mcap_file_id":       "mcap-002",
		"start_timestamp_ns": 3000000000,
		"end_timestamp_ns":   4000000000,
		"reviewer":           "carol",
		"owner":              "team-b",
	}
	resp = env.doJSON(t, "POST", "/api/v1/assets", createBody2)
	if resp.StatusCode != http.StatusCreated {
		body := readJSON(t, resp)
		t.Fatalf("POST /assets (2nd): expected 201, got %d: %v", resp.StatusCode, body)
	}
	readJSON(t, resp) // consume body

	resp = env.doJSON(t, "GET", "/api/v1/assets", nil)
	if resp.StatusCode != http.StatusOK {
		body := readJSON(t, resp)
		t.Fatalf("GET /assets: expected 200, got %d: %v", resp.StatusCode, body)
	}
	listResp := readJSON(t, resp)
	items, ok := listResp["items"].([]interface{})
	if !ok {
		t.Fatalf("GET /assets: items not an array: %v", listResp["items"])
	}
	// The first asset was soft-deleted (archived), so only the second should appear.
	for _, item := range items {
		m := item.(map[string]interface{})
		if m["asset_id"] == assetID {
			t.Fatalf("GET /assets: archived asset should not appear in list")
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Task 12.3: Algo Lifecycle E2E Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestE2E_AlgoLifecycle(t *testing.T) {
	// The Bigtable backend is deprecated (CLAUDE.md). Algorithm state now
	// lives in the asset_algo_latest projection table, which Bigtable never
	// implemented — the e2e wiring for this suite uses no-op repos. Skip
	// the algo-lifecycle path; PostgreSQL's algo_usecase_test.go provides
	// the canonical coverage.
	t.Skip("algo lifecycle no longer supported on the deprecated bigtable backend; see internal/usecase/asset/algo_usecase_test.go")
	env := setupE2E(t)

	// Create an asset first.
	createBody := map[string]interface{}{
		"mcap_file_id":       "mcap-algo-001",
		"start_timestamp_ns": 1000000000,
		"end_timestamp_ns":   2000000000,
		"reviewer":           "alice",
		"owner":              "team-a",
	}
	resp := env.doJSON(t, "POST", "/api/v1/assets", createBody)
	if resp.StatusCode != http.StatusCreated {
		body := readJSON(t, resp)
		t.Fatalf("create asset: expected 201, got %d: %v", resp.StatusCode, body)
	}
	created := readJSON(t, resp)
	assetID := created["asset_id"].(string)

	// Use a valid algo key from algo_registry.yaml: hand_tracking@1.0.0
	algoKey := "hand_tracking@1.0.0"

	// 1. POST /api/v1/assets/:id/algo/:algo_key/start → start algo.
	startBody := map[string]interface{}{
		"method": "dagster",
		"run_id": "run-001",
	}
	resp = env.doJSON(t, "POST", "/api/v1/assets/"+assetID+"/algo/"+algoKey+"/start", startBody)
	if resp.StatusCode != http.StatusOK {
		body := readJSON(t, resp)
		t.Fatalf("start algo: expected 200, got %d: %v", resp.StatusCode, body)
	}
	startResp := readJSON(t, resp)
	if startResp["status"] != "running" {
		t.Fatalf("start algo: expected status=running, got %v", startResp["status"])
	}

	// 2. POST /api/v1/assets/:id/algo/:algo_key/finish → finish algo (ok).
	resultSize := int64(12345)
	finishBody := map[string]interface{}{
		"status":            "ok",
		"output_uri":        "gs://bucket/hand_tracking/output.mcap",
		"run_id":            "run-001",
		"result_size_bytes": resultSize,
		"extra_fields": map[string]interface{}{
			"type": "hand_tracking_v1",
		},
	}
	resp = env.doJSON(t, "POST", "/api/v1/assets/"+assetID+"/algo/"+algoKey+"/finish", finishBody)
	if resp.StatusCode != http.StatusOK {
		body := readJSON(t, resp)
		t.Fatalf("finish algo: expected 200, got %d: %v", resp.StatusCode, body)
	}
	finishResp := readJSON(t, resp)
	if finishResp["status"] != "ok" {
		t.Fatalf("finish algo: expected status=ok, got %v", finishResp["status"])
	}

	// 3. POST /api/v1/assets/:id/algo/:algo_key/reset → reset algo.
	resp = env.doJSON(t, "POST", "/api/v1/assets/"+assetID+"/algo/"+algoKey+"/reset", nil)
	if resp.StatusCode != http.StatusOK {
		body := readJSON(t, resp)
		t.Fatalf("reset algo: expected 200, got %d: %v", resp.StatusCode, body)
	}
	resetResp := readJSON(t, resp)
	if resetResp["status"] != "pending" {
		t.Fatalf("reset algo: expected status=pending, got %v", resetResp["status"])
	}

	// 4. GET /api/v1/assets/:id/algo-events → query events.
	resp = env.doJSON(t, "GET", "/api/v1/assets/"+assetID+"/algo-events", nil)
	if resp.StatusCode != http.StatusOK {
		body := readJSON(t, resp)
		t.Fatalf("list events: expected 200, got %d: %v", resp.StatusCode, body)
	}
	eventsResp := readJSON(t, resp)
	eventItems, ok := eventsResp["items"].([]interface{})
	if !ok {
		t.Fatalf("list events: items not an array: %v", eventsResp["items"])
	}
	// We did start → finish → reset = 3 events for hand_tracking@1.0.0.
	if len(eventItems) < 3 {
		t.Fatalf("list events: expected at least 3 events, got %d", len(eventItems))
	}

	// Verify state machine flow: collect all new_status values.
	// The fakeTable map iteration order is non-deterministic, so check that
	// all expected transitions exist rather than relying on order.
	statusSet := map[string]bool{}
	for _, item := range eventItems {
		ev := item.(map[string]interface{})
		if ev["algo_key"] == algoKey {
			statusSet[ev["new_status"].(string)] = true
		}
	}
	for _, expected := range []string{"running", "ok", "pending"} {
		if !statusSet[expected] {
			t.Fatalf("expected event with new_status=%q, not found in events", expected)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Task 12.4: ListDeliveries E2E Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestE2E_ListDeliveries(t *testing.T) {
	env := setupE2E(t)

	// Create an asset.
	createBody := map[string]interface{}{
		"mcap_file_id":       "mcap-del-001",
		"start_timestamp_ns": 1000000000,
		"end_timestamp_ns":   2000000000,
		"reviewer":           "alice",
		"owner":              "team-a",
	}
	resp := env.doJSON(t, "POST", "/api/v1/assets", createBody)
	if resp.StatusCode != http.StatusCreated {
		body := readJSON(t, resp)
		t.Fatalf("create asset: expected 201, got %d: %v", resp.StatusCode, body)
	}
	created := readJSON(t, resp)
	assetID := created["asset_id"].(string)

	// 1. GET /api/v1/assets/:id/deliveries → empty items initially.
	resp = env.doJSON(t, "GET", "/api/v1/assets/"+assetID+"/deliveries", nil)
	if resp.StatusCode != http.StatusOK {
		body := readJSON(t, resp)
		t.Fatalf("list deliveries (empty): expected 200, got %d: %v", resp.StatusCode, body)
	}
	delResp := readJSON(t, resp)
	items, ok := delResp["items"].([]interface{})
	if !ok {
		t.Fatalf("list deliveries: items not an array: %v", delResp["items"])
	}
	if len(items) != 0 {
		t.Fatalf("list deliveries: expected 0 items, got %d", len(items))
	}

	// 2. Create a delivery via POST /api/v1/deliveries.
	deliveryBody := map[string]interface{}{
		"asset_ids":   []string{assetID},
		"customer_id": "customer-001",
		"contract_id": "contract-001",
		"note":        "test delivery",
		"owner":       "team-a",
	}
	resp = env.doJSON(t, "POST", "/api/v1/deliveries", deliveryBody)
	// Need Idempotency-Key header.
	resp.Body.Close()

	// Retry with Idempotency-Key.
	req, _ := http.NewRequest("POST", env.server.URL+"/api/v1/deliveries", nil)
	b, _ := json.Marshal(deliveryBody)
	req.Body = io.NopCloser(bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Grace-Token", env.token)
	req.Header.Set("Idempotency-Key", "idem-key-001")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create delivery: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		body := readJSON(t, resp)
		t.Fatalf("create delivery: expected 201, got %d: %v", resp.StatusCode, body)
	}
	deliveryResp := readJSON(t, resp)
	deliveryID, ok := deliveryResp["delivery_id"].(string)
	if !ok || deliveryID == "" {
		t.Fatalf("create delivery: missing delivery_id: %v", deliveryResp)
	}

	// 3. GET /api/v1/assets/:id/deliveries → should now contain the delivery.
	resp = env.doJSON(t, "GET", "/api/v1/assets/"+assetID+"/deliveries", nil)
	if resp.StatusCode != http.StatusOK {
		body := readJSON(t, resp)
		t.Fatalf("list deliveries (after): expected 200, got %d: %v", resp.StatusCode, body)
	}
	delResp = readJSON(t, resp)
	items, ok = delResp["items"].([]interface{})
	if !ok {
		t.Fatalf("list deliveries: items not an array: %v", delResp["items"])
	}
	if len(items) != 1 {
		t.Fatalf("list deliveries: expected 1 item, got %d", len(items))
	}
	if items[0].(string) != deliveryID {
		t.Fatalf("list deliveries: expected delivery_id=%s, got %v", deliveryID, items[0])
	}
}
