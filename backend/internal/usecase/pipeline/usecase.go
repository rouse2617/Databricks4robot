package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"errors"

	"log/slog"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	"github.com/CyberOrigin2077/cyber-databrew/internal/argo"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	"github.com/CyberOrigin2077/cyber-databrew/internal/transpiler"
	"github.com/CyberOrigin2077/cyber-databrew/internal/usecase/assetvalidation"
	"gopkg.in/yaml.v3"
)

// Sentinel errors.
var (
	ErrTemplateNotFound        = errors.New("template not found")
	ErrDeploymentNotFound      = errors.New("deployment not found")
	ErrAssetNotFound           = errors.New("asset not found")
	ErrInvalidArgument         = errors.New("invalid argument")
	ErrExecutionTargetNotFound = errors.New("execution target not found")
	ErrWorkflowUnavailable     = errors.New("workflow service unavailable: argo server not configured")
)

// Usecase orchestrates pipeline template management and deployment.
type Usecase struct {
	templateRepo   repository.PipelineTemplateRepository
	deploymentRepo repository.PipelineDeploymentRepository
	targetRepo     repository.ExecutionTargetRepository
	runRepo        repository.PipelineRunRepository
	runNodeRepo    repository.PipelineRunNodeRepository
	runEventRepo   repository.PipelineRunEventRepository
	assetNodeRepo  repository.PipelineRunAssetNodeRepository
	notifyRepo     repository.PipelineRunNotificationRepository
	watcherRepo    repository.PipelineRunWatcherStateRepository
	assetRepo      repository.AssetRepository
	assetEventRepo repository.AssetEventRepository
	relationWriter repository.AssetRelationWriter
	logicalRepo    repository.LogicalAssetRepository
	wfClient       argo.WorkflowClient
	namespace      string
	pricing        *PricingConfig
}

type DeployOptions struct {
	DryRun          bool
	TemplateID      string
	TemplateVersion int
	TargetID        string
}

// SetAssetEventRepo sets the asset event repository (optional, for F4.3+).
func (uc *Usecase) SetAssetEventRepo(r repository.AssetEventRepository) {
	uc.assetEventRepo = r
}

// SetRelationWriter sets the asset relation writer (optional, for F4.7).
func (uc *Usecase) SetRelationWriter(r repository.AssetRelationWriter) {
	uc.relationWriter = r
}

// SetLogicalAssetRepo wires logical_assets persistence (CYB-1013 pipeline outputs).
func (uc *Usecase) SetLogicalAssetRepo(r repository.LogicalAssetRepository) {
	uc.logicalRepo = r
}

// SetRunRepositories wires first-class pipeline run persistence. The legacy
// deployment repo remains required for compatibility endpoints.
func (uc *Usecase) SetRunRepositories(
	targetRepo repository.ExecutionTargetRepository,
	runRepo repository.PipelineRunRepository,
	runNodeRepo repository.PipelineRunNodeRepository,
) {
	uc.targetRepo = targetRepo
	uc.runRepo = runRepo
	uc.runNodeRepo = runNodeRepo
}

// SetRunEventRepo wires durable pipeline run event persistence.
func (uc *Usecase) SetRunEventRepo(r repository.PipelineRunEventRepository) {
	uc.runEventRepo = r
}

// SetObservabilityRepositories wires optional pipeline observability
// persistence for asset-node snapshots, notification candidates, and watcher state.
func (uc *Usecase) SetObservabilityRepositories(
	assetNodeRepo repository.PipelineRunAssetNodeRepository,
	notifyRepo repository.PipelineRunNotificationRepository,
	watcherRepo repository.PipelineRunWatcherStateRepository,
) {
	uc.assetNodeRepo = assetNodeRepo
	uc.notifyRepo = notifyRepo
	uc.watcherRepo = watcherRepo
}

// SetPricing wires the GCP pricing config for cost estimation.
func (uc *Usecase) SetPricing(p *PricingConfig) {
	uc.pricing = p
}

func logPipelineSideEffect(op string, err error) {
	if err != nil {
		slog.Warn("pipeline side effect failed", "op", op, "err", err)
	}
}

const (
	runEventSubmitted            = "run_submitted"
	runEventScheduled            = "run_scheduled"
	runEventWorkflowCreated      = "workflow_created"
	runEventWorkflowObserved     = "workflow_observed"
	runEventWorkflowPhaseChanged = "workflow_phase_changed"
	runEventNodeStarted          = "node_started"
	runEventNodeSucceeded        = "node_succeeded"
	runEventNodeFailed           = "node_failed"
	runEventNodeError            = "node_error"
	runEventPodCreated           = "pod_created"
	runEventPodPhaseChanged      = "pod_phase_changed"
	runEventCompleted            = "run_completed"
	runEventFailed               = "run_failed"
	runEventRetryRequested       = "run_retry_requested"
	runEventResubmitted          = "run_resubmitted"
	runEventStopRequested        = "run_stop_requested"
	runEventDeleteRequested      = "run_delete_requested"
	runEventDeleted              = "run_deleted"
	runEventDeleteFailed         = "run_delete_failed"
)

// New creates a Usecase.
func New(
	templateRepo repository.PipelineTemplateRepository,
	deploymentRepo repository.PipelineDeploymentRepository,
	assetRepo repository.AssetRepository,
	wfClient argo.WorkflowClient,
	namespace string,
) *Usecase {
	return &Usecase{
		templateRepo:   templateRepo,
		deploymentRepo: deploymentRepo,
		assetRepo:      assetRepo,
		wfClient:       wfClient,
		namespace:      namespace,
	}
}

// ListExecutionTargets returns currently available runtime destinations.
func (uc *Usecase) ListExecutionTargets(ctx context.Context) ([]models.ExecutionTarget, error) {
	if uc.targetRepo == nil {
		return []models.ExecutionTarget{uc.defaultExecutionTarget()}, nil
	}
	if err := uc.ensureDefaultExecutionTarget(ctx); err != nil {
		return nil, err
	}
	targets, err := uc.targetRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	for i := range targets {
		uc.normalizeExecutionTarget(&targets[i])
	}
	return targets, nil
}

func (uc *Usecase) defaultExecutionTarget() models.ExecutionTarget {
	status := "available"
	if uc.wfClient == nil || strings.TrimSpace(uc.namespace) == "" {
		status = "unavailable"
	}
	return models.ExecutionTarget{
		ID:                   "default",
		Name:                 "Default Argo target",
		Cluster:              "default",
		Namespace:            uc.namespace,
		ArgoServerConfigured: uc.wfClient != nil,
		Status:               status,
		Enabled:              true,
		IsDefault:            true,
		Description:          "Current backend-configured Argo workflow namespace.",
		ResourceDefaults:     map[string]interface{}{},
		QuotaPolicy:          map[string]interface{}{},
		Labels:               map[string]interface{}{"source": "backend-default"},
	}
}

func (uc *Usecase) ensureDefaultExecutionTarget(ctx context.Context) error {
	if uc.targetRepo == nil {
		return nil
	}
	existing, err := uc.targetRepo.FindDefault(ctx)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}
	target := uc.defaultExecutionTarget()
	return uc.targetRepo.Save(ctx, &target)
}

func (uc *Usecase) normalizeExecutionTarget(target *models.ExecutionTarget) {
	if target == nil {
		return
	}
	if target.Status == "" {
		target.Status = "available"
	}
	if target.ID == "default" && uc.wfClient == nil {
		target.Status = "unavailable"
	}
	if target.ID == "default" && target.Namespace == "" {
		target.Namespace = uc.namespace
	}
	if target.ResourceDefaults == nil {
		target.ResourceDefaults = map[string]interface{}{}
	}
	if target.QuotaPolicy == nil {
		target.QuotaPolicy = map[string]interface{}{}
	}
	if target.Labels == nil {
		target.Labels = map[string]interface{}{}
	}
	target.ArgoServerConfigured = target.ArgoServerURL != "" || uc.wfClient != nil
}

func (uc *Usecase) resolveExecutionTarget(ctx context.Context, targetID string) (*models.ExecutionTarget, error) {
	targetID = strings.TrimSpace(targetID)
	if uc.targetRepo == nil {
		if targetID == "" || targetID == "default" {
			target := uc.defaultExecutionTarget()
			return &target, nil
		}
		return nil, fmt.Errorf("%w: target_id=%q", ErrExecutionTargetNotFound, targetID)
	}
	if err := uc.ensureDefaultExecutionTarget(ctx); err != nil {
		return nil, err
	}
	var (
		target *models.ExecutionTarget
		err    error
	)
	if targetID == "" || targetID == "default" {
		target, err = uc.targetRepo.FindDefault(ctx)
	} else {
		target, err = uc.targetRepo.FindByID(ctx, targetID)
	}
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, fmt.Errorf("%w: target_id=%q", ErrExecutionTargetNotFound, targetID)
	}
	uc.normalizeExecutionTarget(target)
	if !target.Enabled {
		return nil, fmt.Errorf("%w: target_id=%q is disabled", ErrExecutionTargetNotFound, targetID)
	}
	return target, nil
}

func executionTargetSnapshot(target *models.ExecutionTarget) map[string]interface{} {
	if target == nil {
		return map[string]interface{}{}
	}
	return map[string]interface{}{
		"id":                     target.ID,
		"name":                   target.Name,
		"cluster":                target.Cluster,
		"namespace":              target.Namespace,
		"serviceAccount":         target.ServiceAccount,
		"argoServerConfigured":   target.ArgoServerConfigured,
		"status":                 target.Status,
		"isDefault":              target.IsDefault,
		"resourceDefaults":       target.ResourceDefaults,
		"quotaPolicy":            target.QuotaPolicy,
		"labels":                 target.Labels,
		"argoAuthSecretRef":      target.ArgoAuthSecretRef,
		"argoInsecureSkipTls":    target.ArgoInsecureSkipTLS,
		"argoCaCertRef":          target.ArgoCACertRef,
		"argoServerUrlRedacted":  target.ArgoServerURL != "",
		"argoAuthSecretRedacted": target.ArgoAuthSecretRef != "",
	}
}

