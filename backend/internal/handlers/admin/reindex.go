package admin

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	espkg "data-platform/internal/elasticsearch"
	"data-platform/internal/httpresp"
	"data-platform/internal/repository"
	"data-platform/internal/searchindex"
)

// Handler serves administrative maintenance endpoints.
type Handler struct {
	assets  repository.AssetRepository
	tags    repository.AssetTagRepository
	algos   repository.AssetAlgoLatestRepository
	mcap    repository.McapFileRepository
	es      *espkg.Client
	indexer *searchindex.Builder
}

// New constructs an admin Handler. es may be nil (handler returns errors for reindex).
func New(
	assets repository.AssetRepository,
	tags repository.AssetTagRepository,
	algos repository.AssetAlgoLatestRepository,
	mcap repository.McapFileRepository,
	es *espkg.Client,
) *Handler {
	return &Handler{
		assets:  assets,
		tags:    tags,
		algos:   algos,
		mcap:    mcap,
		es:      es,
		indexer: &searchindex.Builder{Assets: assets, Tags: tags, Algos: algos, Mcap: mcap},
	}
}

// SearchReindexRequest is the JSON body for POST /admin/search/reindex.
type SearchReindexRequest struct {
	DryRun   bool `json:"dry_run"`
	PageSize int  `json:"page_size"`
}

// SearchReindex rebuilds all ES documents from PostgreSQL (does not move outbox cursors).
func (h *Handler) SearchReindex(c *gin.Context) {
	if h.es == nil {
		httpresp.Error(c, http.StatusServiceUnavailable, httpresp.CodeServiceUnavailable,
			"Elasticsearch is not configured", nil)
		return
	}
	var req SearchReindexRequest
	if c.Request.ContentLength > 0 {
		_ = c.ShouldBindJSON(&req)
	}
	if req.PageSize <= 0 || req.PageSize > 500 {
		req.PageSize = 200
	}

	ctx := c.Request.Context()
	var scanned, indexed, deleted int64
	page := 1
	for {
		assets, total, err := h.assets.ListWithFilters(ctx, "", nil, page, req.PageSize, "asset_id ASC")
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
				httpresp.Error(c, http.StatusInternalServerError, httpresp.CodeInternalError, err.Error(), nil)
				return
			}
			if !ok {
				if req.DryRun {
					deleted++
					continue
				}
				if err := h.es.DeleteDocument(ctx, a.AssetID); err != nil {
					httpresp.Error(c, http.StatusBadGateway, httpresp.CodeInternalError, err.Error(), nil)
					return
				}
				deleted++
				continue
			}
			if !req.DryRun {
				docs = append(docs, espkg.BulkIndexDoc{ID: a.AssetID, Doc: doc})
			}
			indexed++
		}
		if !req.DryRun && len(docs) > 0 {
			if _, err := h.es.BulkIndex(ctx, docs); err != nil {
				httpresp.Error(c, http.StatusBadGateway, httpresp.CodeInternalError, err.Error(), nil)
				return
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

	c.JSON(http.StatusOK, gin.H{
		"dry_run":                 req.DryRun,
		"assets_scanned":          scanned,
		"documents_indexed":       indexed,
		"documents_deleted":       deleted,
		"elasticsearch_doc_count": esCount,
		"finished_at":             time.Now().UTC().Format(time.RFC3339Nano),
	})
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
