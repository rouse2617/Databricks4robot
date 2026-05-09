package admin

import (
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
)

type SearchAuditResponse struct {
	PGAssets               int64    `json:"pg_assets"`
	ElasticsearchDocs      int64    `json:"elasticsearch_docs"`
	MissingInElasticsearch int64    `json:"missing_in_elasticsearch"`
	OrphanInElasticsearch  int64    `json:"orphan_in_elasticsearch"`
	Consistency            float64  `json:"consistency"`
	Target                 float64  `json:"target"`
	SampleMissingIDs       []string `json:"sample_missing_ids,omitempty"`
	SampleOrphanIDs        []string `json:"sample_orphan_ids,omitempty"`
	DurationMs             int64    `json:"duration_ms"`
}

// SearchAudit performs a PG↔ES audit for the assets index (demo-sized).
// GET /api/v1/admin/search/audit
func (h *Handler) SearchAudit(c *gin.Context) {
	if h.es == nil {
		httpresp.Error(c, http.StatusServiceUnavailable, httpresp.CodeServiceUnavailable,
			"Elasticsearch is not configured", nil)
		return
	}

	ctx := c.Request.Context()
	startedAt := time.Now()

	// Ensure ES reflects recent CDC writes.
	_ = h.es.Refresh(ctx)

	// Collect PG asset IDs (non-deleted).
	const pageSize = 500
	pgIDs := make([]string, 0, 2048)
	page := 1
	for {
		assets, total, err := h.assets.ListWithFilters(ctx, "", nil, page, pageSize, "asset_id ASC")
		if err != nil {
			httpresp.Error(c, http.StatusInternalServerError, httpresp.CodeInternalError, err.Error(), nil)
			return
		}
		for _, a := range assets {
			if a != nil && a.AssetID != "" {
				pgIDs = append(pgIDs, a.AssetID)
			}
		}
		if int64(page*pageSize) >= total || len(assets) == 0 {
			break
		}
		page++
	}

	esIDs, err := h.es.ListAllDocumentIDs(ctx, 1000)
	if err != nil {
		httpresp.Error(c, http.StatusBadGateway, httpresp.CodeInternalError, err.Error(), nil)
		return
	}
	sort.Strings(esIDs)

	pgCount := int64(len(pgIDs))
	esCount := int64(len(esIDs))

	// Build sets for diff.
	pgSet := make(map[string]struct{}, len(pgIDs))
	for _, id := range pgIDs {
		pgSet[id] = struct{}{}
	}
	esSet := make(map[string]struct{}, len(esIDs))
	for _, id := range esIDs {
		esSet[id] = struct{}{}
	}

	var missing, orphan int64
	var missingSample, orphanSample []string
	const sampleN = 10

	for _, id := range pgIDs {
		if _, ok := esSet[id]; ok {
			continue
		}
		missing++
		if len(missingSample) < sampleN {
			missingSample = append(missingSample, id)
		}
	}
	for _, id := range esIDs {
		if _, ok := pgSet[id]; ok {
			continue
		}
		orphan++
		if len(orphanSample) < sampleN {
			orphanSample = append(orphanSample, id)
		}
	}

	den := float64(pgCount)
	if den <= 0 {
		den = 1
	}
	consistency := 1.0 - (float64(missing) / den)
	target := 0.999

	c.JSON(http.StatusOK, SearchAuditResponse{
		PGAssets:               pgCount,
		ElasticsearchDocs:      esCount,
		MissingInElasticsearch: missing,
		OrphanInElasticsearch:  orphan,
		Consistency:            consistency,
		Target:                 target,
		SampleMissingIDs:       missingSample,
		SampleOrphanIDs:        orphanSample,
		DurationMs:             time.Since(startedAt).Milliseconds(),
	})
}
