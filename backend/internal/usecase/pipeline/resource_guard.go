package pipeline

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/transpiler"
)

const defaultUnschedulablePendingThreshold = 15 * time.Minute

// ResourceGuardConfig contains conservative, environment-level runtime guards.
// Empty resource ceiling values disable that specific deploy-time check.
type ResourceGuardConfig struct {
	MaxCPU                        string
	MaxMemory                     string
	MaxDisk                       string
	MaxGPU                        string
	UnschedulablePendingThreshold time.Duration
}

// SetResourceGuardConfig configures static resource ceilings and Pending
// convergence behavior for pipeline runs.
func (uc *Usecase) SetResourceGuardConfig(cfg ResourceGuardConfig) {
	uc.resourceGuard = cfg
}

func (uc *Usecase) nowUTC() time.Time {
	if uc != nil && uc.now != nil {
		return uc.now().UTC()
	}
	return time.Now().UTC()
}

func (uc *Usecase) resourceGuardConfig() ResourceGuardConfig {
	if uc == nil {
		return ResourceGuardConfig{UnschedulablePendingThreshold: defaultUnschedulablePendingThreshold}
	}
	cfg := uc.resourceGuard
	if cfg.UnschedulablePendingThreshold == 0 {
		cfg.UnschedulablePendingThreshold = defaultUnschedulablePendingThreshold
	}
	return cfg
}

func (uc *Usecase) resourceGuardForTarget(target *models.ExecutionTarget) ResourceGuardConfig {
	cfg := uc.resourceGuardConfig()
	if target == nil {
		return cfg
	}
	if len(target.QuotaPolicy) == 0 && !target.IsDefault {
		cfg.MaxCPU = ""
		cfg.MaxMemory = ""
		cfg.MaxDisk = ""
		cfg.MaxGPU = ""
		return cfg
	}
	if len(target.QuotaPolicy) == 0 {
		return cfg
	}
	return applyResourceGuardPolicy(cfg, target.QuotaPolicy)
}

func applyResourceGuardPolicy(cfg ResourceGuardConfig, policy map[string]interface{}) ResourceGuardConfig {
	if nested, ok := mapValue(policy, "resourceCeilings", "resourceLimits", "resources", "limits"); ok {
		if nestedMap, ok := nested.(map[string]interface{}); ok {
			policy = nestedMap
		}
	}
	if value, ok := firstPolicyString(policy, "maxCpu", "maxCPU", "max_cpu", "cpu", "cpuLimit", "cpu_limit"); ok {
		cfg.MaxCPU = value
	}
	if value, ok := firstPolicyString(policy, "maxMemory", "max_memory", "memory", "memoryLimit", "memory_limit"); ok {
		cfg.MaxMemory = value
	}
	if value, ok := firstPolicyString(policy, "maxDisk", "max_disk", "disk", "ephemeralStorage", "maxEphemeralStorage", "ephemeral_storage"); ok {
		cfg.MaxDisk = value
	}
	if value, ok := firstPolicyString(policy, "maxGpu", "maxGPU", "max_gpu", "gpu", "gpuLimit", "gpu_limit"); ok {
		cfg.MaxGPU = value
	}
	return cfg
}

func mapValue(m map[string]interface{}, keys ...string) (interface{}, bool) {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			return value, true
		}
	}
	return nil, false
}

func firstPolicyString(m map[string]interface{}, keys ...string) (string, bool) {
	for _, key := range keys {
		value, ok := m[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v), true
			}
		case float64:
			return fmt.Sprintf("%g", v), true
		case float32:
			return fmt.Sprintf("%g", v), true
		case int:
			return fmt.Sprintf("%d", v), true
		case int64:
			return fmt.Sprintf("%d", v), true
		case int32:
			return fmt.Sprintf("%d", v), true
		case uint:
			return fmt.Sprintf("%d", v), true
		case uint64:
			return fmt.Sprintf("%d", v), true
		case uint32:
			return fmt.Sprintf("%d", v), true
		}
	}
	return "", false
}

type resourceCeilingCheck struct {
	name      string
	requested string
	limit     string
}