func (uc *Usecase) deploymentToRun(dep *models.PipelineDeployment) *models.PipelineRun {
	if dep == nil {
		return nil
	}
	target := uc.defaultExecutionTarget()
	if dep.ExecutionTarget != nil {
		target = *dep.ExecutionTarget
	}
	run := &models.PipelineRun{
		ID:                dep.ID,
		TemplateID:        dep.TemplateID,
		TemplateVersion:   dep.TemplateVersion,
		PipelineName:      dep.PipelineName,
		WorkflowName:      dep.WorkflowName,
		ExecutionTargetID: target.ID,
		TargetSnapshot:    executionTargetSnapshot(&target),
		Status:            dep.Status,
		NodeCount:         dep.NodeCount,
		AssetIDs:          dep.AssetIDs,
		AssetCount:        dep.AssetCount,
		NoAssetRun:        len(dep.AssetIDs) == 0,
		Manifest:          dep.Manifest,
		PipelineJSON:      dep.PipelineJSON,
		ArgoNamespace:     target.Namespace,
		ExecutionTarget:   &target,
		CreatedAt:         dep.CreatedAt,
		UpdatedAt:         dep.UpdatedAt,
		FinishedAt:        dep.FinishedAt,
	}
	if run.ArgoNamespace == "" {
		run.ArgoNamespace = uc.namespace
	}
	return run
}

func runToDeployment(run *models.PipelineRun) *models.PipelineDeployment {
	if run == nil {
		return nil
	}
	dep := &models.PipelineDeployment{
		ID:              run.ID,
		TemplateID:      run.TemplateID,
		TemplateVersion: run.TemplateVersion,
		PipelineName:    run.PipelineName,
		WorkflowName:    run.WorkflowName,
		Status:          run.Status,
		NodeCount:       run.NodeCount,
		AssetIDs:        run.AssetIDs,
		AssetCount:      run.AssetCount,
		ExecutionTarget: run.ExecutionTarget,
		Manifest:        run.Manifest,
		PipelineJSON:    run.PipelineJSON,
		CreatedAt:       run.CreatedAt,
		UpdatedAt:       run.UpdatedAt,
		FinishedAt:      run.FinishedAt,
	}
	return dep
}

func (uc *Usecase) savePipelineRun(ctx context.Context, dep *models.PipelineDeployment, templateVersion int, wfUID string) error {
	if uc.runRepo == nil || dep == nil {
		return nil
	}
	run := uc.deploymentToRun(dep)
	if templateVersion > 0 {
		run.TemplateVersion = &templateVersion
	}
	run.ArgoWorkflowUID = wfUID
	if err := uc.runRepo.Save(ctx, run); err != nil {
		return err
	}
	uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
		EventType:      runEventSubmitted,
		SubjectType:    "run",
		SubjectID:      run.ID,
		Status:         run.Status,
		Message:        "pipeline run submitted",
		OccurredAt:     run.CreatedAt,
		IdempotencyKey: fmt.Sprintf("run_submitted:%s", run.ID),
		Payload: map[string]interface{}{
			"pipelineName": run.PipelineName,
			"templateId":   run.TemplateID,
			"assetCount":   run.AssetCount,
			"targetId":     run.ExecutionTargetID,
		},
	})
	uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
		EventType:      runEventScheduled,
		SubjectType:    "run",
		SubjectID:      run.ID,
		Status:         run.Status,
		Message:        "pipeline run scheduled",
		OccurredAt:     run.CreatedAt,
		IdempotencyKey: fmt.Sprintf("run_scheduled:%s", run.ID),
		Payload: map[string]interface{}{
			"workflowName": run.WorkflowName,
			"namespace":    run.ArgoNamespace,
			"targetId":     run.ExecutionTargetID,
		},
	})
	if run.WorkflowName != "" {
		uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
			EventType:      runEventWorkflowCreated,
			SubjectType:    "workflow",
			SubjectID:      run.WorkflowName,
			Status:         run.Status,
			Message:        "argo workflow created",
			OccurredAt:     run.CreatedAt,
			IdempotencyKey: fmt.Sprintf("workflow_created:%s:%s", run.ID, run.WorkflowName),
			Payload: map[string]interface{}{
				"workflowUid": run.ArgoWorkflowUID,
				"namespace":   run.ArgoNamespace,
			},
		})
	}
	return nil
}

func (uc *Usecase) enrichRun(ctx context.Context, run *models.PipelineRun) {
	if run == nil {
		return
	}
	if len(run.AssetIDs) == 0 {
		run.AssetIDs = assetIDsFromPipelineJSON(run.PipelineJSON)
	}
	run.AssetCount = len(run.AssetIDs)
	if run.ExecutionTarget == nil {
		if uc.targetRepo != nil && run.ExecutionTargetID != "" {
			if target, err := uc.targetRepo.FindByID(ctx, run.ExecutionTargetID); err == nil && target != nil {
				uc.normalizeExecutionTarget(target)
				run.ExecutionTarget = target
			}
		}
		if run.ExecutionTarget == nil {
			target := uc.defaultExecutionTarget()
			run.ExecutionTarget = &target
		}
	}
	if uc.runNodeRepo != nil {
		if nodes, err := uc.runNodeRepo.FindByRunID(ctx, run.ID); err == nil {
			run.Nodes = nodes
		}
	}
	uc.refreshAssetNodes(ctx, run)
}

func timePtrFromMeta(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	out := t.UTC()
	return &out
}

func structToMap(v interface{}) map[string]interface{} {
	raw, err := json.Marshal(v)
	if err != nil || len(raw) == 0 || string(raw) == "null" {
		return map[string]interface{}{}
	}
	out := map[string]interface{}{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]interface{}{}
	}
	return out
}

func phaseRunTerminalEvent(phase wfv1.WorkflowPhase) string {
	switch phase {
	case wfv1.WorkflowSucceeded:
		return runEventCompleted
	case wfv1.WorkflowFailed:
		return runEventFailed
	case wfv1.WorkflowError:
		return runEventFailed
	default:
		return ""
	}
}

func phaseNodeEvent(phase wfv1.NodePhase) string {
	switch phase {
	case wfv1.NodeSucceeded:
		return runEventNodeSucceeded
	case wfv1.NodeFailed:
		return runEventNodeFailed
	case wfv1.NodeError:
		return runEventNodeError
	default:
		return ""
	}
}

func ptrTimeOrNow(t *time.Time) time.Time {
	if t != nil && !t.IsZero() {
		return t.UTC()
	}
	return time.Now().UTC()
}

func argoTimeOrZero(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	out := t.UTC()
	return &out
}

func (uc *Usecase) appendRunEvent(ctx context.Context, run *models.PipelineRun, event models.PipelineRunEvent) {
	if uc.runEventRepo == nil || run == nil {
		return
	}
	event.RunID = run.ID
	if event.WorkflowName == "" {
		event.WorkflowName = run.WorkflowName
	}
	if event.SubjectType == "" {
		event.SubjectType = "run"
	}
	if event.SubjectID == "" {
		event.SubjectID = run.ID
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	if event.ObservedAt.IsZero() {
		event.ObservedAt = time.Now().UTC()
	}
	if event.IdempotencyKey == "" {
		event.IdempotencyKey = strings.Join([]string{
			event.EventType,
			event.SubjectType,
			event.SubjectID,
			event.Status,
			event.OccurredAt.UTC().Format(time.RFC3339Nano),
		}, ":")
	}
	logPipelineSideEffect("append pipeline run event", uc.runEventRepo.Append(ctx, &event))
	uc.createNotificationCandidate(ctx, run, &event)
}

func isFailureEvent(eventType, status string) bool {
	normalized := strings.ToLower(eventType + " " + status)
	return strings.Contains(normalized, "failed") || strings.Contains(normalized, "error")
}

func (uc *Usecase) createNotificationCandidate(ctx context.Context, run *models.PipelineRun, event *models.PipelineRunEvent) {
	if uc.notifyRepo == nil || run == nil || event == nil || !isFailureEvent(event.EventType, event.Status) {
		return
	}
	eventID := event.IdempotencyKey
	if eventID == "" {
		eventID = event.ID
	}
	logPipelineSideEffect("append pipeline run notification candidate", uc.notifyRepo.AppendCandidate(ctx, &models.PipelineRunNotificationCandidate{
		RunID:          run.ID,
		EventID:        eventID,
		EventType:      event.EventType,
		SubjectType:    event.SubjectType,
		SubjectID:      event.SubjectID,
		Status:         event.Status,
		Message:        event.Message,
		SinkType:       "candidate",
		DeliveryStatus: "pending",
		IdempotencyKey: fmt.Sprintf("notify:%s:%s", run.ID, eventID),
	}))
}

func watcherStateWithHealth(state *models.PipelineRunWatcherState) *models.PipelineRunWatcherState {
	if state == nil {
		return nil
	}
	state.Healthy = state.ConsecutiveFailures == 0 && state.LastError == ""
	if state.LastSuccessAt != nil {
		lag := int64(time.Since(*state.LastSuccessAt).Seconds())
		if lag < 0 {
			lag = 0
		}
		state.ScanLagSeconds = &lag
		state.Stale = lag > int64(3*time.Minute/time.Second)
	} else {
		state.Stale = true
	}
	return state
}

func cloneWatcherState(state *models.PipelineRunWatcherState, limit int) models.PipelineRunWatcherState {
	out := models.PipelineRunWatcherState{ID: "default", ActiveScanLimit: limit}
	if state != nil {
		out = *state
	}
	if out.ID == "" {
		out.ID = "default"
	}
	if out.ActiveScanLimit <= 0 {
		out.ActiveScanLimit = limit
	}
	if out.ActiveScanLimit <= 0 {
		out.ActiveScanLimit = 100
	}
	return out
}

func deriveAssetNodeRows(run *models.PipelineRun, nodes []models.PipelineRunNode) []models.PipelineRunAssetNode {
	if run == nil || len(nodes) == 0 {
		return nil
	}
	assetIDs := run.AssetIDs
	if len(assetIDs) == 0 {
		assetIDs = []string{"no-asset"}
	}
	rows := make([]models.PipelineRunAssetNode, 0, len(assetIDs)*len(nodes))
	now := time.Now().UTC()
	for _, assetID := range assetIDs {
		if strings.TrimSpace(assetID) == "" {
			continue
		}
		for _, node := range nodes {
			nodeID := node.PipelineNodeID
			if nodeID == "" {
				nodeID = node.ArgoNodeID
			}
			if nodeID == "" {
				nodeID = node.DisplayName
			}
			if nodeID == "" {
				continue
			}
			costSource := "not_available"
			if node.EstimatedCostUSD != nil {
				costSource = "estimated_resource_duration"
			}
			rows = append(rows, models.PipelineRunAssetNode{
				RunID:            run.ID,
				AssetID:          assetID,
				PipelineNodeID:   nodeID,
				ArgoNodeID:       node.ArgoNodeID,
				DisplayName:      node.DisplayName,
				Status:           node.Phase,
				Message:          node.Message,
				PodName:          node.PodName,
				LogRef:           node.LogRef,
				EstimatedCostUSD: node.EstimatedCostUSD,
				CostSource:       costSource,
				StartedAt:        node.StartedAt,
				FinishedAt:       node.FinishedAt,
				UpdatedAt:        now,
			})
		}
	}
	return rows
}

func (uc *Usecase) refreshAssetNodes(ctx context.Context, run *models.PipelineRun) {
	if uc.assetNodeRepo == nil || run == nil {
		return
	}
	nodes := run.Nodes
	if len(nodes) == 0 && uc.runNodeRepo != nil {
		if stored, err := uc.runNodeRepo.FindByRunID(ctx, run.ID); err == nil {
			nodes = stored
		}
	}
	rows := deriveAssetNodeRows(run, nodes)
	logPipelineSideEffect("replace pipeline run asset nodes", uc.assetNodeRepo.ReplaceByRunID(ctx, run.ID, rows))
}

func (uc *Usecase) appendWorkflowEvents(ctx context.Context, run *models.PipelineRun, wf *wfv1.Workflow) {
	if run == nil || wf == nil {
		return
	}
	workflowUID := string(wf.UID)
	workflowName := wf.Name
	if workflowName == "" {
		workflowName = run.WorkflowName
	}
	uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
		EventType:      runEventWorkflowObserved,
		SubjectType:    "workflow",
		SubjectID:      workflowName,
		Status:         string(wf.Status.Phase),
		Message:        wf.Status.Message,
		OccurredAt:     ptrTimeOrNow(argoTimeOrZero(wf.CreationTimestamp.Time)),
		IdempotencyKey: fmt.Sprintf("workflow_observed:%s:%s", workflowName, workflowUID),
		Payload: map[string]interface{}{
			"workflowUid": workflowUID,
			"namespace":   wf.Namespace,
		},
	})
	if wf.Status.Phase != "" {
		occurredAt := ptrTimeOrNow(argoTimeOrZero(wf.Status.FinishedAt.Time))
		uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
			EventType:      runEventWorkflowPhaseChanged,
			SubjectType:    "workflow",
			SubjectID:      workflowName,
			Status:         string(wf.Status.Phase),
			Message:        wf.Status.Message,
			OccurredAt:     occurredAt,
			IdempotencyKey: fmt.Sprintf("workflow_phase:%s:%s", workflowName, wf.Status.Phase),
			Payload: map[string]interface{}{
				"workflowUid": workflowUID,
				"namespace":   wf.Namespace,
			},
		})
	}
	if terminalEvent := phaseRunTerminalEvent(wf.Status.Phase); terminalEvent != "" {
		uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
			EventType:      terminalEvent,
			SubjectType:    "run",
			SubjectID:      run.ID,
			Status:         string(wf.Status.Phase),
			Message:        wf.Status.Message,
			OccurredAt:     ptrTimeOrNow(argoTimeOrZero(wf.Status.FinishedAt.Time)),
			IdempotencyKey: fmt.Sprintf("run_terminal:%s:%s", run.ID, wf.Status.Phase),
		})
	}
}

