package runs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/google/uuid"

	"github.com/CyberOrigin2077/cyber-databrew/internal/argo"
	"github.com/CyberOrigin2077/cyber-databrew/internal/k8s"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
)

var ErrRunNotFound = errors.New("run not found")

type Usecase struct {
	runRepo            repository.DatabrewRunRepository
	componentBuildRepo repository.ComponentBuildRunRepository
	ragBuildRepo       repository.RAGBuildRunRepository
	componentReleaseRepo repository.ComponentReleaseRepository
	pipelineUC         *pipelineUC.Usecase
	wfClient           argo.WorkflowClient
	podClient          k8s.PodClient
	defaultNamespace   string
}

func New(
	runRepo repository.DatabrewRunRepository,
	componentBuildRepo repository.ComponentBuildRunRepository,
	ragBuildRepo repository.RAGBuildRunRepository,
	componentReleaseRepo repository.ComponentReleaseRepository,
	pipeline *pipelineUC.Usecase,
	wfClient argo.WorkflowClient,
	podClient k8s.PodClient,
	defaultNamespace string,
) *Usecase {
	return &Usecase{
		runRepo:              runRepo,
		componentBuildRepo:   componentBuildRepo,
		ragBuildRepo:         ragBuildRepo,
		componentReleaseRepo: componentReleaseRepo,
		pipelineUC:           pipeline,
		wfClient:             wfClient,
		podClient:            podClient,
		defaultNamespace:     defaultNamespace,
	}
}

func (uc *Usecase) SetPodClient(podClient k8s.PodClient) {
	uc.podClient = podClient
}

type CreateComponentBuildInput struct {
	ComponentID     string
	ComponentName   string
	RepoURL         string
	GitRef          string
	Dockerfile      string
	BuildContext    string
	ImageRepository string
	ImageTag        string
}

type CreateRAGBuildInput struct {
	KnowledgeBaseID string
	EmbeddingModel  string
	VectorIndexName string
	ReleaseVersion  string
	Datasource      map[string]interface{}
}

func (uc *Usecase) ListRuns(ctx context.Context, filter models.DatabrewRunListFilter) (*models.DatabrewRunListResult, error) {
	items, total, err := uc.runRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	for i := range items {
		uc.enrichRunView(ctx, &items[i])
	}
	return &models.DatabrewRunListResult{Items: items, Total: total}, nil
}

func (uc *Usecase) GetRun(ctx context.Context, id string) (*models.DatabrewRun, error) {
	run, err := uc.runRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrRunNotFound
	}
	uc.enrichRunView(ctx, run)
	return run, nil
}

func (uc *Usecase) GetRunByWorkflowName(ctx context.Context, workflowName string) (*models.DatabrewRun, error) {
	run, err := uc.runRepo.FindByRuntimeResource(ctx, uc.defaultNamespace, workflowName)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrRunNotFound
	}
	uc.enrichRunView(ctx, run)
	return run, nil
}

