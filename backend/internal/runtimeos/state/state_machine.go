package state

import (
	"fmt"
	"strings"

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
	summary.AggregateStatus = aggregateStatus(summary)
	return summary
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
