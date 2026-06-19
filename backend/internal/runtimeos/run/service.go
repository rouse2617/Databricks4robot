package run

import (
	"context"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
)

// Service is the Run Kernel boundary for product execution operations. Phase 1
// delegates to the existing pipeline usecase while routes and UI move to Run
// semantics.
type Service interface {
	CreateRun(ctx context.Context, pipeline map[string]interface{}, name string, assetIDs []string, opts ...pipelineUC.DeployOptions) (*models.PipelineRun, error)
	CreateRunByTemplateID(ctx context.Context, templateID, name string, assetIDs []string, opts ...pipelineUC.DeployOptions) (*models.PipelineRun, error)
	CreateRunsByTemplateID(ctx context.Context, templateID, name string, assetIDs []string, opts ...pipelineUC.DeployOptions) ([]models.PipelineRun, error)
	ListRuns(ctx context.Context, refreshActive bool) ([]models.PipelineRun, error)
	ListRunSummaries(ctx context.Context, filter ...models.PipelineRunListFilter) ([]models.PipelineRun, int, error)
	GetRunByWorkflowName(ctx context.Context, workflowName string) (*models.PipelineRun, error)
	GetRun(ctx context.Context, id string) (*models.PipelineRun, error)
	DeleteRun(ctx context.Context, id string) error
	RetryRun(ctx context.Context, id string) (*models.PipelineRun, error)
	RuntimeRetryRun(ctx context.Context, id string) (*models.PipelineRun, error)
	ResubmitRun(ctx context.Context, id string) (*models.PipelineRun, error)
	StopRun(ctx context.Context, id string) error
	SuspendRun(ctx context.Context, id string) error
	ResumeRun(ctx context.Context, id string) error
	TerminateRun(ctx context.Context, id string) error
	ListRunEvents(ctx context.Context, id string, opts models.PipelineRunEventListOptions) (*models.PipelineRunEventListResult, error)
	ListRunAssetNodes(ctx context.Context, id string, opts models.PipelineRunAssetNodeListOptions) (*models.PipelineRunAssetNodeListResult, error)
	GetRunCostSummary(ctx context.Context, id string) (*models.PipelineRunCostSummary, error)
	ListRunNodes(ctx context.Context, id string) ([]models.PipelineRunNode, error)
	ListRunInputs(ctx context.Context, id string) (*models.RunInputList, error)
	ListRunOutputs(ctx context.Context, id string) (*models.RunOutputList, error)
	ListRunChildren(ctx context.Context, id string) (*models.RunChildList, error)
	GetRunRuntime(ctx context.Context, id string) (*models.RunRuntime, error)
	GetRunWatcherStatus(ctx context.Context) (*models.PipelineRunWatcherState, error)
}

// PipelineUsecase is the existing pipeline application boundary that still owns
// Run persistence and Argo orchestration during the phase 1 migration.
type PipelineUsecase interface {
	CreateRun(ctx context.Context, pipeline map[string]interface{}, name string, assetIDs []string, opts ...pipelineUC.DeployOptions) (*models.PipelineRun, error)
	CreateRunByTemplateID(ctx context.Context, templateID, name string, assetIDs []string, opts ...pipelineUC.DeployOptions) (*models.PipelineRun, error)
	CreateRunsByTemplateID(ctx context.Context, templateID, name string, assetIDs []string, opts ...pipelineUC.DeployOptions) ([]models.PipelineRun, error)
	ListRuns(ctx context.Context, refreshActive bool) ([]models.PipelineRun, error)
	ListRunSummaries(ctx context.Context, filter ...models.PipelineRunListFilter) ([]models.PipelineRun, int, error)
	GetRunByWorkflowName(ctx context.Context, workflowName string) (*models.PipelineRun, error)
	GetRun(ctx context.Context, id string) (*models.PipelineRun, error)
	DeleteRun(ctx context.Context, id string) error
	RetryRun(ctx context.Context, id string) (*models.PipelineRun, error)
	RuntimeRetryRun(ctx context.Context, id string) (*models.PipelineRun, error)
	ResubmitRun(ctx context.Context, id string) (*models.PipelineRun, error)
	StopRun(ctx context.Context, id string) error
	SuspendRun(ctx context.Context, id string) error
	ResumeRun(ctx context.Context, id string) error
	TerminateRun(ctx context.Context, id string) error
	ListRunEvents(ctx context.Context, id string, opts models.PipelineRunEventListOptions) (*models.PipelineRunEventListResult, error)
	ListRunAssetNodes(ctx context.Context, id string, opts models.PipelineRunAssetNodeListOptions) (*models.PipelineRunAssetNodeListResult, error)
	GetRunCostSummary(ctx context.Context, id string) (*models.PipelineRunCostSummary, error)
	ListRunNodes(ctx context.Context, id string) ([]models.PipelineRunNode, error)
	ListRunInputs(ctx context.Context, id string) (*models.RunInputList, error)
	ListRunOutputs(ctx context.Context, id string) (*models.RunOutputList, error)
	ListRunChildren(ctx context.Context, id string) (*models.RunChildList, error)
	GetRunRuntime(ctx context.Context, id string) (*models.RunRuntime, error)
	GetRunWatcherStatus(ctx context.Context) (*models.PipelineRunWatcherState, error)
}

