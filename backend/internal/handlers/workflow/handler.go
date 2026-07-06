package workflow

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/CyberOrigin2077/cyber-databrew/internal/argo"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/k8s"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
)

const (
	defaultWorkflowLogTailLines  int64 = 200
	maxWorkflowLogTailLines      int64 = 2000
	defaultWorkflowLogLimitBytes int64 = 262144
	maxWorkflowLogLimitBytes     int64 = 2097152
	workflowLogSourceArgoLive          = "argo-live"
)

const workflowLogPaginationUnavailableReason = "live Argo logs do not provide stable historical cursor pagination"

type Handler struct {
	wfClient        argo.WorkflowClient
	podClient       k8s.PodClient
	execClient      k8s.ExecClient
	namespace       string
	runRepo         repository.PipelineRunRepository
	runEventRepo    repository.PipelineRunEventRepository
	runNodeRepo     repository.PipelineRunNodeRepository
	terminalStore   *terminalSessionStore
	terminalNowFunc func() time.Time
	sseRingBuffers  *ringBufferStore
	archiveStore    ArchiveLogStore
}

type workflowNodeItem struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	DisplayName       string   `json:"displayName"`
	Type              string   `json:"type"`
	TemplateName      string   `json:"templateName"`
	VersionLabel      string   `json:"versionLabel,omitempty"`
	SourceCommit      string   `json:"sourceCommit,omitempty"`
	Image             string   `json:"image,omitempty"`
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
		sseRingBuffers:  newRingBufferStore(),
		archiveStore:    NewNoopArchiveStore(),
	}
}

func (h *Handler) SetPodClient(podClient k8s.PodClient) {
	h.podClient = podClient
}

func (h *Handler) SetExecClient(execClient k8s.ExecClient) {
	h.execClient = execClient
}

func (h *Handler) SetArchiveStore(store ArchiveLogStore) {
	h.archiveStore = store
}

