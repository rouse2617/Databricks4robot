package search

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/elasticsearch"
	searchUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/search"
)

type fakeSearchUsecase struct {
	req searchUC.SearchAssetsRequest
}

func (f *fakeSearchUsecase) SearchAssets(_ context.Context, req searchUC.SearchAssetsRequest) (*elasticsearch.SearchResponse, error) {
	f.req = req
	return &elasticsearch.SearchResponse{
		Total: 1,
		Hits: []elasticsearch.SearchHit{{
			ID:     "asset-child",
			Source: map[string]any{"asset_id": "asset-child", "lineage_relation": map[string]any{"asset_id": req.LineageWith}},
		}},
	}, nil
}

func TestHandler_SyncStatus_WithCallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := New(nil, func() SyncInfo {
		return SyncInfo{
			ElasticsearchOK:           true,
			OutboxRelayEnabled:        true,
			OutboxESSubscriberEnabled: true,
			SearchIndexMode:           "outbox_es_subscriber",
			Env:                       "development",
			AdminSearchEnabled:        true,
		}
	}, nil)
	r := gin.New()
	r.GET("/sync-status", h.SyncStatus)

	req := httptest.NewRequest(http.MethodGet, "/sync-status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"search_index_mode":"outbox_es_subscriber"`) {
		t.Fatalf("body %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"admin_search_enabled":true`) {
		t.Fatalf("body %s", w.Body.String())
	}
}

func TestHandler_SyncStatus_NoCallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := New(nil, nil, nil)
	r := gin.New()
	r.GET("/sync-status", h.SyncStatus)

	req := httptest.NewRequest(http.MethodGet, "/sync-status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"search_index_mode":"unavailable"`) {
		t.Fatalf("body %s", w.Body.String())
	}
}

func TestHandler_SyncProgress_WithCallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := New(nil, nil, func(context.Context) (SyncProgress, error) {
		return SyncProgress{
			PostgresAssetsTotal:     100,
			ElasticsearchDocsTotal:  95,
			PGESGap:                 5,
			PGESSyncRatio:           0.95,
			OutboxPendingEvents:     12,
			OutboxPendingClaimable:  2,
			OutboxProcessingEvents:  3,
			OutboxRelaySafetyLagSec: 2,
			OldestPendingAgeSec:     3.2,
			CheckedAt:               time.Unix(0, 0).UTC(),
		}, nil
	})
	r := gin.New()
	r.GET("/sync-progress", h.SyncProgress)

	req := httptest.NewRequest(http.MethodGet, "/sync-progress", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"pg_es_gap":5`) {
		t.Fatalf("body %s", w.Body.String())
	}
}

func TestHandler_SearchAssetsParsesLineageFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fake := &fakeSearchUsecase{}
	h := &Handler{es: elasticsearch.New("http://example.invalid", "assets", "", ""), search: fake}
	r := gin.New()
	r.GET("/search/assets", h.SearchAssets)

	req := httptest.NewRequest(http.MethodGet, "/search/assets?lineage_with=asset-root&lineage_direction=downstream&lineage_depth=3&relation_types=derived_from,pipeline_output", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
	if fake.req.LineageWith != "asset-root" || fake.req.LineageDirection != "downstream" || fake.req.LineageDepth != 3 {
		t.Fatalf("lineage request = %#v", fake.req)
	}
	if len(fake.req.RelationTypes) != 2 || fake.req.RelationTypes[1] != "pipeline_output" {
		t.Fatalf("relation types = %#v", fake.req.RelationTypes)
	}
	var body struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0]["asset_id"] != "asset-child" {
		t.Fatalf("items = %#v", body.Items)
	}
}

func TestHandler_SearchAssetsRejectsInvalidLineageDirection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{es: elasticsearch.New("http://example.invalid", "assets", "", ""), search: &fakeSearchUsecase{}}
	r := gin.New()
	r.GET("/search/assets", h.SearchAssets)

	req := httptest.NewRequest(http.MethodGet, "/search/assets?lineage_with=asset-root&lineage_direction=sideways", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
}