func (uc *Usecase) appendNodeEvents(ctx context.Context, run *models.PipelineRun, nodes map[string]wfv1.NodeStatus) {
	for id, node := range nodes {
		displayName := node.DisplayName
		if displayName == "" {
			displayName = node.Name
		}
		payload := map[string]interface{}{
			"nodeId":       id,
			"name":         node.Name,
			"displayName":  displayName,
			"templateName": node.TemplateName,
			"type":         string(node.Type),
		}
		if !node.StartedAt.Time.IsZero() {
			uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
				EventType:      runEventNodeStarted,
				SubjectType:    "node",
				SubjectID:      id,
				Status:         string(node.Phase),
				Message:        node.Message,
				OccurredAt:     node.StartedAt.Time.UTC(),
				IdempotencyKey: fmt.Sprintf("node_started:%s:%s", id, node.StartedAt.Time.UTC().Format(time.RFC3339Nano)),
				Payload:        payload,
			})
		}
		if eventType := phaseNodeEvent(node.Phase); eventType != "" {
			occurredAt := ptrTimeOrNow(argoTimeOrZero(node.FinishedAt.Time))
			uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
				EventType:      eventType,
				SubjectType:    "node",
				SubjectID:      id,
				Status:         string(node.Phase),
				Message:        node.Message,
				OccurredAt:     occurredAt,
				IdempotencyKey: fmt.Sprintf("node_phase:%s:%s", id, node.Phase),
				Payload:        payload,
			})
		}
		if node.Type == wfv1.NodeTypePod && node.Name != "" {
			uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
				EventType:      runEventPodCreated,
				SubjectType:    "pod",
				SubjectID:      node.Name,
				Status:         string(node.Phase),
				Message:        node.Message,
				OccurredAt:     ptrTimeOrNow(argoTimeOrZero(node.StartedAt.Time)),
				IdempotencyKey: fmt.Sprintf("pod_created:%s", node.Name),
				Payload:        payload,
			})
			if node.Phase != "" {
				uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
					EventType:      runEventPodPhaseChanged,
					SubjectType:    "pod",
					SubjectID:      node.Name,
					Status:         string(node.Phase),
					Message:        node.Message,
					OccurredAt:     ptrTimeOrNow(argoTimeOrZero(node.FinishedAt.Time)),
					IdempotencyKey: fmt.Sprintf("pod_phase:%s:%s", node.Name, node.Phase),
					Payload:        payload,
				})
			}
		}
	}
}

func runNodesFromWorkflow(runID string, wfName string, nodes map[string]wfv1.NodeStatus) []models.PipelineRunNode {
	out := make([]models.PipelineRunNode, 0, len(nodes))
	for id, node := range nodes {
		podName := ""
		if node.Type == wfv1.NodeTypePod {
			podName = node.Name
		}
		displayName := node.DisplayName
		if displayName == "" {
			displayName = node.Name
		}
		pipelineNodeID := node.TemplateName
		if pipelineNodeID == "" {
			pipelineNodeID = displayName
		}
		logRef := ""
		if podName != "" {
			logRef = fmt.Sprintf("/api/v1/workflows/%s/logs?podName=%s", wfName, podName)
		}
		out = append(out, models.PipelineRunNode{
			ID:                uuid.New().String(),
			RunID:             runID,
			PipelineNodeID:    pipelineNodeID,
			ArgoNodeID:        id,
			ArgoNodeName:      node.Name,
			DisplayName:       displayName,
			TemplateName:      node.TemplateName,
			Type:              string(node.Type),
			Phase:             string(node.Phase),
			Message:           node.Message,
			PodName:           podName,
			HostNodeName:      node.HostNodeName,
			Children:          node.Children,
			Inputs:            structToMap(node.Inputs),
			Outputs:           structToMap(node.Outputs),
			ResourcesDuration: structToMap(node.ResourcesDuration),
			ResourceSummary:   map[string]interface{}{},
			LogRef:            logRef,
			StartedAt:         timePtrFromMeta(node.StartedAt.Time),
			FinishedAt:        timePtrFromMeta(node.FinishedAt.Time),
			CreatedAt:         time.Now().UTC(),
			UpdatedAt:         time.Now().UTC(),
		})
	}
	return out
}

func (uc *Usecase) refreshRunStatus(ctx context.Context, run *models.PipelineRun) {
	if uc.wfClient == nil || run == nil || !isActiveDeploymentStatus(run.Status) {
		return
	}
	namespace := run.ArgoNamespace
	if namespace == "" {
		namespace = uc.namespace
	}
	wf, err := uc.wfClient.GetWorkflow(ctx, run.WorkflowName, namespace)
	if err != nil {
		if errors.Is(err, argo.ErrNotFound) {
			run.Status = deploymentStatusExpired
			logPipelineSideEffect("mark expired pipeline run", uc.runRepo.UpdateStatus(ctx, run.ID, deploymentStatusExpired, nil))
		}
		return
	}
	if wf == nil {
		return
	}
	uc.appendWorkflowEvents(ctx, run, wf)
	if wf.Status.Phase != "" {
		run.Status = string(wf.Status.Phase)
	}
	if string(wf.UID) != "" {
		run.ArgoWorkflowUID = string(wf.UID)
	}
	if wf.Status.Phase == "Succeeded" || wf.Status.Phase == "Failed" || wf.Status.Phase == "Error" {
		now := time.Now().UTC()
		run.FinishedAt = &now
	}
	logPipelineSideEffect("update pipeline run status", uc.runRepo.UpdateStatus(ctx, run.ID, run.Status, run.FinishedAt))
	if uc.runNodeRepo != nil && len(wf.Status.Nodes) > 0 {
		uc.appendNodeEvents(ctx, run, wf.Status.Nodes)
		nodes := runNodesFromWorkflow(run.ID, run.WorkflowName, wf.Status.Nodes)
		for i := range nodes {
			if uc.pricing != nil {
				nodes[i].EstimatedCostUSD = resourcesDurationToCost(nodes[i].ResourcesDuration, uc.pricing)
			}
		}
		logPipelineSideEffect("replace pipeline run nodes", uc.runNodeRepo.ReplaceByRunID(ctx, run.ID, nodes))
		run.Nodes = nodes
		uc.refreshAssetNodes(ctx, run)
	}
}

func (uc *Usecase) refreshPipelineRunStatus(ctx context.Context, run *models.PipelineRun) {
	if uc.runRepo == nil {
		return
	}
	uc.refreshRunStatus(ctx, run)
}