func (h *Handler) SetRunRepositories(runRepo repository.PipelineRunRepository, eventRepo repository.PipelineRunEventRepository, runNodeRepo repository.PipelineRunNodeRepository) {
	h.runRepo = runRepo
	h.runEventRepo = eventRepo
	h.runNodeRepo = runNodeRepo
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
		Message    string            `json:"message,omitempty"`
		NodeCount  int               `json:"nodeCount"`
		CreatedAt  *string           `json:"createdAt,omitempty"`
		StartedAt  *string           `json:"startedAt,omitempty"`
		FinishedAt *string           `json:"finishedAt,omitempty"`
		Labels     map[string]string `json:"labels,omitempty"`
	}
	items := make([]item, 0, len(filtered))
	for _, wf := range filtered {
		created := workflowTimeString(wf.CreationTimestamp.Time)
		it := item{
			Name:      wf.Name,
			Status:    string(wf.Status.Phase),
			Message:   strings.TrimSpace(wf.Status.Message),
			NodeCount: len(wf.Status.Nodes),
			CreatedAt: &created,
			Labels:    wf.Labels,
		}
		if !wf.Status.StartedAt.IsZero() {
			s := workflowTimeString(wf.Status.StartedAt.Time)
			it.StartedAt = &s
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
	ctx := c.Request.Context()

	// CYB-3063: for a run that has already reached a terminal state, avoid
	// querying the live Argo API — reconstruct an equivalent response from
	// already-persisted data instead. Active runs, runs with no matching
	// record, and reconstruction failures all fall through to the original
	// direct-Argo path unchanged.
	run, _ := h.findPipelineRunByWorkflow(ctx, name)
	if run != nil && !pipelineUC.IsActiveDeploymentStatus(run.Status) {
		if wf, runNodes, ok := h.reconstructWorkflowFromDB(ctx, run); ok {
			h.respondWorkflowDetail(c, run, wf, runNodes)
			return
		}
	}

	namespace := h.namespaceForWorkflow(ctx, c, name)
	wf, err := h.wfClient.GetWorkflow(ctx, name, namespace)
	if err != nil {
		if errors.Is(err, argo.ErrNotFound) {
			httpresp.NotFound(c, "WORKFLOW_NOT_FOUND", err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	if run == nil {
		run, _ = h.findPipelineRunByWorkflow(ctx, wf.Name)
	}
	h.respondWorkflowDetail(c, run, wf, nil)
}

// respondWorkflowDetail renders the GetWorkflow response from a *wfv1.Workflow,
// whether it came from a live Argo query or reconstructWorkflowFromDB.
// runNodesForBackfill, when non-nil, supplies Inputs/Outputs/ResourcesDuration
// for the DB-reconstructed path (see backfillNodeItemDataFields).
func (h *Handler) respondWorkflowDetail(c *gin.Context, run *models.PipelineRun, wf *wfv1.Workflow, runNodesForBackfill []models.PipelineRunNode) {
	nodes := buildWorkflowDetailNodes(h, run, wf)
	if runNodesForBackfill != nil {
		nodes = backfillNodeItemDataFields(nodes, runNodesForBackfill)
	}
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
	if !wf.Status.StartedAt.IsZero() {
		resp["startedAt"] = workflowTimeString(wf.Status.StartedAt.Time)
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
		if info := lookupWorkflowNodeRuntimeInfo(run, n.TemplateName); info != nil {
			ni.VersionLabel = info.VersionLabel
			ni.SourceCommit = info.SourceCommit
			ni.Image = info.Image
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

type workflowNodeRuntimeMetadata struct {
	VersionLabel string
	SourceCommit string
	Image        string
}

func lookupWorkflowNodeRuntimeInfo(run *models.PipelineRun, templateName string) *workflowNodeRuntimeMetadata {
	if run == nil || len(run.PipelineJSON) == 0 {
		return nil
	}
	nodeID := strings.TrimSpace(templateName)
	nodeID = strings.TrimPrefix(nodeID, "step-")
	if nodeID == "" {
		return nil
	}
	if info, ok := findWorkflowNodeRuntimeInfo(run.PipelineJSON["nodes"], nodeID); ok {
		return &info
	}
	return nil
}

func findWorkflowNodeRuntimeInfo(rawNodes any, targetID string) (workflowNodeRuntimeMetadata, bool) {
	nodes, ok := interfaceSlice(rawNodes)
	if !ok {
		return workflowNodeRuntimeMetadata{}, false
	}
	for _, raw := range nodes {
		node, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		nodeID, _ := node["id"].(string)
		if strings.TrimSpace(nodeID) == targetID {
			return workflowNodeRuntimeMetadata{
				VersionLabel: firstWorkflowNodeString(node, "componentVersionLabel", "versionLabel", "releaseLabel", "tag"),
				SourceCommit: firstWorkflowNodeString(node, "sourceCommit", "commit"),
				Image:        firstWorkflowNodeString(node, "image"),
			}, true
		}
		if info, ok := findWorkflowNodeRuntimeInfo(node["sub_nodes"], targetID); ok {
			return info, true
		}
	}
	return workflowNodeRuntimeMetadata{}, false
}

func firstWorkflowNodeString(node map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := node[key]; ok {
			if text := strings.TrimSpace(asWorkflowNodeString(value)); text != "" {
				return text
			}
		}
	}
	if component, ok := node["component"].(map[string]any); ok {
		for _, key := range keys {
			if value, ok := component[key]; ok {
				if text := strings.TrimSpace(asWorkflowNodeString(value)); text != "" {
					return text
				}
			}
		}
	}
	return ""
}

func asWorkflowNodeString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return ""
	}
}

func interfaceSlice(in any) ([]any, bool) {
	switch v := in.(type) {
	case []any:
		return v, true
	default:
		return nil, false
	}
}

// GetWorkflowLogs handles GET /api/v1/workflows/:name/logs?nodeId=xxx
func (h *Handler) GetWorkflowLogs(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	nodeId := strings.TrimSpace(c.Query("nodeId"))
	if name == "" || nodeId == "" {
		httpresp.BadRequest(c, "INVALID_ARGUMENT", "workflow name and nodeId are required", nil)
		return
	}
	namespace := h.namespaceForWorkflow(c.Request.Context(), c, name)
	workflow, err := h.wfClient.GetWorkflow(c.Request.Context(), name, namespace)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	podName, ok := resolveCachedWorkflowPodName(workflow, nodeId)
	if !ok {
		// Try archive fallback when live pod is unavailable (e.g. pod GC'd)
		result, archiveErr := h.archiveStore.GetLogs(c.Request.Context(), name, nodeId, "", argo.WorkflowLogOptions{})
		if archiveErr != nil {
			httpresp.Internal(c, archiveErr.Error())
			return
		}
		if result != nil {
			c.JSON(200, gin.H{
				"workflowName": name,
				"nodeId":       nodeId,
				"podName":      "",
				"container":    "",
				"source":       result.Source,
				"logs":         result.Logs,
				"lineCount":    result.LineCount,
				"truncated":    false,
				"truncation": gin.H{
					"bounded": true,
				},
				"pagination": gin.H{
					"available": false,
					"reason":    "archive logs do not support cursor pagination",
				},
				"window": gin.H{
					"mode":  "archive",
					"scope": result.Source,
				},
			})
			return
		}
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
	pagination := gin.H{
		"available":  false,
		"nextCursor": nil,
		"reason":     workflowLogPaginationUnavailableReason,
	}
	window := gin.H{
		"mode":         "tail",
		"tailLines":    meta["tailLines"],
		"limitBytes":   meta["limitBytes"],
		"sinceSeconds": meta["sinceSeconds"],
		"sinceTime":    meta["sinceTime"],
		"previous":     opts.Previous,
		"timestamps":   opts.Timestamps,
		"scope":        "bounded-live-window",
	}
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
		"pagination":   pagination,
		"window":       window,
	})
}

// RetryWorkflow handles POST /api/v1/workflows/:name/retry
func (h *Handler) RetryWorkflow(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		httpresp.BadRequest(c, "INVALID_ARGUMENT", "name is required", nil)
		return
	}
	namespace := h.namespaceForWorkflow(c.Request.Context(), c, name)
	wf, err := h.wfClient.GetWorkflow(c.Request.Context(), name, namespace)
	if err != nil {
		if errors.Is(err, argo.ErrNotFound) {
			httpresp.NotFound(c, "WORKFLOW_NOT_FOUND", err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	if !workflowCanRetry(wf) {
		httpresp.Conflict(c, "WORKFLOW_NOT_RETRYABLE", workflowRetryBlockedMessage(wf), gin.H{
			"hint": "use POST /workflows/:name/resubmit to start a new run from the beginning",
		})
		return
	}
	h.workflowOperation(c, h.wfClient.RetryWorkflow)
}

// ResubmitWorkflow handles POST /api/v1/workflows/:name/resubmit
func (h *Handler) ResubmitWorkflow(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		httpresp.BadRequest(c, "INVALID_ARGUMENT", "name is required", nil)
		return
	}

	namespace := h.namespaceForWorkflow(c.Request.Context(), c, name)
	var newWorkflow *wfv1.Workflow
	var err error
	if client, ok := h.wfClient.(argo.WorkflowResubmitResultClient); ok {
		newWorkflow, err = client.ResubmitWorkflowWithResult(c.Request.Context(), name, namespace)
	} else {
		err = h.wfClient.ResubmitWorkflow(c.Request.Context(), name, namespace)
	}
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}

	run, err := h.syncResubmittedPipelineRun(c.Request.Context(), name, namespace, newWorkflow)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}

	resp := gin.H{"message": "ok"}
	if newWorkflow != nil && strings.TrimSpace(newWorkflow.Name) != "" {
		resp["workflowName"] = newWorkflow.Name
	}
	if run != nil {
		resp["pipelineRunId"] = run.ID
	}
	c.JSON(http.StatusOK, resp)
}

// SuspendWorkflow handles POST /api/v1/workflows/:name/suspend
func (h *Handler) SuspendWorkflow(c *gin.Context) {
	h.workflowOperation(c, h.wfClient.SuspendWorkflow)
}

// StopWorkflow handles POST /api/v1/workflows/:name/stop
func (h *Handler) StopWorkflow(c *gin.Context) {
	if h.workflowOperation(c, h.wfClient.StopWorkflow) {
		h.syncWorkflowOperationPipelineRun(c.Request.Context(), strings.TrimSpace(c.Param("name")), string(wfv1.WorkflowFailed), "workflow shutdown with strategy: Stop")
	}
}

// ResumeWorkflow handles POST /api/v1/workflows/:name/resume
func (h *Handler) ResumeWorkflow(c *gin.Context) {
	h.workflowOperation(c, h.wfClient.ResumeWorkflow)
}

// TerminateWorkflow handles POST /api/v1/workflows/:name/terminate
func (h *Handler) TerminateWorkflow(c *gin.Context) {
	if h.workflowOperation(c, h.wfClient.TerminateWorkflow) {
		h.syncWorkflowOperationPipelineRun(c.Request.Context(), strings.TrimSpace(c.Param("name")), string(wfv1.WorkflowFailed), "Stopped with strategy 'Terminate'")
	}
}

// DeleteWorkflow handles DELETE /api/v1/workflows/:name
func (h *Handler) DeleteWorkflow(c *gin.Context) {
	h.workflowOperation(c, h.wfClient.DeleteWorkflow)
}

func (h *Handler) workflowOperation(c *gin.Context, fn func(context.Context, string, string) error) bool {
	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		httpresp.BadRequest(c, "INVALID_ARGUMENT", "name is required", nil)
		return false
	}
	namespace := h.namespaceForWorkflow(c.Request.Context(), c, name)
	if err := fn(c.Request.Context(), name, namespace); err != nil {
		httpresp.Internal(c, err.Error())
		return false
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
	return true
}

func (h *Handler) syncWorkflowOperationPipelineRun(ctx context.Context, workflowName, status, message string) {
	if h.runRepo == nil || strings.TrimSpace(workflowName) == "" || strings.TrimSpace(status) == "" {
		return
	}
	run, err := h.runRepo.FindByWorkflowName(ctx, workflowName)
	if err != nil || run == nil {
		return
	}
	run.Status = status
	run.Message = strings.TrimSpace(message)
	if !isWorkflowOperationActiveStatus(status) && (run.FinishedAt == nil || run.FinishedAt.IsZero()) {
		finishedAt := h.now()
		run.FinishedAt = &finishedAt
	}
	if err := h.runRepo.Save(ctx, run); err != nil {
		return
	}
	if h.runEventRepo == nil {
		return
	}
	now := h.now()
	eventType := "workflow_operation_synced"
	switch status {
	case string(wfv1.WorkflowFailed), string(wfv1.WorkflowError):
		eventType = "run_failed"
	case string(wfv1.WorkflowSucceeded):
		eventType = "run_completed"
	}
	_ = h.runEventRepo.Append(ctx, &models.PipelineRunEvent{
		RunID:          run.ID,
		WorkflowName:   run.WorkflowName,
		EventType:      eventType,
		SubjectType:    "run",
		SubjectID:      run.ID,
		Status:         run.Status,
		Message:        run.Message,
		IdempotencyKey: strings.Join([]string{"workflow_operation_sync", run.ID, status, message}, ":"),
		OccurredAt:     now,
		ObservedAt:     now,
	})
}

func isWorkflowOperationActiveStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "running", "pending", "unknown", "suspended":
		return true
	default:
		return false
	}
}

