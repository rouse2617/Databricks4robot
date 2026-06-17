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
	"github.com/CyberOrigin2077/cyber-databrew/internal/batchprogress"
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
	templateRepo            repository.PipelineTemplateRepository
	deploymentRepo          repository.PipelineDeploymentRepository
	targetRepo              repository.ExecutionTargetRepository
	runRepo                 repository.PipelineRunRepository
	runNodeRepo             repository.PipelineRunNodeRepository
	runEventRepo            repository.PipelineRunEventRepository
	assetNodeRepo           repository.PipelineRunAssetNodeRepository
	notifyRepo              repository.PipelineRunNotificationRepository
	watcherRepo             repository.PipelineRunWatcherStateRepository
	assetRepo               repository.AssetRepository
	assetEventRepo          repository.AssetEventRepository
	relationWriter          repository.AssetRelationWriter
	logicalRepo             repository.LogicalAssetRepository
	wfClient                argo.WorkflowClient
	namespace               string
	pricing                 *PricingConfig
	workflowTTLSecondsAfter int32
}

type DeployOptions struct {
	DryRun             bool
	TemplateID         string
	TemplateVersion    int
	TargetID           string
	Owner              string
	BatchJobID         string
	PreallocatedRunID  string
	AllowUnknownAssets bool
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

// SetArgoWorkflowTTLSecondsAfterCompletion configures Argo workflow CR TTL
// after completion. Zero keeps transpiler.DefaultTTLSecondsAfterCompletion.
func (uc *Usecase) SetArgoWorkflowTTLSecondsAfterCompletion(seconds int32) {
	uc.workflowTTLSecondsAfter = seconds
}

func (uc *Usecase) argoWorkflowTTLSecondsAfter() int32 {
	if uc.workflowTTLSecondsAfter > 0 {
		return uc.workflowTTLSecondsAfter
	}
	return transpiler.DefaultTTLSecondsAfterCompletion
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
		Scope:             dep.Scope,
		Owner:             dep.Owner,
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
		Scope:           run.Scope,
		Owner:           run.Owner,
		Manifest:        run.Manifest,
		PipelineJSON:    run.PipelineJSON,
		CreatedAt:       run.CreatedAt,
		UpdatedAt:       run.UpdatedAt,
		FinishedAt:      run.FinishedAt,
	}
	return dep
}

func (uc *Usecase) savePipelineRun(ctx context.Context, dep *models.PipelineDeployment, templateVersion int, wfUID string, opts ...DeployOptions) error {
	if uc.runRepo == nil || dep == nil {
		return nil
	}
	run := uc.deploymentToRun(dep)
	if templateVersion > 0 {
		run.TemplateVersion = &templateVersion
	}
	run.ArgoWorkflowUID = wfUID
	if len(opts) > 0 && opts[0].BatchJobID != "" {
		run.BatchJobID = &opts[0].BatchJobID
	}
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
	rowIndexes := make(map[string]int, len(assetIDs)*len(nodes))
	now := time.Now().UTC()
	for _, assetID := range assetIDs {
		if strings.TrimSpace(assetID) == "" {
			continue
		}
		for _, node := range nodes {
			if isWorkflowControlRunNode(node) {
				continue
			}
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
			row := models.PipelineRunAssetNode{
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
			}
			key := assetID + "\x00" + nodeID
			if existingIndex, ok := rowIndexes[key]; ok {
				if assetNodeRowScore(row) > assetNodeRowScore(rows[existingIndex]) {
					rows[existingIndex] = row
				}
				continue
			}
			rowIndexes[key] = len(rows)
			rows = append(rows, row)
		}
	}
	return rows
}

func isWorkflowControlRunNode(node models.PipelineRunNode) bool {
	if node.Type == string(wfv1.NodeTypeDAG) || node.Type == string(wfv1.NodeTypeSteps) {
		return true
	}
	id := strings.TrimSpace(node.PipelineNodeID)
	return id == "dag"
}

func assetNodeRowScore(row models.PipelineRunAssetNode) int {
	score := 0
	if row.PodName != "" {
		score += 10
	}
	if row.EstimatedCostUSD != nil {
		score += 8
	}
	if row.StartedAt != nil {
		score += 4
	}
	if row.FinishedAt != nil {
		score += 4
	}
	switch row.Status {
	case string(wfv1.NodeSucceeded), string(wfv1.NodeFailed), string(wfv1.NodeError):
		score += 3
	case string(wfv1.NodeRunning):
		score += 2
	case string(wfv1.NodePending), "":
		score--
	}
	if row.ArgoNodeID != "" && !strings.HasPrefix(row.ArgoNodeID, "static:") {
		score++
	}
	return score
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

func workflowTaskNameCandidates(node models.PipelineRunNode) []string {
	candidates := []string{
		strings.TrimSpace(node.PipelineNodeID),
		strings.TrimSpace(node.TemplateName),
		strings.TrimSpace(node.DisplayName),
	}
	if node.ArgoNodeName != "" {
		parts := strings.Split(node.ArgoNodeName, ".")
		candidates = append(candidates, strings.TrimSpace(parts[len(parts)-1]))
	}
	out := make([]string, 0, len(candidates))
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		out = append(out, candidate)
	}
	return out
}

