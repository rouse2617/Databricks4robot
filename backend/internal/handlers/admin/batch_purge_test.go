package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	adminuc "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/admin"
)

// purgeFakeOps is a tiny in-package fake for repository.BatchOpsRepository.
// Each test scripts the responses it cares about.
type purgeFakeOps struct {
	idsByBatch   map[string][]string
	mcapsByBatch map[string][]string
	dryRunAssets repository.PurgeCounts
	dryRunMcap   repository.PurgeCounts
	purgeAssets  repository.PurgeCounts
	purgeMcap    repository.PurgeCounts
}

func (f *purgeFakeOps) ListAssetIDsByImportBatch(_ context.Context, b string) ([]string, error) {
	return f.idsByBatch[b], nil
}
func (f *purgeFakeOps) ListMcapFileIDsByImportBatch(_ context.Context, b string) ([]string, error) {
	return f.mcapsByBatch[b], nil
}
func (f *purgeFakeOps) CountAssetReferencesToMcapFiles(_ context.Context, ids []string) (map[string]int64, error) {
	out := map[string]int64{}
	for _, id := range ids {
		out[id] = 0
	}
	return out, nil
}
func (f *purgeFakeOps) PurgeAssetsDryRun(_ context.Context, _ []string) (repository.PurgeCounts, error) {
	return f.dryRunAssets, nil
}
func (f *purgeFakeOps) PurgeAssets(_ context.Context, _ []string, _ int) (repository.PurgeCounts, error) {
	return f.purgeAssets, nil
}
func (f *purgeFakeOps) PurgeMcapFilesDryRun(_ context.Context, _ []string) (repository.PurgeCounts, error) {
	return f.dryRunMcap, nil
}
func (f *purgeFakeOps) PurgeMcapFiles(_ context.Context, _ []string, _ int) (repository.PurgeCounts, error) {
	return f.purgeMcap, nil
}

func newPurgeRouter(ops repository.BatchOpsRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewPurgeHandler(adminuc.New(ops))
	r.DELETE("/internal/assets/:id", h.DeleteAssetHard)
	r.POST("/internal/assets:batch_delete", h.BatchDeleteAssets)
	return r
}

func TestDeleteAssetHard_NotFound(t *testing.T) {
	r := newPurgeRouter(&purgeFakeOps{dryRunAssets: repository.PurgeCounts{Assets: 0}})

	req := httptest.NewRequest(http.MethodDelete, "/internal/assets/AbCd1234", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteAssetHard_OK(t *testing.T) {
	r := newPurgeRouter(&purgeFakeOps{
		dryRunAssets: repository.PurgeCounts{Assets: 1},
		purgeAssets:  repository.PurgeCounts{Assets: 1, AssetTags: 3, AssetEventsByAsset: 2},
	})

	req := httptest.NewRequest(http.MethodDelete, "/internal/assets/AbCd1234", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		AssetID string                 `json:"asset_id"`
		Deleted repository.PurgeCounts `json:"deleted"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.AssetID != "AbCd1234" || resp.Deleted.Assets != 1 || resp.Deleted.AssetTags != 3 {
		t.Fatalf("unexpected response body: %+v", resp)
	}
}

func TestBatchDelete_RejectsInvalidJSON(t *testing.T) {
	r := newPurgeRouter(&purgeFakeOps{})

	req := httptest.NewRequest(http.MethodPost, "/internal/assets:batch_delete",
		strings.NewReader(`{not json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed JSON, got %d", w.Code)
	}
}

func TestBatchDelete_RejectsEmptySelectors(t *testing.T) {
	r := newPurgeRouter(&purgeFakeOps{})

	req := httptest.NewRequest(http.MethodPost, "/internal/assets:batch_delete",
		strings.NewReader(`{"dry_run":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when neither asset_ids nor import_batch is supplied, got %d body=%s",
			w.Code, w.Body.String())
	}
}

func TestBatchDelete_DryRunReturnsCounts(t *testing.T) {
	r := newPurgeRouter(&purgeFakeOps{
		idsByBatch:   map[string][]string{"b1": {"a1", "a2"}},
		mcapsByBatch: map[string][]string{"b1": {"m1"}},
		dryRunAssets: repository.PurgeCounts{Assets: 2, AssetTags: 4},
		dryRunMcap:   repository.PurgeCounts{McapFiles: 1, AssetEventsByMcap: 3},
	})

	body := `{"import_batch":"b1","include_mcap_files":true,"dry_run":true}`
	req := httptest.NewRequest(http.MethodPost, "/internal/assets:batch_delete",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp adminuc.BatchOutput
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.DryRun || !resp.IncludeMcapFiles {
		t.Fatalf("expected dry_run + include_mcap_files=true, got %+v", resp)
	}
	if resp.Resolved.AssetIDs != 2 || resp.Resolved.McapFileIDs != 1 || resp.Resolved.ImportBatch != "b1" {
		t.Fatalf("unexpected resolved: %+v", resp.Resolved)
	}
	if resp.Deleted.Assets != 2 || resp.Deleted.AssetTags != 4 {
		t.Fatalf("expected dry-run asset counts to surface, got %+v", resp.Deleted)
	}
	if resp.Deleted.McapFiles != 1 || resp.Deleted.AssetEventsByMcap != 3 {
		t.Fatalf("expected dry-run mcap counts to surface, got %+v", resp.Deleted)
	}
}

func TestBatchDelete_AssetIDsLiveRun(t *testing.T) {
	r := newPurgeRouter(&purgeFakeOps{
		purgeAssets: repository.PurgeCounts{Assets: 2, AssetTags: 5},
	})

	body := `{"asset_ids":["AbCd1234","EfGh5678"]}`
	req := httptest.NewRequest(http.MethodPost, "/internal/assets:batch_delete",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp adminuc.BatchOutput
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.DryRun {
		t.Fatal("expected dry_run=false")
	}
	if resp.Resolved.AssetIDs != 2 || resp.Resolved.McapFileIDs != 0 {
		t.Fatalf("unexpected resolved: %+v", resp.Resolved)
	}
	if resp.Deleted.Assets != 2 || resp.Deleted.AssetTags != 5 {
		t.Fatalf("unexpected counts: %+v", resp.Deleted)
	}
}