func (uc *Usecase) GetRuntime(ctx context.Context, id string) (map[string]interface{}, error) {
	run, err := uc.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	if uc.wfClient == nil {
		return nil, fmt.Errorf("workflow client not available")
	}
	namespace := run.RuntimeNamespace
	if namespace == "" {
		namespace = uc.defaultNamespace
	}
	wf, err := uc.wfClient.GetWorkflow(ctx, run.RuntimeResourceName, namespace)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(wf)
	if err != nil {
		return nil, err
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (uc *Usecase) GetNodes(ctx context.Context, id string) ([]models.PipelineRunNode, error) {
	if uc.pipelineUC == nil {
		return nil, fmt.Errorf("pipeline usecase not available")
	}
	run, err := uc.pipelineUC.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrRunNotFound
	}
	if run.Nodes == nil {
		return []models.PipelineRunNode{}, nil
	}
	return run.Nodes, nil
}

func (uc *Usecase) GetEvents(ctx context.Context, id string, opts models.PipelineRunEventListOptions) (*models.PipelineRunEventListResult, error) {
	if uc.pipelineUC == nil {
		return nil, fmt.Errorf("pipeline usecase not available")
	}
	return uc.pipelineUC.ListRunEvents(ctx, id, opts)
}

func (uc *Usecase) GetArtifacts(ctx context.Context, id string) ([]map[string]interface{}, error) {
	runtime, err := uc.GetRuntime(ctx, id)
	if err != nil {
		return nil, err
	}
	status, _ := runtime["status"].(map[string]interface{})
	if status == nil {
		return []map[string]interface{}{}, nil
	}
	artifacts := make([]map[string]interface{}, 0)
	if nodes, ok := status["nodes"].(map[string]interface{}); ok {
		for nodeID, raw := range nodes {
			node, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			outputs, _ := node["outputs"].(map[string]interface{})
			if outputs == nil {
				continue
			}
			rawArtifacts, _ := outputs["artifacts"].([]interface{})
			for _, item := range rawArtifacts {
				artifact, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				entry := map[string]interface{}{
					"nodeId": nodeID,
				}
				for k, v := range artifact {
					entry[k] = v
				}
				artifacts = append(artifacts, entry)
			}
		}
	}
	return artifacts, nil
}

func (uc *Usecase) GetPods(ctx context.Context, id string) ([]map[string]interface{}, error) {
	runtime, err := uc.GetRuntime(ctx, id)
	if err != nil {
		return nil, err
	}
	status, _ := runtime["status"].(map[string]interface{})
	if status == nil {
		return []map[string]interface{}{}, nil
	}
	nodes, _ := status["nodes"].(map[string]interface{})
	pods := make([]map[string]interface{}, 0)
	seen := map[string]bool{}
	for nodeID, raw := range nodes {
		node, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		podName, _ := node["podName"].(string)
		if podName == "" || seen[podName] {
			continue
		}
		seen[podName] = true
		entry := map[string]interface{}{
			"nodeId":      nodeID,
			"podName":     podName,
			"displayName": node["displayName"],
			"phase":       node["phase"],
			"message":     node["message"],
			"startedAt":   node["startedAt"],
			"finishedAt":  node["finishedAt"],
		}
		if uc.podClient != nil {
			run, _ := uc.GetRun(ctx, id)
			namespace := uc.defaultNamespace
			if run != nil && run.RuntimeNamespace != "" {
				namespace = run.RuntimeNamespace
			}
			if diag, podErr := uc.podClient.GetPodDiagnostics(ctx, namespace, podName); podErr == nil && diag != nil {
				entry["podIp"] = diag.PodIP
				entry["restartCount"] = diag.RestartCount
				entry["conditions"] = diag.Conditions
				entry["containers"] = diag.Containers
				entry["events"] = diag.Events
				entry["serviceAccountName"] = diag.ServiceAccountName
			}
		}
		pods = append(pods, entry)
	}
	return pods, nil
}

func (uc *Usecase) CreateComponentBuild(ctx context.Context, in CreateComponentBuildInput) (*models.DatabrewRun, error) {
	if in.ComponentName == "" {
		return nil, fmt.Errorf("component name is required")
	}
	if in.RepoURL == "" {
		return nil, fmt.Errorf("repo url is required")
	}
	runName := fmt.Sprintf("build-%s", in.ComponentName)
	workflowName := fmt.Sprintf("%s-%d", runName, time.Now().UTC().Unix())
	run := &models.DatabrewRun{
		Type:                models.RunTypeComponentBuild,
		Name:                runName,
		Status:              string(wfv1.WorkflowPending),
		Runtime:             "argo",
		RuntimeNamespace:    uc.defaultNamespace,
		RuntimeResourceName: workflowName,
	}
	if err := uc.runRepo.Save(ctx, run); err != nil {
		return nil, err
	}
	ext := &models.ComponentBuildRun{
		RunID:           run.ID,
		ComponentID:     in.ComponentID,
		RepoURL:         in.RepoURL,
		GitRef:          in.GitRef,
		Dockerfile:      in.Dockerfile,
		BuildContext:    in.BuildContext,
		ImageRepository: in.ImageRepository,
		ImageTag:        in.ImageTag,
	}
	if ext.Dockerfile == "" {
		ext.Dockerfile = "Dockerfile"
	}
	if ext.BuildContext == "" {
		ext.BuildContext = "."
	}
	if err := uc.componentBuildRepo.Save(ctx, ext); err != nil {
		return nil, err
	}
	if err := uc.submitBuildWorkflow(ctx, run, buildComponentBuildWorkflow(workflowName, uc.defaultNamespace, run.ID, in)); err != nil {
		return nil, err
	}
	uc.enrichRunView(ctx, run)
	return run, nil
}

func (uc *Usecase) CreateRAGBuild(ctx context.Context, in CreateRAGBuildInput) (*models.DatabrewRun, error) {
	if in.KnowledgeBaseID == "" {
		return nil, fmt.Errorf("knowledge base id is required")
	}
	runName := fmt.Sprintf("rag-build-%s", in.KnowledgeBaseID)
	workflowName := fmt.Sprintf("%s-%d", runName, time.Now().UTC().Unix())
	run := &models.DatabrewRun{
		Type:                models.RunTypeRAGBuild,
		Name:                runName,
		Status:              string(wfv1.WorkflowPending),
		Runtime:             "argo",
		RuntimeNamespace:    uc.defaultNamespace,
		RuntimeResourceName: workflowName,
	}
	if err := uc.runRepo.Save(ctx, run); err != nil {
		return nil, err
	}
	ext := &models.RAGBuildRun{
		RunID:              run.ID,
		KnowledgeBaseID:    in.KnowledgeBaseID,
		DatasourceSnapshot: in.Datasource,
		EmbeddingModel:     in.EmbeddingModel,
		VectorIndexName:    in.VectorIndexName,
		ReleaseVersion:     in.ReleaseVersion,
	}
	if err := uc.ragBuildRepo.Save(ctx, ext); err != nil {
		return nil, err
	}
	if err := uc.submitBuildWorkflow(ctx, run, buildRAGBuildWorkflow(workflowName, uc.defaultNamespace, run.ID, in)); err != nil {
		return nil, err
	}
	uc.enrichRunView(ctx, run)
	return run, nil
}

func (uc *Usecase) submitBuildWorkflow(ctx context.Context, run *models.DatabrewRun, wf *wfv1.Workflow) error {
	if uc.wfClient == nil {
		return fmt.Errorf("workflow client not available")
	}
	namespace := run.RuntimeNamespace
	if namespace == "" {
		namespace = uc.defaultNamespace
	}
	if err := uc.wfClient.CreateWorkflow(ctx, wf, namespace); err != nil {
		return fmt.Errorf("create workflow: %w", err)
	}
	if wfDetail, err := uc.wfClient.GetWorkflow(ctx, run.RuntimeResourceName, namespace); err == nil && wfDetail != nil {
		run.RuntimeUID = string(wfDetail.UID)
		if wfDetail.Status.Phase != "" {
			run.Status = string(wfDetail.Status.Phase)
		}
		_ = uc.runRepo.Save(ctx, run)
	}
	return nil
}

func (uc *Usecase) RecordComponentRelease(ctx context.Context, runID string) error {
	run, err := uc.GetRun(ctx, runID)
	if err != nil {
		return err
	}
	if run.Type != models.RunTypeComponentBuild {
		return fmt.Errorf("run is not a component build")
	}
	ext, err := uc.componentBuildRepo.FindByRunID(ctx, runID)
	if err != nil || ext == nil {
		return fmt.Errorf("component build extension not found")
	}
	if uc.componentReleaseRepo == nil {
		return nil
	}
	image := ext.ImageRepository
	if ext.ImageTag != "" && image != "" && !strings.Contains(image, ":") {
		image = fmt.Sprintf("%s:%s", image, ext.ImageTag)
	}
	release := &models.ComponentRelease{
		ID:           uuid.New().String(),
		ComponentID:  ext.ComponentID,
		SourceCommit: ext.CommitSHA,
		Image:        image,
		ImageTag:     ext.ImageTag,
		ImageDigest:  ext.ImageDigest,
		ReleaseLabel: ext.ImageTag,
		BuildRunID:   &runID,
	}
	return uc.componentReleaseRepo.Save(ctx, release)
}

func (uc *Usecase) RetryRun(ctx context.Context, id string) (*models.PipelineRun, error) {
	run, err := uc.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	if run.Type != models.RunTypePipeline {
		return nil, fmt.Errorf("retry is only supported for pipeline runs")
	}
	return uc.pipelineUC.RetryRun(ctx, id)
}

func (uc *Usecase) StopRun(ctx context.Context, id string) error {
	run, err := uc.GetRun(ctx, id)
	if err != nil {
		return err
	}
	if run.Type == models.RunTypePipeline && uc.pipelineUC != nil {
		return uc.pipelineUC.StopRun(ctx, id)
	}
	return uc.directWorkflowOp(ctx, run, uc.wfClient.StopWorkflow)
}

func (uc *Usecase) SuspendRun(ctx context.Context, id string) error {
	run, err := uc.GetRun(ctx, id)
	if err != nil {
		return err
	}
	if run.Type == models.RunTypePipeline && uc.pipelineUC != nil {
		return uc.pipelineUC.SuspendRun(ctx, id)
	}
	return uc.directWorkflowOp(ctx, run, uc.wfClient.SuspendWorkflow)
}

func (uc *Usecase) ResumeRun(ctx context.Context, id string) error {
	run, err := uc.GetRun(ctx, id)
	if err != nil {
		return err
	}
	if run.Type == models.RunTypePipeline && uc.pipelineUC != nil {
		return uc.pipelineUC.ResumeRun(ctx, id)
	}
	return uc.directWorkflowOp(ctx, run, uc.wfClient.ResumeWorkflow)
}

func (uc *Usecase) ResubmitRun(ctx context.Context, id string) error {
	run, err := uc.GetRun(ctx, id)
	if err != nil {
		return err
	}
	if run.Type == models.RunTypePipeline && uc.pipelineUC != nil {
		_, err = uc.pipelineUC.ResubmitRun(ctx, id)
		return err
	}
	return uc.directWorkflowOp(ctx, run, uc.wfClient.ResubmitWorkflow)
}

func (uc *Usecase) TerminateRun(ctx context.Context, id string) error {
	run, err := uc.GetRun(ctx, id)
	if err != nil {
		return err
	}
	if run.Type == models.RunTypePipeline && uc.pipelineUC != nil {
		return uc.pipelineUC.TerminateRun(ctx, id)
	}
	return uc.directWorkflowOp(ctx, run, uc.wfClient.TerminateWorkflow)
}

type workflowOp func(context.Context, string, string) error

func (uc *Usecase) directWorkflowOp(ctx context.Context, run *models.DatabrewRun, op workflowOp) error {
	if uc.wfClient == nil {
		return fmt.Errorf("workflow client not available")
	}
	namespace := run.RuntimeNamespace
	if namespace == "" {
		namespace = uc.defaultNamespace
	}
	return op(ctx, run.RuntimeResourceName, namespace)
}

func (uc *Usecase) enrichRunView(ctx context.Context, run *models.DatabrewRun) {
	// Refresh pipeline status and sync back to the in-memory DatabrewRun.
	// This ensures the list view reflects the latest Argo workflow state
	// instead of a previously-persisted stale status like "Error".
	pipelineRun := uc.refreshPipelineStatus(ctx, run)
	run.StatusLabel = statusLabel(run.Status)
	run.Actions = runActionsForStatus(run.Status)
	run.Summary = uc.buildSummary(ctx, run, pipelineRun)
}

// refreshPipelineStatus fetches the latest pipeline run from the pipeline
// usecase and syncs status/message/finishedAt back to the DatabrewRun.
// Returns the PipelineRun so buildSummary can reuse it without a second fetch.
func (uc *Usecase) refreshPipelineStatus(ctx context.Context, run *models.DatabrewRun) *models.PipelineRun {
	if uc.pipelineUC == nil {
		return nil
	}
	pipelineRun, err := uc.pipelineUC.GetRun(ctx, run.ID)
	if err != nil || pipelineRun == nil {
		return nil
	}
	if pipelineRun.Status != run.Status {
		run.Status = pipelineRun.Status
		run.Message = pipelineRun.Message
		run.FinishedAt = pipelineRun.FinishedAt
	}
	return pipelineRun
}

func (uc *Usecase) buildSummary(ctx context.Context, run *models.DatabrewRun, pipelineRun *models.PipelineRun) any {
	switch run.Type {
	case models.RunTypeComponentBuild:
		ext, err := uc.componentBuildRepo.FindByRunID(ctx, run.ID)
		if err != nil || ext == nil {
			return map[string]string{}
		}
		return map[string]string{
			"component": ext.ComponentID,
			"commit":    ext.CommitSHA,
			"imageTag":  ext.ImageTag,
		}
	case models.RunTypeRAGBuild:
		ext, err := uc.ragBuildRepo.FindByRunID(ctx, run.ID)
		if err != nil || ext == nil {
			return map[string]string{}
		}
		return map[string]string{
			"knowledgeBaseId": ext.KnowledgeBaseID,
			"embeddingModel":  ext.EmbeddingModel,
			"releaseVersion":  ext.ReleaseVersion,
		}
	default:
		if pipelineRun == nil {
			return map[string]interface{}{}
		}
		return map[string]interface{}{
			"pipelineName":    pipelineRun.PipelineName,
			"templateId":      pipelineRun.TemplateID,
			"templateVersion": pipelineRun.TemplateVersion,
			"assetCount":      pipelineRun.AssetCount,
			"nodeCount":       pipelineRun.NodeCount,
		}
	}
}

func statusLabel(status string) string {
	switch status {
	case string(wfv1.WorkflowRunning):
		return "运行中"
	case string(wfv1.WorkflowSucceeded):
		return "成功"
	case string(wfv1.WorkflowFailed):
		return "失败"
	case string(wfv1.WorkflowError):
		return "错误"
	case string(wfv1.WorkflowPending):
		return "等待中"
	case "Suspended":
		return "已暂停"
	case "Terminated":
		return "已终止"
	default:
		return status
	}
}

func runActionsForStatus(status string) *models.RunActions {
	actions := &models.RunActions{}
	switch status {
	case string(wfv1.WorkflowRunning), string(wfv1.WorkflowPending):
		actions.CanStop = true
		actions.CanSuspend = true
		actions.CanTerminate = true
	case "Suspended":
		actions.CanResume = true
		actions.CanTerminate = true
	case string(wfv1.WorkflowFailed), string(wfv1.WorkflowError):
		actions.CanRetry = true
		actions.CanResubmit = true
	case string(wfv1.WorkflowSucceeded):
		actions.CanResubmit = true
	}
	return actions
}
