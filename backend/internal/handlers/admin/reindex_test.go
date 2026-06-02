package admin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	espkg "github.com/CyberOrigin2077/cyber-databrew/internal/elasticsearch"
	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

type reindexAssetRepo struct {
	list []*models.Asset
	get  map[string]*models.Asset
}

func (r *reindexAssetRepo) InsertNew(context.Context, *models.Asset) error { return nil }
func (r *reindexAssetRepo) Get(ctx context.Context, assetID string) (*models.Asset, error) {
	return r.GetAll(ctx, assetID)
}
func (r *reindexAssetRepo) GetAll(_ context.Context, assetID string) (*models.Asset, error) {
	return r.get[assetID], nil
}
func (r *reindexAssetRepo) FindExistingIDs(_ context.Context, assetIDs []string) (map[string]struct{}, error) {
	out := make(map[string]struct{})
	for _, assetID := range assetIDs {
		if r.get[assetID] != nil {
			out[assetID] = struct{}{}
		}
	}
	return out, nil
}
func (r *reindexAssetRepo) Set(context.Context, *models.Asset) error { return nil }
func (r *reindexAssetRepo) SoftDelete(context.Context, string) error { return nil }
func (r *reindexAssetRepo) ListByMcapFile(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}
func (r *reindexAssetRepo) ListByLogicalAssetID(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}
func (r *reindexAssetRepo) WriteSegmentIndex(context.Context, *models.Asset) error { return nil }
func (r *reindexAssetRepo) ListDescendants(_ context.Context, _ string) ([]*models.Asset, error) {
	return nil, nil
}
func (r *reindexAssetRepo) ListWithFilters(_ context.Context, _ string, _ []interface{}, page, pageSize int, _ filter.OrderByClause) ([]*models.Asset, int64, error) {
	start := (page - 1) * pageSize
	if start >= len(r.list) {
		return []*models.Asset{}, int64(len(r.list)), nil
	}
	end := start + pageSize
	if end > len(r.list) {
		end = len(r.list)
	}
	return r.list[start:end], int64(len(r.list)), nil
}

type reindexTagRepo struct{}

func (r *reindexTagRepo) Upsert(context.Context, repository.AssetTagUpsertInput) error {
	return nil
}
func (r *reindexTagRepo) ListByAsset(context.Context, string) ([]*models.AssetTag, error) {
	return []*models.AssetTag{}, nil
}
func (r *reindexTagRepo) Delete(context.Context, string, string, string) error { return nil }

type reindexAlgoRepo struct{}

func (r *reindexAlgoRepo) Upsert(context.Context, *models.AssetAlgoLatest) error { return nil }
func (r *reindexAlgoRepo) GetByAlgo(context.Context, string, string) (*models.AssetAlgoLatest, error) {
	return nil, nil
}
func (r *reindexAlgoRepo) ListByAsset(context.Context, string) ([]*models.AssetAlgoLatest, error) {
	return []*models.AssetAlgoLatest{}, nil
}

type reindexMcapRepo struct{}

func (r *reindexMcapRepo) Get(context.Context, string) (*models.McapFile, error) { return nil, nil }
func (r *reindexMcapRepo) Set(context.Context, *models.McapFile) error           { return nil }
func (r *reindexMcapRepo) UpdateIngestState(context.Context, string, models.IngestState) error {
	return nil
}
func (r *reindexMcapRepo) List(context.Context, int, int, string, string) ([]*models.McapFile, int64, error) {
	return nil, 0, nil
}

type esServerStats struct {
	bulkCalls        int
	deleteCalls      int
	deleteAllCalls   int
	deleteAllDeleted int64
}

func newReindexESTestServer(t *testing.T, failedDocIDs map[string]string, count int64, deleteAllDeleted int64) (*httptest.Server, *esServerStats) {
	t.Helper()

	stats := &esServerStats{deleteAllDeleted: deleteAllDeleted}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/assets/_delete_by_query"):
			stats.deleteAllCalls++
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"deleted": stats.deleteAllDeleted})
		case r.Method == http.MethodPost && r.URL.Path == "/_bulk":
			stats.bulkCalls++
			body, _ := io.ReadAll(r.Body)
			lines := strings.Split(strings.TrimSpace(string(body)), "\n")

			type bulkItem struct {
				Index struct {
					ID     string      `json:"_id"`
					Status int         `json:"status"`
					Error  interface{} `json:"error,omitempty"`
				} `json:"index"`
			}
			items := make([]bulkItem, 0, len(lines)/2)
			hasErrors := false
			for i := 0; i < len(lines)-1; i += 2 {
				var action struct {
					Index struct {
						ID string `json:"_id"`
					} `json:"index"`
				}
				if err := json.Unmarshal([]byte(lines[i]), &action); err != nil {
					t.Fatalf("unmarshal bulk action: %v", err)
				}
				item := bulkItem{}
				item.Index.ID = action.Index.ID
				if reason, fail := failedDocIDs[action.Index.ID]; fail {
					item.Index.Status = http.StatusBadRequest
					item.Index.Error = map[string]string{
						"type":   "mapper_parsing_exception",
						"reason": reason,
					}
					hasErrors = true
				} else {
					item.Index.Status = http.StatusOK
				}
				items = append(items, item)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"errors": hasErrors,
				"items":  items,
			})
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/assets/_doc/"):
			stats.deleteCalls++
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && r.URL.Path == "/assets/_count":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"count": count})
		default:
			t.Fatalf("unexpected ES request: %s %s", r.Method, r.URL.Path)
		}
	}))
	return server, stats
}

