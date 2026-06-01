package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"errors"

	"log/slog"

	"github.com/CyberOrigin2077/cyber-databrew/internal/argo"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	"github.com/CyberOrigin2077/cyber-databrew/internal/transpiler"
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
	assetRepo      repository.AssetRepository
	assetEventRepo repository.AssetEventRepository
	relationWriter repository.AssetRelationWriter
	logicalRepo    repository.LogicalAssetRepository
	wfClient       argo.WorkflowClient
	namespace      string
}

type DeployOptions struct {
	DryRun     bool
	TemplateID string
	TargetID   string
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

func logPipelineSideEffect(op string, err error) {
	if err != nil {
		slog.Warn("pipeline side effect failed", "op", op, "err", err)
	}
}

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
func (uc *Usecase) ListExecutionTargets(_ context.Context) []models.ExecutionTarget {
	return []models.ExecutionTarget{uc.defaultExecutionTarget()}
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
		IsDefault:            true,
		Description:          "Current backend-configured Argo workflow namespace.",
	}
}

func (uc *Usecase) resolveExecutionTarget(targetID string) (*models.ExecutionTarget, error) {
	targetID = strings.TrimSpace(targetID)
	if targetID == "" || targetID == "default" {
		target := uc.defaultExecutionTarget()
		return &target, nil
	}
	return nil, fmt.Errorf("%w: target_id=%q", ErrExecutionTargetNotFound, targetID)
}

// ── Templates ─────────────────────────────────────────────────────

// SaveTemplate persists a pipeline template with auto-incremented version.
func (uc *Usecase) SaveTemplate(ctx context.Context, name string, pipeline map[string]interface{}) (*models.PipelineTemplate, error) {
	version, err := uc.templateRepo.GetNextVersion(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get next version: %w", err)
	}
	t := &models.PipelineTemplate{
		ID:        uuid.New().String(),
		Name:      name,
		Version:   version,
		Pipeline:  pipeline,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if nodes, ok := pipeline["nodes"].([]interface{}); ok {
		t.NodeCount = len(nodes)
	}
	if err := uc.templateRepo.Save(ctx, t); err != nil {
		return nil, fmt.Errorf("save template: %w", err)
	}
	return t, nil
}

// ListVersions returns all versions of a named pipeline template.
func (uc *Usecase) ListVersions(ctx context.Context, name string) ([]models.PipelineTemplate, error) {
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

	pipeName := pipe.Name
	if name != "" {
		pipeName = name
	}

	wfName := pipeName + "-" + uuid.New().String()[:6]
	depID := uuid.New().String()
	templateID := ""
	dryRun := false
	if len(opts) > 0 {
		templateID = opts[0].TemplateID
		dryRun = opts[0].DryRun
	}
	target, err := uc.resolveExecutionTarget("")
	if len(opts) > 0 {
		target, err = uc.resolveExecutionTarget(opts[0].TargetID)
	}
	if err != nil {
		return nil, err
	}

	// Validate all asset IDs exist before proceeding (T-12).
	if len(assetIDs) > 0 && uc.assetRepo != nil {
		for _, aid := range assetIDs {
			a, err := uc.assetRepo.Get(ctx, aid)
			if err != nil || a == nil {
				return nil, fmt.Errorf("%w: asset_id=%q", ErrAssetNotFound, aid)
			}
		}
	}

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
		Namespace:       uc.namespace,
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
		return &models.PipelineDeployment{
			ID:              depID,
			PipelineName:    pipeName,
			WorkflowName:    wfName,
			Status:          "Preview",
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
	if err := uc.wfClient.CreateWorkflow(ctx, wf, uc.namespace); err != nil {
		if strings.Contains(err.Error(), "argo server URL is empty") {
			return nil, fmt.Errorf("%w: create workflow", ErrWorkflowUnavailable)
		}
		return nil, fmt.Errorf("create workflow: %w", err)
	}
	phase, err := uc.wfClient.GetWorkflowStatus(ctx, wfName, uc.namespace)
	if err == nil && phase != "" {
		status = string(phase)
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
	// Embed input asset IDs into PipelineJSON for lineage queries (F4.7).
	if len(assetIDs) > 0 {
		pipelineArg["_input_asset_ids"] = assetIDs
	}

	if err := uc.deploymentRepo.Save(ctx, dep); err != nil {
		return nil, fmt.Errorf("save deployment: %w", err)
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
	if name == "" {
		name = t.Name
	}
	deployOpts := DeployOptions{TemplateID: templateID}
	if len(opts) > 0 {
		deployOpts.TargetID = opts[0].TargetID
	}
	return uc.Deploy(ctx, t.Pipeline, name, assetIDs, deployOpts)
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

// ListDeployments returns all deployments, optionally refreshing active statuses.
func (uc *Usecase) ListDeployments(ctx context.Context) ([]models.PipelineDeployment, error) {
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
			if list[i].Status == "" || list[i].Status == "Running" || list[i].Status == "Pending" || list[i].Status == "Unknown" {
				refreshed++
				phase, err := uc.wfClient.GetWorkflowStatus(ctx, list[i].WorkflowName, uc.namespace)
				if err == nil {
					list[i].Status = string(phase)
					if phase == "Succeeded" || phase == "Failed" || phase == "Error" {
						now := time.Now().UTC()
						list[i].FinishedAt = &now
					}
					logPipelineSideEffect("update deployment status", uc.deploymentRepo.UpdateStatus(ctx, list[i].ID, string(phase)))
				}
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
		return nil, nil
	}
	if uc.wfClient != nil && (d.Status == "" || d.Status == "Running" || d.Status == "Pending" || d.Status == "Unknown") {
		phase, err := uc.wfClient.GetWorkflowStatus(ctx, d.WorkflowName, uc.namespace)
		if err == nil {
			d.Status = string(phase)
			if phase == "Succeeded" || phase == "Failed" || phase == "Error" {
				now := time.Now().UTC()
				d.FinishedAt = &now
			}
			logPipelineSideEffect("update deployment status", uc.deploymentRepo.UpdateStatus(ctx, d.ID, string(phase)))
		}
	}
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
	DeploymentID string             `json:"deployment_id"`
	WorkflowName string             `json:"workflow_name"`
	Status       string             `json:"status"`
	Pods         []PodResourceUsage `json:"pods"`
}

// PodResourceUsage summarizes per-pod runtime duration and template resource requests.
type PodResourceUsage struct {
	PodName       string `json:"pod_name"`
	NodeName      string `json:"node_name,omitempty"`
	CPUUsage      string `json:"cpu_usage"`
	MemoryUsage   string `json:"memory_usage"`
	CPURequest    string `json:"cpu_request"`
	MemoryRequest string `json:"memory_request"`
	CPULimit      string `json:"cpu_limit"`
	MemoryLimit   string `json:"memory_limit"`
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
		Pods:         []PodResourceUsage{},
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
			report.Pods = buildPodResourceUsageReport(wf, d.Manifest)
		}
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
