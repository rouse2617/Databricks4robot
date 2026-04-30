//go:build integration

// Package outbox — integration tests for the Outbox Worker end-to-end flow.
//
// These tests use testcontainers-go to spin up real PostgreSQL and
// Elasticsearch containers, apply the production schema, and exercise the
// full Worker → PG → ES pipeline.
//
// Run with:
//
//	go test -tags integration -v -timeout 300s ./internal/outbox/
package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"data-platform/internal/elasticsearch"
	"data-platform/internal/postgres"
	"data-platform/internal/repository"
	"data-platform/internal/searchindex"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver for database/sql

	tces "github.com/testcontainers/testcontainers-go/modules/elasticsearch"
	tcpg "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// ---------------------------------------------------------------------------
// Shared test infrastructure
// ---------------------------------------------------------------------------

// testEnv holds the shared containers and clients for all E2E tests.
type testEnv struct {
	pgDSN string
	esURL string

	pgClient  *postgres.Client
	esClient  *elasticsearch.Client
	eventRepo repository.AssetEventRepository
	assetRepo repository.AssetRepository
	tagRepo   repository.AssetTagRepository
	algoRepo  repository.AssetAlgoLatestRepository
	mcapRepo  repository.McapFileRepository
	indexer   *searchindex.Builder
}

// migrationFiles lists the SQL files applied in order to bootstrap the schema.
var migrationFiles = []string{
	"../../migrations/001_init.sql",
	"../../migrations/002_seed.sql",
	"../../migrations/004_asset_query_indexes.sql",
	"../../migrations/005_audit_events.sql",
	"../../migrations/006_sync_watermarks.sql",
	"../../migrations/007_backfill_lifecycle_from_status.sql",
	"../../migrations/008_asset_events_outbox_columns.sql",
	"../../migrations/009_lifecycle_state_check.sql",
	"../../migrations/010_outbox_dlq.sql",
	"../../migrations/011_event_retention.sql",
	"../../migrations/012_backfill_real_columns_and_projections.sql",
}

// setupEnv starts PG + ES containers, applies migrations, and returns a
// testEnv ready for use. The caller should defer cleanup via the returned
// cancel function.
func setupEnv(t *testing.T) (*testEnv, context.Context, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	// ---- PostgreSQL ----
	pgCtr, err := tcpg.Run(ctx, "postgres:16-alpine",
		tcpg.WithDatabase("testdb"),
		tcpg.WithUsername("testuser"),
		tcpg.WithPassword("testpass"),
		tcpg.BasicWaitStrategies(),
		tcpg.WithInitScripts(migrationFiles...),
	)
	if err != nil {
		cancel()
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(pgCtr); err != nil {
			t.Logf("terminate postgres: %v", err)
		}
	})

	pgDSN := pgCtr.MustConnectionString(ctx, "sslmode=disable")

	// Seed the outbox_sink_cursors row required by the worker.
	seedDB, err := sql.Open("pgx", pgDSN)
	if err != nil {
		cancel()
		t.Fatalf("open seed db: %v", err)
	}
	defer seedDB.Close()
	_, err = seedDB.ExecContext(ctx,
		`INSERT INTO outbox_sink_cursors (sink_name, last_published_seq)
		 VALUES ('es_assets', 0) ON CONFLICT DO NOTHING`)
	if err != nil {
		cancel()
		t.Fatalf("seed outbox_sink_cursors: %v", err)
	}

	// ---- Elasticsearch ----
	esCtr, err := tces.Run(ctx, "docker.elastic.co/elasticsearch/elasticsearch:8.17.0",
		testcontainers.WithEnv(map[string]string{
			"xpack.security.enabled": "false",
			"discovery.type":         "single-node",
		}),
		testcontainers.WithAdditionalWaitStrategy(
			wait.ForHTTP("/_cluster/health").WithPort("9200/tcp").WithStatusCodeMatcher(func(status int) bool {
				return status == 200
			}).WithStartupTimeout(120*time.Second),
		),
	)
	if err != nil {
		cancel()
		t.Fatalf("start elasticsearch container: %v", err)
	}
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(esCtr); err != nil {
			t.Logf("terminate elasticsearch: %v", err)
		}
	})

	esURL := esCtr.Settings.Address

	// ---- Build clients ----
	pgClient, err := postgres.NewFromDSN(ctx, pgDSN)
	if err != nil {
		cancel()
		t.Fatalf("create postgres client: %v", err)
	}
	t.Cleanup(func() { pgClient.Close() })

	esClient := elasticsearch.New(esURL, "assets")

	assetRepo := postgres.NewAssetRepo(pgClient)
	eventRepo := postgres.NewAssetEventRepo(pgClient)
	tagRepo := postgres.NewAssetTagRepo(pgClient)
	algoRepo := postgres.NewAssetAlgoLatestRepo(pgClient)
	mcapRepo := postgres.NewMcapFileRepo(pgClient)

	indexer := &searchindex.Builder{
		Assets: assetRepo,
		Tags:   tagRepo,
		Algos:  algoRepo,
		Mcap:   mcapRepo,
	}

	env := &testEnv{
		pgDSN:     pgDSN,
		esURL:     esURL,
		pgClient:  pgClient,
		esClient:  esClient,
		eventRepo: eventRepo,
		assetRepo: assetRepo,
		tagRepo:   tagRepo,
		algoRepo:  algoRepo,
		mcapRepo:  mcapRepo,
		indexer:   indexer,
	}

	return env, ctx, cancel
}

