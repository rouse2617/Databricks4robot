package asset

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	assetUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/asset"
)

// previewableLifecycleStates lists asset lifecycle states for which MCAP locator
// and foxglove-source are allowed. Callers still require a linked mcap_files row;
// "created" covers assets that are registered with MCAP bytes before promotion
// to ready (common in dev / Cloud Run smoke assets).
var previewableLifecycleStates = map[string]struct{}{
	"created":    {},
	"ready":      {},
	"delivered":  {},
	"archived":   {},
	"superseded": {},
}

// McapLocatorResponse is returned by GET /assets/:id/mcap-locator.
//
// It bundles the data preview consumers need to fetch the right byte range
// from the underlying MCAP without having to follow two separate endpoints.
type McapLocatorResponse struct {
	AssetID        string            `json:"asset_id"`
	LifecycleState string            `json:"lifecycle_state"`
	Mcap           McapLocatorMcap   `json:"mcap"`
	Window         McapLocatorWindow `json:"window"`
	Version        int64             `json:"version"`
	UpdatedAt      string            `json:"updated_at"`
}

type McapLocatorMcap struct {
	McapFileID string `json:"mcap_file_id"`
	McapURI    string `json:"mcap_uri"`
	SizeBytes  int64  `json:"size_bytes"`
	RawHashMD5 string `json:"raw_hash_md5,omitempty"`
}

// McapLocatorWindow uses the asset row time slice.
//
// StartTimestampNs/EndTimestampNs MUST share the MCAP recording time base
// used in message log_time (typically POSIX time in nanoseconds from Unix epoch).
// Foxglove/hints embed the same pair; mismatched epochs break browser-side window
// intersection with indexed chunk timestamps.
type McapLocatorWindow struct {
	StartTimestampNs int64 `json:"start_timestamp_ns"`
	EndTimestampNs   int64 `json:"end_timestamp_ns"`
	DurationMs       int64 `json:"duration_ms"`
}

// FoxgloveSourceResponse returns a direct-MCAP source contract compatible with
// ds/ds.* URL state in Foxglove/Lichtblick style clients.
type FoxgloveSourceResponse struct {
	AssetID   string            `json:"asset_id"`
	SourceID  string            `json:"source_id"`
	DS        string            `json:"ds"`
	DSParams  map[string]string `json:"ds_params"`
	Hints     map[string]any    `json:"hints,omitempty"`
	ExpiresAt string            `json:"expires_at,omitempty"`
}

// McapLocator handles GET /api/v1/assets/{id}/mcap-locator.
//
// It joins the asset row (time window) with its mcap_files row (gs:// URI,
// size, hash). Preview services use this single call to drive byte-range
// reads against GCS — the locator is read-only and never signs URLs.
//
// @Summary      Get asset MCAP locator (time window + MCAP object)
// @Tags         assets
// @Produce      json
// @Param        id   path      string  true  "Asset ID"
// @Success      200  {object}  asset.McapLocatorResponse
// @Failure      404  {object}  httpresp.ErrorBody
// @Failure      409  {object}  httpresp.ErrorBody
// @Failure      503  {object}  httpresp.ErrorBody
// @Security     DatabrewToken
// @Router       /assets/{id}/mcap-locator [get]
func (h *Handler) McapLocator(c *gin.Context) {
	if h.mcapRepo == nil {
		httpresp.Error(c, http.StatusServiceUnavailable, httpresp.CodeServiceUnavailable, "mcap repository not configured", nil)
		return
	}

	a, err := h.uc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, assetUC.ErrNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}

	state := string(a.LifecycleState)
	if _, ok := previewableLifecycleStates[state]; !ok {
		httpresp.Conflict(c, httpresp.CodeAssetNotPreviewable,
			"asset lifecycle_state does not yet support preview",
			map[string]any{"lifecycle_state": state},
		)
		return
	}

	if a.McapFileID == "" {
		httpresp.Conflict(c, httpresp.CodeAssetNotPreviewable,
			"asset has no associated mcap file",
			map[string]any{"asset_id": a.AssetID},
		)
		return
	}

	mf, err := h.mcapRepo.Get(c.Request.Context(), a.McapFileID)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if mf == nil {
		httpresp.NotFound(c, httpresp.CodeMcapFileNotFound, "mcap file not found")
		return
	}

	c.JSON(http.StatusOK, buildMcapLocator(a, mf))
}

// FoxgloveSource handles GET /api/v1/assets/{id}/foxglove-source.
//
// It exposes a direct-MCAP source contract (`ds=remote-file`) so frontend
// players can read MCAP without going through segment.mp4 remux.
func (h *Handler) FoxgloveSource(c *gin.Context) {
	if h.mcapRepo == nil {
		httpresp.Error(c, http.StatusServiceUnavailable, httpresp.CodeServiceUnavailable, "mcap repository not configured", nil)
		return
	}

	a, err := h.uc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, assetUC.ErrNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}

	state := string(a.LifecycleState)
	if _, ok := previewableLifecycleStates[state]; !ok {
		httpresp.Conflict(c, httpresp.CodeAssetNotPreviewable,
			"asset lifecycle_state does not yet support preview",
			map[string]any{"lifecycle_state": state},
		)
		return
	}

	if a.McapFileID == "" {
		httpresp.Conflict(c, httpresp.CodeAssetNotPreviewable,
			"asset has no associated mcap file",
			map[string]any{"asset_id": a.AssetID},
		)
		return
	}

	mf, err := h.mcapRepo.Get(c.Request.Context(), a.McapFileID)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if mf == nil {
		httpresp.NotFound(c, httpresp.CodeMcapFileNotFound, "mcap file not found")
		return
	}

	c.JSON(http.StatusOK, buildFoxgloveSource(a, mf))
}

func buildMcapLocator(a *models.Asset, mf *models.McapFile) McapLocatorResponse {
	return McapLocatorResponse{
		AssetID:        a.AssetID,
		LifecycleState: string(a.LifecycleState),
		Mcap: McapLocatorMcap{
			McapFileID: mf.McapFileID,
			McapURI:    mf.GCSPath,
			SizeBytes:  mf.SizeBytes,
			RawHashMD5: mf.RawHashMD5,
		},
		Window: McapLocatorWindow{
			StartTimestampNs: a.StartTimestampNs,
			EndTimestampNs:   a.EndTimestampNs,
			DurationMs:       a.DurationMs,
		},
		Version:   a.Version,
		UpdatedAt: a.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

func buildFoxgloveSource(a *models.Asset, mf *models.McapFile) FoxgloveSourceResponse {
	sourceID := "remote-file"
	proxyURL := fmt.Sprintf("/api/v1/mcap-files/%s/bytes", mf.McapFileID)
	return FoxgloveSourceResponse{
		AssetID:  a.AssetID,
		SourceID: sourceID,
		DS:       sourceID,
		DSParams: map[string]string{
			// Same-origin proxy so browser-based players can fetch with CORS-free Range.
			"url":          proxyURL,
			"mcap_file_id": mf.McapFileID,
		},
		// hints.window echoes mcap-locator time windows (POSIX ns with MCAP log_time).
		Hints: map[string]any{
			"lifecycle_state": a.LifecycleState,
			"gcs_path":        mf.GCSPath,
			"window": map[string]any{
				"start_timestamp_ns": a.StartTimestampNs,
				"end_timestamp_ns":   a.EndTimestampNs,
				"duration_ms":        a.DurationMs,
			},
		},
	}
}
