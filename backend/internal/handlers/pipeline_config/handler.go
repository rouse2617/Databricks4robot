package pipeline_config

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/middleware"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	uc "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline_config"
)

// Handler bundles standalone pipeline config endpoints.
type Handler struct {
	uc *uc.Usecase
}

// New constructs a Handler.
func New(uc *uc.Usecase) *Handler { return &Handler{uc: uc} }

type createConfigRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	FileType    string   `json:"fileType"`
	Lifecycle   string   `json:"lifecycle"`
	Scope       string   `json:"scope"`
	Content     string   `json:"content"`
	Summary     string   `json:"summary"`
}

type updateConfigRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	FileType    string   `json:"fileType"`
	Lifecycle   string   `json:"lifecycle"`
}

type createVersionRequest struct {
	Status  string `json:"status"`
	Content string `json:"content"`
	Summary string `json:"summary"`
}

// List handles GET /api/v1/pipeline-configs.
func (h *Handler) List(c *gin.Context) {
	user, admin := currentUser(c)
	filter := repository.PipelineConfigFilter{
		Query:     strings.TrimSpace(c.Query("q")),
		Scope:     strings.TrimSpace(c.Query("scope")),
		Lifecycle: strings.TrimSpace(c.Query("lifecycle")),
	}
	if admin {
		filter.Owner = strings.TrimSpace(c.Query("owner"))
	} else {
		filter.Owner = user
	}
	items, err := h.uc.List(c.Request.Context(), filter)
	if err != nil {
		writeConfigError(c, err)
		return
	}
	if items == nil {
		items = []models.PipelineConfig{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// Create handles POST /api/v1/pipeline-configs.
func (h *Handler) Create(c *gin.Context) {
	var req createConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	user, _ := currentUser(c)
	created, err := h.uc.Create(c.Request.Context(), uc.CreateConfigInput{
		Name:        req.Name,
		Description: req.Description,
		Tags:        req.Tags,
		FileType:    req.FileType,
		Lifecycle:   req.Lifecycle,
		Content:     req.Content,
		Summary:     req.Summary,
		Owner:       user,
		Scope:       req.Scope,
	})
	if err != nil {
		writeConfigError(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

// Get handles GET /api/v1/pipeline-configs/:id.
func (h *Handler) Get(c *gin.Context) {
	cfg, ok := h.authorizedConfig(c, c.Param("id"))
	if !ok {
		return
	}
	c.JSON(http.StatusOK, cfg)
}

// Update handles PUT /api/v1/pipeline-configs/:id.
func (h *Handler) Update(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	existing, ok := h.authorizedConfig(c, id)
	if !ok {
		return
	}
	var req updateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	updated, err := h.uc.Update(c.Request.Context(), existing.ID, uc.UpdateConfigInput{
		Name:        req.Name,
		Description: req.Description,
		Tags:        req.Tags,
		FileType:    req.FileType,
		Lifecycle:   req.Lifecycle,
	})
	if err != nil {
		writeConfigError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

// UpdateVersionStatus handles PUT /api/v1/pipeline-configs/:id/versions/:version/status.
func (h *Handler) UpdateVersionStatus(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	versionStr := strings.TrimSpace(c.Param("version"))
	version, parseErr := strconv.Atoi(versionStr)
	if parseErr != nil || version <= 0 {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid version", nil)
		return
	}
	var req struct {
		Status  string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	updated, err := h.uc.UpdateVersionStatus(c.Request.Context(), id, version, req.Status)
	if err != nil {
		writeConfigError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

// CreateVersion handles POST /api/v1/pipeline-configs/:id/versions.
func (h *Handler) CreateVersion(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	existing, ok := h.authorizedConfig(c, id)
	if !ok {
		return
	}
	var req createVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	version, err := h.uc.CreateVersion(c.Request.Context(), existing.ID, uc.CreateVersionInput{
		Status:  req.Status,
		Content: req.Content,
		Summary: req.Summary,
		Author:  middleware.GetUserEmail(c),
	})
	if err != nil {
		writeConfigError(c, err)
		return
	}
	version.Content = ""
	c.JSON(http.StatusCreated, version)
}

// GetVersion handles GET /api/v1/pipeline-configs/:id/versions/:version.
func (h *Handler) GetVersion(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	existing, ok := h.authorizedConfig(c, id)
	if !ok {
		return
	}
	versionNo, err := strconv.Atoi(strings.TrimSpace(c.Param("version")))
	if err != nil || versionNo <= 0 {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "version must be a positive integer", nil)
		return
	}
	version, err := h.uc.GetVersion(c.Request.Context(), existing.ID, versionNo)
	if err != nil {
		writeConfigError(c, err)
		return
	}
	c.JSON(http.StatusOK, version)
}

// Deprecate handles POST /api/v1/pipeline-configs/:id/deprecate.
func (h *Handler) Deprecate(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	existing, ok := h.authorizedConfig(c, id)
	if !ok {
		return
	}
	if err := h.uc.Deprecate(c.Request.Context(), existing.ID); err != nil {
		writeConfigError(c, err)
		return
	}
	cfg, err := h.uc.Get(c.Request.Context(), existing.ID)
	if err != nil {
		writeConfigError(c, err)
		return
	}
	c.JSON(http.StatusOK, cfg)
}

func (h *Handler) authorizedConfig(c *gin.Context, id string) (*models.PipelineConfig, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return nil, false
	}
	cfg, err := h.uc.Get(c.Request.Context(), id)
	if err != nil {
		writeConfigError(c, err)
		return nil, false
	}
	user, admin := currentUser(c)
	if !admin && cfg.Owner != user {
		httpresp.NotFound(c, httpresp.CodeConfigNotFound, "config not found")
		return nil, false
	}
	return cfg, true
}

func currentUser(c *gin.Context) (string, bool) {
	user := middleware.GetUserEmail(c)
	role, _ := c.Get(middleware.CtxKeyRole)
	roleText, _ := role.(string)
	return user, roleText == "admin" || user == "sdk"
}

func writeConfigError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, uc.ErrConfigNotFound):
		httpresp.NotFound(c, httpresp.CodeConfigNotFound, "config not found")
	case errors.Is(err, uc.ErrConfigNameExists):
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
	case errors.Is(err, uc.ErrInvalidConfig):
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
	default:
		httpresp.Internal(c, err.Error())
	}
}
