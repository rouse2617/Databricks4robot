package pipeline_component

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/middleware"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	uc "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline_component"
)

// Handler bundles the pipeline component endpoints.
type Handler struct {
	uc *uc.Usecase
}

// New constructs a Handler.
func New(uc *uc.Usecase) *Handler { return &Handler{uc: uc} }

type syncReleasesRequest struct {
	Items []models.PipelineComponentRelease `json:"items"`
}

// CreateComponent handles POST /api/v1/components.
func (h *Handler) CreateComponent(c *gin.Context) {
	var pc models.PipelineComponent
	if err := c.ShouldBindJSON(&pc); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	pc.Scope = "dev"
	pc.Owner = middleware.GetUserEmail(c)
	created, err := h.uc.Create(c.Request.Context(), &pc)
	if err != nil {
		writeComponentError(c, err)
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
		writeComponentError(c, err)
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
		writeComponentError(c, err)
		return
	}
	c.Status(204)
}

// ListReleases handles GET /api/v1/pipeline-component-releases.
func (h *Handler) ListReleases(c *gin.Context) {
	filter := repository.ComponentReleaseFilter{
		Query:       strings.TrimSpace(c.Query("q")),
		ComponentID: strings.TrimSpace(c.Query("componentId")),
		TaskName:    strings.TrimSpace(c.Query("taskName")),
		Status:      strings.TrimSpace(c.Query("status")),
		Channel:     strings.TrimSpace(c.Query("channel")),
	}
	if raw := strings.TrimSpace(c.Query("selectable")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "selectable must be a boolean", nil)
			return
		}
		filter.Selectable = &value
	}
	items, err := h.uc.ListReleases(c.Request.Context(), filter)
	if err != nil {
		writeReleaseError(c, err)
		return
	}
	if items == nil {
		items = []models.PipelineComponentRelease{}
	}
	c.JSON(200, gin.H{"items": items})
}

// GetRelease handles GET /api/v1/pipeline-component-releases/:id.
func (h *Handler) GetRelease(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	release, err := h.uc.GetRelease(c.Request.Context(), id)
	if err != nil {
		writeReleaseError(c, err)
		return
	}
	if release == nil {
		httpresp.NotFound(c, httpresp.CodeAssetNotFound, "component release not found")
		return
	}
	c.JSON(200, release)
}

// SyncReleases handles POST /api/v1/pipeline-component-releases/sync.
func (h *Handler) SyncReleases(c *gin.Context) {
	var req syncReleasesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	items, err := h.uc.SyncReleases(c.Request.Context(), req.Items)
	if err != nil {
		writeReleaseError(c, err)
		return
	}
	if items == nil {
		items = []models.PipelineComponentRelease{}
	}
	c.JSON(200, gin.H{"items": items})
}

func writeComponentError(c *gin.Context, err error) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "not found"):
		httpresp.NotFound(c, httpresp.CodeAssetNotFound, "component not found")
	case strings.Contains(msg, "required") || strings.Contains(msg, "type must be"):
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, msg, nil)
	case strings.Contains(msg, "system components cannot"):
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, msg, nil)
	default:
		httpresp.Internal(c, msg)
	}
}

func writeReleaseError(c *gin.Context, err error) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "not found"):
		httpresp.NotFound(c, httpresp.CodeAssetNotFound, "component release not found")
	case strings.Contains(msg, "required") || strings.Contains(msg, "must be"):
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, msg, nil)
	default:
		httpresp.Internal(c, msg)
	}
}