// SyncActiveRunEvents refreshes active runs from Argo and records durable
// workflow/node/pod transition events. It is safe to call repeatedly because
// event writes are idempotent.
func (uc *Usecase) SyncActiveRunEvents(ctx context.Context, limit int) (int, error) {
	if uc.runRepo == nil || uc.wfClient == nil || uc.runEventRepo == nil {
		return 0, nil
	}
	if limit <= 0 {
		limit = 100
	}
	var prior *models.PipelineRunWatcherState
	if uc.watcherRepo != nil {
		if state, err := uc.watcherRepo.FindByID(ctx, "default"); err == nil && state != nil && state.ActiveScanLimit > 0 {
			limit = state.ActiveScanLimit
			prior = state
		}
	}
	scanStartedAt := time.Now().UTC()
	nextState := cloneWatcherState(prior, limit)
	nextState.LastScanStartedAt = &scanStartedAt
	nextState.ActiveScanLimit = limit
	nextState.TotalScans++
	runs, err := uc.runRepo.FindAll(ctx)
	if err != nil {
		if uc.watcherRepo != nil {
			now := time.Now().UTC()
			nextState.LastScanFinishedAt = &now
			nextState.LastErrorAt = &now
			nextState.LastError = err.Error()
			nextState.ConsecutiveFailures++
			nextState.TotalErrors++
			logPipelineSideEffect("save pipeline watcher state", uc.watcherRepo.Save(ctx, &nextState))
		}
		return 0, err
	}
	synced := 0
	for i := range runs {
		if synced >= limit {
			break
		}
		if !isActiveDeploymentStatus(runs[i].Status) {
			continue
		}
		uc.refreshPipelineRunStatus(ctx, &runs[i])
		synced++
	}
	if uc.watcherRepo != nil {
		now := time.Now().UTC()
		lag := int64(0)
		if nextState.LastSuccessAt != nil {
			lag = int64(now.Sub(*nextState.LastSuccessAt).Seconds())
			if lag < 0 {
				lag = 0
			}
		}
		nextState.LastSyncedAt = &now
		nextState.LastScanFinishedAt = &now
		nextState.LastSuccessAt = &now
		nextState.LastSyncedRunCount = synced
		nextState.ConsecutiveFailures = 0
		nextState.LastError = ""
		nextState.ScanLagSeconds = &lag
		logPipelineSideEffect("save pipeline watcher state", uc.watcherRepo.Save(ctx, &nextState))
	}
	return synced, nil
}

// GetRunWatcherStatus returns the persisted pipeline watcher health snapshot.
func (uc *Usecase) GetRunWatcherStatus(ctx context.Context) (*models.PipelineRunWatcherState, error) {
	if uc.watcherRepo == nil {
		return watcherStateWithHealth(&models.PipelineRunWatcherState{
			ID:              "default",
			ActiveScanLimit: 100,
			LastError:       "pipeline run watcher state repository is not configured",
		}), nil
	}
	state, err := uc.watcherRepo.FindByID(ctx, "default")
	if err != nil {
		return nil, err
	}
	if state == nil {
		state = &models.PipelineRunWatcherState{ID: "default", ActiveScanLimit: 100}
	}
	return watcherStateWithHealth(state), nil
}

// StartRunEventWatcher starts a polling watcher for active pipeline runs.
func (uc *Usecase) StartRunEventWatcher(ctx context.Context, interval time.Duration, limit int) {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := uc.SyncActiveRunEvents(ctx, limit); err != nil {
					slog.Warn("pipeline run event watcher sync failed", "err", err)
				}
			}
		}
	}()
}

func (uc *Usecase) resolveExecutionTargetForCompatibility(targetID string) (*models.ExecutionTarget, error) {
	if targetID == "" || targetID == "default" {
		target := uc.defaultExecutionTarget()
		return &target, nil
	}
	return nil, fmt.Errorf("%w: target_id=%q", ErrExecutionTargetNotFound, targetID)
}

// ── Templates ─────────────────────────────────────────────────────

// SaveTemplate persists a pipeline template with auto-incremented version.
func (uc *Usecase) SaveTemplate(ctx context.Context, name string, pipeline map[string]interface{}) (*models.PipelineTemplate, error) {
	raw, err := json.Marshal(pipeline)
	if err != nil {
		return nil, fmt.Errorf("marshal pipeline: %w", err)
	}
	pipe, _, err := rawToPipeline(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: parse pipeline: %v", ErrInvalidArgument, err)
	}
	transpiler.NormalizePipeline(pipe)
	if err := transpiler.ValidatePipeline(pipe); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidArgument, err)
	}
	normalizedRaw, err := json.Marshal(pipe)
	if err != nil {
		return nil, fmt.Errorf("marshal normalized pipeline: %w", err)
	}
	normalizedPipeline := map[string]interface{}{}
	if err := json.Unmarshal(normalizedRaw, &normalizedPipeline); err != nil {
		return nil, fmt.Errorf("unmarshal normalized pipeline: %w", err)
	}

	version, err := uc.templateRepo.GetNextVersion(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get next version: %w", err)
	}
	t := &models.PipelineTemplate{
		ID:        uuid.New().String(),
		Name:      name,
		Version:   version,
		Pipeline:  normalizedPipeline,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	t.NodeCount = len(pipe.Nodes)
	if err := uc.templateRepo.Save(ctx, t); err != nil {
		return nil, fmt.Errorf("save template: %w", err)
	}
	return t, nil
}

// ListVersions returns all versions of a pipeline template. The identifier is
// normally a template id; name fallback preserves compatibility with older
// callers that used the route param as a template name.
func (uc *Usecase) ListVersions(ctx context.Context, templateIDOrName string) ([]models.PipelineTemplate, error) {
	name := templateIDOrName
	if t, err := uc.templateRepo.FindByID(ctx, templateIDOrName); err != nil {
		return nil, fmt.Errorf("find template: %w", err)
	} else if t != nil {
		name = t.Name
	}
	return uc.templateRepo.FindVersionsByName(ctx, name)
}

// ListTemplates returns all pipeline templates.
func (uc *Usecase) ListTemplates(ctx context.Context) ([]models.PipelineTemplate, error) {
	return uc.templateRepo.FindAll(ctx)
}

// GetTemplate returns a pipeline template by id.
func (uc *Usecase) GetTemplate(ctx context.Context, id string) (*models.PipelineTemplate, error) {
	return uc.templateRepo.FindByID(ctx, id)
}

// DeleteTemplate removes a pipeline template.
func (uc *Usecase) DeleteTemplate(ctx context.Context, id string) error {
	if uc.runRepo != nil {
		if err := uc.runRepo.DeleteByTemplateID(ctx, id); err != nil {
			return fmt.Errorf("delete template runs: %w", err)
		}
	}
	if err := uc.deploymentRepo.DeleteByTemplateID(ctx, id); err != nil {
		return fmt.Errorf("delete template deployments: %w", err)
	}
	return uc.templateRepo.Delete(ctx, id)
}

// ── Deploy ────────────────────────────────────────────────────────

// Deploy transpiles a pipeline and submits it as an Argo Workflow.
// pipelineArg is the raw pipeline JSON map. name overrides the workflow name.
// assetIDs are passed as workflow-level parameters (F4.1).
func (uc *Usecase) Deploy(
	ctx context.Context,
	pipelineArg map[string]interface{},
	name string,
	assetIDs []string,
	opts ...DeployOptions,
) (*models.PipelineDeployment, error) {
	// Marshal pipeline to JSON for transpiler.
	raw, err := json.Marshal(pipelineArg)
	if err != nil {
		return nil, fmt.Errorf("marshal pipeline: %w", err)
	}

	pipe, nodeCount, err := rawToPipeline(raw)
	if err != nil {
		return nil, fmt.Errorf("parse pipeline: %w", err)
	}
	transpiler.NormalizePipeline(pipe)
	if err := transpiler.ValidatePipeline(pipe); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidArgument, err)
	}
	normalizedRaw, err := json.Marshal(pipe)
	if err != nil {
		return nil, fmt.Errorf("marshal normalized pipeline: %w", err)
	}
	if err := json.Unmarshal(normalizedRaw, &pipelineArg); err != nil {
		return nil, fmt.Errorf("unmarshal normalized pipeline: %w", err)
	}

	pipeName := pipe.Name
	if name != "" {
		pipeName = name
	}

	wfName := pipeName + "-" + uuid.New().String()[:6]
	depID := uuid.New().String()
	templateID := ""
	templateVersion := 0
	dryRun := false
	if len(opts) > 0 {
		templateID = opts[0].TemplateID
		templateVersion = opts[0].TemplateVersion
		dryRun = opts[0].DryRun
	}
	target, err := uc.resolveExecutionTarget(ctx, "")
	if len(opts) > 0 {
		target, err = uc.resolveExecutionTarget(ctx, opts[0].TargetID)
	}
	if err != nil {
		return nil, err
	}
	targetNamespace := target.Namespace
	if targetNamespace == "" {
		targetNamespace = uc.namespace
	}

	normalizedAssetIDs, err := assetvalidation.Validate(ctx, uc.assetRepo, "asset_ids", assetIDs)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAssetNotFound, err)
	}
	assetIDs = normalizedAssetIDs

	// Assemble workflow-level params and global env vars from asset IDs.
	var wfParams []transpiler.Param
	globalEnv := []transpiler.EnvVar{
		{Name: "PIPELINE_DEPLOYMENT_ID", Value: depID},
	}
	if len(assetIDs) > 0 {
		wfParams = append(wfParams, transpiler.Param{
			Name:  "asset_ids",
			Value: strings.Join(assetIDs, ","),
		})
		globalEnv = append(globalEnv,
			transpiler.EnvVar{Name: "ASSET_IDS", Value: strings.Join(assetIDs, ",")},
			transpiler.EnvVar{Name: "ASSET_COUNT", Value: fmt.Sprintf("%d", len(assetIDs))},
		)
		for i, aid := range assetIDs {
			prefix := fmt.Sprintf("ASSET_%d_", i)
			globalEnv = append(globalEnv, transpiler.EnvVar{Name: prefix + "ID", Value: aid})
			if uc.assetRepo != nil {
				a, err := uc.assetRepo.Get(ctx, aid)
				if err == nil && a != nil {
					if a.StorageURI != "" {
						globalEnv = append(globalEnv, transpiler.EnvVar{Name: prefix + "STORAGE_URI", Value: a.StorageURI})
					}
					if a.AssetType != "" {
						globalEnv = append(globalEnv, transpiler.EnvVar{Name: prefix + "TYPE", Value: a.AssetType})
					}
				}
			}
		}
	}

	// Transpile to Argo Workflow.
	wfOpts := &transpiler.Options{
		Name:            wfName,
		Namespace:       targetNamespace,
		TTLSecondsAfter: 3600,
		WorkflowParams:  wfParams,
		GlobalEnv:       globalEnv,
	}
	wf, err := transpiler.Transpile(pipe, wfOpts)
	if err != nil {
		return nil, fmt.Errorf("transpile: %w", err)
	}

	manifestBytes, err := yaml.Marshal(wf)
	if err != nil {
		return nil, fmt.Errorf("marshal manifest: %w", err)
	}
	manifest := string(manifestBytes)
	if manifest == "" {
		manifest = "{}\n"
	}

	if dryRun {
		var templateVersionPtr *int
		if templateVersion > 0 {
			templateVersionPtr = &templateVersion
		}
		return &models.PipelineDeployment{
			ID:              depID,
			PipelineName:    pipeName,
			WorkflowName:    wfName,
			Status:          "Preview",
			TemplateVersion: templateVersionPtr,
			NodeCount:       nodeCount,
			Manifest:        &manifest,
			PipelineJSON:    pipelineArg,
			AssetIDs:        assetIDs,
			AssetCount:      len(assetIDs),
			ExecutionTarget: target,
			CreatedAt:       time.Now().UTC(),
			UpdatedAt:       time.Now().UTC(),
		}, nil
	}

	// Submit to Argo workflow engine.
	if uc.wfClient == nil {
		return nil, ErrWorkflowUnavailable
	}
	status := "Pending"
	if err := uc.wfClient.CreateWorkflow(ctx, wf, targetNamespace); err != nil {
		if strings.Contains(err.Error(), "argo server URL is empty") {
			return nil, fmt.Errorf("%w: create workflow", ErrWorkflowUnavailable)
		}
		return nil, fmt.Errorf("create workflow: %w", err)
	}
	wfUID := ""
	phase, err := uc.wfClient.GetWorkflowStatus(ctx, wfName, targetNamespace)
	if err == nil && phase != "" {
		status = string(phase)
	}
	if wfDetail, err := uc.wfClient.GetWorkflow(ctx, wfName, targetNamespace); err == nil && wfDetail != nil {
		wfUID = string(wfDetail.UID)
		if wfDetail.Status.Phase != "" {
			status = string(wfDetail.Status.Phase)
		}
	}

	dep := &models.PipelineDeployment{
		ID:              depID,
		PipelineName:    pipeName,
		WorkflowName:    wfName,
		Status:          status,
		NodeCount:       nodeCount,
		Manifest:        &manifest,
		PipelineJSON:    pipelineArg,
		AssetIDs:        assetIDs,
		AssetCount:      len(assetIDs),
		ExecutionTarget: target,
		CreatedAt:       time.Now().UTC(),
	}
	if templateID != "" {
		dep.TemplateID = &templateID
	}
	if templateVersion > 0 {
		dep.TemplateVersion = &templateVersion
	}
	// Embed input asset IDs into PipelineJSON for lineage queries (F4.7).
	if len(assetIDs) > 0 {
		pipelineArg["_input_asset_ids"] = assetIDs
	}

	if err := uc.deploymentRepo.Save(ctx, dep); err != nil {
		return nil, fmt.Errorf("save deployment: %w", err)
	}
	if err := uc.savePipelineRun(ctx, dep, templateVersion, wfUID); err != nil {
		return nil, fmt.Errorf("save pipeline run: %w", err)
	}

	// Record asset_events for input assets (F4.4).
	if uc.assetEventRepo != nil && len(assetIDs) > 0 {
		payload, _ := json.Marshal(map[string]interface{}{
			"deployment_id": depID,
			"pipeline_name": pipeName,
			"workflow_name": wfName,
		})
		for _, aid := range assetIDs {
			logPipelineSideEffect("append pipeline_processing event", uc.assetEventRepo.Append(ctx, repository.AssetEventAppendInput{
				EventType:     "pipeline_processing",
				AggregateType: "asset",
				AssetID:       aid,
				RunID:         depID,
				EventPayload:  payload,
			}))
		}
	}

	return dep, nil
}