type service struct {
	pipeline PipelineUsecase
}

// NewService adapts the current pipeline usecase into the Run Kernel boundary.
func NewService(pipeline PipelineUsecase) Service {
	return &service{pipeline: pipeline}
}

func (s *service) CreateRun(ctx context.Context, pipeline map[string]interface{}, name string, assetIDs []string, opts ...pipelineUC.DeployOptions) (*models.PipelineRun, error) {
	return s.pipeline.CreateRun(ctx, pipeline, name, assetIDs, opts...)
}

func (s *service) CreateRunByTemplateID(ctx context.Context, templateID, name string, assetIDs []string, opts ...pipelineUC.DeployOptions) (*models.PipelineRun, error) {
	return s.pipeline.CreateRunByTemplateID(ctx, templateID, name, assetIDs, opts...)
}

func (s *service) CreateRunsByTemplateID(ctx context.Context, templateID, name string, assetIDs []string, opts ...pipelineUC.DeployOptions) ([]models.PipelineRun, error) {
	return s.pipeline.CreateRunsByTemplateID(ctx, templateID, name, assetIDs, opts...)
}

func (s *service) ListRuns(ctx context.Context, refreshActive bool) ([]models.PipelineRun, error) {
	return s.pipeline.ListRuns(ctx, refreshActive)
}

func (s *service) ListRunSummaries(ctx context.Context, filter ...models.PipelineRunListFilter) ([]models.PipelineRun, int, error) {
	return s.pipeline.ListRunSummaries(ctx, filter...)
}

func (s *service) GetRunByWorkflowName(ctx context.Context, workflowName string) (*models.PipelineRun, error) {
	return s.pipeline.GetRunByWorkflowName(ctx, workflowName)
}

func (s *service) GetRun(ctx context.Context, id string) (*models.PipelineRun, error) {
	return s.pipeline.GetRun(ctx, id)
}

func (s *service) DeleteRun(ctx context.Context, id string) error {
	return s.pipeline.DeleteRun(ctx, id)
}

func (s *service) RetryRun(ctx context.Context, id string) (*models.PipelineRun, error) {
	return s.pipeline.RetryRun(ctx, id)
}

func (s *service) RuntimeRetryRun(ctx context.Context, id string) (*models.PipelineRun, error) {
	return s.pipeline.RuntimeRetryRun(ctx, id)
}

func (s *service) ResubmitRun(ctx context.Context, id string) (*models.PipelineRun, error) {
	return s.pipeline.ResubmitRun(ctx, id)
}

func (s *service) StopRun(ctx context.Context, id string) error {
	return s.pipeline.StopRun(ctx, id)
}

func (s *service) SuspendRun(ctx context.Context, id string) error {
	return s.pipeline.SuspendRun(ctx, id)
}

func (s *service) ResumeRun(ctx context.Context, id string) error {
	return s.pipeline.ResumeRun(ctx, id)
}

func (s *service) TerminateRun(ctx context.Context, id string) error {
	return s.pipeline.TerminateRun(ctx, id)
}

func (s *service) ListRunEvents(ctx context.Context, id string, opts models.PipelineRunEventListOptions) (*models.PipelineRunEventListResult, error) {
	return s.pipeline.ListRunEvents(ctx, id, opts)
}

func (s *service) ListRunAssetNodes(ctx context.Context, id string, opts models.PipelineRunAssetNodeListOptions) (*models.PipelineRunAssetNodeListResult, error) {
	return s.pipeline.ListRunAssetNodes(ctx, id, opts)
}

func (s *service) GetRunCostSummary(ctx context.Context, id string) (*models.PipelineRunCostSummary, error) {
	return s.pipeline.GetRunCostSummary(ctx, id)
}

func (s *service) ListRunNodes(ctx context.Context, id string) ([]models.PipelineRunNode, error) {
	return s.pipeline.ListRunNodes(ctx, id)
}

func (s *service) ListRunInputs(ctx context.Context, id string) (*models.RunInputList, error) {
	return s.pipeline.ListRunInputs(ctx, id)
}

func (s *service) ListRunOutputs(ctx context.Context, id string) (*models.RunOutputList, error) {
	return s.pipeline.ListRunOutputs(ctx, id)
}

func (s *service) ListRunChildren(ctx context.Context, id string) (*models.RunChildList, error) {
	return s.pipeline.ListRunChildren(ctx, id)
}

func (s *service) GetRunRuntime(ctx context.Context, id string) (*models.RunRuntime, error) {
	return s.pipeline.GetRunRuntime(ctx, id)
}

func (s *service) GetRunWatcherStatus(ctx context.Context) (*models.PipelineRunWatcherState, error) {
	return s.pipeline.GetRunWatcherStatus(ctx)
}
