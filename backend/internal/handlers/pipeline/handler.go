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

// ListVersions handles GET /api/v1/pipelines/:id/versions.
func (h *Handler) ListVersions(c *gin.Context) {
	name := strings.TrimSpace(c.Param("id"))
	if name == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "name is required", nil)
		return
	}

	items, err := h.uc.ListVersions(c.Request.Context(), name)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if items == nil {
		items = []models.PipelineTemplate{}
	}
	c.JSON(200, gin.H{"items": items})
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

// RetryDeployment handles POST /api/v1/deployments/:id/retry.
func (h *Handler) RetryDeployment(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	dep, err := h.uc.RetryDeployment(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pipelineUC.ErrDeploymentNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(201, dep)
}

// StopDeployment handles POST /api/v1/deployments/:id/stop.
func (h *Handler) StopDeployment(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	if err := h.uc.StopDeployment(c.Request.Context(), id); err != nil {
		if errors.Is(err, pipelineUC.ErrDeploymentNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, gin.H{"message": "workflow stopped"})
}

// SaveFromDeployment handles POST /api/v1/deployments/:id/save-template.
func (h *Handler) SaveFromDeployment(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "deployment id is required", nil)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	_ = c.ShouldBindJSON(&req)

	t, err := h.uc.SaveFromDeployment(c.Request.Context(), id, req.Name)
	if err != nil {
		if errors.Is(err, pipelineUC.ErrDeploymentNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(201, t)
}

// GetResourceUsage handles GET /api/v1/deployments/:id/resources.
func (h *Handler) GetResourceUsage(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "deployment id is required", nil)
		return
	}

	report, err := h.uc.GetResourceUsage(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pipelineUC.ErrDeploymentNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, report)
}

// DiffTemplates handles GET /api/v1/pipelines/:id1/diff/:id2.
func (h *Handler) DiffTemplates(c *gin.Context) {
	id1 := strings.TrimSpace(c.Param("id"))
	id2 := strings.TrimSpace(c.Param("id2"))
	if id1 == "" || id2 == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "both id1 and id2 are required", nil)
		return
	}

	diff, err := h.uc.DiffTemplates(c.Request.Context(), id1, id2)
	if err != nil {
		if errors.Is(err, pipelineUC.ErrTemplateNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, diff)
}

// RegisterOutput handles POST /api/v1/pipeline-assets.
// Pipeline containers call back to register processing results as new assets.
func (h *Handler) RegisterOutput(c *gin.Context) {
	var req pipelineUC.RegisterPipelineOutputInput
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	if req.DeploymentID == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "deployment_id is required", nil)
		return
	}
	asset, err := h.uc.RegisterOutput(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, pipelineUC.ErrDeploymentNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(201, asset)
}

// GetLineage handles GET /api/v1/assets/:id/pipeline-lineage.
func (h *Handler) GetLineage(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "asset id is required", nil)
		return
	}
	lineage, err := h.uc.GetLineage(c.Request.Context(), id)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, lineage)
}
