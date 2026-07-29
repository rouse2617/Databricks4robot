package mcap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

type mockMcapRepo struct {
	getFn               func(ctx context.Context, mcapFileID string) (*models.McapFile, error)
	setFn               func(ctx context.Context, f *models.McapFile) error
	updateIngestStateFn func(ctx context.Context, mcapFileID string, state models.IngestState) error
	listFn              func(ctx context.Context, page, pageSize int, ingestState, owner, mcapFileID string) ([]*models.McapFile, int64, error)
}

func (m *mockMcapRepo) Get(ctx context.Context, mcapFileID string) (*models.McapFile, error) {
	if m.getFn != nil {
		return m.getFn(ctx, mcapFileID)
	}
	return nil, nil
}
func (m *mockMcapRepo) Set(ctx context.Context, f *models.McapFile) error {
	if m.setFn != nil {
		return m.setFn(ctx, f)
	}
	return nil
}
func (m *mockMcapRepo) UpdateIngestState(ctx context.Context, mcapFileID string, state models.IngestState) error {
	if m.updateIngestStateFn != nil {
		return m.updateIngestStateFn(ctx, mcapFileID, state)
	}
	return nil
}
func (m *mockMcapRepo) List(ctx context.Context, page, pageSize int, ingestState, owner, mcapFileID string) ([]*models.McapFile, int64, error) {
	if m.listFn != nil {
		return m.listFn(ctx, page, pageSize, ingestState, owner, mcapFileID)
	}
	return []*models.McapFile{}, 0, nil
}

type mcapEventRepo struct {
	appendFn func(ctx context.Context, in repository.AssetEventAppendInput) error
}

func (m *mcapEventRepo) Append(ctx context.Context, in repository.AssetEventAppendInput) error {
	if m.appendFn != nil {
		return m.appendFn(ctx, in)
	}
	return nil
}
func (m *mcapEventRepo) ListPending(context.Context, int) ([]*models.AssetEvent, error) {
	return nil, nil
}
func (m *mcapEventRepo) ListPendingSafe(context.Context, time.Duration, int) ([]*models.AssetEvent, error) {
	return nil, nil
}
func (m *mcapEventRepo) ListByAsset(context.Context, string, repository.AssetEventListOptions) ([]*models.AssetEvent, error) {
	return nil, nil
}
func (m *mcapEventRepo) ListVersionPromotedByLogical(context.Context, string) ([]*models.AssetEvent, error) {
	return nil, nil
}

func (m *mcapEventRepo) ListGlobal(_ context.Context, opts repository.AssetEventListOptions) ([]*models.AssetEvent, error) {
	return nil, nil
}

func (m *mcapEventRepo) MarkPublished(context.Context, []int64) error { return nil }
func (m *mcapEventRepo) MarkFailed(context.Context, int64, string) error {
	return nil
}
func (m *mcapEventRepo) CountPending(context.Context) (int64, error) { return 0, nil }
func (m *mcapEventRepo) CountPendingClaimable(context.Context, time.Duration) (int64, error) {
	return 0, nil
}
func (m *mcapEventRepo) CountProcessing(context.Context) (int64, error) { return 0, nil }
func (m *mcapEventRepo) OldestPendingAge(context.Context) (float64, error) {
	return 0, nil
}
func (m *mcapEventRepo) ListBetweenSeq(context.Context, int64, int64, int) ([]*models.AssetEvent, error) {
	return nil, nil
}

func (m *mcapEventRepo) PublishStateCounts(context.Context) (map[string]int64, error) {
	return map[string]int64{}, nil
}

// stubAssetRepo satisfies repository.AssetRepository just enough for
// mcap CreateFile tests: only InsertNew is exercised by the code under
// test, so the other methods embed a nil interface and will panic if
// invoked — making accidental couplings noisy in a test rather than
// silently green.
type stubAssetRepo struct {
	repository.AssetRepository
	insertNewFn func(context.Context, *models.Asset) error
	getFn       func(context.Context, string) (*models.Asset, error)
	setFn       func(context.Context, *models.Asset) error
}

func (s *stubAssetRepo) InsertNew(ctx context.Context, a *models.Asset) error {
	if s.insertNewFn != nil {
		return s.insertNewFn(ctx, a)
	}
	return nil
}

// getFn/setFn added for CYB-4011 BackfillGraceVideoID (mirror-column update).
func (s *stubAssetRepo) Get(ctx context.Context, assetID string) (*models.Asset, error) {
	if s.getFn != nil {
		return s.getFn(ctx, assetID)
	}
	return nil, nil
}
func (s *stubAssetRepo) Set(ctx context.Context, a *models.Asset) error {
	if s.setFn != nil {
		return s.setFn(ctx, a)
	}
	return nil
}

