// Package search provides the GET /api/v1/search/assets handler that
// builds an Elasticsearch bool query from query parameters.
package search

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"data-platform/internal/elasticsearch"
	"data-platform/internal/httpresp"
)

// Handler serves search endpoints backed by Elasticsearch.
type Handler struct {
	es *elasticsearch.Client
}

// New creates a search Handler. If esClient is nil the handler will
// return 503 for all requests (graceful degradation).
func New(esClient *elasticsearch.Client) *Handler {
	return &Handler{es: esClient}
}

// SearchAssets handles GET /api/v1/search/assets.
//
// Query parameters:
//
//	q          — free-text query (multi_match across notes, owner, reviewer, task)
//	filter     — repeated, format "field:op:value" where op is eq|ne|gt|gte|lt|lte
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

	// Parse filter params: "field:op:value" where op can be eq, ne, gt, gte, lt, lte
	var filters []elasticsearch.FilterOp
	validOps := map[string]bool{"eq": true, "ne": true, "gt": true, "gte": true, "lt": true, "lte": true}
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