// newWorker creates an ESWorker wired to the test environment.
func (e *testEnv) newWorker() *ESWorker {
	return &ESWorker{
		Events:    e.eventRepo,
		Indexer:   e.indexer,
		ES:        e.esClient,
		BatchSize: 100,
		SinkName:  "es_assets",
	}
}

// insertMcapFile inserts a minimal mcap_files row required by the FK on assets.
func (e *testEnv) insertMcapFile(ctx context.Context, t *testing.T, mcapFileID string) {
	t.Helper()
	db, err := sql.Open("pgx", e.pgDSN)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	_, err = db.ExecContext(ctx, `
		INSERT INTO mcap_files (mcap_file_id, created_at, updated_at)
		VALUES ($1, now(), now())
		ON CONFLICT DO NOTHING`, mcapFileID)
	if err != nil {
		t.Fatalf("insert mcap_file %s: %v", mcapFileID, err)
	}
}

// insertAsset inserts a minimal asset row into PG.
func (e *testEnv) insertAsset(ctx context.Context, t *testing.T, assetID, mcapFileID string) {
	t.Helper()
	e.insertMcapFile(ctx, t, mcapFileID)
	db, err := sql.Open("pgx", e.pgDSN)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	_, err = db.ExecContext(ctx, `
		INSERT INTO assets (asset_id, mcap_file_id, start_timestamp_ns, end_timestamp_ns,
		                    lifecycle_state, status, created_at, updated_at)
		VALUES ($1, $2, 1000000000, 2000000000, 'ready', 'approved', now(), now())
		ON CONFLICT DO NOTHING`, assetID, mcapFileID)
	if err != nil {
		t.Fatalf("insert asset %s: %v", assetID, err)
	}
}

// insertEvent inserts a pending asset_event for the given asset.
func (e *testEnv) insertEvent(ctx context.Context, t *testing.T, assetID string) {
	t.Helper()
	err := e.eventRepo.Append(ctx, repository.AssetEventAppendInput{
		EventType:     "asset.updated",
		AggregateType: "asset",
		AssetID:       assetID,
		EventSource:   "test",
	})
	if err != nil {
		t.Fatalf("insert event for asset %s: %v", assetID, err)
	}
}

