package asset

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// VersionHistoryEntry is one row in the version timeline for a logical asset.
type VersionHistoryEntry struct {
	Version    int64     `json:"version"`
	AssetID    string    `json:"asset_id,omitempty"`
	PromotedAt time.Time `json:"promoted_at"`
	ByRunID    string    `json:"by_run_id,omitempty"`
	Reason     string    `json:"reason,omitempty"`
}

// RevisionSummary is a compact revision row for UI switching.
type RevisionSummary struct {
	AssetID   string    `json:"asset_id"`
	Revision  int64     `json:"revision"`
	IsCurrent bool      `json:"is_current"`
	CreatedAt time.Time `json:"created_at"`
}

// ProvenanceResult aggregates version + revision metadata for one asset view.
type ProvenanceResult struct {
	AssetID        string                `json:"asset_id"`
	LogicalAssetID string                `json:"logical_asset_id,omitempty"`
	Revisions      []RevisionSummary     `json:"revisions"`
	VersionHistory []VersionHistoryEntry `json:"version_history"`
}

// GetProvenance returns revision list and version history for the asset's logical family.
func (u *Usecase) GetProvenance(ctx context.Context, assetID string) (*ProvenanceResult, error) {
	a, err := u.Get(ctx, assetID)
	if err != nil {
		return nil, err
	}
	res := &ProvenanceResult{
		AssetID:        a.AssetID,
		LogicalAssetID: a.LogicalAssetID,
	}
	if a.LogicalAssetID == "" {
		rev := a.Revision
		if rev <= 0 {
			rev = 1
		}
		res.Revisions = []RevisionSummary{{
			AssetID:   a.AssetID,
			Revision:  rev,
			IsCurrent: a.IsCurrent,
			CreatedAt: a.CreatedAt,
		}}
		res.VersionHistory = []VersionHistoryEntry{{
			Version:    rev,
			AssetID:    a.AssetID,
			PromotedAt: a.CreatedAt,
		}}
		return res, nil
	}

	revisions, err := u.repo.ListByLogicalAssetID(ctx, a.LogicalAssetID)
	if err != nil {
		return nil, err
	}
	res.Revisions = make([]RevisionSummary, 0, len(revisions))
	for _, row := range revisions {
		rev := row.Revision
		if rev <= 0 {
			rev = 1
		}
		res.Revisions = append(res.Revisions, RevisionSummary{
			AssetID:   row.AssetID,
			Revision:  rev,
			IsCurrent: row.IsCurrent,
			CreatedAt: row.CreatedAt,
		})
	}

	promotedByAsset := map[string]*models.AssetEvent{}
	if u.eventRepo != nil {
		events, err := u.eventRepo.ListVersionPromotedByLogical(ctx, a.LogicalAssetID)
		if err != nil {
			return nil, err
		}
		for _, e := range events {
			promotedByAsset[e.AssetID] = e
		}
	}

	history := make([]VersionHistoryEntry, 0, len(revisions))
	for _, row := range revisions {
		rev := row.Revision
		if rev <= 0 {
			rev = 1
		}
		entry := VersionHistoryEntry{
			Version: rev,
			AssetID: row.AssetID,
		}
		if ev, ok := promotedByAsset[row.AssetID]; ok {
			entry.PromotedAt = ev.OccurredAt
			entry.ByRunID, entry.Reason = parseVersionPromotedPayload(ev.EventPayload)
		} else {
			entry.PromotedAt = row.CreatedAt
		}
		history = append(history, entry)
	}
	sort.Slice(history, func(i, j int) bool {
		return history[i].Version < history[j].Version
	})
	res.VersionHistory = history
	return res, nil
}

func parseVersionPromotedPayload(raw json.RawMessage) (runID, reason string) {
	if len(raw) == 0 {
		return "", ""
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", ""
	}
	if v, ok := payload["run_id"].(string); ok {
		runID = v
	}
	if v, ok := payload["reason"].(string); ok {
		reason = v
	}
	return runID, reason
}