// DeployByTemplateID deploys a saved template.
func (uc *Usecase) DeployByTemplateID(ctx context.Context, templateID, name string, assetIDs []string, opts ...DeployOptions) (*models.PipelineDeployment, error) {
	t, err := uc.templateRepo.FindByID(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("find template: %w", err)
	}
	if t == nil {
		return nil, ErrTemplateNotFound
	}
	if len(opts) > 0 && opts[0].TemplateVersion > 0 && opts[0].TemplateVersion != t.Version {
		versioned, err := uc.templateRepo.FindByNameAndVersion(ctx, t.Name, opts[0].TemplateVersion)
		if err != nil {
			return nil, fmt.Errorf("find template version: %w", err)
		}
		if versioned == nil {
			return nil, ErrTemplateNotFound
		}
		t = versioned
		templateID = t.ID
	}
	if name == "" {
		name = t.Name
	}
	deployOpts := DeployOptions{TemplateID: templateID, TemplateVersion: t.Version}
	if len(opts) > 0 {
		deployOpts.TargetID = opts[0].TargetID
	}
	return uc.Deploy(ctx, t.Pipeline, name, assetIDs, deployOpts)
}

// ── Pipeline Runs ───────────────────────────────────────────────────────

// CreateRun creates a first-class pipeline run while keeping deployment
// compatibility storage in sync.
func (uc *Usecase) CreateRun(
	ctx context.Context,
	pipelineArg map[string]interface{},
	name string,
	assetIDs []string,
	opts ...DeployOptions,
) (*models.PipelineRun, error) {
	dep, err := uc.Deploy(ctx, pipelineArg, name, assetIDs, opts...)
	if err != nil {
		return nil, err
	}
	if uc.runRepo == nil {
		return uc.deploymentToRun(dep), nil
	}
	run, err := uc.runRepo.FindByID(ctx, dep.ID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return uc.deploymentToRun(dep), nil
	}
	uc.enrichRun(ctx, run)
	return run, nil
}

// CreateRunByTemplateID creates a first-class pipeline run from a saved template.
func (uc *Usecase) CreateRunByTemplateID(ctx context.Context, templateID, name string, assetIDs []string, opts ...DeployOptions) (*models.PipelineRun, error) {
	dep, err := uc.DeployByTemplateID(ctx, templateID, name, assetIDs, opts...)
	if err != nil {
		return nil, err
	}
	if uc.runRepo == nil {
		return uc.deploymentToRun(dep), nil
	}
	run, err := uc.runRepo.FindByID(ctx, dep.ID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return uc.deploymentToRun(dep), nil
	}
	uc.enrichRun(ctx, run)
	return run, nil
}

// ListRuns returns all first-class pipeline runs. When the run table is not
// wired, it projects legacy deployments for compatibility.
func (uc *Usecase) ListRuns(ctx context.Context) ([]models.PipelineRun, error) {
	if uc.runRepo == nil {
		deps, err := uc.ListDeployments(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]models.PipelineRun, 0, len(deps))
		for i := range deps {
			out = append(out, *uc.deploymentToRun(&deps[i]))
		}
		return out, nil
	}
	list, err := uc.runRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	if uc.wfClient != nil {
		refreshed := 0
		for i := range list {
			if refreshed >= maxActiveDeploymentStatusRefresh {
				break
			}
			if isActiveDeploymentStatus(list[i].Status) {
				refreshed++
				uc.refreshPipelineRunStatus(ctx, &list[i])
			}
		}
	}
	for i := range list {
		uc.enrichRun(ctx, &list[i])
	}
	return list, nil
}

// GetRun returns a single first-class pipeline run.
func (uc *Usecase) GetRun(ctx context.Context, id string) (*models.PipelineRun, error) {
	if uc.runRepo == nil {
		dep, err := uc.GetDeployment(ctx, id)
		if err != nil || dep == nil {
			return nil, err
		}
		return uc.deploymentToRun(dep), nil
	}
	run, err := uc.runRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, nil
	}
	uc.refreshPipelineRunStatus(ctx, run)
	uc.enrichRun(ctx, run)
	return run, nil
}

// DeleteRun removes a first-class run and the legacy deployment record.
func (uc *Usecase) DeleteRun(ctx context.Context, id string) error {
	run, err := uc.GetRun(ctx, id)
	if err != nil {
		return err
	}
	if run == nil {
		return ErrDeploymentNotFound
	}
	uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
		EventType:      runEventDeleteRequested,
		SubjectType:    "run",
		SubjectID:      run.ID,
		Status:         run.Status,
		Message:        "pipeline run delete requested",
		IdempotencyKey: fmt.Sprintf("run_delete_requested:%s:%d", run.ID, time.Now().UTC().UnixNano()),
	})
	if uc.wfClient != nil {
		namespace := run.ArgoNamespace
		if namespace == "" {
			namespace = uc.namespace
		}
		if err := uc.wfClient.DeleteWorkflow(ctx, run.WorkflowName, namespace); err != nil {
			uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
				EventType:      runEventDeleteFailed,
				SubjectType:    "run",
				SubjectID:      run.ID,
				Status:         run.Status,
				Message:        "pipeline run delete failed",
				Reason:         err.Error(),
				IdempotencyKey: fmt.Sprintf("run_delete_failed:%s:%d", run.ID, time.Now().UTC().UnixNano()),
				Payload: map[string]interface{}{
					"workflowName": run.WorkflowName,
					"namespace":    namespace,
				},
			})
			logPipelineSideEffect("delete workflow", err)
		} else {
			uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
				EventType:      runEventDeleted,
				SubjectType:    "run",
				SubjectID:      run.ID,
				Status:         "Deleted",
				Message:        "pipeline run deleted",
				IdempotencyKey: fmt.Sprintf("run_deleted:%s", run.ID),
				Payload: map[string]interface{}{
					"workflowName": run.WorkflowName,
					"namespace":    namespace,
				},
			})
		}
	}
	if uc.runNodeRepo != nil {
		logPipelineSideEffect("delete pipeline run nodes", uc.runNodeRepo.DeleteByRunID(ctx, id))
	}
	if uc.runRepo != nil {
		if err := uc.runRepo.Delete(ctx, id); err != nil {
			return err
		}
	}
	return uc.deploymentRepo.Delete(ctx, id)
}

