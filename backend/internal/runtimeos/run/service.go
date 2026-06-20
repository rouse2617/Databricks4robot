package run

import (
	"context"
	"fmt"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
)

// LifecycleOperation names product Run lifecycle controls. Runtime-specific
// execution remains behind the pipeline/runtime adapter boundary.
type LifecycleOperation string

const (
	LifecycleRuntimeRetry LifecycleOperation = "runtime_retry"
	LifecycleStop         LifecycleOperation = "stop"
	LifecycleSuspend      LifecycleOperation = "suspend"
	LifecycleResume       LifecycleOperation = "resume"
	LifecycleTerminate    LifecycleOperation = "terminate"
)

// Service is the Run Kernel boundary for product execution operations. It
// keeps product routes Run-centric while the existing pipeline usecase still
// owns persistence during the incremental Runtime OS migration.
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
	RerunRun(ctx context.Context, id string) (*models.PipelineRun, error)
	ControlRun(ctx context.Context, id string, op LifecycleOperation) (*models.PipelineRun, error)
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
	RerunRun(ctx context.Context, id string) (*models.PipelineRun, error)
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
	return s.ControlRun(ctx, id, LifecycleRuntimeRetry)
}

func (s *service) ResubmitRun(ctx context.Context, id string) (*models.PipelineRun, error) {
	return s.pipeline.ResubmitRun(ctx, id)
}

func (s *service) RerunRun(ctx context.Context, id string) (*models.PipelineRun, error) {
	return s.pipeline.RerunRun(ctx, id)
}

func (s *service) ControlRun(ctx context.Context, id string, op LifecycleOperation) (*models.PipelineRun, error) {
	switch op {
	case LifecycleRuntimeRetry:
		return s.pipeline.RuntimeRetryRun(ctx, id)
	case LifecycleStop:
		return nil, s.pipeline.StopRun(ctx, id)
	case LifecycleSuspend:
		return nil, s.pipeline.SuspendRun(ctx, id)
	case LifecycleResume:
		return nil, s.pipeline.ResumeRun(ctx, id)
	case LifecycleTerminate:
		return nil, s.pipeline.TerminateRun(ctx, id)
	default:
		return nil, fmt.Errorf("unsupported run lifecycle operation %q", op)
	}
}

func (s *service) StopRun(ctx context.Context, id string) error {
	_, err := s.ControlRun(ctx, id, LifecycleStop)
	return err
}

func (s *service) SuspendRun(ctx context.Context, id string) error {
	_, err := s.ControlRun(ctx, id, LifecycleSuspend)
	return err
}

func (s *service) ResumeRun(ctx context.Context, id string) error {
	_, err := s.ControlRun(ctx, id, LifecycleResume)
	return err
}

func (s *service) TerminateRun(ctx context.Context, id string) error {
	_, err := s.ControlRun(ctx, id, LifecycleTerminate)
	return err
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
