package workflow

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/argo"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/k8s"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

const (
	defaultWorkflowLogTailLines  int64 = 200
	maxWorkflowLogTailLines      int64 = 2000
	defaultWorkflowLogLimitBytes int64 = 262144
	maxWorkflowLogLimitBytes     int64 = 2097152
	workflowLogSourceArgoLive          = "argo-live"
)

type Handler struct {
	wfClient        argo.WorkflowClient
	podClient       k8s.PodClient
	execClient      k8s.ExecClient
	namespace       string
	runRepo         repository.PipelineRunRepository
	runEventRepo    repository.PipelineRunEventRepository
	terminalStore   *terminalSessionStore
	terminalNowFunc func() time.Time
}

type workflowNodeItem struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	DisplayName       string   `json:"displayName"`
	Type              string   `json:"type"`
	TemplateName      string   `json:"templateName"`
	Phase             string   `json:"phase"`
	Message           string   `json:"message,omitempty"`
	PodName           string   `json:"podName,omitempty"`
	Inputs            any      `json:"inputs,omitempty"`
	Outputs           any      `json:"outputs,omitempty"`
	ResourcesDuration any      `json:"resourcesDuration,omitempty"`
	HostNodeName      string   `json:"hostNodeName,omitempty"`
	Progress          string   `json:"progress,omitempty"`
	EstimatedDuration int64    `json:"estimatedDuration,omitempty"`
	Children          []string `json:"children,omitempty"`
	StartedAt         *string  `json:"startedAt,omitempty"`
	FinishedAt        *string  `json:"finishedAt,omitempty"`
	Debug             any      `json:"debug,omitempty"`
}

func New(wfClient argo.WorkflowClient, namespace string) *Handler {
	return &Handler{
		wfClient:        wfClient,
		namespace:       namespace,
		terminalStore:   newTerminalSessionStore(),
		terminalNowFunc: time.Now,
	}
}

func (h *Handler) SetPodClient(podClient k8s.PodClient) {
	h.podClient = podClient
}

func (h *Handler) SetExecClient(execClient k8s.ExecClient) {
	h.execClient = execClient
}

func (h *Handler) SetRunRepositories(runRepo repository.PipelineRunRepository, eventRepo repository.PipelineRunEventRepository) {
	h.runRepo = runRepo
	h.runEventRepo = eventRepo
}

