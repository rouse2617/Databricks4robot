package backfill

import (
	"context"
	"strings"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

func (uc *Usecase) resolveCompletionStatus(ctx context.Context, job *models.BackfillJob, item models.BackfillItem) string {
	reportID, version, ok := expectedReportFromJob(job)
	if !ok || uc.resultRepo == nil {
		return "completed"
	}
	has, err := uc.resultRepo.HasAlgoRunResult(ctx, item.AssetID, reportID, version)
	if err != nil || has {
		return "completed"
	}
	return "awaiting_result"
}

func expectedReportFromJob(job *models.BackfillJob) (reportID, version string, ok bool) {
	if job == nil || len(job.FilterJSON) == 0 {
		return "", "", false
	}
	reportID = stringFromFilter(job.FilterJSON, "expectedReportId", "expected_report_id")
	version = stringFromFilter(job.FilterJSON, "expectedReportVersion", "expected_report_version")
	if reportID == "" {
		return "", "", false
	}
	if version == "" {
		version = "1.0.0"
	}
	return reportID, version, true
}

func stringFromFilter(filter map[string]any, keys ...string) string {
	for _, key := range keys {
		raw, exists := filter[key]
		if !exists {
			continue
		}
		if v, ok := raw.(string); ok {
			if trimmed := strings.TrimSpace(v); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func itemExpectsReport(item models.BackfillItem, reportID, version string) bool {
	_ = reportID
	_ = version
	return item.Status == "awaiting_result"
}