// RetryRun re-runs a first-class pipeline run.
func (uc *Usecase) RetryRun(ctx context.Context, id string) (*models.PipelineRun, error) {
	run, err := uc.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrDeploymentNotFound
	}
	uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
		EventType:      runEventRetryRequested,
		SubjectType:    "run",
		SubjectID:      run.ID,
		Status:         run.Status,
		Message:        "pipeline run retry requested",
		IdempotencyKey: fmt.Sprintf("run_retry_requested:%s:%d", run.ID, time.Now().UTC().UnixNano()),
	})
	targetID := run.ExecutionTargetID
	if targetID == "" && run.ExecutionTarget != nil {
		targetID = run.ExecutionTarget.ID
	}
	next, err := uc.CreateRun(ctx, run.PipelineJSON, run.PipelineName+"-retry", run.AssetIDs, DeployOptions{TargetID: targetID})
	if err == nil && next != nil {
		uc.appendRunEvent(ctx, next, models.PipelineRunEvent{
			EventType:      runEventResubmitted,
			SubjectType:    "run",
			SubjectID:      next.ID,
			Status:         next.Status,
			Message:        "pipeline run created from retry",
			IdempotencyKey: fmt.Sprintf("run_resubmitted:%s:%s", next.ID, run.ID),
			Payload: map[string]interface{}{
				"sourceRunId": run.ID,
			},
		})
	}
	return next, err
}

// StopRun stops a run's workflow.
func (uc *Usecase) StopRun(ctx context.Context, id string) error {
	run, err := uc.GetRun(ctx, id)
	if err != nil {
		return err
	}
	if run == nil {
		return ErrDeploymentNotFound
	}
	if uc.wfClient == nil {
		return fmt.Errorf("workflow client not available")
	}
	uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
		EventType:      runEventStopRequested,
		SubjectType:    "run",
		SubjectID:      run.ID,
		Status:         run.Status,
		Message:        "pipeline run stop requested",
		IdempotencyKey: fmt.Sprintf("run_stop_requested:%s:%d", run.ID, time.Now().UTC().UnixNano()),
	})
	namespace := run.ArgoNamespace
	if namespace == "" {
		namespace = uc.namespace
	}
	return uc.wfClient.StopWorkflow(ctx, run.WorkflowName, namespace)
}

// ListRunEvents returns a chronological page of stored events for a run.
func (uc *Usecase) ListRunEvents(ctx context.Context, id string, opts models.PipelineRunEventListOptions) (*models.PipelineRunEventListResult, error) {
	if uc.runRepo == nil {
		return nil, ErrDeploymentNotFound
	}
	run, err := uc.runRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrDeploymentNotFound
	}
	uc.refreshPipelineRunStatus(ctx, run)
	if uc.runEventRepo == nil {
		return &models.PipelineRunEventListResult{Items: []models.PipelineRunEvent{}}, nil
	}
	result, err := uc.runEventRepo.ListByRunID(ctx, id, opts)
	if err != nil {
		return nil, err
	}
	if result.Items == nil {
		result.Items = []models.PipelineRunEvent{}
	}
	return result, nil
}

// ListRunAssetNodes returns derived asset × node snapshots for a run.
func (uc *Usecase) ListRunAssetNodes(ctx context.Context, id string, opts models.PipelineRunAssetNodeListOptions) (*models.PipelineRunAssetNodeListResult, error) {
	run, err := uc.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrDeploymentNotFound
	}
	if uc.assetNodeRepo == nil {
		rows := deriveAssetNodeRows(run, run.Nodes)
		summary := models.PipelineRunAssetNodeSummary{
			Statuses:   map[string]int{},
			CostSource: "not_available",
		}
		assets := map[string]bool{}
		nodes := map[string]bool{}
		var totalCost float64
		hasCost := false
		for _, row := range rows {
			summary.Statuses[row.Status]++
			assets[row.AssetID] = true
			nodes[row.PipelineNodeID] = true
			if row.EstimatedCostUSD != nil {
				totalCost += *row.EstimatedCostUSD
				hasCost = true
			}
		}
		summary.AssetCount = len(assets)
		summary.NodeCount = len(nodes)
		if hasCost {
			summary.TotalEstimatedCostUSD = &totalCost
			summary.CostSource = "estimated_resource_duration"
		}
		return &models.PipelineRunAssetNodeListResult{Items: rows, Total: len(rows), Summary: summary}, nil
	}
	return uc.assetNodeRepo.ListByRunID(ctx, id, opts)
}

func durationSeconds(startedAt, finishedAt *time.Time) *int64 {
	if startedAt == nil || finishedAt == nil || finishedAt.Before(*startedAt) {
		return nil
	}
	seconds := int64(finishedAt.Sub(*startedAt).Seconds())
	return &seconds
}

// GetRunCostSummary returns estimated cost and duration summaries for a run.
func (uc *Usecase) GetRunCostSummary(ctx context.Context, id string) (*models.PipelineRunCostSummary, error) {
	run, err := uc.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrDeploymentNotFound
	}
	summary := &models.PipelineRunCostSummary{
		RunID:              id,
		CostSource:         "not_available",
		NodeSummaries:      []models.PipelineRunNodeCostSummary{},
		AssetNodeSummaries: []models.PipelineRunAssetNodeCostSummary{},
		GeneratedAt:        time.Now().UTC(),
	}
	var total float64
	hasCost := false
	for _, node := range run.Nodes {
		if node.EstimatedCostUSD != nil {
			total += *node.EstimatedCostUSD
			hasCost = true
		}
		podCount := 0
		if node.PodName != "" {
			podCount = 1
		}
		summary.NodeSummaries = append(summary.NodeSummaries, models.PipelineRunNodeCostSummary{
			NodeID:           node.PipelineNodeID,
			DisplayName:      node.DisplayName,
			Status:           node.Phase,
			PodCount:         podCount,
			EstimatedCostUSD: node.EstimatedCostUSD,
			CostSource:       costSourceFromPtr(node.EstimatedCostUSD),
			DurationSeconds:  durationSeconds(node.StartedAt, node.FinishedAt),
		})
	}
	assetNodes, err := uc.ListRunAssetNodes(ctx, id, models.PipelineRunAssetNodeListOptions{Limit: 500})
	if err == nil && assetNodes != nil {
		for _, row := range assetNodes.Items {
			summary.AssetNodeSummaries = append(summary.AssetNodeSummaries, models.PipelineRunAssetNodeCostSummary{
				AssetID:          row.AssetID,
				NodeID:           row.PipelineNodeID,
				DisplayName:      row.DisplayName,
				Status:           row.Status,
				EstimatedCostUSD: row.EstimatedCostUSD,
				CostSource:       row.CostSource,
			})
		}
	}
	if hasCost {
		summary.TotalEstimatedCostUSD = &total
		summary.CostSource = "estimated_resource_duration"
	}
	return summary, nil
}

func costSourceFromPtr(v *float64) string {
	if v == nil {
		return "not_available"
	}
	return "estimated_resource_duration"
}

// SaveFromDeployment creates a new template from a deployment's pipeline JSON (F7.8).
func (uc *Usecase) SaveFromDeployment(ctx context.Context, deploymentID, templateName string) (*models.PipelineTemplate, error) {
	d, err := uc.deploymentRepo.FindByID(ctx, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("find deployment: %w", err)
	}
	if d == nil {
		return nil, ErrDeploymentNotFound
	}
	if d.PipelineJSON == nil {
		return nil, fmt.Errorf("deployment %s has no pipeline JSON", deploymentID)
	}
	name := templateName
	if name == "" {
		name = d.PipelineName + "-from-deployment"
	}
	return uc.SaveTemplate(ctx, name, d.PipelineJSON)
}

// ── Deployments ───────────────────────────────────────────────────

// maxActiveDeploymentStatusRefresh caps Argo status polls per ListDeployments call.
const maxActiveDeploymentStatusRefresh = 50

const deploymentStatusExpired = "Expired"

func isActiveDeploymentStatus(status string) bool {
	return status == "" || status == "Running" || status == "Pending" || status == "Unknown"
}

func (uc *Usecase) refreshDeploymentStatus(ctx context.Context, d *models.PipelineDeployment) {
	if uc.wfClient == nil || d == nil || !isActiveDeploymentStatus(d.Status) {
		return
	}
	phase, err := uc.wfClient.GetWorkflowStatus(ctx, d.WorkflowName, uc.namespace)
	if err != nil {
		if errors.Is(err, argo.ErrNotFound) {
			d.Status = deploymentStatusExpired
			logPipelineSideEffect("mark expired deployment", uc.deploymentRepo.UpdateStatus(ctx, d.ID, deploymentStatusExpired))
		}
		return
	}
	d.Status = string(phase)
	if phase == "Succeeded" || phase == "Failed" || phase == "Error" {
		now := time.Now().UTC()
		d.FinishedAt = &now
	}
	logPipelineSideEffect("update deployment status", uc.deploymentRepo.UpdateStatus(ctx, d.ID, string(phase)))
}

// ListDeployments returns all deployments, optionally refreshing active statuses.
func (uc *Usecase) ListDeployments(ctx context.Context) ([]models.PipelineDeployment, error) {
	if uc.runRepo != nil {
		runs, err := uc.ListRuns(ctx)
		if err != nil {
			return nil, err
		}
		legacy, err := uc.deploymentRepo.FindAll(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]models.PipelineDeployment, 0, len(runs)+len(legacy))
		seen := make(map[string]struct{}, len(runs))
		for i := range runs {
			dep := runToDeployment(&runs[i])
			out = append(out, *dep)
			seen[dep.ID] = struct{}{}
		}
		for i := range legacy {
			if _, ok := seen[legacy[i].ID]; ok {
				continue
			}
			uc.enrichDeployment(&legacy[i])
			out = append(out, legacy[i])
		}
		sort.SliceStable(out, func(i, j int) bool {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		})
		return out, nil
	}
	list, err := uc.deploymentRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	// Refresh status for active workflows (capped to avoid N+1 storms).
	if uc.wfClient != nil {
		refreshed := 0
		for i := range list {
			if refreshed >= maxActiveDeploymentStatusRefresh {
				break
			}
			if isActiveDeploymentStatus(list[i].Status) {
				refreshed++
				uc.refreshDeploymentStatus(ctx, &list[i])
			}
		}
	}
	for i := range list {
		uc.enrichDeployment(&list[i])
	}
	return list, nil
}

// GetDeployment returns a single deployment with refreshed status.
func (uc *Usecase) GetDeployment(ctx context.Context, id string) (*models.PipelineDeployment, error) {
	d, err := uc.deploymentRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		if uc.runRepo != nil {
			run, err := uc.GetRun(ctx, id)
			if err != nil || run == nil {
				return nil, err
			}
			return runToDeployment(run), nil
		}
		return nil, nil
	}
	uc.refreshDeploymentStatus(ctx, d)
	uc.enrichDeployment(d)
	return d, nil
}