// ListWorkflows handles GET /api/v1/workflows
func (h *Handler) ListWorkflows(c *gin.Context) {
	nameFilter := strings.TrimSpace(c.Query("name"))
	statusFilter := strings.TrimSpace(c.Query("status"))
	labelFilters := c.QueryArray("label")
	createdAfter, createdAfterGiven, ok := parseRFC3339QueryParam(c, "createdAfter")
	if !ok {
		return
	}
	finishedBefore, finishedBeforeGiven, ok := parseRFC3339QueryParam(c, "finishedBefore")
	if !ok {
		return
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
		created := workflowTimeString(wf.CreationTimestamp.Time)
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
			t := workflowTimeString(wf.Status.FinishedAt.Time)
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
		if errors.Is(err, argo.ErrNotFound) {
			httpresp.NotFound(c, "WORKFLOW_NOT_FOUND", err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	run, _ := h.findPipelineRunByWorkflow(c.Request.Context(), wf.Name)
	nodes := buildWorkflowDetailNodes(h, run, wf)
	created := workflowTimeString(wf.CreationTimestamp.Time)
	resp := gin.H{
		"name":              wf.Name,
		"status":            string(wf.Status.Phase),
		"message":           wf.Status.Message,
		"nodes":             nodes,
		"edges":             buildWorkflowDagEdges(wf),
		"createdAt":         created,
		"labels":            wf.Labels,
		"estimatedDuration": int64(wf.Status.EstimatedDuration),
		"progress":          string(wf.Status.Progress),
	}
	if !wf.Status.FinishedAt.IsZero() {
		t := workflowTimeString(wf.Status.FinishedAt.Time)
		resp["finishedAt"] = t
	}
	c.JSON(200, resp)
}

func buildWorkflowDetailNodes(h *Handler, run *models.PipelineRun, wf *wfv1.Workflow) []workflowNodeItem {
	if wf == nil {
		return nil
	}
	templatesByName := make(map[string]*wfv1.Template, len(wf.Spec.Templates))
	for i := range wf.Spec.Templates {
		tmpl := &wf.Spec.Templates[i]
		templatesByName[tmpl.Name] = tmpl
	}
	nodes := make([]workflowNodeItem, 0, len(wf.Status.Nodes))
	seenTaskNames := make(map[string]struct{})
	for _, n := range wf.Status.Nodes {
		ni := workflowNodeItem{
			ID:                n.ID,
			Name:              n.Name,
			DisplayName:       n.DisplayName,
			Type:              string(n.Type),
			TemplateName:      n.TemplateName,
			Phase:             string(n.Phase),
			Message:           n.Message,
			PodName:           "",
			Inputs:            n.Inputs,
			Outputs:           n.Outputs,
			ResourcesDuration: n.ResourcesDuration,
			HostNodeName:      n.HostNodeName,
			Progress:          string(n.Progress),
			EstimatedDuration: int64(n.EstimatedDuration),
			Children:          n.Children,
		}
		if !n.StartedAt.IsZero() {
			t := workflowTimeString(n.StartedAt.Time)
			ni.StartedAt = &t
		}
		if !n.FinishedAt.IsZero() {
			t := workflowTimeString(n.FinishedAt.Time)
			ni.FinishedAt = &t
		}
		if podName, ok := resolveWorkflowPodName(wf, n.ID); ok {
			ni.PodName = podName
		}
		if h != nil {
			ni.Debug = h.terminalCapabilityForNode(run, wf, n, ni.PodName)
		}
		nodes = append(nodes, ni)
		for _, taskName := range workflowNodeTaskNameCandidates(n) {
			seenTaskNames[taskName] = struct{}{}
		}
	}
	for _, task := range workflowStaticDAGTasks(wf) {
		taskName := strings.TrimSpace(task.Name)
		if taskName == "" {
			continue
		}
		if _, ok := seenTaskNames[taskName]; ok {
			continue
		}
		nodes = append(nodes, workflowStaticTaskNodeItem(task, templatesByName))
		seenTaskNames[taskName] = struct{}{}
	}
	return nodes
}

func workflowStaticTaskNodeItem(task wfv1.DAGTask, templatesByName map[string]*wfv1.Template) workflowNodeItem {
	taskName := strings.TrimSpace(task.Name)
	templateName := strings.TrimSpace(task.Template)
	nodeType := string(wfv1.NodeTypePod)
	if tmpl := templatesByName[templateName]; tmpl != nil {
		nodeType = workflowTemplateDisplayType(*tmpl)
	}
	return workflowNodeItem{
		ID:           taskName,
		Name:         taskName,
		DisplayName:  taskName,
		Type:         nodeType,
		TemplateName: templateName,
		Phase:        string(wfv1.NodePending),
		Inputs:       task.Arguments,
	}
}

func workflowTemplateDisplayType(t wfv1.Template) string {
	switch {
	case t.DAG != nil:
		return string(wfv1.NodeTypeDAG)
	case t.Steps != nil:
		return string(wfv1.NodeTypeSteps)
	default:
		return string(wfv1.NodeTypePod)
	}
}

func workflowTimeString(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// GetWorkflowLogs handles GET /api/v1/workflows/:name/logs?nodeId=xxx
func (h *Handler) GetWorkflowLogs(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	nodeId := strings.TrimSpace(c.Query("nodeId"))
	if name == "" || nodeId == "" {
		httpresp.BadRequest(c, "INVALID_ARGUMENT", "workflow name and nodeId are required", nil)
		return
	}
	namespace := h.namespaceFor(c)
	workflow, err := h.wfClient.GetWorkflow(c.Request.Context(), name, namespace)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	podName, ok := resolveWorkflowPodName(workflow, nodeId)
	if !ok {
		httpresp.BadRequest(c, "INVALID_ARGUMENT", "workflow pod node not found", nil)
		return
	}
	opts, meta, ok := parseWorkflowLogOptions(c)
	if !ok {
		return
	}

	result, err := h.wfClient.GetWorkflowLogs(c.Request.Context(), name, podName, namespace, opts)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	meta["bytesTruncated"] = result.Truncated
	c.JSON(200, gin.H{
		"workflowName": name,
		"nodeId":       nodeId,
		"podName":      podName,
		"container":    opts.Container,
		"source":       workflowLogSourceArgoLive,
		"logs":         result.Logs,
		"lineCount":    result.LineCount,
		"truncated":    result.Truncated,
		"nextCursor":   nil,
		"truncation":   meta,
	})
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

func parseWorkflowLogOptions(c *gin.Context) (argo.WorkflowLogOptions, gin.H, bool) {
	tailLines, tailClamped, ok := parseBoundedInt64Query(
		c,
		"tailLines",
		defaultWorkflowLogTailLines,
		maxWorkflowLogTailLines,
		false,
	)
	if !ok {
		return argo.WorkflowLogOptions{}, nil, false
	}
	limitBytes, limitClamped, ok := parseBoundedInt64Query(
		c,
		"limitBytes",
		defaultWorkflowLogLimitBytes,
		maxWorkflowLogLimitBytes,
		false,
	)
	if !ok {
		return argo.WorkflowLogOptions{}, nil, false
	}
	opts := argo.WorkflowLogOptions{
		Container:  strings.TrimSpace(c.DefaultQuery("container", defaultWorkflowLogContainer)),
		TailLines:  &tailLines,
		LimitBytes: &limitBytes,
	}
	if opts.Container == "" {
		opts.Container = defaultWorkflowLogContainer
	}
	if raw := strings.TrimSpace(c.Query("sinceSeconds")); raw != "" {
		seconds, _, parsed := parseBoundedInt64Query(c, "sinceSeconds", 0, 0, true)
		if !parsed {
			return argo.WorkflowLogOptions{}, nil, false
		}
		opts.SinceSeconds = &seconds
	}
	if raw := strings.TrimSpace(c.Query("sinceTime")); raw != "" {
		if _, err := time.Parse(time.RFC3339, raw); err != nil {
			httpresp.BadRequest(c, "INVALID_ARGUMENT", "sinceTime must be RFC3339", nil)
			return argo.WorkflowLogOptions{}, nil, false
		}
		opts.SinceTime = raw
	}
	if raw := strings.TrimSpace(c.Query("previous")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			httpresp.BadRequest(c, "INVALID_ARGUMENT", "previous must be a boolean", nil)
			return argo.WorkflowLogOptions{}, nil, false
		}
		opts.Previous = value
	}
	if raw := strings.TrimSpace(c.Query("timestamps")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			httpresp.BadRequest(c, "INVALID_ARGUMENT", "timestamps must be a boolean", nil)
			return argo.WorkflowLogOptions{}, nil, false
		}
		opts.Timestamps = value
	}
	meta := gin.H{
		"bounded":           true,
		"tailLines":         tailLines,
		"maxTailLines":      maxWorkflowLogTailLines,
		"tailLinesClamped":  tailClamped,
		"limitBytes":        limitBytes,
		"maxLimitBytes":     maxWorkflowLogLimitBytes,
		"limitBytesClamped": limitClamped,
	}
	return opts, meta, true
}

func parseBoundedInt64Query(
	c *gin.Context,
	name string,
	defaultValue int64,
	maxValue int64,
	allowNoMax bool,
) (int64, bool, bool) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return defaultValue, false, true
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		httpresp.BadRequest(c, "INVALID_ARGUMENT", name+" must be a non-negative integer", nil)
		return 0, false, false
	}
	if !allowNoMax && value > maxValue {
		return maxValue, true, true
	}
	return value, false, true
}
