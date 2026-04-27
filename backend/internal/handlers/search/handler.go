// Package search provides the GET /api/v1/search/assets handler that
// builds an OpenSearch bool query from query parameters.
package search

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"data-platform/internal/httpresp"
	"data-platform/internal/opensearch"
)

// Handler serves search endpoints backed by OpenSearch.
type Handler struct {
	os *opensearch.Client
}

// New creates a search Handler. If osClient is nil the handler will
// return 503 for all requests (graceful degradation).
func New(osClient *opensearch.Client) *Handler {
	return &Handler{os: osClient}
}

// SearchAssets handles GET /api/v1/search/assets.
//
// Query parameters:
//
//	q          — free-text query (multi_match across notes, owner, reviewer, task)
//	filter     — repeated, format "field:eq:value" (only eq supported for now)
//	page       — 1-based page number (default 1)
//	page_size  — results per page (default 20, max 200)
func (h *Handler) SearchAssets(c *gin.Context) {
	if h.os == nil {
		httpresp.Error(c, http.StatusServiceUnavailable,
			httpresp.CodeServiceUnavailable,
			"OpenSearch is not available", nil)
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

	// Parse filter params: "field:eq:value"
	filters := make(map[string]string)
	for _, f := range c.QueryArray("filter") {
		parts := strings.SplitN(f, ":", 3)
		if len(parts) == 3 && parts[1] == "eq" {
			filters[parts[0]] = parts[2]
		}
	}

	req := opensearch.SearchRequest{
		Query:    q,
		Filters:  filters,
		Page:     page,
		PageSize: pageSize,
	}

	result, err := h.os.Search(c.Request.Context(), req)
	if err != nil {
		httpresp.Error(c, http.StatusServiceUnavailable,
			httpresp.CodeServiceUnavailable,
			"OpenSearch query failed: "+err.Error(), nil)
		return
	}

	// Map hits to items (flatten _source)
	items := make([]map[string]any, 0, len(result.Hits))
	for _, hit := range result.Hits {
		doc := hit.Source
		doc["_score"] = hit.Score
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
