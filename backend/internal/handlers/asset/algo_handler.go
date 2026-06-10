package asset

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/audit"
	"github.com/CyberOrigin2077/cyber-databrew/internal/handlers"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	assetUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/asset"
)

// AlgoHandler handles algorithm lifecycle HTTP requests.
type AlgoHandler struct {
	uc *assetUC.AlgoUsecase
}

// NewAlgoHandler creates a new AlgoHandler.
func NewAlgoHandler(uc *assetUC.AlgoUsecase) *AlgoHandler {
	return &AlgoHandler{uc: uc}
}

// Start begins an algorithm run on an asset.
// @Summary      Start algorithm
// @Description  Mark an algorithm as running on the given asset
// @Tags         algorithms
// @Accept       json
// @Produce      json
// @Param        id       path string true "Asset ID"
// @Param        algo_key path string true "Algorithm key (name@version)"
// @Param        body     body object true "Start algo request"
// @Success      200 {object} object
// @Failure      400 {object} httpresp.ErrorBody
// @Failure      404 {object} httpresp.ErrorBody
// @Failure      409 {object} httpresp.ErrorBody
// @Security     DatabrewToken
// @Router       /assets/{id}/algo/{algo_key}/start [post]
func (h *AlgoHandler) Start(c *gin.Context) {
	assetID, ok := handlers.RequirePathAssetID(c)
	if !ok {
		return
	}
	algoKey := c.Param("algo_key")

	var req struct {
		Method string  `json:"method" binding:"required"`
		RunID  *string `json:"run_id,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}

	err := h.uc.StartAlgo(c.Request.Context(), assetID, algoKey, assetUC.StartAlgoInput{
		Method: req.Method,
		RunID:  req.RunID,
	})
	if err != nil {
		h.mapError(c, err)
		return
	}
	c.JSON(200, gin.H{"asset_id": assetID, "algo_key": algoKey, "status": "running"})
}

// Finish completes an algorithm run on an asset.
// @Summary      Finish algorithm
// @Description  Mark an algorithm as ok or failed on the given asset
// @Tags         algorithms
// @Accept       json
// @Produce      json
// @Param        id       path string true "Asset ID"
// @Param        algo_key path string true "Algorithm key (name@version)"
// @Param        body     body object true "Finish algo request"
// @Success      200 {object} object
// @Failure      400 {object} httpresp.ErrorBody
// @Failure      404 {object} httpresp.ErrorBody
// @Failure      409 {object} httpresp.ErrorBody
// @Failure      422 {object} httpresp.ErrorBody
// @Security     DatabrewToken
// @Router       /assets/{id}/algo/{algo_key}/finish [post]
func (h *AlgoHandler) Finish(c *gin.Context) {
	assetID, ok := handlers.RequirePathAssetID(c)
	if !ok {
		return
	}
	algoKey := c.Param("algo_key")

	var req struct {
		Status          string                 `json:"status" binding:"required"`
		OutputURI       *string                `json:"output_uri,omitempty"`
		RunID           *string                `json:"run_id,omitempty"`
		Reason          *string                `json:"reason,omitempty"`
		ResultSizeBytes *int64                 `json:"result_size_bytes,omitempty"`
		ExtraFields     map[string]interface{} `json:"extra_fields,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}

	err := h.uc.FinishAlgo(c.Request.Context(), assetID, algoKey, assetUC.FinishAlgoInput{
		Status:          req.Status,
		OutputURI:       req.OutputURI,
		RunID:           req.RunID,
		Reason:          req.Reason,
		ResultSizeBytes: req.ResultSizeBytes,
		ExtraFields:     req.ExtraFields,
	})
	if err != nil {
		h.mapError(c, err)
		return
	}
	c.JSON(200, gin.H{"asset_id": assetID, "algo_key": algoKey, "status": req.Status})
}

// Reset resets a failed or ok algorithm back to pending.
// @Summary      Reset algorithm
// @Description  Reset a failed or ok algorithm back to pending state
// @Tags         algorithms
// @Produce      json
// @Param        id       path string true "Asset ID"
// @Param        algo_key path string true "Algorithm key (name@version)"
// @Success      200 {object} object
// @Failure      400 {object} httpresp.ErrorBody
// @Failure      404 {object} httpresp.ErrorBody
// @Failure      409 {object} httpresp.ErrorBody
// @Security     DatabrewToken
// @Router       /assets/{id}/algo/{algo_key}/reset [post]
func (h *AlgoHandler) Reset(c *gin.Context) {
	assetID := c.Param("id")
	algoKey := c.Param("algo_key")

	err := h.uc.ResetAlgo(c.Request.Context(), assetID, algoKey)
	if err != nil {
		h.mapError(c, err)
		return
	}
	audit.Log(c.Request.Context(), "algo.reset", "asset", []string{assetID}, map[string]any{"algo_key": algoKey})
	c.JSON(200, gin.H{"asset_id": assetID, "algo_key": algoKey, "status": "pending"})
}

// GET /api/v1/assets/:id/algo
func (h *AlgoHandler) ListCurrent(c *gin.Context) {
	assetID := c.Param("id")
	rows, err := h.uc.ListCurrentStates(c.Request.Context(), assetID)
	if err != nil {
		h.mapError(c, err)
		return
	}
	c.JSON(200, gin.H{"items": rows})
}

// mapError maps usecase errors to appropriate HTTP responses.
func (h *AlgoHandler) mapError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, assetUC.ErrInvalidAlgoKey):
		httpresp.BadRequest(c, httpresp.CodeInvalidAlgoKey, err.Error(), nil)
	case errors.Is(err, assetUC.ErrAssetNotFound):
		httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
	case errors.Is(err, assetUC.ErrAlgoAlreadyRunning):
		httpresp.Conflict(c, httpresp.CodeAlgoAlreadyRunning, err.Error(), nil)
	case errors.Is(err, assetUC.ErrInvalidStateTransition):
		httpresp.Conflict(c, httpresp.CodeInvalidStateTransition, err.Error(), nil)
	case errors.Is(err, assetUC.ErrConcurrentConflict):
		httpresp.Conflict(c, httpresp.CodeConcurrentConflict, err.Error(), nil)
	case errors.Is(err, assetUC.ErrMissingRequiredField):
		httpresp.Unprocessable(c, httpresp.CodeMissingRequiredField, err.Error(), nil)
	case errors.Is(err, assetUC.ErrMissingReason):
		httpresp.Unprocessable(c, httpresp.CodeMissingReason, err.Error(), nil)
	case errors.Is(err, assetUC.ErrRunNotFound):
		httpresp.BadRequest(c, httpresp.CodeAlgoRunNotFound, err.Error(), nil)
	default:
		httpresp.Internal(c, err.Error())
	}
}