func newReindexHandlerForTest(t *testing.T, repo repository.AssetRepository, failedDocIDs map[string]string, count int64, deleteAllDeleted int64) (*Handler, *esServerStats) {
	t.Helper()
	esServer, stats := newReindexESTestServer(t, failedDocIDs, count, deleteAllDeleted)
	t.Cleanup(esServer.Close)
	return New(
		repo,
		&reindexTagRepo{},
		&reindexAlgoRepo{},
		&reindexMcapRepo{},
		nil,
		espkg.New(esServer.URL, "assets", "", ""),
		nil,
		nil,
		nil,
	), stats
}

func makeAsset(id string) *models.Asset {
	now := time.Date(2026, 4, 30, 10, 0, 0, 0, time.UTC)
	return &models.Asset{
		AssetID:          id,
		McapFileID:       "m1",
		LifecycleState:   "ready",
		Owner:            "owner",
		Reviewer:         "reviewer",
		StartTimestampNs: 1,
		EndTimestampNs:   2,
		DurationMs:       1,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func TestSearchReindex_DryRunDoesNotWriteToES(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &reindexAssetRepo{
		list: []*models.Asset{makeAsset("a1"), makeAsset("a2")},
		get: map[string]*models.Asset{
			"a1": makeAsset("a1"),
			"a2": makeAsset("a2"),
		},
	}
	handler, stats := newReindexHandlerForTest(t, repo, nil, 0, 0)

	r := gin.New()
	r.POST("/search/reindex", handler.SearchReindex)

	req := httptest.NewRequest(http.MethodPost, "/search/reindex", strings.NewReader(`{"dry_run":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if stats.bulkCalls != 0 || stats.deleteCalls != 0 {
		t.Fatalf("dry run should not hit ES writes, got bulk=%d delete=%d", stats.bulkCalls, stats.deleteCalls)
	}
	if stats.deleteAllCalls != 0 {
		t.Fatalf("dry run should not clear the index, got delete_all=%d", stats.deleteAllCalls)
	}

	var resp SearchReindexResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !resp.DryRun {
		t.Fatal("expected dry_run=true")
	}
	if resp.TotalAssets != 2 || resp.Indexed != 2 || resp.Deleted != 0 || resp.Failed != 0 {
		t.Fatalf("unexpected summary: %+v", resp)
	}
}

func TestSearchReindex_PartialBulkFailureIsReported(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &reindexAssetRepo{
		list: []*models.Asset{makeAsset("a1"), makeAsset("a2")},
		get: map[string]*models.Asset{
			"a1": makeAsset("a1"),
			"a2": makeAsset("a2"),
		},
	}
	handler, stats := newReindexHandlerForTest(t, repo, map[string]string{"a2": "failed to parse field"}, 1, 3)

	r := gin.New()
	r.POST("/search/reindex", handler.SearchReindex)

	req := httptest.NewRequest(http.MethodPost, "/search/reindex", strings.NewReader(`{"dry_run":false}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if stats.bulkCalls != 1 {
		t.Fatalf("expected 1 bulk call, got %d", stats.bulkCalls)
	}
	if stats.deleteAllCalls != 1 {
		t.Fatalf("expected 1 delete_by_query call, got %d", stats.deleteAllCalls)
	}

	var resp SearchReindexResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Indexed != 1 || resp.Failed != 1 {
		t.Fatalf("unexpected partial failure summary: %+v", resp)
	}
	if resp.Deleted != 3 {
		t.Fatalf("expected deleted count to include cleared docs, got %+v", resp)
	}
	if len(resp.Errors) != 1 || !strings.Contains(resp.Errors[0], "a2") {
		t.Fatalf("expected failed doc in errors, got %+v", resp.Errors)
	}
}

func TestSearchReindex_InvalidJSONReturns400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &reindexAssetRepo{}
	handler, _ := newReindexHandlerForTest(t, repo, nil, 0, 0)

	r := gin.New()
	r.POST("/search/reindex", handler.SearchReindex)

	req := httptest.NewRequest(http.MethodPost, "/search/reindex", strings.NewReader(`{"dry_run":`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}