func (uc *Usecase) enrichDeployment(d *models.PipelineDeployment) {
	if d == nil {
		return
	}
	if len(d.AssetIDs) == 0 {
		d.AssetIDs = assetIDsFromPipelineJSON(d.PipelineJSON)
	}
	d.AssetCount = len(d.AssetIDs)
	target := uc.defaultExecutionTarget()
	d.ExecutionTarget = &target
}

// DeleteDeployment removes a deployment record and optionally deletes the K8s workflow.
func (uc *Usecase) DeleteDeployment(ctx context.Context, id string) error {
	if uc.runRepo != nil {
		return uc.DeleteRun(ctx, id)
	}
	if uc.wfClient != nil {
		d, err := uc.deploymentRepo.FindByID(ctx, id)
		if err == nil && d != nil {
			logPipelineSideEffect("delete workflow", uc.wfClient.DeleteWorkflow(ctx, d.WorkflowName, uc.namespace))
		}
	}
	return uc.deploymentRepo.Delete(ctx, id)
}

// RetryDeployment re-deploys from a saved deployment's pipeline JSON.
func (uc *Usecase) RetryDeployment(ctx context.Context, id string) (*models.PipelineDeployment, error) {
	if uc.runRepo != nil {
		run, err := uc.RetryRun(ctx, id)
		if err != nil {
			return nil, err
		}
		return runToDeployment(run), nil
	}
	d, err := uc.deploymentRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find deployment: %w", err)
	}
	if d == nil {
		return nil, ErrDeploymentNotFound
	}
	assetIDs := assetIDsFromPipelineJSON(d.PipelineJSON)
	return uc.Deploy(ctx, d.PipelineJSON, d.PipelineName+"-retry", assetIDs)
}

// StopDeployment stops a running workflow by setting its Shutdown strategy.
func (uc *Usecase) StopDeployment(ctx context.Context, id string) error {
	if uc.runRepo != nil {
		return uc.StopRun(ctx, id)
	}
	d, err := uc.deploymentRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("find deployment: %w", err)
	}
	if d == nil {
		return ErrDeploymentNotFound
	}
	if uc.wfClient == nil {
		return fmt.Errorf("workflow client not available")
	}
	return uc.wfClient.StopWorkflow(ctx, d.WorkflowName, uc.namespace)
}

// ── Pipeline Output Registration (F4.3) ──────────────────────────

