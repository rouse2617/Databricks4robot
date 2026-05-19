package admin

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/audit"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	adminuc "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/admin"
)

// PurgeHandler serves the internal hard-delete endpoints. It is constructed
// separately from the search-admin Handler so dependencies stay minimal.
type PurgeHandler struct {
	uc *adminuc.Usecase
}

// NewPurgeHandler returns a PurgeHandler ready to be registered against the
// routes in /api/v1/internal/*.
func NewPurgeHandler(uc *adminuc.Usecase) *PurgeHandler {
	return &PurgeHandler{uc: uc}
}

// BatchDeleteRequest is the JSON body for POST /api/v1/internal/assets:batch_delete.
// Exactly one of AssetIDs or ImportBatch must be provided.
type BatchDeleteRequest struct {
	AssetIDs         []string `json:"asset_ids,omitempty"`
	ImportBatch      string   `json:"import_batch,omitempty"`
	IncludeMcapFiles bool     `json:"include_mcap_files,omitempty"`
	DryRun           bool     `json:"dry_run,omitempty"`
	ChunkSize        int      `json:"chunk_size,omitempty"`
}

// DeleteAssetHard handles DELETE /api/v1/internal/assets/:id.
// Hard-deletes the asset and all of its child rows (tags, algo, eval, events
// referencing the asset_id, etc.). The associated mcap_file is left in place
// because shared metadata is out of scope for a point delete.
func (h *PurgeHandler) DeleteAssetHard(c *gin.Context) {
	assetID := c.Param("id")
	counts, err := h.uc.PurgeOne(c.Request.Context(), assetID)
	if err != nil {
		switch {
		case errors.Is(err, adminuc.ErrInvalidInput):
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		case errors.Is(err, adminuc.ErrNotFound):
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
		default:
			httpresp.Internal(c, err.Error())
		}
		return
	}
	audit.Log(c.Request.Context(), "asset.hard_delete", "asset", []string{assetID}, nil)
	c.JSON(http.StatusOK, gin.H{
		"asset_id": assetID,
		"deleted":  counts,
	})
}

// BatchDeleteAssets handles POST /api/v1/internal/assets:batch_delete.
// Accepts either an explicit list of asset_ids or an import_batch tag and
// optionally extends the operation to include mcap_files cleanup.
func (h *PurgeHandler) BatchDeleteAssets(c *gin.Context) {
	var req BatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}

	out, err := h.uc.PurgeBatch(c.Request.Context(), adminuc.BatchInput{
		AssetIDs:         req.AssetIDs,
		ImportBatch:      req.ImportBatch,
		IncludeMcapFiles: req.IncludeMcapFiles,
		DryRun:           req.DryRun,
		ChunkSize:        req.ChunkSize,
	})
	if err != nil {
		if errors.Is(err, adminuc.ErrInvalidInput) {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}

	if !req.DryRun {
		audit.Log(c.Request.Context(), "asset.batch_hard_delete", "asset", req.AssetIDs, map[string]any{
			"import_batch":       req.ImportBatch,
			"include_mcap_files": req.IncludeMcapFiles,
			"resolved":           out.Resolved,
		})
	}
	c.JSON(http.StatusOK, out)
}