func setupMcapRouter(route, path string, fn gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Handle(route, path, fn)
	return r
}

func doMcapReq(t *testing.T, r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
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

func TestFinalizeUpload(t *testing.T) {
	repo := &mockMcapRepo{}
	h := New(repo)
	r := setupMcapRouter(http.MethodPost, "/mcap/upload/finalize", h.FinalizeUpload)

	w := doMcapReq(t, r, http.MethodPost, "/mcap/upload/finalize", map[string]any{"x": 1})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	repo.updateIngestStateFn = func(context.Context, string, models.IngestState) error { return errors.New("boom") }
	w = doMcapReq(t, r, http.MethodPost, "/mcap/upload/finalize", map[string]any{"mcap_file_id": "abcd1234"})
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	repo.updateIngestStateFn = func(context.Context, string, models.IngestState) error { return nil }
	w = doMcapReq(t, r, http.MethodPost, "/mcap/upload/finalize", map[string]any{"mcap_file_id": "abcd1234"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCreateFile(t *testing.T) {
	repo := &mockMcapRepo{}
	h := New(repo)
	r := setupMcapRouter(http.MethodPost, "/mcap-files", h.CreateFile)

	w := doMcapReq(t, r, http.MethodPost, "/mcap-files", map[string]any{"mcap_file_id": "not8chars"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	repo.setFn = func(context.Context, *models.McapFile) error { return errors.New("boom") }
	w = doMcapReq(t, r, http.MethodPost, "/mcap-files", map[string]any{
		"mcap_file_id": "abcd1234",
		"gcs_path":     "gs://bucket/a.mcap",
	})
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	var got *models.McapFile
	repo.setFn = func(_ context.Context, f *models.McapFile) error {
		got = f
		return nil
	}
	w = doMcapReq(t, r, http.MethodPost, "/mcap-files", map[string]any{
		"mcap_file_id":      "abcd1234",
		"gcs_path":          "gs://bucket/a.mcap",
		"raw_hash_md5":      "md5-1",
		"ingest_state":      "summarized",
		"size_bytes":        123,
		"channel_count":     4,
		"chunk_count":       8,
		"owner":             "team-a",
		"vendor_id":         "vendor-1",
		"scene_id":          "kitchen",
		"metadata":          map[string]any{"source": "test"},
		"process_state":     map[string]string{"hand_tracking": "completed"},
		"retention_tier":    "standard",
		"storage_uri":       "ignored-by-handler",
		"collection_method": "manual_import",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
	}
	if got == nil || got.GCSPath != "gs://bucket/a.mcap" || got.Owner != "team-a" || got.IngestState != models.IngestStateSummarized {
		t.Fatalf("unexpected mcap file passed to repo: %+v", got)
	}
}

func TestCreateFile_UniqueViolations(t *testing.T) {
	// Regression: auto-gen mcap_file_id must NOT retry on raw_hash_md5
	// UNIQUE violations (uq_mcap_files_hash_md5) — those can never be
	// resolved by picking a new ID. Pre-fix behavior was to burn all 16
	// retries and return 500.
	pgHash := &pgconn.PgError{Code: "23505", ConstraintName: "uq_mcap_files_hash_md5"}
	pgID := &pgconn.PgError{Code: "23505", ConstraintName: "mcap_files_pkey"}

	decode := func(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
		t.Helper()
		var body map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		return body
	}

	t.Run("auto-gen + hash conflict returns 409 DUPLICATE_HASH", func(t *testing.T) {
		repo := &mockMcapRepo{}
		calls := 0
		repo.setFn = func(context.Context, *models.McapFile) error {
			calls++
			return fmt.Errorf("mcap repo set: %w", pgHash)
		}
		h := New(repo)
		r := setupMcapRouter(http.MethodPost, "/mcap-files", h.CreateFile)

		w := doMcapReq(t, r, http.MethodPost, "/mcap-files", map[string]any{
			"raw_hash_md5": "deadbeefdeadbeefdeadbeefdeadbeef",
			"gcs_path":     "gs://bucket/a.mcap",
		})
		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d body=%s", w.Code, w.Body.String())
		}
		body := decode(t, w)
		if got, _ := body["code"].(string); got != "DUPLICATE_HASH" {
			t.Fatalf("expected code=DUPLICATE_HASH, got %v", body["code"])
		}
		if calls != 1 {
			t.Fatalf("expected 1 attempt (no retry on hash conflict), got %d", calls)
		}
	})

	t.Run("auto-gen + id collision retries then succeeds", func(t *testing.T) {
		repo := &mockMcapRepo{}
		calls := 0
		repo.setFn = func(context.Context, *models.McapFile) error {
			calls++
			if calls < 3 {
				return fmt.Errorf("mcap repo set: %w", pgID)
			}
			return nil
		}
		h := New(repo)
		r := setupMcapRouter(http.MethodPost, "/mcap-files", h.CreateFile)

		w := doMcapReq(t, r, http.MethodPost, "/mcap-files", map[string]any{
			"raw_hash_md5": "cafef00d00000000000000000000ffff",
			"gcs_path":     "gs://bucket/b.mcap",
		})
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
		}
		if calls != 3 {
			t.Fatalf("expected 3 attempts (2 retries), got %d", calls)
		}
	})

	t.Run("explicit id + hash conflict returns 409 DUPLICATE_HASH", func(t *testing.T) {
		repo := &mockMcapRepo{}
		repo.setFn = func(context.Context, *models.McapFile) error {
			return fmt.Errorf("mcap repo set: %w", pgHash)
		}
		h := New(repo)
		r := setupMcapRouter(http.MethodPost, "/mcap-files", h.CreateFile)

		w := doMcapReq(t, r, http.MethodPost, "/mcap-files", map[string]any{
			"mcap_file_id": "abcd1234",
			"raw_hash_md5": "deadbeefdeadbeefdeadbeefdeadbeef",
			"gcs_path":     "gs://bucket/c.mcap",
		})
		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d body=%s", w.Code, w.Body.String())
		}
		body := decode(t, w)
		if got, _ := body["code"].(string); got != "DUPLICATE_HASH" {
			t.Fatalf("expected code=DUPLICATE_HASH, got %v", body["code"])
		}
	})

	t.Run("explicit id + pkey collision returns 409 DUPLICATE_MCAP_FILE_ID", func(t *testing.T) {
		repo := &mockMcapRepo{}
		repo.setFn = func(context.Context, *models.McapFile) error {
			return fmt.Errorf("mcap repo set: %w", pgID)
		}
		h := New(repo)
		r := setupMcapRouter(http.MethodPost, "/mcap-files", h.CreateFile)

		w := doMcapReq(t, r, http.MethodPost, "/mcap-files", map[string]any{
			"mcap_file_id": "abcd1234",
			"gcs_path":     "gs://bucket/d.mcap",
		})
		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d body=%s", w.Code, w.Body.String())
		}
		body := decode(t, w)
		if got, _ := body["code"].(string); got != "DUPLICATE_MCAP_FILE_ID" {
			t.Fatalf("expected code=DUPLICATE_MCAP_FILE_ID, got %v", body["code"])
		}
	})

	// AssetRepo.InsertNew wraps assets_pkey unique_violation into the
	// ErrDuplicateAssetID sentinel, so uniqueViolationKind's assets_pkey case
	// is unreachable via the auto-derived raw_mcap asset path. The handler
	// must catch the sentinel explicitly; previously this leaked as HTTP 500
	// "duplicate asset id".
	t.Run("explicit id + asset sentinel returns 409 DUPLICATE_MCAP_FILE_ID", func(t *testing.T) {
		repo := &mockMcapRepo{}
		asset := &stubAssetRepo{
			insertNewFn: func(context.Context, *models.Asset) error {
				return repository.ErrDuplicateAssetID
			},
		}
		h := New(repo)
		h.SetAssetRepo(asset)
		r := setupMcapRouter(http.MethodPost, "/mcap-files", h.CreateFile)

		w := doMcapReq(t, r, http.MethodPost, "/mcap-files", map[string]any{
			"mcap_file_id": "abcd1234",
			"raw_hash_md5": "deadbeefcafebabe1234deadbeefcafe", // pragma: allowlist secret
			"gcs_path":     "gs://bucket/e.mcap",
		})
		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d body=%s", w.Code, w.Body.String())
		}
		body := decode(t, w)
		if got, _ := body["code"].(string); got != "DUPLICATE_MCAP_FILE_ID" {
			t.Fatalf("expected code=DUPLICATE_MCAP_FILE_ID, got %v", body["code"])
		}
	})

	t.Run("auto-gen + asset sentinel retries then succeeds", func(t *testing.T) {
		repo := &mockMcapRepo{}
		calls := 0
		asset := &stubAssetRepo{
			insertNewFn: func(context.Context, *models.Asset) error {
				calls++
				if calls < 3 {
					return repository.ErrDuplicateAssetID
				}
				return nil
			},
		}
		h := New(repo)
		h.SetAssetRepo(asset)
		r := setupMcapRouter(http.MethodPost, "/mcap-files", h.CreateFile)

		w := doMcapReq(t, r, http.MethodPost, "/mcap-files", map[string]any{
			"raw_hash_md5": "aa11bb22cc33dd44ee55ff66aa11bb22", // pragma: allowlist secret
			"gcs_path":     "gs://bucket/f.mcap",
		})
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
		}
		if calls != 3 {
			t.Fatalf("expected 3 InsertNew attempts (2 retries), got %d", calls)
		}
	})
}

func TestGetFileAndStaticEndpoints(t *testing.T) {
	repo := &mockMcapRepo{}
	h := New(repo)

	r := setupMcapRouter(http.MethodGet, "/mcap-files/:id", h.GetFile)
	repo.getFn = func(context.Context, string) (*models.McapFile, error) { return nil, nil }
	w := doMcapReq(t, r, http.MethodGet, "/mcap-files/abcd1234", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	repo.getFn = func(context.Context, string) (*models.McapFile, error) { return nil, errors.New("boom") }
	w = doMcapReq(t, r, http.MethodGet, "/mcap-files/abcd1234", nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	repo.getFn = func(context.Context, string) (*models.McapFile, error) {
		return &models.McapFile{McapFileID: "abcd1234"}, nil
	}
	w = doMcapReq(t, r, http.MethodGet, "/mcap-files/abcd1234", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	r = setupMcapRouter(http.MethodGet, "/mcap/:id/messages", h.IterMessages)
	w = doMcapReq(t, r, http.MethodGet, "/mcap/abcd1234/messages", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	r = setupMcapRouter(http.MethodGet, "/mcap-files", h.ListFiles)
	w = doMcapReq(t, r, http.MethodGet, "/mcap-files", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// CYB-4445: mcap_file_id query param is rejected when not 8 alphanumeric and
// forwarded to repo.List when valid. Combined with owner filter via AND.
func TestListFiles_McapFileIDFilter(t *testing.T) {
	repo := &mockMcapRepo{}
	var (
		gotID      string
		gotOwner   string
		gotResults []*models.McapFile
	)
	repo.listFn = func(_ context.Context, _, _ int, _, owner, mcapFileID string) ([]*models.McapFile, int64, error) {
		gotID = mcapFileID
		gotOwner = owner
		gotResults = []*models.McapFile{{McapFileID: mcapFileID}}
		return gotResults, int64(len(gotResults)), nil
	}
	h := New(repo)
	r := setupMcapRouter(http.MethodGet, "/mcap-files", h.ListFiles)

	// happy exact match
	w := doMcapReq(t, r, http.MethodGet, "/mcap-files?mcap_file_id=LEMpjOmB", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("happy: expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if gotID != "LEMpjOmB" {
		t.Fatalf("happy: expected mcapFileID forwarded LEMpjOmB, got %q", gotID)
	}
	if gotOwner != "" {
		t.Fatalf("happy: expected owner empty, got %q", gotOwner)
	}

	// combined: id + owner filter passes both through unchanged
	gotID, gotOwner = "", ""
	w = doMcapReq(t, r, http.MethodGet, "/mcap-files?mcap_file_id=abcd1234&owner=grace-pu", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("combined: expected 200, got %d", w.Code)
	}
	if gotID != "abcd1234" || gotOwner != "grace-pu" {
		t.Fatalf("combined: expected id=abcd1234 owner=grace-pu, got id=%q owner=%q", gotID, gotOwner)
	}

	// omit → empty string, no filter applied
	gotID = ""
	w = doMcapReq(t, r, http.MethodGet, "/mcap-files", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("omit: expected 200, got %d", w.Code)
	}
	if gotID != "" {
		t.Fatalf("omit: expected empty forwarded id, got %q", gotID)
	}

	// bad length → 400, repo never called
	repo.listFn = func(context.Context, int, int, string, string, string) ([]*models.McapFile, int64, error) {
		t.Fatalf("repo.List should not be called when validation fails")
		return nil, 0, nil
	}
	w = doMcapReq(t, r, http.MethodGet, "/mcap-files?mcap_file_id=SHORT", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bad length: expected 400, got %d body=%s", w.Code, w.Body.String())
	}

	// non-alphanumeric 8-char value → 400
	w = doMcapReq(t, r, http.MethodGet, "/mcap-files?mcap_file_id=ABCD_EFG", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("non-alphanumeric: expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestMcapHandlers_AppendOutboxEvents(t *testing.T) {
	repo := &mockMcapRepo{}
	var eventTypes []string
	eventRepo := &mcapEventRepo{
		appendFn: func(_ context.Context, in repository.AssetEventAppendInput) error {
			eventTypes = append(eventTypes, in.EventType)
			return nil
		},
	}
	h := New(repo)
	h.SetEventRepo(eventRepo)

	rCreate := setupMcapRouter(http.MethodPost, "/mcap-files", h.CreateFile)
	w := doMcapReq(t, rCreate, http.MethodPost, "/mcap-files", map[string]any{
		"mcap_file_id": "abcd1234",
		"gcs_path":     "gs://bucket/a.mcap",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create expected 201, got %d", w.Code)
	}

	repo.updateIngestStateFn = func(context.Context, string, models.IngestState) error { return nil }
	rFinalize := setupMcapRouter(http.MethodPost, "/mcap/upload/finalize", h.FinalizeUpload)
	w = doMcapReq(t, rFinalize, http.MethodPost, "/mcap/upload/finalize", map[string]any{"mcap_file_id": "abcd1234"})
	if w.Code != http.StatusOK {
		t.Fatalf("finalize expected 200, got %d", w.Code)
	}
	if len(eventTypes) != 2 || eventTypes[0] != "mcap_file_created" || eventTypes[1] != "mcap_upload_finalized" {
		t.Fatalf("unexpected event types: %#v", eventTypes)
	}
}

// CYB-4011: PATCH /internal/mcap-files/:id/grace-video-id backfill endpoint.
func TestBackfillGraceVideoID(t *testing.T) {
	const gvid = "019f9893-3456-7376-ae68-30a89227eb46"

	build := func() (*Handler, *mockMcapRepo, *stubAssetRepo, *mcapEventRepo) {
		repo := &mockMcapRepo{
			getFn: func(_ context.Context, id string) (*models.McapFile, error) {
				return &models.McapFile{McapFileID: id, RawHashMD5: "md5-1"}, nil
			},
		}
		asset := &stubAssetRepo{
			getFn: func(_ context.Context, id string) (*models.Asset, error) {
				return &models.Asset{AssetID: id, McapFileID: id, AssetType: "raw_mcap", Version: 1}, nil
			},
		}
		ev := &mcapEventRepo{}
		h := New(repo)
		h.SetAssetRepo(asset)
		h.SetEventRepo(ev)
		return h, repo, asset, ev
	}

	route := func(h *Handler) *gin.Engine {
		return setupMcapRouter(http.MethodPatch, "/internal/mcap-files/:id/grace-video-id", h.BackfillGraceVideoID)
	}

	// missing grace_video_id -> 400
	h, _, _, _ := build()
	w := doMcapReq(t, route(h), http.MethodPatch, "/internal/mcap-files/abcd1234/grace-video-id", map[string]any{})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("empty body: expected 400, got %d", w.Code)
	}

	// not found -> 404
	h, repo, _, _ := build()
	repo.getFn = func(context.Context, string) (*models.McapFile, error) { return nil, nil }
	w = doMcapReq(t, route(h), http.MethodPatch, "/internal/mcap-files/abcd1234/grace-video-id", map[string]any{"grace_video_id": gvid})
	if w.Code != http.StatusNotFound {
		t.Fatalf("not found: expected 404, got %d", w.Code)
	}

	// happy path: mcap Set + asset Set + asset_updated event
	h, repo, asset, ev := build()
	var mcapSet, assetSet string
	var eventTypes []string
	repo.setFn = func(_ context.Context, f *models.McapFile) error { mcapSet = f.GraceVideoID; return nil }
	asset.setFn = func(_ context.Context, a *models.Asset) error { assetSet = a.GraceVideoID; return nil }
	ev.appendFn = func(_ context.Context, in repository.AssetEventAppendInput) error {
		eventTypes = append(eventTypes, in.EventType)
		return nil
	}
	w = doMcapReq(t, route(h), http.MethodPatch, "/internal/mcap-files/abcd1234/grace-video-id", map[string]any{"grace_video_id": gvid})
	if w.Code != http.StatusOK {
		t.Fatalf("happy: expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	if mcapSet != gvid {
		t.Fatalf("mcap grace_video_id not written: %q", mcapSet)
	}
	if assetSet != gvid {
		t.Fatalf("asset mirror grace_video_id not written: %q", assetSet)
	}
	if len(eventTypes) != 1 || eventTypes[0] != "asset_updated" {
		t.Fatalf("expected one asset_updated event, got %#v", eventTypes)
	}
}
