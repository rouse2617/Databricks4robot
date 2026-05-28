package workflow

import (
	"context"
	"net/http"
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/argo"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
)

type Handler struct {
	wfClient  argo.WorkflowClient
	namespace string
}

func New(wfClient argo.WorkflowClient, namespace string) *Handler {
	return &Handler{wfClient: wfClient, namespace: namespace}
}

// ListWorkflows handles GET /api/v1/workflows
func (h *Handler) ListWorkflows(c *gin.Context) {
	nameFilter := strings.TrimSpace(c.Query("name"))
	statusFilter := strings.TrimSpace(c.Query("status"))
	labelFilters := c.QueryArray("label")
	createdAfterRaw := strings.TrimSpace(c.Query("createdAfter"))
	finishedBeforeRaw := strings.TrimSpace(c.Query("finishedBefore"))

	var (
		createdAfter        time.Time
		createdAfterGiven   bool
		finishedBefore      time.Time
		finishedBeforeGiven bool
	)
	if createdAfterRaw != "" {
		if parsed, err := time.Parse(time.RFC3339, createdAfterRaw); err == nil {
			createdAfter = parsed
			createdAfterGiven = true
		}
	}
	if finishedBeforeRaw != "" {
		if parsed, err := time.Parse(time.RFC3339, finishedBeforeRaw); err == nil {
			finishedBefore = parsed
			finishedBeforeGiven = true
		}
	}

	parsedLabels := make([][2]string, 0, len(labelFilters))
	for _, raw := range labelFilters {
		key, value, found := strings.Cut(strings.TrimSpace(raw), "=")
		if !found || key == "" || value == "" {
			continue
		}
		parsedLabels = append(parsedLabels, [2]string{key, value})
	}

	list, err := h.wfClient.ListWorkflows(c.Request.Context(), h.namespaceFor(c), "")
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}

	filtered := make([]wfv1.Workflow, 0, len(list))
	for _, wf := range list {
		if nameFilter != "" &&
			!strings.Contains(strings.ToLower(wf.Name), strings.ToLower(nameFilter)) {
			continue
		}
		if statusFilter != "" && string(wf.Status.Phase) != statusFilter {
			continue
		}
		labelMatch := true
		for _, item := range parsedLabels {
			v, ok := wf.Labels[item[0]]
			if !ok || v != item[1] {
				labelMatch = false
				break
			}
		}
		if !labelMatch {
			continue
		}
		if createdAfterGiven && wf.CreationTimestamp.Time.Before(createdAfter) {
			continue
		}
		if finishedBeforeGiven {
			if wf.Status.FinishedAt.IsZero() {
				continue
			}
			if wf.Status.FinishedAt.Time.After(finishedBefore) {
				continue
			}
		}

		filtered = append(filtered, wf)
	}

	type item struct {
		Name       string            `json:"name"`
		Status     string            `json:"status"`
		NodeCount  int               `json:"nodeCount"`
		CreatedAt  *string           `json:"createdAt,omitempty"`
		FinishedAt *string           `json:"finishedAt,omitempty"`
		Labels     map[string]string `json:"labels,omitempty"`
	}
	items := make([]item, 0, len(filtered))
	for _, wf := range filtered {
		created := wf.CreationTimestamp.Time.Format("2006-01-02T15:04:05Z")
		it := item{
			Name:      wf.Name,
			Status:    string(wf.Status.Phase),
			NodeCount: len(wf.Status.Nodes),
			CreatedAt: &created,
			Labels:    wf.Labels,
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
	wf, err := h.wfClient.GetWorkflow(c.Request.Context(), name, h.namespaceFor(c))
	if err != nil {
		httpresp.NotFound(c, "WORKFLOW_NOT_FOUND", err.Error())
		return
	}
	type nodeItem struct {
		ID                string   `json:"id"`
		Name              string   `json:"name"`
		DisplayName       string   `json:"displayName"`
		Type              string   `json:"type"`
		TemplateName      string   `json:"templateName"`
		Phase             string   `json:"phase"`
		Message           string   `json:"message,omitempty"`
		Inputs            any      `json:"inputs,omitempty"`
		Outputs           any      `json:"outputs,omitempty"`
		ResourcesDuration any      `json:"resourcesDuration,omitempty"`
		HostNodeName      string   `json:"hostNodeName,omitempty"`
		Progress          string   `json:"progress,omitempty"`
		EstimatedDuration int64    `json:"estimatedDuration,omitempty"`
		Children          []string `json:"children,omitempty"`
		StartedAt         *string  `json:"startedAt,omitempty"`
		FinishedAt        *string  `json:"finishedAt,omitempty"`
	}
	nodes := make([]nodeItem, 0, len(wf.Status.Nodes))
	for _, n := range wf.Status.Nodes {
		ni := nodeItem{
			ID:                n.ID,
			Name:              n.Name,
			DisplayName:       n.DisplayName,
			Type:              string(n.Type),
			TemplateName:      n.TemplateName,
			Phase:             string(n.Phase),
			Message:           n.Message,
			Inputs:            n.Inputs,
			Outputs:           n.Outputs,
			ResourcesDuration: n.ResourcesDuration,
			HostNodeName:      n.HostNodeName,
			Progress:          string(n.Progress),
			EstimatedDuration: int64(n.EstimatedDuration),
			Children:          n.Children,
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
		"name":              wf.Name,
		"status":            string(wf.Status.Phase),
		"message":           wf.Status.Message,
		"nodes":             nodes,
		"createdAt":         created,
		"labels":            wf.Labels,
		"estimatedDuration": int64(wf.Status.EstimatedDuration),
		"progress":          string(wf.Status.Progress),
	}
	if !wf.Status.FinishedAt.IsZero() {
		t := wf.Status.FinishedAt.Time.Format("2006-01-02T15:04:05Z")
		resp["finishedAt"] = t
	}
	c.JSON(200, resp)
}

// GetWorkflowLogs handles GET /api/v1/workflows/:name/logs?nodeId=xxx
func (h *Handler) GetWorkflowLogs(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	nodeId := strings.TrimSpace(c.Query("nodeId"))
	if name == "" || nodeId == "" {
		httpresp.BadRequest(c, "INVALID_ARGUMENT", "workflow name and nodeId are required", nil)
		return
	}
	logs, err := h.wfClient.GetWorkflowLogs(c.Request.Context(), name, nodeId, h.namespaceFor(c))
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, gin.H{"logs": logs})
}

// RetryWorkflow handles POST /api/v1/workflows/:name/retry
func (h *Handler) RetryWorkflow(c *gin.Context) {
	h.workflowOperation(c, h.wfClient.RetryWorkflow)
}

// ResubmitWorkflow handles POST /api/v1/workflows/:name/resubmit
func (h *Handler) ResubmitWorkflow(c *gin.Context) {
	h.workflowOperation(c, h.wfClient.ResubmitWorkflow)
}

// SuspendWorkflow handles POST /api/v1/workflows/:name/suspend
func (h *Handler) SuspendWorkflow(c *gin.Context) {
	h.workflowOperation(c, h.wfClient.SuspendWorkflow)
}

// StopWorkflow handles POST /api/v1/workflows/:name/stop
func (h *Handler) StopWorkflow(c *gin.Context) {
	h.workflowOperation(c, h.wfClient.StopWorkflow)
}

// ResumeWorkflow handles POST /api/v1/workflows/:name/resume
func (h *Handler) ResumeWorkflow(c *gin.Context) {
	h.workflowOperation(c, h.wfClient.ResumeWorkflow)
}

// TerminateWorkflow handles POST /api/v1/workflows/:name/terminate
func (h *Handler) TerminateWorkflow(c *gin.Context) {
	h.workflowOperation(c, h.wfClient.TerminateWorkflow)
}

// DeleteWorkflow handles DELETE /api/v1/workflows/:name
func (h *Handler) DeleteWorkflow(c *gin.Context) {
	h.workflowOperation(c, h.wfClient.DeleteWorkflow)
}

func (h *Handler) workflowOperation(c *gin.Context, fn func(context.Context, string, string) error) {
	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		httpresp.BadRequest(c, "INVALID_ARGUMENT", "name is required", nil)
		return
	}
	if err := fn(c.Request.Context(), name, h.namespaceFor(c)); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (h *Handler) namespaceFor(c *gin.Context) string {
	if namespace := strings.TrimSpace(c.GetString("namespace")); namespace != "" {
		return namespace
	}
	return h.namespace
}