func templateNodeType(tmpl *wfv1.Template) string {
	if tmpl == nil {
		return string(wfv1.NodeTypePod)
	}
	switch {
	case tmpl.DAG != nil:
		return string(wfv1.NodeTypeDAG)
	case tmpl.Steps != nil:
		return string(wfv1.NodeTypeSteps)
	case tmpl.Container != nil || tmpl.Script != nil:
		return string(wfv1.NodeTypePod)
	case tmpl.Suspend != nil:
		return string(wfv1.NodeTypeSuspend)
	default:
		return string(wfv1.NodeTypePod)
	}
}

func appendMissingStaticRunNodesFromWorkflow(runID string, wf *wfv1.Workflow, nodes []models.PipelineRunNode) []models.PipelineRunNode {
	if wf == nil || len(wf.Spec.Templates) == 0 {
		return nodes
	}
	templatesByName := make(map[string]*wfv1.Template, len(wf.Spec.Templates))
	for i := range wf.Spec.Templates {
		tmpl := &wf.Spec.Templates[i]
		templatesByName[tmpl.Name] = tmpl
	}
	seen := map[string]struct{}{}
	for _, node := range nodes {
		for _, candidate := range workflowTaskNameCandidates(node) {
			seen[candidate] = struct{}{}
		}
	}
	now := time.Now().UTC()
	for i := range wf.Spec.Templates {
		tmpl := &wf.Spec.Templates[i]
		if tmpl.DAG == nil {
			continue
		}
		for _, task := range tmpl.DAG.Tasks {
			taskName := strings.TrimSpace(task.Name)
			if taskName == "" {
				continue
			}
			if _, ok := seen[taskName]; ok {
				continue
			}
			templateName := strings.TrimSpace(task.Template)
			phase := "Pending"
			if wf.Status.Phase == wfv1.WorkflowSucceeded {
				phase = string(wfv1.NodeSucceeded)
			}
			nodes = append(nodes, models.PipelineRunNode{
				ID:              uuid.New().String(),
				RunID:           runID,
				PipelineNodeID:  taskName,
				ArgoNodeID:      fmt.Sprintf("static:%s", taskName),
				DisplayName:     taskName,
				TemplateName:    templateName,
				Type:            templateNodeType(templatesByName[templateName]),
				Phase:           phase,
				Children:        []string{},
				ResourceSummary: map[string]interface{}{},
				CreatedAt:       now,
				UpdatedAt:       now,
			})
			seen[taskName] = struct{}{}
		}
	}
	return nodes
}

func (uc *Usecase) replaceRunNodesFromWorkflow(ctx context.Context, run *models.PipelineRun, wf *wfv1.Workflow) {
	if uc.runNodeRepo == nil || run == nil || wf == nil {
		return
	}
	uc.appendNodeEvents(ctx, run, wf.Status.Nodes)
	nodes := runNodesFromWorkflow(run.ID, run.WorkflowName, wf.Status.Nodes)
	nodes = appendMissingStaticRunNodesFromWorkflow(run.ID, wf, nodes)
	if len(nodes) == 0 {
		return
	}
	for i := range nodes {
		if uc.pricing != nil {
			nodes[i].EstimatedCostUSD = resourcesDurationToCost(nodes[i].ResourcesDuration, uc.pricing)
		}
	}
	logPipelineSideEffect("replace pipeline run nodes", uc.runNodeRepo.ReplaceByRunID(ctx, run.ID, nodes))
	run.Nodes = nodes
	uc.refreshAssetNodes(ctx, run)
}

func (uc *Usecase) runCostSnapshotMissing(run *models.PipelineRun) bool {
	if run == nil {
		return false
	}
	if len(run.Nodes) == 0 {
		return true
	}
	businessNodeCount := 0
	for _, node := range run.Nodes {
		if node.PipelineNodeID != "dag" {
			businessNodeCount++
		}
		if uc.pricing != nil && len(node.ResourcesDuration) > 0 && node.EstimatedCostUSD == nil {
			return true
		}
	}
	if run.NodeCount > 0 && businessNodeCount < run.NodeCount {
		return true
	}
	return false
}

