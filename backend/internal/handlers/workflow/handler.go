package workflow

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/k8s"
)

type Handler struct {
	wfClient  k8s.WorkflowClient
	namespace string
}

func New(wfClient k8s.WorkflowClient, namespace string) *Handler {
	return &Handler{wfClient: wfClient, namespace: namespace}
}

// ListWorkflows handles GET /api/v1/workflows
func (h *Handler) ListWorkflows(c *gin.Context) {
	list, err := h.wfClient.ListWorkflows(c.Request.Context(), h.namespace, "")
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	type item struct {
		Name       string  `json:"name"`
		Status     string  `json:"status"`
		NodeCount  int     `json:"nodeCount"`
		CreatedAt  *string `json:"createdAt,omitempty"`
		FinishedAt *string `json:"finishedAt,omitempty"`
	}
	items := make([]item, 0, len(list))
	for _, wf := range list {
		created := wf.CreationTimestamp.Time.Format("2006-01-02T15:04:05Z")
		it := item{
			Name:      wf.Name,
			Status:    string(wf.Status.Phase),
			NodeCount: len(wf.Status.Nodes),
			CreatedAt: &created,
		}
		if wf.Status.FinishedAt.IsZero() {
			it.FinishedAt = nil
		} else {
			t := wf.Status.FinishedAt.Time.Format("2006-01-02T15:04:05Z")
			it.FinishedAt = &t
		}
		items = append(items, it)
	}
	c.JSON(200, gin.H{"items": items})
}

// GetWorkflow handles GET /api/v1/workflows/:name
func (h *Handler) GetWorkflow(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		httpresp.BadRequest(c, "INVALID_ARGUMENT", "name is required", nil)
		return
	}
	wf, err := h.wfClient.GetWorkflow(c.Request.Context(), name, h.namespace)
	if err != nil {
		httpresp.NotFound(c, "WORKFLOW_NOT_FOUND", err.Error())
		return
	}
	type nodeItem struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		DisplayName string  `json:"displayName"`
		Phase       string  `json:"phase"`
		Message     string  `json:"message,omitempty"`
		StartedAt   *string `json:"startedAt,omitempty"`
		FinishedAt  *string `json:"finishedAt,omitempty"`
	}
	nodes := make([]nodeItem, 0, len(wf.Status.Nodes))
	for _, n := range wf.Status.Nodes {
		ni := nodeItem{
			ID:          n.ID,
			Name:        n.TemplateName,
			DisplayName: n.DisplayName,
			Phase:       string(n.Phase),
			Message:     n.Message,
		}
		if !n.StartedAt.IsZero() {
			t := n.StartedAt.Time.Format("2006-01-02T15:04:05Z")
			ni.StartedAt = &t
		}
		if !n.FinishedAt.IsZero() {
			t := n.FinishedAt.Time.Format("2006-01-02T15:04:05Z")
			ni.FinishedAt = &t
		}
		nodes = append(nodes, ni)
	}
	created := wf.CreationTimestamp.Time.Format("2006-01-02T15:04:05Z")
	resp := gin.H{
		"name":      wf.Name,
		"status":    string(wf.Status.Phase),
		"message":   wf.Status.Message,
		"nodes":     nodes,
		"createdAt": created,
	}
	if !wf.Status.FinishedAt.IsZero() {
		t := wf.Status.FinishedAt.Time.Format("2006-01-02T15:04:05Z")
		resp["finishedAt"] = t
	}
	c.JSON(200, resp)
}