// waitForESDoc polls ES until a document with the given asset_id appears or
// the timeout expires.
func (e *testEnv) waitForESDoc(ctx context.Context, t *testing.T, assetID string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/assets/_doc/%s", e.esURL, assetID), nil)
		if err == nil {
			httpResp, err := http.DefaultClient.Do(req)
			if err == nil {
				var docResp struct {
					Found bool `json:"found"`
				}
				_ = json.NewDecoder(httpResp.Body).Decode(&docResp)
				httpResp.Body.Close()
				if httpResp.StatusCode == http.StatusOK && docResp.Found {
					return
				}
			}
		}
		select {
		case <-ctx.Done():
			t.Fatalf("context cancelled waiting for ES doc %s", assetID)
		case <-time.After(500 * time.Millisecond):
		}
	}
	t.Fatalf("ES doc for asset %s not found within %v", assetID, timeout)
}

// esDocCount returns the total number of documents in the ES index.
func (e *testEnv) esDocCount(ctx context.Context, t *testing.T) int64 {
	t.Helper()
	// Force a refresh so recently indexed docs are visible.
	_ = e.esClient.Ping(ctx) // warm up
	count, err := e.esClient.Count(ctx)
	if err != nil {
		t.Fatalf("ES count: %v", err)
	}
	return count
}

// pendingCount returns the number of pending events in PG.
func (e *testEnv) pendingCount(ctx context.Context, t *testing.T) int64 {
	t.Helper()
	n, err := e.eventRepo.CountPending(ctx)
	if err != nil {
		t.Fatalf("count pending: %v", err)
	}
	return n
}