func (uc *Usecase) applyWorkflowToRun(ctx context.Context, run *models.PipelineRun, wf *wfv1.Workflow) {
	if uc.runRepo == nil || run == nil || wf == nil {
		return
	}
	uc.appendWorkflowEvents(ctx, run, wf)
	status := run.Status
	if wf.Status.Phase != "" {
		status = string(wf.Status.Phase)
	}
	message := wf.Status.Message
	finishedAt := argoTimeOrZero(wf.Status.FinishedAt.Time)
	if derivedStatus, derivedMessage, derivedFinishedAt, ok := deriveTerminalRunFromWorkflowNodes(wf); ok {
		status = derivedStatus
		message = derivedMessage
		if derivedFinishedAt != nil {
			finishedAt = derivedFinishedAt
		}
	}
	run.Status = status
	if string(wf.UID) != "" {
		run.ArgoWorkflowUID = string(wf.UID)
	}
	if isActiveDeploymentStatus(run.Status) {
		run.FinishedAt = nil
		run.Message = ""
	}
	if wf.Status.Phase == wfv1.WorkflowSucceeded || wf.Status.Phase == wfv1.WorkflowFailed || wf.Status.Phase == wfv1.WorkflowError {
		if finishedAt != nil {
			run.FinishedAt = finishedAt
		} else if run.FinishedAt == nil || run.FinishedAt.IsZero() {
			now := time.Now().UTC()
			run.FinishedAt = &now
		}
	}
	if !isActiveDeploymentStatus(run.Status) {
		run.Message = strings.TrimSpace(message)
		if finishedAt != nil {
			run.FinishedAt = finishedAt
		} else if run.FinishedAt == nil || run.FinishedAt.IsZero() {
			now := time.Now().UTC()
			run.FinishedAt = &now
		}
	}
	uc.persistRunObservation(ctx, run)
	uc.replaceRunNodesFromWorkflow(ctx, run, wf)
}

func deriveTerminalRunFromWorkflowNodes(wf *wfv1.Workflow) (string, string, *time.Time, bool) {
	if wf == nil || !isActiveDeploymentStatus(string(wf.Status.Phase)) {
		return "", "", nil, false
	}
	var latestFinishedAt *time.Time
	derivedMessage := ""
	for _, node := range wf.Status.Nodes {
		phase := node.Phase
		if phase != wfv1.NodeFailed && phase != wfv1.NodeError {
			continue
		}
		message := strings.TrimSpace(node.Message)
		if !isWorkflowShutdownMessage(message) {
			continue
		}
		if finishedAt := argoTimeOrZero(node.FinishedAt.Time); finishedAt != nil {
			if latestFinishedAt == nil || finishedAt.After(*latestFinishedAt) {
				latestFinishedAt = finishedAt
				derivedMessage = message
			}
		} else if derivedMessage == "" {
			derivedMessage = message
		}
	}
	if derivedMessage == "" {
		return "", "", nil, false
	}
	return string(wfv1.WorkflowFailed), derivedMessage, latestFinishedAt, true
}

func isWorkflowShutdownMessage(message string) bool {
	normalized := strings.ToLower(strings.TrimSpace(message))
	return strings.Contains(normalized, "workflow shutdown with strategy:") ||
		strings.Contains(normalized, "stopped with strategy")
}

func (uc *Usecase) persistRunObservation(ctx context.Context, run *models.PipelineRun) {
	if uc.runRepo == nil || run == nil || strings.TrimSpace(run.ID) == "" {
		return
	}
	existing, err := uc.runRepo.FindByID(ctx, run.ID)
	if err != nil || existing == nil {
		logPipelineSideEffect("save pipeline run observation", uc.runRepo.Save(ctx, run))
		return
	}
	existing.Status = run.Status
	if isActiveDeploymentStatus(run.Status) {
		existing.FinishedAt = nil
	} else if run.FinishedAt != nil && !run.FinishedAt.IsZero() {
		if existing.FinishedAt == nil || existing.FinishedAt.IsZero() || run.FinishedAt.Before(*existing.FinishedAt) {
			existing.FinishedAt = run.FinishedAt
		}
	}
	existing.Message = run.Message
	if uid := strings.TrimSpace(run.ArgoWorkflowUID); uid != "" {
		existing.ArgoWorkflowUID = uid
	}
	if name := strings.TrimSpace(run.WorkflowName); name != "" {
		existing.WorkflowName = name
	}
	if run.StartedAt != nil {
		existing.StartedAt = run.StartedAt
	}
	logPipelineSideEffect("save pipeline run observation", uc.runRepo.Save(ctx, existing))
	*run = *existing
}

func isStaleWorkflowUnavailableMessage(message string) bool {
	switch strings.TrimSpace(message) {
	case staleWorkflowTTLCleanupMessage, messageWorkflowAwaitingDeploy, messageWorkflowUnavailable:
		return true
	default:
		return false
	}
}

func workflowUnavailableMessage(run *models.PipelineRun) string {
	if run == nil {
		return messageWorkflowUnavailable
	}
	if strings.TrimSpace(run.ArgoWorkflowUID) == "" && isBatchSubtaskPlaceholderWorkflowName(run.WorkflowName) {
		return messageWorkflowAwaitingDeploy
	}
	return messageWorkflowUnavailable
}