func (h *Handler) syncResubmittedPipelineRun(ctx context.Context, sourceWorkflowName, namespace string, wf *wfv1.Workflow) (*models.PipelineRun, error) {
	if h.runRepo == nil || wf == nil {
		return nil, nil
	}
	newWorkflowName := strings.TrimSpace(wf.Name)
	if newWorkflowName == "" || newWorkflowName == sourceWorkflowName {
		return nil, nil
	}
	sourceRun, err := h.runRepo.FindByWorkflowName(ctx, sourceWorkflowName)
	if err != nil || sourceRun == nil {
		return nil, err
	}
	existingRun, err := h.runRepo.FindByWorkflowName(ctx, newWorkflowName)
	if err != nil || existingRun != nil {
		return existingRun, err
	}

	createdAt := h.now()
	if !wf.CreationTimestamp.IsZero() {
		createdAt = wf.CreationTimestamp.Time.UTC()
	}
	run := &models.PipelineRun{
		ID:                 uuid.NewString(),
		TemplateID:         sourceRun.TemplateID,
		PipelineName:       sourceRun.PipelineName,
		TemplateVersion:    sourceRun.TemplateVersion,
		WorkflowName:       newWorkflowName,
		ExecutionTargetID:  sourceRun.ExecutionTargetID,
		TargetSnapshot:     sourceRun.TargetSnapshot,
		Status:             workflowStatusOrDefault(wf),
		NodeCount:          sourceRun.NodeCount,
		AssetIDs:           append([]string(nil), sourceRun.AssetIDs...),
		AssetCount:         sourceRun.AssetCount,
		NoAssetRun:         sourceRun.NoAssetRun,
		Manifest:           sourceRun.Manifest,
		PipelineJSON:       sourceRun.PipelineJSON,
		ArgoNamespace:      namespace,
		ArgoWorkflowUID:    string(wf.UID),
		Message:            strings.TrimSpace(wf.Status.Message),
		ExecutionTarget:    sourceRun.ExecutionTarget,
		TotalEstimatedCost: sourceRun.TotalEstimatedCost,
		Scope:              sourceRun.Scope,
		Owner:              sourceRun.Owner,
		BatchJobID:         sourceRun.BatchJobID,
		LedgerState:        sourceRun.LedgerState,
		CreatedAt:          createdAt,
		UpdatedAt:          createdAt,
		StartedAt:          workflowTimeOrNil(wf.Status.StartedAt.Time),
		FinishedAt:         workflowTimeOrNil(wf.Status.FinishedAt.Time),
	}
	if err := h.runRepo.Save(ctx, run); err != nil {
		return nil, err
	}
	h.appendResubmittedRunEvent(ctx, run, sourceRun)
	return run, nil
}

