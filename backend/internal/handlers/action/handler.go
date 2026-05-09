// Package action exposes the seg-internal action HTTP API. See
// docs/review/api-guide.md §2.7 for the wire contract.
package action

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/id"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	actionUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/action"
)

// Handler bundles the action endpoints.
type Handler struct {
	uc *actionUC.Usecase
}

// New constructs a Handler.
func New(uc *actionUC.Usecase) *Handler { return &Handler{uc: uc} }

// requirePathAssetID validates :id as an 8-character alphanumeric asset id.
func requirePathAssetID(c *gin.Context) (string, bool) {
	raw := strings.TrimSpace(c.Param("id"))
	if raw == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "asset id is required", nil)
		return "", false
	}
	if !id.ValidateAssetID(raw) {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid asset id: must be 8 alphanumeric characters", nil)
		return "", false
	}
	return raw, true
}

func parseInt64Ptr(c *gin.Context, name string) (*int64, bool) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return nil, true
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid "+name, map[string]any{"error": err.Error()})
		return nil, false
	}
	return &v, true
}

// Create handles POST /assets/:id/actions.
func (h *Handler) Create(c *gin.Context) {
	assetID, ok := requirePathAssetID(c)
	if !ok {
		return
	}
	var req struct {
		StartNs       int64                  `json:"start_ns" binding:"required"`
		EndNs         int64                  `json:"end_ns" binding:"required"`
		ActionIndex   *int                   `json:"action_index"`
		PrimaryLabel  string                 `json:"primary_label"`
		Labels        []string               `json:"labels"`
		Description   string                 `json:"description"`
		Attrs         map[string]interface{} `json:"attrs"`
		SourceType    string                 `json:"source_type"`
		SourceName    string                 `json:"source_name"`
		SourceVersion string                 `json:"source_version"`
		RunID         string                 `json:"run_id"`
		Confidence    *float64               `json:"confidence"`
		ExternalID    string                 `json:"external_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}

	row, err := h.uc.Create(c.Request.Context(), actionUC.CreateInput{
		AssetID:       assetID,
		StartNs:       req.StartNs,
		EndNs:         req.EndNs,
		ActionIndex:   req.ActionIndex,
		PrimaryLabel:  req.PrimaryLabel,
		Labels:        req.Labels,
		Description:   req.Description,
		Attrs:         req.Attrs,
		SourceType:    req.SourceType,
		SourceName:    req.SourceName,
		SourceVersion: req.SourceVersion,
		RunID:         req.RunID,
		Confidence:    req.Confidence,
		ExternalID:    req.ExternalID,
	})
	if err != nil {
		switch {
		case errors.Is(err, actionUC.ErrSegNotFound):
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
		case errors.Is(err, actionUC.ErrParentNotSeg),
			errors.Is(err, actionUC.ErrRangeOutsideSeg),
			errors.Is(err, actionUC.ErrInvalidLabel):
			httpresp.Unprocessable(c, "INVALID_ACTION", err.Error(), nil)
		case errors.Is(err, actionUC.ErrInvalidRange),
			errors.Is(err, actionUC.ErrInvalidSourceType):
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		case errors.Is(err, repository.ErrOptimisticLock):
			httpresp.Conflict(c, httpresp.CodeConcurrentConflict,
				"action with the same external_id already exists", nil)
		case errors.Is(err, repository.ErrSchemaMismatch):
			httpresp.Internal(c, "actions schema mismatch: run migration 018_actions_id_to_short_id.sql")
		default:
			httpresp.Internal(c, err.Error())
		}
		return
	}
	c.JSON(201, row)
}

// List handles GET /assets/:id/actions[?at|from|to|label|limit].
func (h *Handler) List(c *gin.Context) {
	assetID, ok := requirePathAssetID(c)
	if !ok {
		return
	}
	at, ok := parseInt64Ptr(c, "at")
	if !ok {
		return
	}
	from, ok := parseInt64Ptr(c, "from")
	if !ok {
		return
	}
	to, ok := parseInt64Ptr(c, "to")
	if !ok {
		return
	}
	limit := 200
	if raw := c.Query("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 && v <= 1000 {
			limit = v
		}
	}

	items, err := h.uc.List(c.Request.Context(), actionUC.ListInput{
		AssetID:   assetID,
		PointAtNs: at,
		FromNs:    from,
		ToNs:      to,
		Label:     strings.TrimSpace(c.Query("label")),
		Limit:     limit,
	})
	if err != nil {
		switch {
		case errors.Is(err, actionUC.ErrSegNotFound):
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
		default:
			httpresp.Internal(c, err.Error())
		}
		return
	}
	if items == nil {
		items = []*models.Action{}
	}
	c.JSON(200, gin.H{"items": items, "asset_id": assetID, "total": len(items)})
}