func (uc *Usecase) markRunWorkflowNotFound(ctx context.Context, run *models.PipelineRun) {
	if uc.reconcileTerminalRunFromLedger(ctx, run) {
		return
	}
	if isPendingBatchWorkflowCreation(run) {
		if isStaleWorkflowUnavailableMessage(run.Message) {
			run.Message = ""
			uc.persistRunObservation(ctx, run)
		}
		return
	}
	if run.FinishedAt == nil || run.FinishedAt.IsZero() {
		now := time.Now().UTC()
		run.FinishedAt = &now
	}
	run.Status = deploymentStatusExpired
	run.Message = workflowUnavailableMessage(run)
	uc.persistRunObservation(ctx, run)
}

func (uc *Usecase) reconcileMisclassifiedRunFromArgo(ctx context.Context, run *models.PipelineRun) {
	if uc.wfClient == nil || run == nil {
		return
	}
	if strings.TrimSpace(run.WorkflowName) == "" {
		return
	}
	if !isMisclassifiedTerminalRunStatus(run.Status) && !isStaleWorkflowUnavailableMessage(run.Message) {
		return
	}
	if isPendingBatchWorkflowCreation(run) && isStaleWorkflowUnavailableMessage(run.Message) {
		run.Message = ""
		uc.persistRunObservation(ctx, run)
		return
	}
	namespace := run.ArgoNamespace
	if namespace == "" {
		namespace = uc.namespace
	}
	wf, err := uc.wfClient.GetWorkflow(ctx, run.WorkflowName, namespace)
	if err != nil || wf == nil {
		return
	}
	uc.applyWorkflowToRun(ctx, run, wf)
}

// RefreshRunForList performs a bounded status refresh for batch list views.
// It avoids full run enrichment and is safe to call per page item.
func (uc *Usecase) RefreshRunForList(ctx context.Context, run *models.PipelineRun) {
	if uc.runRepo == nil || run == nil || strings.TrimSpace(run.ID) == "" {
		return
	}
	if !needsRunListRefresh(run) && !needsMisclassifiedReconcile(run) {
		return
	}
	if isActiveDeploymentStatus(run.Status) {
		uc.refreshRunStatus(ctx, run)
	}
	uc.reconcileMisclassifiedRunFromArgo(ctx, run)
	uc.reconcileTerminalRunFromLedger(ctx, run)
	if fresh, err := uc.runRepo.FindByID(ctx, run.ID); err == nil && fresh != nil {
		*run = *fresh
	}
}

func needsMisclassifiedReconcile(run *models.PipelineRun) bool {
	if run == nil {
		return false
	}
	return isMisclassifiedTerminalRunStatus(run.Status) || isStaleWorkflowUnavailableMessage(run.Message)
}

func needsRunListRefresh(run *models.PipelineRun) bool {
	if run == nil {
		return false
	}
	if isActiveDeploymentStatus(run.Status) {
		return true
	}
	if needsLedgerReconcile(run) {
		return true
	}
	return needsMisclassifiedReconcile(run)
}

func (uc *Usecase) refreshRunSummariesForList(ctx context.Context, items []models.PipelineRun) {
	if uc.runRepo == nil || uc.wfClient == nil || len(items) == 0 {
		return
	}
	refreshed := 0
	for i := range items {
		if refreshed >= maxActiveDeploymentStatusRefresh {
			break
		}
		if !needsRunListRefresh(&items[i]) {
			continue
		}
		refreshed++
		uc.RefreshRunForList(ctx, &items[i])
	}
}

func needsLedgerReconcile(run *models.PipelineRun) bool {
	if run == nil || run.FinishedAt == nil || run.FinishedAt.IsZero() {
		return false
	}
	return isActiveDeploymentStatus(run.Status)
}

// reconcileTerminalRunFromLedger infers a terminal run status from durable
// asset-node rows when Argo has already TTL'd the workflow CR.
func (uc *Usecase) reconcileTerminalRunFromLedger(ctx context.Context, run *models.PipelineRun) bool {
	if uc.runRepo == nil || run == nil || !needsLedgerReconcile(run) {
		return false
	}
	if uc.assetNodeRepo == nil {
		return false
	}
	result, err := uc.assetNodeRepo.ListByRunID(ctx, run.ID, models.PipelineRunAssetNodeListOptions{Limit: 500})
	if err != nil || result == nil || len(result.Items) == 0 {
		return false
	}
	status, ok := inferRunStatusFromAssetNodes(result.Items)
	if !ok {
		return false
	}
	run.Status = status
	uc.persistRunObservation(ctx, run)
	return true
}

func inferRunStatusFromAssetNodes(nodes []models.PipelineRunAssetNode) (string, bool) {
	if len(nodes) == 0 {
		return "", false
	}
	hasFailed := false
	hasActive := false
	for _, node := range nodes {
		switch strings.ToLower(strings.TrimSpace(node.Status)) {
		case "failed", "error":
			hasFailed = true
		case "running", "pending":
			hasActive = true
		case "succeeded", "success", "skipped", "omitted", "completed":
			// terminal success path
		default:
			hasActive = true
		}
	}
	if hasActive {
		return "", false
	}
	if hasFailed {
		return string(wfv1.WorkflowFailed), true
	}
	return string(wfv1.WorkflowSucceeded), true
}