func workflowStatusOrDefault(wf *wfv1.Workflow) string {
	if wf == nil || wf.Status.Phase == "" {
		return "Pending"
	}
	return string(wf.Status.Phase)
}

func workflowTimeOrNil(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	out := t.UTC()
	return &out
}

func (h *Handler) appendResubmittedRunEvent(ctx context.Context, run, sourceRun *models.PipelineRun) {
	if h.runEventRepo == nil || run == nil {
		return
	}
	now := h.now()
	event := &models.PipelineRunEvent{
		RunID:        run.ID,
		WorkflowName: run.WorkflowName,
		EventType:    "run_resubmitted",
		SubjectType:  "run",
		SubjectID:    run.ID,
		Status:       run.Status,
		Message:      "pipeline run created from workflow resubmit",
		Payload: map[string]interface{}{
			"sourceRunId":        sourceRun.ID,
			"sourceWorkflowName": sourceRun.WorkflowName,
			"workflowName":       run.WorkflowName,
		},
		IdempotencyKey: strings.Join([]string{"workflow_resubmitted", run.ID, sourceRun.ID}, ":"),
		OccurredAt:     now,
		ObservedAt:     now,
	}
	_ = h.runEventRepo.Append(ctx, event)
}

func (h *Handler) namespaceFor(c *gin.Context) string {
	if namespace := strings.TrimSpace(c.GetString("namespace")); namespace != "" {
		return namespace
	}
	if namespace := strings.TrimSpace(c.Query("namespace")); namespace != "" {
		return namespace
	}
	return h.namespace
}

