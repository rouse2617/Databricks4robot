package state

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

const (
	StatusPending   = "Pending"
	StatusRunning   = "Running"
	StatusSucceeded = "Succeeded"
	StatusFailed    = "Failed"
	StatusError     = "Error"
	StatusCancelled = "Cancelled"
	StatusExpired   = "Expired"
	StatusSuspended = "Suspended"
)

// NormalizeRunStatus maps Argo, batch, and compatibility statuses into the
// product Run status vocabulary used by Run Tree aggregation.
func NormalizeRunStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "succeeded", "success", "completed", "complete":
		return StatusSucceeded
	case "failed", "failure":
		return StatusFailed
	case "error":
		return StatusError
	case "cancelled", "canceled", "terminated", "stopped":
		return StatusCancelled
	case "expired":
		return StatusExpired
	case "suspended", "paused", "pilot_review":
		return StatusSuspended
	case "running", "pilot_running":
		return StatusRunning
	case "pending", "created", "validated", "submitted", "queued", "":
		return StatusPending
	default:
		return strings.TrimSpace(status)
	}
}

func IsActiveStatus(status string) bool {
	switch NormalizeRunStatus(status) {
	case StatusPending, StatusRunning, StatusSuspended:
		return true
	default:
		return false
	}
}

func IsTerminalStatus(status string) bool {
	return !IsActiveStatus(status)
}

func IsSuccessStatus(status string) bool {
	return NormalizeRunStatus(status) == StatusSucceeded
}

func IsFailureStatus(status string) bool {
	switch NormalizeRunStatus(status) {
	case StatusFailed, StatusError, StatusExpired:
		return true
	default:
		return false
	}
}

func IsCancelledStatus(status string) bool {
	return NormalizeRunStatus(status) == StatusCancelled
}

// stalledObservationThreshold — read-side hint threshold. An active run that
// has not been updated in this long is annotated BlockingReason=stalled so
// the UI surfaces "任务长时间未更新" without the writer touching the status.
// Kept in-package (not env-configurable yet) to keep the change small; can be
// promoted to config when we see real load with different SLAs.
const stalledObservationThreshold = 30 * time.Minute

// nowUTC is a package-level clock so tests can inject a deterministic time.
var nowUTC = func() time.Time { return time.Now().UTC() }

// AnnotateRunDiagnostics attaches normalized failure/blocking fields to a Run
// without changing its persisted status or raw runtime message.
//
// CYB-3491: additionally surfaces a "stalled" hint on active runs that have
// gone quiet for stalledObservationThreshold. This is display-only — no
// database write, no status flip — so a webhook / poll that later delivers
// progress naturally clears it (updates UpdatedAt → freshness returns).
func AnnotateRunDiagnostics(run *models.PipelineRun) {
	if run == nil {
		return
	}
	run.FailureReason = ""
	run.BlockingReason = ""
	run.BlockingMessage = ""
	diag, ok := ClassifyRunDiagnostic(*run)
	if ok {
		status := NormalizeRunStatus(run.Status)
		if IsFailureStatus(status) || IsCancelledStatus(status) {
			run.FailureReason = diag.Reason
			return
		}
		run.BlockingReason = diag.Reason
		run.BlockingMessage = diag.Message
	}
	// Stall hint: only for active runs whose classifier gave nothing more
	// specific (unschedulable etc. already tell the user why it's slow — we
	// don't want to overwrite those with a generic "stalled").
	if run.BlockingReason == "" && IsActiveStatus(run.Status) {
		ref := run.UpdatedAt
		if ref.IsZero() {
			return
		}
		if nowUTC().Sub(ref) >= stalledObservationThreshold {
			run.BlockingReason = "stalled"
			run.BlockingMessage = "任务长时间未收到状态更新（可能停滞或平台跟丢了状态)"
		}
	}
}

// ClassifyRunDiagnostic returns a stable reason code and representative
// message for product surfaces that need to explain why a Run is blocked or
// failed without exposing Argo/Kubernetes as the primary product model.
func ClassifyRunDiagnostic(run models.PipelineRun) (models.RunBlockingReason, bool) {
	status := NormalizeRunStatus(run.Status)
	message := strings.TrimSpace(run.Message)
	if message == "" {
		message = representativeNodeDiagnosticMessage(run.Nodes)
	}
	reason := classifyRunDiagnosticReason(status, message, run.WorkflowName, run.ArgoWorkflowUID)
	if reason == "" {
		return models.RunBlockingReason{}, false
	}
	if message == "" {
		message = defaultDiagnosticMessage(reason)
	}
	diag := models.RunBlockingReason{
		Reason:       reason,
		Message:      message,
		Count:        1,
		ExampleRunID: strings.TrimSpace(run.ID),
		Source:       "pipeline_runs",
	}
	if len(run.AssetIDs) == 1 {
		diag.ExampleAsset = run.AssetIDs[0]
	}
	return diag, true
}