func (uc *Usecase) validateResourceCeilings(pipe *transpiler.Pipeline, target *models.ExecutionTarget) error {
	if pipe == nil {
		return nil
	}
	cfg := uc.resourceGuardForTarget(target)
	var walk func(nodes []transpiler.Node) error
	walk = func(nodes []transpiler.Node) error {
		for i := range nodes {
			node := nodes[i]
			if err := validateNodeResourceCeilings(node, cfg, target); err != nil {
				return err
			}
			if len(node.SubNodes) > 0 {
				if err := walk(node.SubNodes); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walk(pipe.Nodes)
}

func validateNodeResourceCeilings(node transpiler.Node, cfg ResourceGuardConfig, target *models.ExecutionTarget) error {
	res := node.Component.Resources
	if res == nil {
		return nil
	}
	checks := []resourceCeilingCheck{
		{name: "cpu", requested: res.CPU, limit: cfg.MaxCPU},
		{name: "memory", requested: res.Memory, limit: cfg.MaxMemory},
		{name: "ephemeral-storage", requested: res.Disk, limit: cfg.MaxDisk},
		{name: "gpu", requested: res.GPU, limit: cfg.MaxGPU},
	}
	for _, check := range checks {
		if strings.TrimSpace(check.requested) == "" || strings.TrimSpace(check.limit) == "" {
			continue
		}
		requested, err := resource.ParseQuantity(strings.TrimSpace(check.requested))
		if err != nil {
			return fmt.Errorf("%w: node %q has invalid %s resource quantity %q", ErrInvalidArgument, nodeDisplayName(node), check.name, check.requested)
		}
		limit, err := resource.ParseQuantity(strings.TrimSpace(check.limit))
		if err != nil {
			slog.Warn("pipeline resource ceiling ignored because configured limit is invalid",
				"resource", check.name,
				"limit", check.limit,
				"targetId", targetID(target),
			)
			continue
		}
		if requested.Cmp(limit) <= 0 {
			continue
		}
		return fmt.Errorf("%w: %s不支持该资源规格：节点 %q 请求 %s=%s，最大可用 %s=%s。请降低资源、切换计算档位或联系管理员扩容",
			ErrInvalidArgument,
			targetLabel(target),
			nodeDisplayName(node),
			check.name,
			strings.TrimSpace(check.requested),
			check.name,
			strings.TrimSpace(check.limit),
		)
	}
	return nil
}

func targetID(target *models.ExecutionTarget) string {
	if target == nil {
		return ""
	}
	return strings.TrimSpace(target.ID)
}

func targetLabel(target *models.ExecutionTarget) string {
	if target == nil {
		return "当前执行环境"
	}
	if strings.TrimSpace(target.Name) != "" {
		return fmt.Sprintf("执行目标 %q", strings.TrimSpace(target.Name))
	}
	if strings.TrimSpace(target.ID) != "" {
		return fmt.Sprintf("执行目标 %q", strings.TrimSpace(target.ID))
	}
	return "当前执行环境"
}

func nodeDisplayName(node transpiler.Node) string {
	if strings.TrimSpace(node.Component.Name) != "" {
		return strings.TrimSpace(node.Component.Name)
	}
	if strings.TrimSpace(node.ID) != "" {
		return strings.TrimSpace(node.ID)
	}
	return "unknown"
}

func (uc *Usecase) deriveUnschedulableRunFromWorkflow(
	run *models.PipelineRun,
	wf *wfv1.Workflow,
) (string, string, *time.Time, bool) {
	cfg := uc.resourceGuardConfig()
	if cfg.UnschedulablePendingThreshold <= 0 {
		return "", "", nil, false
	}
	if wf == nil || !isActiveDeploymentStatus(string(wf.Status.Phase)) {
		return "", "", nil, false
	}
	now := uc.nowUTC()
	var selectedName string
	var selectedMessage string
	var selectedSince time.Time
	for _, node := range wf.Status.Nodes {
		if node.Phase != wfv1.NodePending {
			continue
		}
		message := strings.TrimSpace(node.Message)
		if message == "" {
			message = strings.TrimSpace(wf.Status.Message)
		}
		if !isUnschedulableSchedulerMessage(message) {
			continue
		}
		pendingSince := pendingReferenceTime(run, wf, node)
		if pendingSince.IsZero() || now.Sub(pendingSince) < cfg.UnschedulablePendingThreshold {
			continue
		}
		if selectedSince.IsZero() || pendingSince.Before(selectedSince) {
			selectedSince = pendingSince
			selectedName = workflowNodeDisplayName(node)
			selectedMessage = message
		}
	}
	if selectedMessage == "" && isUnschedulableSchedulerMessage(wf.Status.Message) {
		pendingSince := workflowPendingReferenceTime(run, wf)
		if !pendingSince.IsZero() && now.Sub(pendingSince) >= cfg.UnschedulablePendingThreshold {
			selectedSince = pendingSince
			selectedName = strings.TrimSpace(wf.Name)
			selectedMessage = strings.TrimSpace(wf.Status.Message)
		}
	}
	if selectedMessage == "" {
		return "", "", nil, false
	}
	finishedAt := now
	return string(wfv1.WorkflowError), formatUnschedulableRunMessage(selectedName, now.Sub(selectedSince), selectedMessage), &finishedAt, true
}

func (uc *Usecase) deriveImageStartupRunFromWorkflow(
	run *models.PipelineRun,
	wf *wfv1.Workflow,
) (string, string, *time.Time, bool) {
	cfg := uc.resourceGuardConfig()
	if cfg.UnschedulablePendingThreshold <= 0 {
		return "", "", nil, false
	}
	if wf == nil || !isActiveDeploymentStatus(string(wf.Status.Phase)) {
		return "", "", nil, false
	}
	now := uc.nowUTC()
	var selectedName string
	var selectedMessage string
	var selectedSince time.Time
	for _, node := range wf.Status.Nodes {
		if node.Phase != wfv1.NodePending {
			continue
		}
		message := strings.TrimSpace(node.Message)
		if !isImageStartupFailureMessage(message) {
			continue
		}
		pendingSince := pendingReferenceTime(run, wf, node)
		if pendingSince.IsZero() || now.Sub(pendingSince) < cfg.UnschedulablePendingThreshold {
			continue
		}
		if selectedSince.IsZero() || pendingSince.Before(selectedSince) {
			selectedSince = pendingSince
			selectedName = workflowNodeDisplayName(node)
			selectedMessage = message
		}
	}
	if selectedMessage == "" {
		return "", "", nil, false
	}
	finishedAt := now
	return string(wfv1.WorkflowError), formatImageStartupRunMessage(selectedName, now.Sub(selectedSince), selectedMessage), &finishedAt, true
}

func isImageStartupFailureMessage(message string) bool {
	normalized := strings.ToLower(strings.TrimSpace(message))
	if normalized == "" {
		return false
	}
	signals := []string{
		"invalidimagename",
		"invalid image name",
		"invalid reference format",
		"failed to apply default image tag",
		"couldn't parse image name",
		"errimagepull",
		"imagepullbackoff",
		"failed to pull image",
		"pull access denied",
		"manifest unknown",
		"unauthorized: authentication required",
	}
	for _, signal := range signals {
		if strings.Contains(normalized, signal) {
			return true
		}
	}
	return false
}

func isUnschedulableSchedulerMessage(message string) bool {
	normalized := strings.ToLower(strings.TrimSpace(message))
	if normalized == "" {
		return false
	}
	signals := []string{
		"unschedulable",
		"insufficient cpu",
		"insufficient memory",
		"insufficient ephemeral-storage",
		"insufficient nvidia.com/gpu",
		"untolerated taint",
		"didn't match pod affinity",
		"didn't match pod anti-affinity",
		"didn't match pod's node affinity",
		"didn't match node selector",
		"preemption is not helpful",
		"exceeded quota",
	}
	for _, signal := range signals {
		if strings.Contains(normalized, signal) {
			return true
		}
	}
	return false
}

func pendingReferenceTime(run *models.PipelineRun, wf *wfv1.Workflow, node wfv1.NodeStatus) time.Time {
	if !node.StartedAt.Time.IsZero() {
		return node.StartedAt.Time.UTC()
	}
	return workflowPendingReferenceTime(run, wf)
}

func workflowPendingReferenceTime(run *models.PipelineRun, wf *wfv1.Workflow) time.Time {
	if wf != nil {
		if !wf.Status.StartedAt.Time.IsZero() {
			return wf.Status.StartedAt.Time.UTC()
		}
		if !wf.CreationTimestamp.Time.IsZero() {
			return wf.CreationTimestamp.Time.UTC()
		}
	}
	if run != nil {
		if run.StartedAt != nil && !run.StartedAt.IsZero() {
			return run.StartedAt.UTC()
		}
		if !run.CreatedAt.IsZero() {
			return run.CreatedAt.UTC()
		}
	}
	return time.Time{}
}

func workflowNodeDisplayName(node wfv1.NodeStatus) string {
	for _, value := range []string{node.DisplayName, node.TemplateName, node.Name, node.ID} {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "unknown"
}

func formatUnschedulableRunMessage(nodeName string, pendingFor time.Duration, schedulerMessage string) string {
	nodeName = strings.TrimSpace(nodeName)
	if nodeName == "" {
		nodeName = "unknown"
	}
	if pendingFor < 0 {
		pendingFor = 0
	}
	return fmt.Sprintf("Kubernetes 调度失败：节点 %q 已 Pending %s，%s",
		nodeName,
		pendingFor.Round(time.Second),
		strings.TrimSpace(schedulerMessage),
	)
}

func formatImageStartupRunMessage(nodeName string, pendingFor time.Duration, imageMessage string) string {
	nodeName = strings.TrimSpace(nodeName)
	if nodeName == "" {
		nodeName = "unknown"
	}
	if pendingFor < 0 {
		pendingFor = 0
	}
	return fmt.Sprintf("Kubernetes 镜像启动失败：节点 %q 已 Pending %s，%s",
		nodeName,
		pendingFor.Round(time.Second),
		strings.TrimSpace(imageMessage),
	)
}
