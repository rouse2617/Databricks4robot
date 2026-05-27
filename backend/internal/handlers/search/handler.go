// Package search provides the GET /api/v1/search/assets handler that
// builds an Elasticsearch bool query from query parameters.
package search

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/elasticsearch"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	searchUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/search"
)

type assetSearchUsecase interface {
	SearchAssets(context.Context, searchUC.SearchAssetsRequest) (*elasticsearch.SearchResponse, error)
}

// Handler serves search endpoints backed by Elasticsearch.
type Handler struct {
	es       *elasticsearch.Client
	search   assetSearchUsecase
	sync     func() SyncInfo
	progress func(context.Context) (SyncProgress, error)
}

// New creates a search Handler. If esClient is nil the handler will
// return 503 for search requests (graceful degradation). sync may be nil;
// SyncStatus then returns a minimal snapshot based only on whether ES is wired.
func New(esClient *elasticsearch.Client, sync func() SyncInfo, progress func(context.Context) (SyncProgress, error)) *Handler {
	return &Handler{es: esClient, search: searchUC.New(esClient), sync: sync, progress: progress}
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

	lineageWith := strings.TrimSpace(c.Query("lineage_with"))
	lineageDirection := strings.TrimSpace(c.Query("lineage_direction"))
	if lineageDirection == "" {
		lineageDirection = searchUC.DirectionBoth
	}
	lineageDirection = strings.ToLower(lineageDirection)
	if lineageDirection != searchUC.DirectionUpstream &&
		lineageDirection != searchUC.DirectionDownstream &&
		lineageDirection != searchUC.DirectionBoth {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "lineage_direction must be upstream, downstream, or both", nil)
		return
	}
	lineageDepth := 1
	if raw := strings.TrimSpace(c.Query("lineage_depth")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "lineage_depth must be a positive integer", nil)
			return
		}
		if v > 3 {
			v = 3
		}
		lineageDepth = v
	}
	relationTypes, err := parseRelationTypes(c.QueryArray("relation_types"), c.Query("relation_types"))
	if err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		return
	}

	req := searchUC.SearchAssetsRequest{
		Mode:             strings.TrimSpace(c.Query("mode")),
		Query:            q,
		Filters:          filters,
		Page:             page,
		PageSize:         pageSize,
		LineageWith:      lineageWith,
		LineageDirection: lineageDirection,
		LineageDepth:     lineageDepth,
		RelationTypes:    relationTypes,
	}

	result, err := h.search.SearchAssets(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, searchUC.ErrLineageSeedNotFound) {
			httpresp.Error(c, http.StatusNotFound, httpresp.CodeInvalidArgument, "lineage_with asset not found", nil)
			return
		}
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
			} else {
				doc["asset_id"] = "unknown"
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

var supportedRelationTypes = map[string]struct{}{
	"split_from":   {},
	"contains":     {},
	"derived_from": {},
	"merged_from":  {},
	"sampled_from": {},
	"revision_of":  {},
	"pipeline_output": {},
}

func parseRelationTypes(repeated []string, single string) ([]string, error) {
	rawValues := repeated
	if len(rawValues) == 0 && strings.TrimSpace(single) != "" {
		rawValues = []string{single}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, raw := range rawValues {
		for _, part := range strings.Split(raw, ",") {
			relationType := strings.ToLower(strings.TrimSpace(part))
			if relationType == "" {
				return nil, errors.New("relation_types must be a comma-separated list of supported relation types")
			}
			if _, ok := supportedRelationTypes[relationType]; !ok {
				return nil, errors.New("unsupported relation_type " + strconv.Quote(relationType))
			}
			if _, ok := seen[relationType]; ok {
				continue
			}
			seen[relationType] = struct{}{}
			out = append(out, relationType)
		}
	}
	return out, nil
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

// SyncProgress handles GET /api/v1/search/sync-progress.
func (h *Handler) SyncProgress(c *gin.Context) {
	if h.progress == nil {
		httpresp.Error(c, http.StatusServiceUnavailable,
			httpresp.CodeServiceUnavailable,
			"sync progress is not available", nil)
		return
	}
	progress, err := h.progress(c.Request.Context())
	if err != nil {
		httpresp.Error(c, http.StatusServiceUnavailable,
			httpresp.CodeServiceUnavailable,
			"failed to read sync progress: "+err.Error(), nil)
		return
	}
	c.JSON(http.StatusOK, progress)
}
