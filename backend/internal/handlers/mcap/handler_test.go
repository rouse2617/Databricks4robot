package mcap

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

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

type mockMcapRepo struct {
	getFn               func(ctx context.Context, mcapFileID string) (*models.McapFile, error)
	setFn               func(ctx context.Context, f *models.McapFile) error
	updateIngestStateFn func(ctx context.Context, mcapFileID string, state models.IngestState) error
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
func (m *mockMcapRepo) List(ctx context.Context, page, pageSize int, ingestState, owner string) ([]*models.McapFile, int64, error) {
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