// RegisterPipelineOutputInput describes an asset produced by a pipeline step.
type RegisterPipelineOutputInput struct {
	DeploymentID string                 `json:"deployment_id"`
	NodeID       string                 `json:"node_id"`
	AssetID      string                 `json:"asset_id"`
	StorageURI   string                 `json:"storage_uri"`
	AssetType    string                 `json:"asset_type"`
	Files        map[string]string      `json:"files,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// RegisterOutput creates a new asset from a pipeline step output and records
// an asset_event for lineage tracing. The container calls back to this endpoint
// (via the PIPELINE_DEPLOYMENT_ID injected env var) to register results.
func (uc *Usecase) RegisterOutput(ctx context.Context, in RegisterPipelineOutputInput) (*models.Asset, error) {
	// Validate deployment exists.
	dep, err := uc.deploymentRepo.FindByID(ctx, in.DeploymentID)
	if err != nil {
		return nil, fmt.Errorf("find deployment: %w", err)
	}
	if dep == nil {
		return nil, ErrDeploymentNotFound
	}

	assetID := in.AssetID
	if assetID == "" {
		assetID = uuid.New().String()
	}

	asset := &models.Asset{
		AssetID:    assetID,
		StorageURI: in.StorageURI,
		AssetType:  in.AssetType,
		Files:      in.Files,
		Metadata:   in.Metadata,
		CreatedAt:  time.Now().UTC(),
	}
	if err := seedPipelineOutputVersion(ctx, uc.logicalRepo, asset); err != nil {
		return nil, fmt.Errorf("seed pipeline output version: %w", err)
	}
	if err := uc.assetRepo.InsertNew(ctx, asset); err != nil {
		return nil, fmt.Errorf("insert asset: %w", err)
	}

	// Record asset_event for lineage.
	if uc.assetEventRepo != nil {
		eventPayload, _ := json.Marshal(map[string]interface{}{
			"deployment_id": in.DeploymentID,
			"node_id":       in.NodeID,
			"pipeline_name": dep.PipelineName,
		})
		logPipelineSideEffect("append pipeline_output event", uc.assetEventRepo.Append(ctx, repository.AssetEventAppendInput{
			EventType:     "pipeline_output",
			AggregateType: "asset",
			AssetID:       assetID,
			RunID:         in.DeploymentID,
			EventPayload:  eventPayload,
		}))
	}

	// Create asset_relations from input assets to output asset (F4.7).
	if uc.relationWriter != nil {
		switch inputIDs := dep.PipelineJSON["_input_asset_ids"].(type) {
		case []interface{}:
			for _, id := range inputIDs {
				if s, ok := id.(string); ok {
					logPipelineSideEffect("insert pipeline_output relation", uc.relationWriter.InsertRelation(ctx, s, assetID, "pipeline_output", in.DeploymentID))
				}
			}
		case []string:
			for _, s := range inputIDs {
				logPipelineSideEffect("insert pipeline_output relation", uc.relationWriter.InsertRelation(ctx, s, assetID, "pipeline_output", in.DeploymentID))
			}
		}
	}

	return asset, nil
}

// ── Lineage Query (F4.5) ──────────────────────────────────────────

// AssetLineage describes how an asset was produced by a pipeline.
type AssetLineage struct {
	AssetID      string   `json:"asset_id"`
	DeploymentID string   `json:"deployment_id,omitempty"`
	PipelineName string   `json:"pipeline_name,omitempty"`
	WorkflowName string   `json:"workflow_name,omitempty"`
	NodeID       string   `json:"node_id,omitempty"`
	InputAssets  []string `json:"input_assets,omitempty"`
	ProducedAt   string   `json:"produced_at,omitempty"`
}

// GetLineage returns pipeline provenance for an asset (which run + step produced it).
func (uc *Usecase) GetLineage(ctx context.Context, assetID string) (*AssetLineage, error) {
	if uc.assetEventRepo == nil {
		return nil, fmt.Errorf("asset event repo not available")
	}

	events, err := uc.assetEventRepo.ListByAsset(ctx, assetID, repository.AssetEventListOptions{
		EventTypes: []string{"pipeline_output"},
		Limit:      1,
	})
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	if len(events) == 0 {
		return &AssetLineage{AssetID: assetID}, nil
	}

	ev := events[0]
	lineage := &AssetLineage{
		AssetID:    assetID,
		ProducedAt: ev.OccurredAt.Format(time.RFC3339),
	}

	// Parse event payload for deployment_id, node_id, pipeline_name.
	var payload map[string]interface{}
	if len(ev.EventPayload) > 0 {
		_ = json.Unmarshal(ev.EventPayload, &payload)
		if did, ok := payload["deployment_id"].(string); ok {
			lineage.DeploymentID = did
		}
		if nid, ok := payload["node_id"].(string); ok {
			lineage.NodeID = nid
		}
		if pn, ok := payload["pipeline_name"].(string); ok {
			lineage.PipelineName = pn
		}
	}

	// Look up deployment for pipeline details + input assets.
	dep, err := uc.deploymentRepo.FindByID(ctx, lineage.DeploymentID)
	if err == nil && dep != nil {
		if lineage.PipelineName == "" {
			lineage.PipelineName = dep.PipelineName
		}
		lineage.WorkflowName = dep.WorkflowName
		// Extract input asset IDs from PipelineJSON.
		if dep.PipelineJSON != nil {
			switch raw := dep.PipelineJSON["_input_asset_ids"].(type) {
			case []interface{}:
				for _, id := range raw {
					if s, ok := id.(string); ok {
						lineage.InputAssets = append(lineage.InputAssets, s)
					}
				}
			case []string:
				lineage.InputAssets = raw
			}
		}
	}

	return lineage, nil
}

// ── Resource Usage (F5.8) ────────────────────────────────────────

// ResourceUsageReport describes per-pod resource usage for a deployment.
type ResourceUsageReport struct {
	DeploymentID         string              `json:"deployment_id,omitempty"`
	WorkflowName         string              `json:"workflow_name"`
	Status               string              `json:"status"`
	ObservedAt           string              `json:"observed_at"`
	Source               ResourceUsageSource `json:"source"`
	LiveMetricsAvailable bool                `json:"live_metrics_available"`
	Pods                 []PodResourceUsage  `json:"pods"`
}

// ResourceUsageSource describes where each resource signal came from.
type ResourceUsageSource struct {
	Workflow string `json:"workflow"`
	Metrics  string `json:"metrics"`
	Spec     string `json:"spec"`
}

// ResourceValues carries CPU and memory resource quantities.
type ResourceValues struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// PodResourceUsage summarizes per-pod runtime duration and template resource requests.
type PodResourceUsage struct {
	PodName                string         `json:"pod_name"`
	NodeID                 string         `json:"node_id,omitempty"`
	NodeName               string         `json:"node_name,omitempty"`
	TemplateName           string         `json:"template_name,omitempty"`
	ObservedAt             string         `json:"observed_at,omitempty"`
	LiveMetricsAvailable   bool           `json:"live_metrics_available"`
	Requests               ResourceValues `json:"requests"`
	Limits                 ResourceValues `json:"limits"`
	ResourceDuration       ResourceValues `json:"resource_duration"`
	CPUUsage               string         `json:"cpu_usage"`
	MemoryUsage            string         `json:"memory_usage"`
	CPUResourceDuration    string         `json:"cpu_resource_duration,omitempty"`
	MemoryResourceDuration string         `json:"memory_resource_duration,omitempty"`
	CPURequest             string         `json:"cpu_request"`
	MemoryRequest          string         `json:"memory_request"`
	CPULimit               string         `json:"cpu_limit"`
	MemoryLimit            string         `json:"memory_limit"`
}

// GetResourceUsage returns resource usage for pods belonging to a deployment.
func (uc *Usecase) GetResourceUsage(ctx context.Context, deploymentID string) (*ResourceUsageReport, error) {
	d, err := uc.deploymentRepo.FindByID(ctx, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("find deployment: %w", err)
	}
	if d == nil {
		return nil, ErrDeploymentNotFound
	}

	report := &ResourceUsageReport{
		DeploymentID: deploymentID,
		WorkflowName: d.WorkflowName,
		Status:       d.Status,
		ObservedAt:   time.Now().UTC().Format(time.RFC3339),
		Source: ResourceUsageSource{
			Workflow: "unavailable",
			Metrics:  "unavailable",
			Spec:     specSource(d.Manifest),
		},
		LiveMetricsAvailable: false,
		Pods:                 []PodResourceUsage{},
	}

	if uc.wfClient != nil && d.WorkflowName != "" {
		wf, err := uc.wfClient.GetWorkflow(ctx, d.WorkflowName, uc.namespace)
		if err != nil {
			return nil, fmt.Errorf("get workflow: %w", err)
		}
		if wf != nil {
			if wf.Status.Phase != "" {
				report.Status = string(wf.Status.Phase)
			}
			report.Source.Workflow = "argo-live"
			report.Pods = buildPodResourceUsageReport(wf, d.Manifest, report.ObservedAt, "")
		}
	}

	return report, nil
}

// GetWorkflowResourceUsage returns resource usage metadata for a workflow.
func (uc *Usecase) GetWorkflowResourceUsage(ctx context.Context, workflowName string) (*ResourceUsageReport, error) {
	return uc.getWorkflowResourceUsage(ctx, workflowName, "")
}

// GetWorkflowNodeResourceUsage returns resource usage metadata for one workflow node.
func (uc *Usecase) GetWorkflowNodeResourceUsage(ctx context.Context, workflowName, nodeID string) (*ResourceUsageReport, error) {
	return uc.getWorkflowResourceUsage(ctx, workflowName, nodeID)
}

func (uc *Usecase) getWorkflowResourceUsage(ctx context.Context, workflowName, nodeID string) (*ResourceUsageReport, error) {
	if strings.TrimSpace(workflowName) == "" {
		return nil, ErrInvalidArgument
	}
	if uc.wfClient == nil {
		return nil, ErrWorkflowUnavailable
	}

	var manifest *string
	var deploymentID string
	// Prefer the first-class pipeline_runs table (workflow_name UNIQUE) so we
	// avoid a full table scan over pipeline_deployments. Fall back to the
	// legacy compatibility read only when the run repo has no matching row.
	if uc.runRepo != nil {
		if run, err := uc.runRepo.FindByWorkflowName(ctx, workflowName); err == nil && run != nil {
			manifest = run.Manifest
			deploymentID = run.ID
		}
	}
	if manifest == nil && uc.deploymentRepo != nil {
		if deployments, err := uc.deploymentRepo.FindAll(ctx); err == nil {
			for i := range deployments {
				if deployments[i].WorkflowName == workflowName {
					manifest = deployments[i].Manifest
					deploymentID = deployments[i].ID
					break
				}
			}
		}
	}

	wf, err := uc.wfClient.GetWorkflow(ctx, workflowName, uc.namespace)
	if err != nil {
		return nil, fmt.Errorf("get workflow: %w", err)
	}
	observedAt := time.Now().UTC().Format(time.RFC3339)
	report := &ResourceUsageReport{
		DeploymentID: deploymentID,
		WorkflowName: workflowName,
		Status:       string(wf.Status.Phase),
		ObservedAt:   observedAt,
		Source: ResourceUsageSource{
			Workflow: "argo-live",
			Metrics:  "unavailable",
			Spec:     specSource(manifest),
		},
		LiveMetricsAvailable: false,
		Pods:                 buildPodResourceUsageReport(wf, manifest, observedAt, nodeID),
	}
	if nodeID != "" && len(report.Pods) == 0 {
		return nil, ErrDeploymentNotFound
	}
	return report, nil
}

// ── Pipeline Diff (F2.12) ────────────────────────────────────────

// PipelineDiff describes the structural diff between two template versions.
type PipelineDiff struct {
	AddedNodes    []DiffNode `json:"added_nodes"`
	RemovedNodes  []DiffNode `json:"removed_nodes"`
	ModifiedNodes []DiffNode `json:"modified_nodes"`
	AddedEdges    []DiffEdge `json:"added_edges"`
	RemovedEdges  []DiffEdge `json:"removed_edges"`
}

// DiffNode is a node in a diff result.
type DiffNode struct {
	ID        string                 `json:"id"`
	Component map[string]interface{} `json:"component,omitempty"`
	Inputs    []interface{}          `json:"inputs,omitempty"`
	Outputs   []interface{}          `json:"outputs,omitempty"`
}

// DiffEdge is an edge in a diff result.
type DiffEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// DiffTemplates compares two pipeline templates and returns a structural diff.
func (uc *Usecase) DiffTemplates(ctx context.Context, id1, id2 string) (*PipelineDiff, error) {
	t1, err := uc.templateRepo.FindByID(ctx, id1)
	if err != nil {
		return nil, fmt.Errorf("find template %s: %w", id1, err)
	}
	if t1 == nil {
		return nil, fmt.Errorf("template %s: %w", id1, ErrTemplateNotFound)
	}
	t2, err := uc.templateRepo.FindByID(ctx, id2)
	if err != nil {
		return nil, fmt.Errorf("find template %s: %w", id2, err)
	}
	if t2 == nil {
		return nil, fmt.Errorf("template %s: %w", id2, ErrTemplateNotFound)
	}

	p1Raw, err := json.Marshal(t1.Pipeline)
	if err != nil {
		return nil, fmt.Errorf("marshal t1 pipeline: %w", err)
	}
	p2Raw, err := json.Marshal(t2.Pipeline)
	if err != nil {
		return nil, fmt.Errorf("marshal t2 pipeline: %w", err)
	}

	p1, _, err := rawToPipeline(p1Raw)
	if err != nil {
		return nil, fmt.Errorf("parse t1 pipeline: %w", err)
	}
	p2, _, err := rawToPipeline(p2Raw)
	if err != nil {
		return nil, fmt.Errorf("parse t2 pipeline: %w", err)
	}

	// Build lookup maps.
	nodes1 := make(map[string]map[string]interface{})
	for _, n := range p1.Nodes {
		nodes1[n.ID] = nodeToMap(n)
	}
	nodes2 := make(map[string]map[string]interface{})
	for _, n := range p2.Nodes {
		nodes2[n.ID] = nodeToMap(n)
	}

	edges1 := make(map[string]bool)
	for _, e := range p1.Edges {
		edges1[e.Source+"|"+e.Target] = true
	}
	edges2 := make(map[string]bool)
	for _, e := range p2.Edges {
		edges2[e.Source+"|"+e.Target] = true
	}

	diff := &PipelineDiff{}

	// Find added/modified nodes.
	for _, n2 := range p2.Nodes {
		n1Raw, exists := nodes1[n2.ID]
		if !exists {
			diff.AddedNodes = append(diff.AddedNodes, nodeToDiffNode(n2))
		} else {
			n2Raw := nodeToMap(n2)
			if !mapsEqual(n1Raw, n2Raw) {
				diff.ModifiedNodes = append(diff.ModifiedNodes, nodeToDiffNode(n2))
			}
		}
	}

	// Find removed nodes.
	for _, n1 := range p1.Nodes {
		if _, exists := nodes2[n1.ID]; !exists {
			diff.RemovedNodes = append(diff.RemovedNodes, nodeToDiffNode(n1))
		}
	}

	// Find added edges.
	for _, e2 := range p2.Edges {
		key := e2.Source + "|" + e2.Target
		if !edges1[key] {
			diff.AddedEdges = append(diff.AddedEdges, DiffEdge{Source: e2.Source, Target: e2.Target})
		}
	}

	// Find removed edges.
	for _, e1 := range p1.Edges {
		key := e1.Source + "|" + e1.Target
		if !edges2[key] {
			diff.RemovedEdges = append(diff.RemovedEdges, DiffEdge{Source: e1.Source, Target: e1.Target})
		}
	}

	return diff, nil
}

func nodeToMap(n transpiler.Node) map[string]interface{} {
	m := map[string]interface{}{
		"id": n.ID,
	}
	if n.Component.Name != "" || n.Component.Image != "" {
		comp := map[string]interface{}{
			"name":  n.Component.Name,
			"image": n.Component.Image,
		}
		if len(n.Component.Command) > 0 {
			comp["command"] = n.Component.Command
		}
		if n.Component.Resources != nil {
			comp["resources"] = n.Component.Resources
		}
		m["component"] = comp
	}
	if len(n.Inputs) > 0 {
		m["inputs"] = n.Inputs
	}
	if len(n.Outputs) > 0 {
		m["outputs"] = n.Outputs
	}
	return m
}

func nodeToDiffNode(n transpiler.Node) DiffNode {
	dn := DiffNode{ID: n.ID}
	if n.Component.Name != "" || n.Component.Image != "" {
		comp := map[string]interface{}{
			"name":  n.Component.Name,
			"image": n.Component.Image,
		}
		if len(n.Component.Command) > 0 {
			comp["command"] = n.Component.Command
		}
		if n.Component.Resources != nil {
			comp["resources"] = n.Component.Resources
		}
		dn.Component = comp
	}
	if len(n.Inputs) > 0 {
		dn.Inputs = make([]interface{}, len(n.Inputs))
		for i, p := range n.Inputs {
			dn.Inputs[i] = map[string]string{"name": p.Name, "type": p.Type}
		}
	}
	if len(n.Outputs) > 0 {
		dn.Outputs = make([]interface{}, len(n.Outputs))
		for i, p := range n.Outputs {
			dn.Outputs[i] = map[string]string{"name": p.Name, "type": p.Type}
		}
	}
	return dn
}

func mapsEqual(a, b map[string]interface{}) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}

// ── Helpers ───────────────────────────────────────────────────────

func rawToPipeline(raw json.RawMessage) (*transpiler.Pipeline, int, error) {
	var pipe transpiler.Pipeline
	if err := json.Unmarshal(raw, &pipe); err != nil {
		return nil, 0, err
	}
	return &pipe, len(pipe.Nodes), nil
}

func assetIDsFromPipelineJSON(pipeline map[string]interface{}) []string {
	if pipeline == nil {
		return nil
	}
	switch raw := pipeline["_input_asset_ids"].(type) {
	case []string:
		return raw
	case []interface{}:
		out := make([]string, 0, len(raw))
		for _, id := range raw {
			if s, ok := id.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
