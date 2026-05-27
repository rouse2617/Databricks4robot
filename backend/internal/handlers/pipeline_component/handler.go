package pipeline_component

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	uc "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline_component"
)

// Handler bundles the pipeline component endpoints.
type Handler struct {
	uc *uc.Usecase
}

// New constructs a Handler.
func New(uc *uc.Usecase) *Handler { return &Handler{uc: uc} }

// CreateComponent handles POST /api/v1/components.
func (h *Handler) CreateComponent(c *gin.Context) {
	var pc models.PipelineComponent
	if err := c.ShouldBindJSON(&pc); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	created, err := h.uc.Create(c.Request.Context(), &pc)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(201, created)
}

// ListComponents handles GET /api/v1/components?q=&source=.
func (h *Handler) ListComponents(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	source := strings.TrimSpace(c.Query("source"))
	items, err := h.uc.List(c.Request.Context(), q, source)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if items == nil {
		items = []models.PipelineComponent{}
	}
	c.JSON(200, gin.H{"items": items})
}

// GetComponent handles GET /api/v1/components/:id.
func (h *Handler) GetComponent(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	pc, err := h.uc.Get(c.Request.Context(), id)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if pc == nil {
		httpresp.NotFound(c, httpresp.CodeAssetNotFound, "component not found")
		return
	}
	c.JSON(200, pc)
}

// UpdateComponent handles PUT /api/v1/components/:id.
func (h *Handler) UpdateComponent(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	var pc models.PipelineComponent
	if err := c.ShouldBindJSON(&pc); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	pc.ID = id
	if err := h.uc.Update(c.Request.Context(), &pc); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, &pc)
}

// DeleteComponent handles DELETE /api/v1/components/:id.
func (h *Handler) DeleteComponent(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	if err := h.uc.Delete(c.Request.Context(), id); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.Status(204)
}
