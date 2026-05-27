package pipeline

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
)

// Handler bundles the pipeline endpoints.
type Handler struct {
	uc *pipelineUC.Usecase
}

// New constructs a Handler.
func New(uc *pipelineUC.Usecase) *Handler { return &Handler{uc: uc} }

// SaveTemplate handles POST /api/v1/pipelines.
func (h *Handler) SaveTemplate(c *gin.Context) {
	var req struct {
		Name     string                 `json:"name" binding:"required"`
		Pipeline map[string]interface{} `json:"pipeline" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}

	t, err := h.uc.SaveTemplate(c.Request.Context(), req.Name, req.Pipeline)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(201, t)
}

// ListTemplates handles GET /api/v1/pipelines.
func (h *Handler) ListTemplates(c *gin.Context) {
	items, err := h.uc.ListTemplates(c.Request.Context())
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if items == nil {
		items = []models.PipelineTemplate{}
	}
	c.JSON(200, gin.H{"items": items})
}

// GetTemplate handles GET /api/v1/pipelines/:id.
func (h *Handler) GetTemplate(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}

	t, err := h.uc.GetTemplate(c.Request.Context(), id)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if t == nil {
		httpresp.NotFound(c, httpresp.CodeAssetNotFound, "template not found")
		return
	}
	c.JSON(200, t)
}

// DeleteTemplate handles DELETE /api/v1/pipelines/:id.
func (h *Handler) DeleteTemplate(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}

	if err := h.uc.DeleteTemplate(c.Request.Context(), id); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.Status(204)
}

// Deploy handles POST /api/v1/deploy.
func (h *Handler) Deploy(c *gin.Context) {
	var req struct {
		Pipeline map[string]interface{} `json:"pipeline" binding:"required"`
		Name     string                 `json:"name"`
		AssetIDs []string               `json:"asset_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}

	dep, err := h.uc.Deploy(c.Request.Context(), req.Pipeline, req.Name, req.AssetIDs)
	if err != nil {
		mapDeployError(c, err)
		return
	}
	c.JSON(201, dep)
}

// DeployByTemplate handles POST /api/v1/deploy/template/:id.
func (h *Handler) DeployByTemplate(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "template id is required", nil)
		return
	}

	var req struct {
		Name     string   `json:"name"`
		AssetIDs []string `json:"asset_ids"`
	}
	_ = c.ShouldBindJSON(&req) // name and asset_ids are optional

	dep, err := h.uc.DeployByTemplateID(c.Request.Context(), id, req.Name, req.AssetIDs)
	if err != nil {
		mapDeployError(c, err)
		return
	}
	c.JSON(201, dep)
}

// ListDeployments handles GET /api/v1/deployments.
func (h *Handler) ListDeployments(c *gin.Context) {
	items, err := h.uc.ListDeployments(c.Request.Context())
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if items == nil {
		items = []models.PipelineDeployment{}
	}
	c.JSON(200, gin.H{"items": items})
}

// GetDeployment handles GET /api/v1/deployments/:id.
func (h *Handler) GetDeployment(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}

	d, err := h.uc.GetDeployment(c.Request.Context(), id)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if d == nil {
		httpresp.NotFound(c, httpresp.CodeAssetNotFound, "deployment not found")
		return
	}
	c.JSON(200, d)
}

// DeleteDeployment handles DELETE /api/v1/deployments/:id.
func (h *Handler) DeleteDeployment(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}

	if err := h.uc.DeleteDeployment(c.Request.Context(), id); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.Status(204)
}

func mapDeployError(c *gin.Context, err error) {
	if errors.Is(err, pipelineUC.ErrTemplateNotFound) {
		httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
		return
	}
	httpresp.Internal(c, err.Error())
}