func representativeNodeDiagnosticMessage(nodes []models.PipelineRunNode) string {
	for _, node := range nodes {
		if IsFailureStatus(node.Phase) || IsCancelledStatus(node.Phase) {
			if message := strings.TrimSpace(node.Message); message != "" {
				return message
			}
		}
	}
	for _, node := range nodes {
		if IsActiveStatus(node.Phase) {
			if message := strings.TrimSpace(node.Message); message != "" {
				return message
			}
		}
	}
	for _, node := range nodes {
		if message := strings.TrimSpace(node.Message); message != "" {
			return message
		}
	}
	return ""
}

func classifyRunDiagnosticReason(status, message, workflowName, workflowUID string) string {
	normalizedMessage := strings.ToLower(strings.TrimSpace(message))
	// Batch parent runs have synthetic workflow names and never have a real
	// Argo workflow. Their terminal status comes from child aggregation.
	if strings.HasPrefix(strings.TrimSpace(workflowName), "batch-parent-") || strings.HasPrefix(strings.TrimSpace(workflowName), "backfill-parent-") {
		if strings.EqualFold(status, StatusSucceeded) {
			return "" // no diagnostic — succeeded
		}
		if IsFailureStatus(status) {
			return "run_failed"
		}
		return "runtime_not_submitted"
	}
	switch {
	case containsAny(normalizedMessage, "unschedulable", "insufficient cpu", "insufficient memory", "insufficient ephemeral-storage", "didn't match pod affinity", "didn't match pod anti-affinity"):
		return "unschedulable"
	case containsAny(normalizedMessage, "不支持该资源规格", "最大可用", "resource limit", "resource quota", "quota exceeded", "exceeds target"):
		return "resource_incompatible"
	case containsAny(normalizedMessage, "imagepullbackoff", "errimagepull", "invalidimagename", "invalid image", "failed to apply default image tag", "couldn't parse image name", "manifest unknown", "pull access denied"):
		return "image_startup"
	case containsAny(normalizedMessage, "create runtime config projection", "runtime config projection", "configmap", "insufficient privileges", "forbidden", "unauthorized"):
		return "runtime_config_projection_failed"
	case containsAny(normalizedMessage, "stale run: exceeded maximum active duration"):
		return "stale_running"
	case containsAny(normalizedMessage, "argo 工作流已被 ttl 清理", "workflow not found", "workflow service unavailable"):
		return "runtime_missing"
	case strings.EqualFold(status, StatusPending) && strings.TrimSpace(workflowName) == "" && strings.TrimSpace(workflowUID) == "":
		return "runtime_not_submitted"
	case IsCancelledStatus(status):
		return "cancelled"
	case IsFailureStatus(status):
		return "run_failed"
	default:
		return ""
	}
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func defaultDiagnosticMessage(reason string) string {
	switch reason {
	case "runtime_not_submitted":
		return "Run 已创建，正在等待提交到运行时。"
	case "cancelled":
		return "Run 已取消。"
	case "runtime_config_projection_failed":
		return "运行配置投影失败。"
	case "run_failed":
		return "Run 已失败，暂无更具体的运行时诊断。"
	default:
		return ""
	}
}

// AggregateChildRuns computes a stable summary for child Run rows without
// reaching into the runtime system.
func AggregateChildRuns(children []models.PipelineRun) models.RunChildSummary {
	statuses := make(map[string]int)
	summary := models.RunChildSummary{
		Total:    len(children),
		Statuses: statuses,
	}
	for _, child := range children {
		status := NormalizeRunStatus(child.Status)
		statuses[status]++
		switch status {
		case StatusSucceeded:
			summary.SucceededCount++
			summary.TerminalCount++
		case StatusFailed, StatusError, StatusExpired:
			summary.FailedCount++
			summary.TerminalCount++
		case StatusCancelled:
			summary.CancelledCount++
			summary.TerminalCount++
		case StatusRunning:
			summary.RunningCount++
			summary.ActiveCount++
		case StatusPending:
			summary.PendingCount++
			summary.ActiveCount++
		case StatusSuspended:
			summary.SuspendedCount++
			summary.ActiveCount++
		default:
			if IsActiveStatus(status) {
				summary.ActiveCount++
			} else {
				summary.TerminalCount++
			}
		}
	}
	summary.HasFailures = summary.FailedCount > 0 || summary.CancelledCount > 0
	summary.AggregateStatus = aggregateStatus(summary)
	summary.TopFailureReasons = topRunDiagnostics(children, 5)
	summary.HasBlocking = summary.PendingCount > 0 || summary.SuspendedCount > 0
	if !summary.HasBlocking {
		for _, reason := range summary.TopFailureReasons {
			switch reason.Reason {
			case "runtime_not_submitted", "unschedulable", "resource_incompatible", "stale_running":
				summary.HasBlocking = true
			}
		}
	}
	summary.HealthStatus = aggregateHealthStatus(summary)
	return summary
}

func aggregateHealthStatus(summary models.RunChildSummary) string {
	switch {
	case summary.Total == 0:
		return "pending"
	case summary.FailedCount == summary.Total || summary.CancelledCount == summary.Total:
		return "failed"
	case summary.HasFailures:
		return "degraded"
	case summary.HasBlocking:
		return "warning"
	default:
		return "healthy"
	}
}

func topRunDiagnostics(children []models.PipelineRun, limit int) []models.RunBlockingReason {
	if len(children) == 0 || limit <= 0 {
		return nil
	}
	type bucket struct {
		reason string
		count  int
		first  models.RunBlockingReason
	}
	buckets := map[string]*bucket{}
	for _, child := range children {
		diag, ok := ClassifyRunDiagnostic(child)
		if !ok {
			continue
		}
		key := diag.Reason
		if key == "" {
			continue
		}
		current := buckets[key]
		if current == nil {
			diag.Count = 0
			current = &bucket{reason: key, first: diag}
			buckets[key] = current
		}
		current.count++
		if current.first.Message == "" && diag.Message != "" {
			current.first.Message = diag.Message
		}
	}
	if len(buckets) == 0 {
		return nil
	}
	out := make([]models.RunBlockingReason, 0, len(buckets))
	for _, bucket := range buckets {
		item := bucket.first
		item.Count = bucket.count
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Reason < out[j].Reason
		}
		return out[i].Count > out[j].Count
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func aggregateStatus(summary models.RunChildSummary) string {
	if summary.Total == 0 {
		return StatusPending
	}
	if summary.RunningCount > 0 {
		return StatusRunning
	}
	if summary.PendingCount > 0 {
		return StatusPending
	}
	if summary.SuspendedCount > 0 {
		return StatusSuspended
	}
	if summary.FailedCount > 0 {
		if summary.Statuses[StatusError] > 0 {
			return StatusError
		}
		if summary.Statuses[StatusExpired] > 0 {
			return StatusExpired
		}
		return StatusFailed
	}
	if summary.CancelledCount > 0 {
		return StatusCancelled
	}
	if summary.SucceededCount == summary.Total {
		return StatusSucceeded
	}
	return StatusPending
}

// BuildBatchChildRelations projects batch child relationships from the current
// PipelineRun.BatchJobID metadata. It deliberately does not invent relations
// for rows that lack parent metadata.
func BuildBatchChildRelations(parentRunID string, children []models.PipelineRun) []models.RunRelation {
	parentRunID = strings.TrimSpace(parentRunID)
	if parentRunID == "" {
		return nil
	}
	relations := make([]models.RunRelation, 0, len(children))
	for _, child := range children {
		if child.BatchJobID == nil || strings.TrimSpace(*child.BatchJobID) != parentRunID {
			continue
		}
		childID := strings.TrimSpace(child.ID)
		if childID == "" || childID == parentRunID {
			continue
		}
		relation := models.RunRelation{
			ID:           fmt.Sprintf("%s:%s:batch_child", parentRunID, childID),
			ParentRunID:  parentRunID,
			ChildRunID:   childID,
			RelationType: "batch_child",
			Source:       "pipeline_runs.batch_job_id",
		}
		if len(child.AssetIDs) == 1 {
			relation.AssetID = child.AssetIDs[0]
		}
		relations = append(relations, relation)
	}
	return relations
}