func (uc *Usecase) refreshRunStatus(ctx context.Context, run *models.PipelineRun) {
	if uc.wfClient == nil || run == nil || !isActiveDeploymentStatus(run.Status) {
		return
	}
	if strings.TrimSpace(run.WorkflowName) == "" {
		return
	}
	namespace := run.ArgoNamespace
	if namespace == "" {
		namespace = uc.namespace
	}
	wf, err := uc.wfClient.GetWorkflow(ctx, run.WorkflowName, namespace)
	if err != nil {
		if errors.Is(err, argo.ErrNotFound) {
			if shouldWaitForWorkflowCreation(run, time.Now().UTC()) {
				if isPendingBatchWorkflowCreation(run) && isStaleWorkflowUnavailableMessage(run.Message) {
					run.Message = ""
					uc.persistRunObservation(ctx, run)
				}
				return
			}
			uc.markRunWorkflowNotFound(ctx, run)
		}
		return
	}
	if wf == nil {
		return
	}
	uc.applyWorkflowToRun(ctx, run, wf)
	uc.maybeMarkStaleRun(ctx, run, wf)
}

func (uc *Usecase) maybeMarkStaleRun(ctx context.Context, run *models.PipelineRun, wf *wfv1.Workflow) {
	if uc.runRepo == nil || run == nil || wf == nil || !isActiveDeploymentStatus(run.Status) {
		return
	}
	ref := run.UpdatedAt
	if run.StartedAt != nil && !run.StartedAt.IsZero() {
		ref = *run.StartedAt
	}
	if ref.IsZero() || time.Since(ref) < staleActiveRunMaxAge {
		return
	}
	phase := wf.Status.Phase
	if phase != wfv1.WorkflowRunning && phase != wfv1.WorkflowPending && phase != wfv1.WorkflowPhase("Suspended") {
		return
	}
	now := time.Now().UTC()
	run.Status = string(wfv1.WorkflowFailed)
	run.FinishedAt = &now
	run.Message = fmt.Sprintf("stale run: exceeded maximum active duration (%s)", staleActiveRunMaxAge.Truncate(time.Hour))
	logPipelineSideEffect("mark stale pipeline run failed", uc.runRepo.UpdateStatus(ctx, run.ID, run.Status, run.FinishedAt))
}

func (uc *Usecase) refreshPipelineRunStatus(ctx context.Context, run *models.PipelineRun) {
	if uc.runRepo == nil {
		return
	}
	uc.refreshRunStatus(ctx, run)
}

// backfillRunStatus refreshes a run from Argo regardless of its current status.
// Unlike refreshRunStatus, this does NOT skip completed runs — it's used by the
// watcher backfill to sync events for runs that completed before the watcher scanned them.
func (uc *Usecase) backfillRunStatus(ctx context.Context, run *models.PipelineRun) {
	if uc.wfClient == nil || run == nil {
		return
	}
	namespace := run.ArgoNamespace
	if namespace == "" {
		namespace = uc.namespace
	}
	wf, err := uc.wfClient.GetWorkflow(ctx, run.WorkflowName, namespace)
	if err != nil {
		if errors.Is(err, argo.ErrNotFound) {
			run.LedgerState = "no_ledger"
			logPipelineSideEffect("update pipeline run ledger state",
				uc.runRepo.UpdateLedgerState(ctx, run.ID, "no_ledger"))
		}
		return
	}
	if wf == nil {
		return
	}
	uc.appendWorkflowEvents(ctx, run, wf)
	uc.replaceRunNodesFromWorkflow(ctx, run, wf)
	run.LedgerState = "has_ledger"
	logPipelineSideEffect("update pipeline run ledger state + backfill completed",
		uc.runRepo.UpdateLedgerState(ctx, run.ID, "has_ledger"))
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
	// Backfill: scan completed runs (within the last 7 days) that may be
	// missing their ledger events (e.g. they completed before watcher scan).
	backfillLimit := limit / 2
	if backfillLimit < 5 {
		backfillLimit = 5
	}
	backfilled := 0
	since := time.Now().UTC().AddDate(0, 0, -7)
	for i := range runs {
		if backfilled >= backfillLimit {
			break
		}
		if isActiveDeploymentStatus(runs[i].Status) {
			continue
		}
		if runs[i].FinishedAt == nil || runs[i].FinishedAt.Before(since) {
			continue
		}
		if runs[i].LedgerState == "has_ledger" {
			continue
		}
		uc.backfillRunStatus(ctx, &runs[i])
		backfilled++
	}
	synced += backfilled
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
	if runs, err := uc.runRepo.FindAll(ctx); err == nil {
		total := len(runs)
		hasEvents := 0
		for _, r := range runs {
			if r.LedgerState == "has_ledger" {
				hasEvents++
			}
		}
		state.LedgerHealth = models.LedgerHealth{
			TotalRuns:      total,
			RunsWithEvents: hasEvents,
			RunsWithout:    total - hasEvents,
			LastBackfillAt: state.LastSyncedAt,
		}
	}
	return watcherStateWithHealth(state), nil
}