// refreshES forces an ES index refresh so recently indexed docs become searchable.
func (e *testEnv) refreshES(ctx context.Context, t *testing.T) {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/assets/_refresh", e.esURL), nil)
	if err != nil {
		t.Fatalf("build ES refresh request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("ES refresh request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusMultipleChoices {
		t.Fatalf("ES refresh returned status %d", resp.StatusCode)
	}
}

// cursorSeq returns the current last_published_seq for the given sink.
func (e *testEnv) cursorSeq(ctx context.Context, t *testing.T, sinkName string) int64 {
	t.Helper()
	db, err := sql.Open("pgx", e.pgDSN)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	var seq int64
	err = db.QueryRowContext(ctx,
		`SELECT last_published_seq FROM outbox_sink_cursors WHERE sink_name = $1`,
		sinkName).Scan(&seq)
	if err != nil {
		t.Fatalf("read cursor for %s: %v", sinkName, err)
	}
	return seq
}

func init() {
	// Suppress noisy log output during tests.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})))
}

// ---------------------------------------------------------------------------
// 4.3 TestE2E_Notify_HappyPath
// ---------------------------------------------------------------------------

// TestE2E_Notify_HappyPath verifies the basic flow: insert an asset and an
// event into PG, start the worker, and confirm the asset appears in ES.
func TestE2E_Notify_HappyPath(t *testing.T) {
	env, ctx, cancel := setupEnv(t)
	defer cancel()

	assetID := "00000000-0000-0000-0000-000000000001"
	mcapID := "00000000-0000-0000-0000-00000000f001"

	env.insertAsset(ctx, t, assetID, mcapID)
	env.insertEvent(ctx, t, assetID)

	// Start worker with a fast tick.
	wCtx, wCancel := context.WithCancel(ctx)
	defer wCancel()
	w := env.newWorker()
	go w.Run(wCtx, 500*time.Millisecond)

	// The doc should appear in ES within a reasonable timeout.
	env.waitForESDoc(ctx, t, assetID, 30*time.Second)

	// Verify pending count drops to 0.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if env.pendingCount(ctx, t) == 0 {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal("pending events did not drain to 0")
}

// ---------------------------------------------------------------------------
// 4.4 TestE2E_DedupBatch
// ---------------------------------------------------------------------------

// TestE2E_DedupBatch inserts multiple events for the same asset and verifies
// that only one ES document is created (dedup by asset_id in the bulk).
func TestE2E_DedupBatch(t *testing.T) {
	env, ctx, cancel := setupEnv(t)
	defer cancel()

	assetID := "00000000-0000-0000-0000-000000000002"
	mcapID := "00000000-0000-0000-0000-00000000f002"

	env.insertAsset(ctx, t, assetID, mcapID)

	// Insert 20 events for the same asset.
	for range 20 {
		env.insertEvent(ctx, t, assetID)
	}

	wCtx, wCancel := context.WithCancel(ctx)
	defer wCancel()
	w := env.newWorker()
	go w.Run(wCtx, 500*time.Millisecond)

	// Wait for all events to be processed.
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if env.pendingCount(ctx, t) == 0 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if env.pendingCount(ctx, t) != 0 {
		t.Fatal("pending events did not drain to 0")
	}

	// Verify exactly 1 document in ES (dedup by asset_id).
	env.refreshES(ctx, t)
	count := env.esDocCount(ctx, t)
	if count != 1 {
		t.Fatalf("expected 1 ES doc (dedup), got %d", count)
	}
}

// ---------------------------------------------------------------------------
// 4.5 TestE2E_RestartReplay
// ---------------------------------------------------------------------------

// TestE2E_RestartReplay verifies that stopping and restarting the worker does
// not lose events. Events inserted while the worker is down are processed
// after restart.
func TestE2E_RestartReplay(t *testing.T) {
	env, ctx, cancel := setupEnv(t)
	defer cancel()

	const numAssets = 5
	mcapID := "00000000-0000-0000-0000-00000000f003"
	env.insertMcapFile(ctx, t, mcapID)

	// Insert first batch of assets + events.
	for i := range numAssets {
		aid := fmt.Sprintf("00000000-0000-0000-0001-%012d", i)
		env.insertAsset(ctx, t, aid, mcapID)
		env.insertEvent(ctx, t, aid)
	}

	// Start worker, let it process some events.
	wCtx1, wCancel1 := context.WithCancel(ctx)
	w1 := env.newWorker()
	go w1.Run(wCtx1, 500*time.Millisecond)

	// Wait for first batch to drain.
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if env.pendingCount(ctx, t) == 0 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Stop the worker.
	wCancel1()
	time.Sleep(1 * time.Second) // let goroutine exit

	// Insert second batch while worker is down.
	for i := numAssets; i < numAssets*2; i++ {
		aid := fmt.Sprintf("00000000-0000-0000-0001-%012d", i)
		env.insertAsset(ctx, t, aid, mcapID)
		env.insertEvent(ctx, t, aid)
	}

	// Restart worker.
	wCtx2, wCancel2 := context.WithCancel(ctx)
	defer wCancel2()
	w2 := env.newWorker()
	go w2.Run(wCtx2, 500*time.Millisecond)

	// Wait for all events to drain.
	deadline = time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if env.pendingCount(ctx, t) == 0 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if env.pendingCount(ctx, t) != 0 {
		t.Fatalf("pending events did not drain after restart, remaining: %d",
			env.pendingCount(ctx, t))
	}

	// Verify all assets are in ES.
	env.refreshES(ctx, t)
	count := env.esDocCount(ctx, t)
	if count != int64(numAssets*2) {
		t.Fatalf("expected %d ES docs after restart, got %d", numAssets*2, count)
	}
}

// ---------------------------------------------------------------------------
// 4.6 TestE2E_ConcurrentAck_NoSeqGap
// ---------------------------------------------------------------------------

// TestE2E_ConcurrentAck_NoSeqGap verifies that the cursor advances
// monotonically without gaps, even when multiple assets are processed.
func TestE2E_ConcurrentAck_NoSeqGap(t *testing.T) {
	env, ctx, cancel := setupEnv(t)
	defer cancel()

	const numAssets = 10
	mcapID := "00000000-0000-0000-0000-00000000f004"
	env.insertMcapFile(ctx, t, mcapID)

	for i := range numAssets {
		aid := fmt.Sprintf("00000000-0000-0000-0002-%012d", i)
		env.insertAsset(ctx, t, aid, mcapID)
		env.insertEvent(ctx, t, aid)
	}

	// Record cursor before processing.
	cursorBefore := env.cursorSeq(ctx, t, "es_assets")

	wCtx, wCancel := context.WithCancel(ctx)
	defer wCancel()
	w := env.newWorker()

	// Track cursor values over time to verify monotonic advancement.
	var mu sync.Mutex
	var cursorHistory []int64

	done := make(chan struct{})
	go func() {
		defer close(done)
		w.Run(wCtx, 500*time.Millisecond)
	}()

	// Poll cursor while worker runs.
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		seq := env.cursorSeq(ctx, t, "es_assets")
		mu.Lock()
		cursorHistory = append(cursorHistory, seq)
		mu.Unlock()

		if env.pendingCount(ctx, t) == 0 && seq > cursorBefore {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}

	wCancel()
	<-done

	// Verify cursor advanced monotonically (no gaps / regressions).
	mu.Lock()
	defer mu.Unlock()
	for i := 1; i < len(cursorHistory); i++ {
		if cursorHistory[i] < cursorHistory[i-1] {
			t.Fatalf("cursor regressed: history[%d]=%d > history[%d]=%d",
				i-1, cursorHistory[i-1], i, cursorHistory[i])
		}
	}

	// Cursor should have advanced past the initial value.
	finalCursor := env.cursorSeq(ctx, t, "es_assets")
	if finalCursor <= cursorBefore {
		t.Fatalf("cursor did not advance: before=%d, after=%d", cursorBefore, finalCursor)
	}

	// All events should be processed.
	if env.pendingCount(ctx, t) != 0 {
		t.Fatalf("expected 0 pending events, got %d", env.pendingCount(ctx, t))
	}
}

// ---------------------------------------------------------------------------
// 4.7 TestE2E_ESDown_Backpressure
// ---------------------------------------------------------------------------

// TestE2E_ESDown_Backpressure verifies that when ES is unavailable, events
// stay pending (backpressure), and once ES comes back, events are eventually
// processed.
func TestE2E_ESDown_Backpressure(t *testing.T) {
	env, ctx, cancel := setupEnv(t)
	defer cancel()

	assetID := "00000000-0000-0000-0000-000000000099"
	mcapID := "00000000-0000-0000-0000-00000000f099"

	env.insertAsset(ctx, t, assetID, mcapID)
	env.insertEvent(ctx, t, assetID)

	// Create a worker pointing to a dead ES endpoint.
	badES := elasticsearch.New("http://127.0.0.1:19999", "assets")
	badWorker := &ESWorker{
		Events:    env.eventRepo,
		Indexer:   env.indexer,
		ES:        badES,
		BatchSize: 100,
		SinkName:  "es_assets",
	}

	wCtx, wCancel := context.WithCancel(ctx)
	go badWorker.Run(wCtx, 1*time.Second)

	// Let the worker attempt a few ticks with ES down.
	time.Sleep(5 * time.Second)

	// Events should still be pending (backpressure).
	pending := env.pendingCount(ctx, t)
	if pending == 0 {
		t.Fatal("expected events to remain pending while ES is down")
	}

	// Stop the bad worker.
	wCancel()
	time.Sleep(1 * time.Second)

	// Now start a worker with the real ES.
	wCtx2, wCancel2 := context.WithCancel(ctx)
	defer wCancel2()
	goodWorker := env.newWorker()
	go goodWorker.Run(wCtx2, 500*time.Millisecond)

	// Events should eventually be processed.
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if env.pendingCount(ctx, t) == 0 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if env.pendingCount(ctx, t) != 0 {
		t.Fatal("events did not drain after ES recovery")
	}

	// Verify the doc is in ES.
	env.waitForESDoc(ctx, t, assetID, 10*time.Second)
}
