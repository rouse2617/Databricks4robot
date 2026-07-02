package batchprogress

import (
	"sort"
	"strings"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// Derive builds a compact node summary for batch subtask list and attempt history.
func Derive(rows []models.PipelineRunAssetNode, overallStatus string) *models.PipelineRunNodeProgress {
	return DeriveWithMessage(rows, overallStatus, "")
}

// DeriveWithMessage builds a compact node summary and can surface the run-level
// terminal message when the stored node snapshot was taken before the last
// active node reached a terminal phase.
func DeriveWithMessage(rows []models.PipelineRunAssetNode, overallStatus, overallMessage string) *models.PipelineRunNodeProgress {
	if len(rows) == 0 {
		switch strings.TrimSpace(overallStatus) {
		case "Pending", "":
			return &models.PipelineRunNodeProgress{Label: "未开始"}
		case "Succeeded", "Success":
			return &models.PipelineRunNodeProgress{Label: "已完成"}
		default:
			return nil
		}
	}

	sorted := append([]models.PipelineRunAssetNode(nil), rows...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].PipelineNodeID < sorted[j].PipelineNodeID
	})

	for _, row := range sorted {
		status := strings.TrimSpace(row.Status)
		if status == "Failed" || status == "Error" {
			name := strings.TrimSpace(row.DisplayName)
			if name == "" {
				name = row.PipelineNodeID
			}
			return &models.PipelineRunNodeProgress{
				FocusNodeID:   row.PipelineNodeID,
				FocusNodeName: name,
				FocusStatus:   status,
				Message:       strings.TrimSpace(row.Message),
			}
		}
	}

	if isTerminalFailureStatus(overallStatus) {
		for _, row := range sorted {
			status := strings.TrimSpace(row.Status)
			if status != "Running" {
				continue
			}
			name := strings.TrimSpace(row.DisplayName)
			if name == "" {
				name = row.PipelineNodeID
			}
			message := strings.TrimSpace(row.Message)
			if message == "" {
				message = strings.TrimSpace(overallMessage)
			}
			return &models.PipelineRunNodeProgress{
				FocusNodeID:   row.PipelineNodeID,
				FocusNodeName: name,
				FocusStatus:   "Error",
				Message:       message,
			}
		}
	}

	running := make([]models.PipelineRunAssetNode, 0)
	for _, row := range sorted {
		if strings.TrimSpace(row.Status) == "Running" {
			running = append(running, row)
		}
	}
	if len(running) > 0 {
		row := running[len(running)-1]
		name := strings.TrimSpace(row.DisplayName)
		if name == "" {
			name = row.PipelineNodeID
		}
		progress := &models.PipelineRunNodeProgress{
			FocusNodeID:   row.PipelineNodeID,
			FocusNodeName: name,
			FocusStatus:   "Running",
		}
		if extra := len(running) - 1; extra > 0 {
			progress.ParallelRunning = extra
		}
		return progress
	}

	succeeded := make([]models.PipelineRunAssetNode, 0)
	for _, row := range sorted {
		if strings.TrimSpace(row.Status) == "Succeeded" {
			succeeded = append(succeeded, row)
		}
	}
	if len(succeeded) > 0 {
		row := succeeded[len(succeeded)-1]
		name := strings.TrimSpace(row.DisplayName)
		if name == "" {
			name = row.PipelineNodeID
		}
		overall := strings.TrimSpace(overallStatus)
		if overall == "Succeeded" || overall == "Success" {
			return &models.PipelineRunNodeProgress{
				FocusNodeID:   row.PipelineNodeID,
				FocusNodeName: name,
				FocusStatus:   "Succeeded",
			}
		}
	}

	switch strings.TrimSpace(overallStatus) {
	case "Succeeded", "Success":
		return &models.PipelineRunNodeProgress{Label: "已完成"}
	default:
		return &models.PipelineRunNodeProgress{Label: "进行中"}
	}
}

func isTerminalFailureStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed", "error", "expired":
		return true
	default:
		return false
	}
}

// ByRunID maps run IDs to node progress summaries.
func ByRunID(rows []models.PipelineRunAssetNode, runs []models.PipelineRun) map[string]*models.PipelineRunNodeProgress {
	statusByRun := make(map[string]string, len(runs))
	messageByRun := make(map[string]string, len(runs))
	for _, run := range runs {
		statusByRun[run.ID] = run.Status
		messageByRun[run.ID] = run.Message
	}
	grouped := make(map[string][]models.PipelineRunAssetNode)
	for _, row := range rows {
		grouped[row.RunID] = append(grouped[row.RunID], row)
	}
	out := make(map[string]*models.PipelineRunNodeProgress, len(runs))
	for _, run := range runs {
		out[run.ID] = DeriveWithMessage(grouped[run.ID], statusByRun[run.ID], messageByRun[run.ID])
	}
	return out
}
