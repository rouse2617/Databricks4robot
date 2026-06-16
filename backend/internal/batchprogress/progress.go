package batchprogress

import (
	"sort"
	"strings"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// Derive builds a compact node summary for batch subtask list and attempt history.
func Derive(rows []models.PipelineRunAssetNode, overallStatus string) *models.PipelineRunNodeProgress {
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

	switch strings.TrimSpace(overallStatus) {
	case "Succeeded", "Success":
		return &models.PipelineRunNodeProgress{Label: "已完成"}
	default:
		return &models.PipelineRunNodeProgress{Label: "进行中"}
	}
}

// ByRunID maps run IDs to node progress summaries.
func ByRunID(rows []models.PipelineRunAssetNode, runs []models.PipelineRun) map[string]*models.PipelineRunNodeProgress {
	statusByRun := make(map[string]string, len(runs))
	for _, run := range runs {
		statusByRun[run.ID] = run.Status
	}
	grouped := make(map[string][]models.PipelineRunAssetNode)
	for _, row := range rows {
		grouped[row.RunID] = append(grouped[row.RunID], row)
	}
	out := make(map[string]*models.PipelineRunNodeProgress, len(runs))
	for _, run := range runs {
		out[run.ID] = Derive(grouped[run.ID], statusByRun[run.ID])
	}
	return out
}
