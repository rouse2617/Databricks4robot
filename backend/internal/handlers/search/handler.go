// Package search provides the GET /api/v1/search/assets handler that
// builds an Elasticsearch bool query from query parameters.
package search

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/elasticsearch"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
)

// Handler serves search endpoints backed by Elasticsearch.
type Handler struct {
	es   *elasticsearch.Client
	sync func() SyncInfo
}

// New creates a search Handler. If esClient is nil the handler will
// return 503 for search requests (graceful degradation). sync may be nil;
// SyncStatus then returns a minimal snapshot based only on whether ES is wired.
func New(esClient *elasticsearch.Client, sync func() SyncInfo) *Handler {
	return &Handler{es: esClient, sync: sync}
}

// SearchAssets handles GET /api/v1/search/assets.
//
// Query parameters:
//
//	q          — free-text query (multi_match over notes/owner.text/reviewer.text/asset_id)
//	filter     — repeated, format "field:op:value" where op is one of
//	             eq | ne | gt | gte | lt | lte | between.
//	             "between" expects value="lower,upper" (e.g. "9000,11000").
//	             Field routing is handled by the ES client; nested paths
//	             tags.<key> and algos.<name>[.<attr>] are auto-detected.
//	page       — 1-based page number (default 1)
//	page_size  — results per page (default 20, max 200)
func (h *Handler) SearchAssets(c *gin.Context) {
	if h.es == nil {
		httpresp.Error(c, http.StatusServiceUnavailable,
			httpresp.CodeServiceUnavailable,
			"Elasticsearch is not available", nil)
		return
	}

	q := strings.TrimSpace(c.Query("q"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}

	// Parse filter params: "field:op:value"
	var filters []elasticsearch.FilterOp
	validOps := map[string]bool{
		"eq": true, "ne": true,
		"gt": true, "gte": true, "lt": true, "lte": true,
		"between": true,
		"ilike":   true,
	}
	for _, f := range c.QueryArray("filter") {
		parts := strings.SplitN(f, ":", 3)
		if len(parts) == 3 && validOps[parts[1]] {
			filters = append(filters, elasticsearch.FilterOp{
				Field: parts[0],
				Op:    parts[1],
				Value: parts[2],
			})
		}
	}

	req := elasticsearch.SearchRequest{
		Mode:     strings.TrimSpace(c.Query("mode")),
		Query:    q,
		Filters:  filters,
		Page:     page,
		PageSize: pageSize,
	}

	result, err := h.es.Search(c.Request.Context(), req)
	if err != nil {
		httpresp.Error(c, http.StatusServiceUnavailable,
			httpresp.CodeServiceUnavailable,
			"Elasticsearch query failed: "+err.Error(), nil)
		return
	}

	// Map hits to items (flatten _source, include highlight)
	items := make([]map[string]any, 0, len(result.Hits))
	for _, hit := range result.Hits {
		doc := hit.Source
		if doc == nil {
			doc = map[string]any{}
		}
		// Some legacy / partial index rows omit asset_id; the document _id is the
		// canonical Postgres asset UUID when indexing uses BulkIndexDoc{ID: assetID}.
		if s, ok := doc["asset_id"].(string); !ok || strings.TrimSpace(s) == "" {
			if hit.ID != "" {
				doc["asset_id"] = hit.ID
			}
		}
		doc["_score"] = hit.Score
		if len(hit.Highlight) > 0 {
			doc["_highlight"] = hit.Highlight
		}
		items = append(items, doc)
	}

	c.JSON(http.StatusOK, gin.H{
		"items":        items,
		"total":        result.Total,
		"page":         page,
		"page_size":    pageSize,
		"aggregations": result.Aggregations,
	})
}

// SyncStatus handles GET /api/v1/search/sync-status (Elasticsearch index path).
func (h *Handler) SyncStatus(c *gin.Context) {
	if h.sync != nil {
		c.JSON(http.StatusOK, h.sync())
		return
	}
	ok := h.es != nil
	mode := "unavailable"
	if ok {
		mode = "manual"
	}
	c.JSON(http.StatusOK, SyncInfo{
		ElasticsearchOK: ok,
		SearchIndexMode: mode,
	})
}
