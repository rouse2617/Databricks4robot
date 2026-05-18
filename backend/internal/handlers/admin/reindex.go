package admin

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	espkg "github.com/CyberOrigin2077/cyber-databrew/internal/elasticsearch"
	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	"github.com/CyberOrigin2077/cyber-databrew/internal/searchindex"
)

// Handler serves administrative maintenance endpoints.
type Handler struct {
	assets  repository.AssetRepository
	tags    repository.AssetTagRepository
	algos   repository.AssetAlgoLatestRepository
	mcap    repository.McapFileRepository
	actions repository.ActionRepository
	events  repository.AssetEventRepository
	dlq     repository.OutboxDLQRepository
	jobs    repository.SearchReindexJobRepository
	es      *espkg.Client
	indexer *searchindex.Builder

	jobMu       sync.Mutex
	runningJobs map[string]struct{}
}

// New constructs an admin Handler. es may be nil (handler returns errors for reindex).
// actions, events, and dlq may be nil; optional endpoints degrade gracefully when unset.
func New(
	assets repository.AssetRepository,
	tags repository.AssetTagRepository,
	algos repository.AssetAlgoLatestRepository,
	mcap repository.McapFileRepository,
	actions repository.ActionRepository,
	es *espkg.Client,
	events repository.AssetEventRepository,
	dlq repository.OutboxDLQRepository,
	jobs repository.SearchReindexJobRepository,
) *Handler {
	b := &searchindex.Builder{Assets: assets, Tags: tags, Algos: algos, Mcap: mcap, Actions: actions}
	return &Handler{
		assets:      assets,
		tags:        tags,
		algos:       algos,
		mcap:        mcap,
		actions:     actions,
		events:      events,
		dlq:         dlq,
		jobs:        jobs,
		es:          es,
		indexer:     b,
		runningJobs: map[string]struct{}{},
	}
}

// SearchReindexRequest is the JSON body for POST /admin/search/reindex.
type SearchReindexRequest struct {
	DryRun   bool `json:"dry_run"`
	PageSize int  `json:"page_size"`
}

type SearchReindexResponse struct {
	DryRun                bool     `json:"dry_run"`
	TotalAssets           int64    `json:"total_assets"`
	Indexed               int64    `json:"indexed"`
	Deleted               int64    `json:"deleted"`
	Failed                int64    `json:"failed"`
	DurationMs            int64    `json:"duration_ms"`
	Errors                []string `json:"errors,omitempty"`
	ElasticsearchDocCount int64    `json:"elasticsearch_doc_count,omitempty"`
	AssetsScanned         int64    `json:"assets_scanned"`
	DocumentsIndexed      int64    `json:"documents_indexed"`
	DocumentsDeleted      int64    `json:"documents_deleted"`
}

const maxReportedReindexErrors = 20

func parseSearchReindexRequest(c *gin.Context) (SearchReindexRequest, error) {
	var req SearchReindexRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			return SearchReindexRequest{}, fmt.Errorf("invalid request body: %w", err)
		}
	}

	if raw := c.Query("dry_run"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return SearchReindexRequest{}, fmt.Errorf("invalid dry_run: %w", err)
		}
		req.DryRun = v
	}

	if raw := c.Query("page_size"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return SearchReindexRequest{}, fmt.Errorf("invalid page_size: %w", err)
		}
		req.PageSize = v
	}

	if req.PageSize <= 0 || req.PageSize > 500 {
		req.PageSize = 200
	}
	return req, nil
}

func appendReindexError(errors []string, msg string) []string {
	if len(errors) >= maxReportedReindexErrors {
		return errors
	}
	return append(errors, msg)
}

// SearchReindex rebuilds all ES documents from PostgreSQL (does not move outbox cursors).
func (h *Handler) SearchReindex(c *gin.Context) {
	if h.es == nil {
		httpresp.Error(c, http.StatusServiceUnavailable, httpresp.CodeServiceUnavailable,
			"Elasticsearch is not configured", nil)
		return
	}
	req, err := parseSearchReindexRequest(c)
	if err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid reindex request", map[string]any{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	startedAt := time.Now()
	var scanned, indexed, deleted, failed int64
	var errors []string

	if !req.DryRun {
		cleared, err := h.es.DeleteAllDocuments(ctx)
		if err != nil {
			httpresp.Error(c, http.StatusBadGateway, httpresp.CodeInternalError, err.Error(), nil)
			return
		}
		deleted += cleared
	}

	page := 1
	for {
		assets, total, err := h.assets.ListWithFilters(ctx, "", nil, page, req.PageSize, filter.OrderByClause{SQL: "asset_id ASC"})
		if err != nil {
			httpresp.Error(c, http.StatusInternalServerError, httpresp.CodeInternalError, err.Error(), nil)
			return
		}
		if len(assets) == 0 {
			break
		}
		var docs []espkg.BulkIndexDoc
		for _, a := range assets {
			scanned++
			doc, ok, err := h.indexer.Build(ctx, a.AssetID)
			if err != nil {
				failed++
				errors = appendReindexError(errors, fmt.Sprintf("build %s: %v", a.AssetID, err))
				continue
			}
			if !ok {
				if req.DryRun {
					deleted++
					continue
				}
				if err := h.es.DeleteDocument(ctx, a.AssetID); err != nil {
					failed++
					errors = appendReindexError(errors, fmt.Sprintf("delete %s: %v", a.AssetID, err))
					continue
				}
				deleted++
				continue
			}
			if req.DryRun {
				indexed++
				continue
			}
			docs = append(docs, espkg.BulkIndexDoc{ID: a.AssetID, Doc: doc})
		}
		if !req.DryRun && len(docs) > 0 {
			bulkResult, err := h.es.BulkIndex(ctx, docs)
			if err != nil {
				httpresp.Error(c, http.StatusBadGateway, httpresp.CodeInternalError, err.Error(), nil)
				return
			}
			indexed += int64(len(bulkResult.Succeeded))
			failed += int64(len(bulkResult.Failed))
			for _, item := range bulkResult.Failed {
				reason := item.Error
				if reason == "" {
					reason = fmt.Sprintf("status %d", item.Status)
				}
				errors = appendReindexError(errors, fmt.Sprintf("index %s: %s", item.ID, reason))
			}
		}
		if int64(page*req.PageSize) >= total {
			break
		}
		page++
	}

	var esCount int64
	if !req.DryRun {
		var err error
		esCount, err = h.es.Count(ctx)
		if err != nil {
			esCount = -1
		}
	}

	resp := SearchReindexResponse{
		DryRun:                req.DryRun,
		TotalAssets:           scanned,
		Indexed:               indexed,
		Deleted:               deleted,
		Failed:                failed,
		DurationMs:            time.Since(startedAt).Milliseconds(),
		Errors:                errors,
		ElasticsearchDocCount: esCount,
		AssetsScanned:         scanned,
		DocumentsIndexed:      indexed,
		DocumentsDeleted:      deleted,
	}
	c.JSON(http.StatusOK, resp)
}

// AdminTokenAuth validates X-Admin-Token (or admin_token query for scripts).
func AdminTokenAuth(expected string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if expected == "" {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "admin disabled"})
			return
		}
		tok := c.GetHeader("X-Admin-Token")
		if tok == "" {
			tok = c.Query("admin_token")
		}
		if tok != expected {
			httpresp.Unauthorized(c, httpresp.CodeUnauthorized, "invalid admin token")
			c.Abort()
			return
		}
		c.Next()
	}
}
