// Package searchindex builds Elasticsearch documents for assets from PostgreSQL projections.
package searchindex

import (
	"context"
	"fmt"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// Builder loads PG rows and produces ES _source-shaped maps (see deploy/local/elasticsearch/init-index.sh).
type Builder struct {
	Assets  repository.AssetRepository
	Tags    repository.AssetTagRepository
	Algos   repository.AssetAlgoLatestRepository
	Mcap    repository.McapFileRepository
	Actions repository.ActionRepository // optional; nil disables actions[] projection
	Lineage repository.AssetLineageRepository
}

// Build returns the document for an asset suitable for ES index API.
// If the asset is soft-deleted / missing, ok is false (caller should delete the ES doc).
func (b *Builder) Build(ctx context.Context, assetID string) (doc map[string]any, ok bool, err error) {
	a, err := b.Assets.Get(ctx, assetID)
	if err != nil {
		return nil, false, err
	}
	if a == nil {
		return nil, false, nil
	}

	meta := map[string]any{}
	if a.Metadata != nil {
		for k, v := range a.Metadata {
			meta[k] = v
		}
	}

	// lifecycle_state is the primary aggregation/facet field (P0-3).
	// Fall back to legacy status for rows not yet backfilled.
	lifecycleState := a.LifecycleState
	if lifecycleState == "" {
		lifecycleState = string(a.Status)
	}

doc = map[string]any{
		"asset_id":           a.AssetID,
		"mcap_file_id":       a.McapFileID,
		"segment_locator":    a.SegmentLocator,
		"asset_type":         a.AssetType,
		"lifecycle_state":    lifecycleState,
		"status":             string(a.Status), // deprecated — retained during dual-write window (§5.8.1)
		"is_deleted":         false,
		"version":            a.Version,
		"retention_tier":     a.RetentionTier,
		"storage_uri":        a.StorageURI,
		"owner":              a.Owner,
		"reviewer":           a.Reviewer,
		"start_timestamp_ns": a.StartTimestampNs,
		"end_timestamp_ns":   a.EndTimestampNs,
		"duration_ms":        a.DurationMs,
		"delivery_count":     a.DeliveryCount,
		"asset_level":        a.AssetLevel,
		"metadata":           meta,
		"tags_flat":          map[string]any{},
		"tags":               []map[string]any{},
		"algos":              []map[string]any{},
		"mcap":               map[string]any{},
		"lineage_upstream_ids":     []string{},
		"lineage_downstream_ids":   []string{},
		"lineage_relation_types":   []string{},
	}

	if a.ParentAssetID != "" {
		doc["parent_asset_id"] = a.ParentAssetID
	}
	if a.RootAssetID != "" {
		doc["root_asset_id"] = a.RootAssetID
	}
	if a.TenantID != "" {
		doc["tenant_id"] = a.TenantID
	}
	if a.ProjectID != "" {
		doc["project_id"] = a.ProjectID
	}
	if a.LastDeliveredTo != "" {
		doc["last_delivered_to"] = a.LastDeliveredTo
	}
	if a.ExpireAt != nil {
		doc["expire_at"] = a.ExpireAt.UTC().Format(time.RFC3339Nano)
	}
	if a.LastDeliveredAt != nil {
		doc["last_delivered_at"] = a.LastDeliveredAt.UTC().Format(time.RFC3339Nano)
	}
	doc["created_at"] = a.CreatedAt.UTC().Format(time.RFC3339Nano)
	doc["updated_at"] = a.UpdatedAt.UTC().Format(time.RFC3339Nano)

	if a.LogicalAssetID != "" {
		doc["logical_asset_id"] = a.LogicalAssetID
		doc["revision"] = a.Revision
		doc["is_current"] = a.IsCurrent
	}

	notes := ""
	if v, exists := meta["notes"]; exists {
		switch t := v.(type) {
		case string:
			notes = t
		default:
			notes = fmt.Sprint(t)
		}
	}

	tags, err := b.Tags.ListByAsset(ctx, assetID)
	if err != nil {
		return nil, false, err
	}
	tagsNested := make([]map[string]any, 0, len(tags))
	tagsFlat := doc["tags_flat"].(map[string]any)
	for _, t := range tags {
		entry := map[string]any{
			"key":         t.TagKey,
			"value":       t.TagValue,
			"source_type": t.SourceType,
			"source_name": t.SourceName,
		}
		if !t.UpdatedAt.IsZero() {
			entry["tagged_at"] = t.UpdatedAt.UTC().Format(time.RFC3339Nano)
		}
		tagsNested = append(tagsNested, entry)
		tagsFlat[t.TagKey] = t.TagValue
		if t.TagKey == "notes" && t.TagValue != "" {
			notes = t.TagValue
		}
	}
	doc["tags"] = tagsNested
	doc["notes"] = notes

	algos, err := b.Algos.ListByAsset(ctx, assetID)
	if err != nil {
		return nil, false, err
	}
	algoNested := make([]map[string]any, 0, len(algos))
	for _, al := range algos {
		entry := map[string]any{
			"name":    al.AlgoName,
			"version": al.AlgoVersion,
			"status":  al.Status,
		}
		if al.ResultTag != "" {
			entry["result_tag"] = al.ResultTag
		}
		if al.ResultScore != nil {
			entry["result_score"] = *al.ResultScore
		}
		if al.RunID != "" {
			entry["run_id"] = al.RunID
		}
		if al.FinishedAt != nil {
			entry["finished_at"] = al.FinishedAt.UTC().Format(time.RFC3339Nano)
		}
		algoNested = append(algoNested, entry)
	}
	doc["algos"] = algoNested

	if b.Mcap != nil && a.McapFileID != "" {
		mf, err := b.Mcap.Get(ctx, a.McapFileID)
		if err != nil {
			return nil, false, err
		}
		if mf != nil {
			mcapObj := map[string]any{}
			if mf.StartTimestampNs > 0 {
				mcapObj["recorded_at"] = time.Unix(0, mf.StartTimestampNs).UTC().Format(time.RFC3339Nano)
			}
			doc["mcap"] = mcapObj
		}
	}

	if a.StartTimestampNs > 0 {
		doc["recorded_at"] = time.Unix(0, a.StartTimestampNs).UTC().Format(time.RFC3339Nano)
	}

	if b.Lineage != nil {
		projection, err := b.Lineage.GetLineageProjection(ctx, assetID)
		if err != nil {
			return nil, false, err
		}
		if projection != nil {
			doc["lineage_upstream_ids"] = projection.UpstreamIDs
			doc["lineage_downstream_ids"] = projection.DownstreamIDs
			doc["lineage_relation_types"] = projection.RelationTypes
		}
	}

	// actions[] nested: re-read the seg's full action set on every projection.
	// CDC writes are at-least-once and reads are idempotent, so this is safe.
	if b.Actions != nil {
		actions, err := b.Actions.ListByAsset(ctx, assetID, repository.ActionListOptions{Limit: 1000})
		if err != nil {
			return nil, false, err
		}
		actionsNested := make([]map[string]any, 0, len(actions))
		for _, ax := range actions {
			if ax == nil {
				continue
			}
			entry := map[string]any{
				"action_id":   ax.ActionID,
				"start_ns":    ax.StartNs,
				"end_ns":      ax.EndNs,
				"labels":      ax.Labels,
				"source_type": ax.SourceType,
			}
			if ax.PrimaryLabel != "" {
				entry["primary_label"] = ax.PrimaryLabel
			}
			if ax.Description != "" {
				entry["description"] = ax.Description
			}
			if ax.SourceName != "" {
				entry["source_name"] = ax.SourceName
			}
			if ax.RunID != "" {
				entry["run_id"] = ax.RunID
			}
			if ax.Confidence != nil {
				entry["confidence"] = *ax.Confidence
			}
			if !ax.UpdatedAt.IsZero() {
				entry["updated_at"] = ax.UpdatedAt.UTC().Format(time.RFC3339Nano)
			}
			actionsNested = append(actionsNested, entry)
		}
		doc["actions"] = actionsNested
	}

	return doc, true, nil
}
