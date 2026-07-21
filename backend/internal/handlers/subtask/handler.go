package subtask

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/middleware"
	st "github.com/CyberOrigin2077/cyber-databrew/internal/subtask"
)

type Repo interface {
	List(ctx context.Context) ([]st.Task, error)
	Get(ctx context.Context, id string) (*st.Task, error)
	Create(ctx context.Context, t *st.Task) error
	Update(ctx context.Context, t *st.Task) error
	Delete(ctx context.Context, id string) error
	SetEnabled(ctx context.Context, id string, enabled bool) error
}

type Handler struct {
	repo Repo
}

func New(repo Repo) *Handler { return &Handler{repo: repo} }

type taskRequest struct {
	Name               string `json:"name"`
	Enabled            *bool  `json:"enabled,omitempty"`
	TemplateID         string `json:"templateId"`
	TemplateVersion    *int   `json:"templateVersion,omitempty"`
	TargetID           string `json:"targetId"`
	ProjectID          string `json:"projectId"`
	SubscriptionID     string `json:"subscriptionId"`
	PullIntervalSec    *int   `json:"pullIntervalSeconds,omitempty"`
	MaxMessagesPerPull *int   `json:"maxMessagesPerPull,omitempty"`
}

func (h *Handler) List(c *gin.Context) {
	tasks, err := h.repo.List(c.Request.Context())
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	q := strings.ToLower(strings.TrimSpace(c.Query("q")))
	enabled := c.Query("enabled")
	filtered := make([]st.Task, 0, len(tasks))
	for _, t := range tasks {
		if q != "" && !strings.Contains(strings.ToLower(t.Name), q) && !strings.Contains(strings.ToLower(t.ID), q) {
			continue
		}
		if enabled == "true" && !t.Enabled {
			continue
		}
		if enabled == "false" && t.Enabled {
			continue
		}
		filtered = append(filtered, t)
	}
	page, pageSize := parsePaging(c)
	total := len(filtered)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	c.JSON(http.StatusOK, gin.H{
		"items":    filtered[start:end],
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handler) Get(c *gin.Context) {
	id := c.Param("id")
	t, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if t == nil {
		httpresp.NotFound(c, httpresp.CodeSubscriptionTaskNotFound, "subscription task not found")
		return
	}
	c.JSON(http.StatusOK, t)
}

func (h *Handler) Create(c *gin.Context) {
	var req taskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	if err := validateRequest(req, true); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		return
	}
	task := requestToTask(req, "")
	task.ID = "sub_" + uuid.New().String()
	task.CreatedBy = middleware.GetUserEmail(c)
	task.CreatedAt = time.Now().UTC()
	task.UpdatedAt = task.CreatedAt
	if err := h.repo.Create(c.Request.Context(), &task); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(http.StatusCreated, task)
}

func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")
	existing, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if existing == nil {
		httpresp.NotFound(c, httpresp.CodeSubscriptionTaskNotFound, "subscription task not found")
		return
	}
	var req taskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	if err := validateRequest(req, false); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		return
	}
	task := requestToTask(req, id)
	task.CreatedBy = existing.CreatedBy
	task.CreatedAt = existing.CreatedAt
	if req.Enabled == nil {
		task.Enabled = existing.Enabled
	}
	if err := h.repo.Update(c.Request.Context(), &task); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	fresh, _ := h.repo.Get(c.Request.Context(), id)
	if fresh != nil {
		c.JSON(http.StatusOK, fresh)
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Pause(c *gin.Context) {
	h.setEnabled(c, false)
}

func (h *Handler) Resume(c *gin.Context) {
	h.setEnabled(c, true)
}

func (h *Handler) setEnabled(c *gin.Context, enabled bool) {
	id := c.Param("id")
	if err := h.repo.SetEnabled(c.Request.Context(), id, enabled); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	t, _ := h.repo.Get(c.Request.Context(), id)
	if t == nil {
		httpresp.NotFound(c, httpresp.CodeSubscriptionTaskNotFound, "subscription task not found")
		return
	}
	c.JSON(http.StatusOK, t)
}

func parsePaging(c *gin.Context) (page, pageSize int) {
	page = 1
	pageSize = 50
	if v := c.Query("page"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &page); err != nil || page < 1 {
			page = 1
		}
	}
	if v := c.Query("pageSize"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &pageSize); err != nil || pageSize < 1 {
			pageSize = 50
		}
	}
	if pageSize > 500 {
		pageSize = 500
	}
	return
}

func validateRequest(req taskRequest, forCreate bool) error {
	if forCreate && strings.TrimSpace(req.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(req.TemplateID) == "" {
		return errors.New("templateId is required")
	}
	if strings.TrimSpace(req.TargetID) == "" {
		return errors.New("targetId is required")
	}
	if strings.TrimSpace(req.ProjectID) == "" {
		return errors.New("projectId is required")
	}
	if strings.TrimSpace(req.SubscriptionID) == "" {
		return errors.New("subscriptionId is required")
	}
	if req.PullIntervalSec != nil && *req.PullIntervalSec <= 0 {
		return errors.New("pullIntervalSeconds must be positive")
	}
	if req.MaxMessagesPerPull != nil && (*req.MaxMessagesPerPull <= 0 || *req.MaxMessagesPerPull > 10000) {
		return errors.New("maxMessagesPerPull must be 1-10000")
	}
	return nil
}

func requestToTask(req taskRequest, id string) st.Task {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	pullInterval := 10
	if req.PullIntervalSec != nil {
		pullInterval = *req.PullIntervalSec
	}
	maxMsg := 1000
	if req.MaxMessagesPerPull != nil {
		maxMsg = *req.MaxMessagesPerPull
	}
	return st.Task{
		ID:                 id,
		Name:               strings.TrimSpace(req.Name),
		Enabled:            enabled,
		TemplateID:         strings.TrimSpace(req.TemplateID),
		TemplateVersion:    req.TemplateVersion,
		TargetID:           strings.TrimSpace(req.TargetID),
		ProjectID:          strings.TrimSpace(req.ProjectID),
		SubscriptionID:     strings.TrimSpace(req.SubscriptionID),
		PullIntervalSec:    pullInterval,
		MaxMessagesPerPull: maxMsg,
	}
}