func (h *Handler) namespaceForWorkflow(ctx context.Context, c *gin.Context, workflowName string) string {
	if namespace := strings.TrimSpace(c.GetString("namespace")); namespace != "" {
		return namespace
	}
	if namespace := strings.TrimSpace(c.Query("namespace")); namespace != "" {
		return namespace
	}
	if run, _ := h.findPipelineRunByWorkflow(ctx, workflowName); run != nil {
		if namespace := strings.TrimSpace(run.ArgoNamespace); namespace != "" {
			return namespace
		}
		if run.ExecutionTarget != nil {
			if namespace := strings.TrimSpace(run.ExecutionTarget.Namespace); namespace != "" {
				return namespace
			}
		}
		if namespace := strings.TrimSpace(targetString(run.TargetSnapshot, "namespace")); namespace != "" {
			return namespace
		}
	}
	return h.namespace
}

func parseWorkflowLogOptions(c *gin.Context) (argo.WorkflowLogOptions, gin.H, bool) {
	if cursor := strings.TrimSpace(c.Query("cursor")); cursor != "" {
		httpresp.BadRequest(c, "INVALID_ARGUMENT", "cursor pagination is not available for live Argo logs", gin.H{
			"reason": workflowLogPaginationUnavailableReason,
		})
		return argo.WorkflowLogOptions{}, nil, false
	}
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
	if opts.SinceSeconds != nil {
		meta["sinceSeconds"] = *opts.SinceSeconds
	}
	if opts.SinceTime != "" {
		meta["sinceTime"] = opts.SinceTime
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