// StartRunEventWatcher starts a polling watcher for active pipeline runs.
func (uc *Usecase) StartRunEventWatcher(ctx context.Context, interval time.Duration, limit int) {
	if interval <= 0 {
		interval = 3 * time.Second
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
func (uc *Usecase) SaveTemplate(ctx context.Context, name string, pipeline map[string]interface{}, scope string, owner string) (*models.PipelineTemplate, error) {
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
	// Propagate active version from the existing latest (if any).
	activeVersion := 0
	if prev, _ := uc.templateRepo.FindByNameAndVersion(ctx, name, version-1); prev != nil {
		activeVersion = prev.ActiveVersion
	}
	t := &models.PipelineTemplate{
		ID:            uuid.New().String(),
		Name:          name,
		Version:       version,
		ActiveVersion: activeVersion,
		Scope:         scope,
		Owner:         owner,
		Pipeline:      normalizedPipeline,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
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

// SetActiveVersion pins a pipeline's default run version. When version is 0,
// the pin is cleared (latest = active).
func (uc *Usecase) SetActiveVersion(ctx context.Context, templateIDOrName string, version int) error {
	t, err := uc.templateRepo.FindByID(ctx, templateIDOrName)
	if err != nil {
		return fmt.Errorf("find template: %w", err)
	}
	if t == nil {
		return ErrTemplateNotFound
	}
	return uc.templateRepo.SetActiveVersion(ctx, t.Name, version)
}

// Promote copies a dev template to the prod scope.
var ErrProdLocked = errors.New("prod templates are read-only")

func (uc *Usecase) Promote(ctx context.Context, templateID string) (*models.PipelineTemplate, error) {
	t, err := uc.templateRepo.FindByID(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("find template: %w", err)
	}
	if t == nil {
		return nil, ErrTemplateNotFound
	}
	if t.Scope == "prod" {
		return nil, fmt.Errorf("template %s is already in prod scope", t.Name)
	}
	return uc.SaveTemplate(ctx, t.Name, t.Pipeline, "prod", t.Owner)
}

// ListTemplates returns all pipeline templates.
func (uc *Usecase) ListTemplates(ctx context.Context) ([]models.PipelineTemplate, error) {
	return uc.templateRepo.FindAll(ctx)
}

// ListTemplatesPaged returns a paginated pipeline template list.
func (uc *Usecase) ListTemplatesPaged(ctx context.Context, filter models.PipelineTemplateListFilter) ([]models.PipelineTemplate, int, error) {
	return uc.templateRepo.FindLatestPaged(ctx, filter)
}

// GetTemplate returns a pipeline template by id.
func (uc *Usecase) GetTemplate(ctx context.Context, id string) (*models.PipelineTemplate, error) {
	return uc.templateRepo.FindByID(ctx, id)
}

// DeleteTemplate removes a pipeline template.
var ErrTemplateNotOwned = errors.New("template is not owned by current user")

func (uc *Usecase) DeleteTemplate(ctx context.Context, id, owner string) error {
	t, err := uc.templateRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("find template: %w", err)
	}
	if t == nil {
		return nil
	}
	if t.Scope == "prod" {
		return ErrProdLocked
	}
	if owner != "" && t.Owner != "" && t.Owner != owner {
		return ErrTemplateNotOwned
	}
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
		if strings.TrimSpace(opts[0].PreallocatedRunID) != "" {
			depID = strings.TrimSpace(opts[0].PreallocatedRunID)
		}
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

	normalizedAssetIDs, err := uc.validateDeployAssetIDs(ctx, assetIDs, opts...)
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
		TTLSecondsAfter: uc.argoWorkflowTTLSecondsAfter(),
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

	runScope := "dev"
	runOwner := ""
	if len(opts) > 0 {
		runOwner = opts[0].Owner
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
		Scope:           runScope,
		Owner:           runOwner,
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
	if err := uc.savePipelineRun(ctx, dep, templateVersion, wfUID, opts...); err != nil {
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
	requestedVersion := 0
	if len(opts) > 0 {
		requestedVersion = opts[0].TemplateVersion
	}
	// Resolve which version to deploy: explicit request > active pin > latest.
	resolvedVersion := t.Version
	if requestedVersion > 0 && requestedVersion != t.Version {
		resolvedVersion = requestedVersion
	} else if requestedVersion == 0 && t.ActiveVersion > 0 && t.ActiveVersion != t.Version {
		resolvedVersion = t.ActiveVersion
	}
	if resolvedVersion != t.Version {
		versioned, err := uc.templateRepo.FindByNameAndVersion(ctx, t.Name, resolvedVersion)
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
		deployOpts.Owner = opts[0].Owner
		deployOpts.BatchJobID = opts[0].BatchJobID
		deployOpts.DryRun = opts[0].DryRun
		deployOpts.AllowUnknownAssets = opts[0].AllowUnknownAssets
		deployOpts.PreallocatedRunID = opts[0].PreallocatedRunID
	}
	return uc.Deploy(ctx, t.Pipeline, name, assetIDs, deployOpts)
}

func (uc *Usecase) validateDeployAssetIDs(ctx context.Context, assetIDs []string, opts ...DeployOptions) ([]string, error) {
	allowUnknown := len(opts) > 0 && opts[0].AllowUnknownAssets
	if allowUnknown {
		return assetvalidation.NormalizeAssetIDs("asset_ids", assetIDs)
	}
	return assetvalidation.Validate(ctx, uc.assetRepo, "asset_ids", assetIDs)
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

// CreateRunsByTemplateID deploys one pipeline run per asset when multiple asset
// IDs are provided; zero or one asset uses a single run as before.
func (uc *Usecase) CreateRunsByTemplateID(ctx context.Context, templateID, name string, assetIDs []string, opts ...DeployOptions) ([]models.PipelineRun, error) {
	if len(assetIDs) <= 1 {
		run, err := uc.CreateRunByTemplateID(ctx, templateID, name, assetIDs, opts...)
		if err != nil {
			return nil, err
		}
		return []models.PipelineRun{*run}, nil
	}
	runs := make([]models.PipelineRun, 0, len(assetIDs))
	for _, assetID := range assetIDs {
		run, err := uc.CreateRunByTemplateID(ctx, templateID, name, []string{assetID}, opts...)
		if err != nil {
			return runs, err
		}
		runs = append(runs, *run)
	}
	return runs, nil
}

// ListRuns returns all first-class pipeline runs. When the run table is not
// wired, it projects legacy deployments for compatibility.
// refreshActive triggers live Argo status polls (capped); list endpoints should
// pass false and rely on the run watcher + GetRun for on-demand refresh.
func (uc *Usecase) ListRuns(ctx context.Context, refreshActive bool) ([]models.PipelineRun, error) {
	if uc.runRepo == nil {
		deps, err := uc.listDeployments(ctx, refreshActive)
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
	if refreshActive && uc.wfClient != nil {
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

// ListRunSummaries returns lightweight pipeline runs for list UIs. It skips
// manifest/pipeline_json hydration and per-run node/asset enrichment.
func (uc *Usecase) ListRunSummaries(ctx context.Context, filter ...models.PipelineRunListFilter) ([]models.PipelineRun, int, error) {
	if uc.runRepo == nil {
		deps, err := uc.deploymentRepo.FindAll(ctx)
		if err != nil {
			return nil, 0, err
		}
		out := make([]models.PipelineRun, 0, len(deps))
		for i := range deps {
			run := uc.deploymentToRun(&deps[i])
			stripRunHeavyFields(run)
			out = append(out, *run)
		}
		return out, len(out), nil
	}
	if len(filter) > 0 && (filter[0].BatchJobID != "" || filter[0].ExcludeBatch || filter[0].Status != "" || filter[0].Page > 0 || filter[0].PageSize > 0) {
		items, total, err := uc.runRepo.ListSummaries(ctx, filter[0])
		if err != nil {
			return nil, 0, err
		}
		uc.refreshRunSummariesForList(ctx, items)
		if filter[0].BatchJobID != "" {
			uc.attachBatchNodeProgress(ctx, items)
		}
		return items, total, nil
	}
	items, err := uc.runRepo.FindAllSummaries(ctx)
	if err != nil {
		return nil, 0, err
	}
	uc.refreshRunSummariesForList(ctx, items)
	return items, len(items), nil
}

func (uc *Usecase) attachBatchNodeProgress(ctx context.Context, items []models.PipelineRun) {
	if uc.assetNodeRepo == nil || len(items) == 0 {
		return
	}
	runIDs := make([]string, 0, len(items))
	for i := range items {
		if items[i].ID != "" {
			runIDs = append(runIDs, items[i].ID)
		}
	}
	rows, err := uc.assetNodeRepo.ListByRunIDs(ctx, runIDs)
	if err != nil {
		return
	}
	progressByRun := batchprogress.ByRunID(rows, items)
	for i := range items {
		items[i].NodeProgress = progressByRun[items[i].ID]
	}
}

// ListBatchAssetRuns returns all pipeline runs for one asset within a batch job.
func (uc *Usecase) ListBatchAssetRuns(ctx context.Context, batchJobID, assetID string) ([]models.PipelineRun, error) {
	if uc.runRepo == nil {
		return nil, nil
	}
	return uc.runRepo.FindAllByBatchJobAndAssetID(ctx, batchJobID, assetID)
}

// ListAssetNodesByRunIDs returns asset-node rows for multiple runs.
func (uc *Usecase) ListAssetNodesByRunIDs(ctx context.Context, runIDs []string) ([]models.PipelineRunAssetNode, error) {
	if uc.assetNodeRepo == nil || len(runIDs) == 0 {
		return nil, nil
	}
	return uc.assetNodeRepo.ListByRunIDs(ctx, runIDs)
}

func stripRunHeavyFields(run *models.PipelineRun) {
	if run == nil {
		return
	}
	run.PipelineJSON = nil
	run.Manifest = nil
	run.TargetSnapshot = nil
	run.Nodes = nil
	run.ExecutionTarget = nil
}

// GetRunByWorkflowName returns a pipeline run by its Argo workflow name.
func (uc *Usecase) GetRunByWorkflowName(ctx context.Context, workflowName string) (*models.PipelineRun, error) {
	workflowName = strings.TrimSpace(workflowName)
	if workflowName == "" {
		return nil, nil
	}
	if uc.runRepo == nil {
		return nil, nil
	}
	run, err := uc.runRepo.FindByWorkflowName(ctx, workflowName)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, nil
	}
	uc.refreshPipelineRunStatus(ctx, run)
	uc.reconcileTerminalRunFromLedger(ctx, run)
	uc.reconcileMisclassifiedRunFromArgo(ctx, run)
	uc.enrichRun(ctx, run)
	return run, nil
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
	uc.reconcileTerminalRunFromLedger(ctx, run)
	uc.reconcileMisclassifiedRunFromArgo(ctx, run)
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
	if uc.runCostSnapshotMissing(run) {
		if isActiveDeploymentStatus(run.Status) {
			uc.refreshRunStatus(ctx, run)
		} else {
			uc.backfillRunStatus(ctx, run)
		}
		uc.enrichRun(ctx, run)
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
		if isWorkflowControlRunNode(node) {
			continue
		}
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
	return uc.SaveTemplate(ctx, name, d.PipelineJSON, "dev", "")
}

// ── Deployments ───────────────────────────────────────────────────

// maxActiveDeploymentStatusRefresh caps Argo status polls per ListDeployments call.
const maxActiveDeploymentStatusRefresh = 50

const deploymentStatusExpired = "Expired"

const (
	staleWorkflowTTLCleanupMessage = "Argo 工作流已被 TTL 清理"
	messageWorkflowAwaitingDeploy  = "等待绑定 Argo workflow"
	messageWorkflowUnavailable     = "Argo workflow 在集群中不可访问（可能已 TTL 清理）"
)

// staleActiveRunMaxAge is the maximum duration a run may stay in an active
// Argo phase before the watcher marks it failed as a zombie run.
const staleActiveRunMaxAge = 48 * time.Hour

// workflowCreateVisibilityGracePeriod avoids marking brand-new runs as expired
// while Argo is still creating the workflow CR.
const workflowCreateVisibilityGracePeriod = 5 * time.Minute

func isActiveDeploymentStatus(status string) bool {
	return status == "" || status == "Running" || status == "Pending" || status == "Unknown"
}

func shouldWaitForWorkflowCreation(run *models.PipelineRun, now time.Time) bool {
	if run == nil {
		return false
	}
	if strings.TrimSpace(run.WorkflowName) == "" {
		return true
	}
	if isPendingBatchWorkflowCreation(run) {
		return true
	}
	if !isActiveDeploymentStatus(run.Status) {
		return false
	}
	if strings.TrimSpace(run.ArgoWorkflowUID) != "" {
		return false
	}
	if run.CreatedAt.IsZero() {
		return false
	}
	return now.Sub(run.CreatedAt) < workflowCreateVisibilityGracePeriod
}

func isPendingBatchWorkflowCreation(run *models.PipelineRun) bool {
	if run == nil || run.BatchJobID == nil || strings.TrimSpace(*run.BatchJobID) == "" {
		return false
	}
	if strings.TrimSpace(run.ArgoWorkflowUID) != "" {
		return false
	}
	if isBatchSubtaskPlaceholderWorkflowName(run.WorkflowName) {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(run.Status), "Pending")
}

func isMisclassifiedTerminalRunStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "error", "expired":
		return true
	default:
		return false
	}
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

// ListDeployments returns all deployments without live Argo refresh (fast list).
func (uc *Usecase) ListDeployments(ctx context.Context) ([]models.PipelineDeployment, error) {
	return uc.listDeployments(ctx, false)
}

func (uc *Usecase) listDeployments(ctx context.Context, refreshActive bool) ([]models.PipelineDeployment, error) {
	if uc.runRepo != nil {
		runs, _, err := uc.ListRunSummaries(ctx)
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
	if refreshActive && uc.wfClient != nil {
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
