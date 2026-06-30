package pipeline

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// GetUserPipelineStats returns aggregated usage statistics for a user's pipelines
// for smart grouping (e.g. "most used").
// Window parameter: "7d", "30d", "60d", "90d"
func (uc *Usecase) GetUserPipelineStats(ctx context.Context, userEmail, window string) (*models.PipelineUserStats, error) {
	// Parse window to duration
	dur, err := parseWindow(window)
	if err != nil {
		return nil, err
	}

	cutoff := uc.now().Add(-dur)

	// Fetch user's pipeline stats from repository
	stats, err := uc.templateRepo.GetUserPipelineStatsAfter(ctx, userEmail, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pipeline stats: %w", err)
	}

	// Enrich with recommendations (sorted by score)
	return stats, nil
}

// CheckPipelineVersionConflict checks if baseVersion matches current version.
func (uc *Usecase) CheckPipelineVersionConflict(ctx context.Context, pipelineID string, baseVersion int) (*models.PipelineTemplate, bool, error) {
	return uc.templateRepo.GetLatestVersionWithConflictCheck(ctx, pipelineID, baseVersion)
}

// GetRecommendations returns top pipeline recommendations for smart grouping
func (uc *Usecase) GetRecommendations(ctx context.Context, userEmail, window string, limit int) ([]*models.PipelineRecommendation, error) {
	stats, err := uc.GetUserPipelineStats(ctx, userEmail, window)
	if err != nil {
		return nil, err
	}

	// Build recommendations from stats
	recommendations := make([]*models.PipelineRecommendation, 0, len(stats.ClickCounts))

	for pipelineID, clickCount := range stats.ClickCounts {
		runCount := stats.RunCounts[pipelineID]
		execTime := stats.ExecutionTimes[pipelineID]
		lastAccess := stats.LastAccessTimes[pipelineID]

		// Calculate composite score
		score := calcRecommendationScore(clickCount, runCount, lastAccess, uc.now())

		recommendations = append(recommendations, &models.PipelineRecommendation{
			PipelineID: pipelineID,
			Score:      score,
			ClickCount: clickCount,
			RunCount:   runCount,
			LastAccess: &lastAccess,
			Reason:     generateRecommendationReason(clickCount, runCount, execTime),
		})
	}

	// Sort by score descending
	sortRecommendationsByScore(recommendations)

	// Limit results
	if limit > 0 && len(recommendations) > limit {
		recommendations = recommendations[:limit]
	}

	return recommendations, nil
}

// calcRecommendationScore computes a composite score for ranking pipelines.
// Formula: (clickCount * 0.4) + (runCount * 0.35) + (recency * 0.25)
// Returns a score between 0-10
func calcRecommendationScore(clickCount, runCount int, lastAccess time.Time, now time.Time) float64 {
	const (
		maxScore      = 10.0
		clickWeight   = 0.4
		runWeight     = 0.35
		recencyWeight = 0.25
		maxDaysOld    = 30
	)

	// Normalize click count (assume max 50 clicks = 10 points)
	clickScore := math.Min(10.0, float64(clickCount)/5.0)

	// Normalize run count (assume max 20 runs = 10 points)
	runScore := math.Min(10.0, float64(runCount)/2.0)

	// Normalize recency (assume 30 days old = 0 points, today = 10 points)
	daysOld := now.Sub(lastAccess).Hours() / 24.0
	recencyScore := math.Max(0, 10.0-daysOld/float64(maxDaysOld)*10.0)

	// Composite score
	score := (clickScore * clickWeight) + (runScore * runWeight) + (recencyScore * recencyWeight)
	return math.Min(maxScore, math.Max(0, score))
}

// generateRecommendationReason creates a human-readable explanation
func generateRecommendationReason(clickCount, runCount int, execTimeMs int64) string {
	parts := []string{}

	if clickCount > 0 {
		parts = append(parts, fmt.Sprintf("过去30天点击%d次", clickCount))
	}
	if runCount > 0 {
		parts = append(parts, fmt.Sprintf("运行%d次", runCount))
	}

	if len(parts) == 0 {
		return "最近使用"
	}

	return fmt.Sprintf("%s，%s", parts[0], parts[len(parts)-1])
}

// parseWindow converts window string to duration
func parseWindow(window string) (time.Duration, error) {
	switch window {
	case "7d":
		return 7 * 24 * time.Hour, nil
	case "30d":
		return 30 * 24 * time.Hour, nil
	case "60d":
		return 60 * 24 * time.Hour, nil
	case "90d":
		return 90 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("invalid window: %s", window)
	}
}

// sortRecommendationsByScore sorts recommendations by score descending
func sortRecommendationsByScore(recs []*models.PipelineRecommendation) {
	// Simple bubble sort for small lists; use sort.Slice for production
	for i := 0; i < len(recs); i++ {
		for j := i + 1; j < len(recs); j++ {
			if recs[j].Score > recs[i].Score {
				recs[i], recs[j] = recs[j], recs[i]
			}
		}
	}
}
