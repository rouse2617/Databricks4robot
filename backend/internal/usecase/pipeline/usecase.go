package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"errors"

	"github.com/CyberOrigin2077/cyber-databrew/internal/k8s"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	"github.com/CyberOrigin2077/cyber-databrew/internal/transpiler"
)

// Sentinel errors.
var ErrTemplateNotFound = errors.New("template not found")

// Usecase orchestrates pipeline template management and deployment.
type Usecase struct {
	templateRepo   repository.PipelineTemplateRepository
	deploymentRepo repository.PipelineDeploymentRepository
	wfClient       k8s.WorkflowClient
	namespace      string
}

// New creates a Usecase. wfClient may be nil (no K8s/Argo integration).
func New(
	templateRepo repository.PipelineTemplateRepository,
	deploymentRepo repository.PipelineDeploymentRepository,
	wfClient k8s.WorkflowClient,
	namespace string,
) *Usecase {
	return &Usecase{
		templateRepo:   templateRepo,
		deploymentRepo: deploymentRepo,
		wfClient:       wfClient,
		namespace:      namespace,
	}
}

// ── Templates ─────────────────────────────────────────────────────

// SaveTemplate persists a pipeline template.
func (uc *Usecase) SaveTemplate(ctx context.Context, name string, pipeline map[string]interface{}) (*models.PipelineTemplate, error) {
	t := &models.PipelineTemplate{
		ID:        uuid.New().String(),
		Name:      name,
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
func (uc *Usecase) Deploy(ctx context.Context, pipelineArg map[string]interface{}, name string) (*models.PipelineDeployment, error) {
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

	// Transpile to Argo Workflow.
	opts := &transpiler.Options{
		Name:            wfName,
		Namespace:       uc.namespace,
		TTLSecondsAfter: 3600,
	}
	wf, err := transpiler.Transpile(pipe, opts)
	if err != nil {
		return nil, fmt.Errorf("transpile: %w", err)
	}

	// Submit to K8s/Argo if client is available.
	status := "Pending"
	if uc.wfClient != nil {
		if err := uc.wfClient.CreateWorkflow(ctx, wf, uc.namespace); err != nil {
			return nil, fmt.Errorf("create workflow: %w", err)
		}
		phase, err := uc.wfClient.GetWorkflowStatus(ctx, wfName, uc.namespace)
		if err == nil {
			status = string(phase)
		}
	}

	manifest := ""
	if m, err := json.MarshalIndent(wf, "", "  "); err == nil {
		manifest = string(m)
	}

	dep := &models.PipelineDeployment{
		ID:           uuid.New().String(),
		PipelineName: pipeName,
		WorkflowName: wfName,
		Status:       status,
		NodeCount:    nodeCount,
		Manifest:     &manifest,
		PipelineJSON: pipelineArg,
		CreatedAt:    time.Now().UTC(),
	}
	if err := uc.deploymentRepo.Save(ctx, dep); err != nil {
		return nil, fmt.Errorf("save deployment: %w", err)
	}
	return dep, nil
}

// DeployByTemplateID deploys a saved template.
func (uc *Usecase) DeployByTemplateID(ctx context.Context, templateID, name string) (*models.PipelineDeployment, error) {
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
	return uc.Deploy(ctx, t.Pipeline, name)
}

// ── Deployments ───────────────────────────────────────────────────

// ListDeployments returns all deployments, optionally refreshing active statuses.
func (uc *Usecase) ListDeployments(ctx context.Context) ([]models.PipelineDeployment, error) {
	list, err := uc.deploymentRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	// Refresh status for active workflows.
	if uc.wfClient != nil {
		for i := range list {
			if list[i].Status == "Running" || list[i].Status == "Pending" || list[i].Status == "Unknown" {
				phase, err := uc.wfClient.GetWorkflowStatus(ctx, list[i].WorkflowName, uc.namespace)
				if err == nil {
					list[i].Status = string(phase)
					if phase == "Succeeded" || phase == "Failed" || phase == "Error" {
						now := time.Now().UTC()
						list[i].FinishedAt = &now
					}
					_ = uc.deploymentRepo.UpdateStatus(ctx, list[i].ID, string(phase))
				}
			}
		}
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
	if uc.wfClient != nil && (d.Status == "Running" || d.Status == "Pending" || d.Status == "Unknown") {
		phase, err := uc.wfClient.GetWorkflowStatus(ctx, d.WorkflowName, uc.namespace)
		if err == nil {
			d.Status = string(phase)
			if phase == "Succeeded" || phase == "Failed" || phase == "Error" {
				now := time.Now().UTC()
				d.FinishedAt = &now
			}
			_ = uc.deploymentRepo.UpdateStatus(ctx, d.ID, string(phase))
		}
	}
	return d, nil
}

// DeleteDeployment removes a deployment record and optionally deletes the K8s workflow.
func (uc *Usecase) DeleteDeployment(ctx context.Context, id string) error {
	if uc.wfClient != nil {
		d, err := uc.deploymentRepo.FindByID(ctx, id)
		if err == nil && d != nil {
			_ = uc.wfClient.DeleteWorkflow(ctx, d.WorkflowName, uc.namespace)
		}
	}
	return uc.deploymentRepo.Delete(ctx, id)
}

// ── Helpers ───────────────────────────────────────────────────────

func rawToPipeline(raw json.RawMessage) (*transpiler.Pipeline, int, error) {
	var pipe transpiler.Pipeline
	if err := json.Unmarshal(raw, &pipe); err != nil {
		return nil, 0, err
	}
	return &pipe, len(pipe.Nodes), nil
}
