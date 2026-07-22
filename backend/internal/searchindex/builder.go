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
		"asset_id":               a.AssetID,
		"mcap_file_id":           a.McapFileID,
		"segment_locator":        a.SegmentLocator,
		"asset_type":             a.AssetType,
		"lifecycle_state":        lifecycleState,
		"status":                 string(a.Status), // deprecated — retained during dual-write window (§5.8.1)
		"is_deleted":             false,
		"version":                a.Version,
		"retention_tier":         a.RetentionTier,
		"storage_uri":            a.StorageURI,
		"thumb_uri":              a.ThumbURI,
		"files":                  a.Files, // CYB-3233: object-key URIs (algo_input_*/annot_*/delivery_*/raw_mcap/…)
		"owner":                  a.Owner,
		"reviewer":               a.Reviewer,
		"start_timestamp_ns":     a.StartTimestampNs,
		"end_timestamp_ns":       a.EndTimestampNs,
		"duration_ms":            a.DurationMs,
		"delivery_count":         a.DeliveryCount,
		"asset_level":            a.AssetLevel,
		"metadata":               meta,
		"tags_flat":              map[string]any{},
		"tags":                   []map[string]any{},
		"algos":                  []map[string]any{},
		"mcap":                   map[string]any{},
		"lineage_upstream_ids":   []string{},
		"lineage_downstream_ids": []string{},
		"lineage_relation_types": []string{},
	}

	// CYB-3268: action is a first-class asset but never traverses the
	// created→processing→ready lifecycle. Omit lifecycle_state so ES facets don't
	// surface a meaningless lifecycle:ready bucket for actions (PG hard-codes
	// 'ready' only to satisfy the NOT NULL + CHECK column).
	if a.AssetType == "action" {
		delete(doc, "lifecycle_state")
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
	// CYB-3715: top-level flatten fields mirrored from mcap_files onto assets.
	// Distinct from the nested mcap.<col> object below — the top-level form is
	// what filter/facet paths address after CYB-3715 landed in #526/#527.
	// Emit only when non-empty so dynamic mapping doesn't create every asset
	// doc with 7 empty-string properties.
	if a.CameraModel != "" {
		doc["camera_model"] = a.CameraModel
	}
	if a.DeviceID != "" {
		doc["device_id"] = a.DeviceID
	}
	if a.CollectorID != "" {
		doc["collector_id"] = a.CollectorID
	}
	if a.SceneID != "" {
		doc["scene_id"] = a.SceneID
	}
	if a.DataSource != "" {
		doc["data_source"] = a.DataSource
	}
	if a.CollectionMethod != "" {
		doc["collection_method"] = a.CollectionMethod
	}
	if a.SourcePlatform != "" {
		doc["source_platform"] = a.SourcePlatform
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
	addTypedMetadataProjection(doc, a.AssetType, meta)

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
			// CYB-3233: mcap object URI (mcap_uri column scans into GCSPath).
			if mf.GCSPath != "" {
				mcapObj["mcap_uri"] = mf.GCSPath
			}
			// CYB-3297 (Phase C): denormalize the mcap capture fields so the
			// "采集" filters/facets (vendor/device/scene/...) that the mapping,
			// registry and UI already advertise actually resolve. These columns
			// exist on mcap_files and are scanned by McapFileRepo.Get; only the
			// builder was omitting them. Additive: unset fields stay absent.
			if mf.VendorID != "" {
				mcapObj["vendor_id"] = mf.VendorID
			}
			if mf.DeviceID != "" {
				mcapObj["device_id"] = mf.DeviceID
			}
			if mf.CameraModel != "" {
				mcapObj["camera_model"] = mf.CameraModel
			}
			if mf.DataSource != "" {
				mcapObj["data_source"] = mf.DataSource
			}
			if mf.LocationID != "" {
				mcapObj["location_id"] = mf.LocationID
			}
			if mf.SceneID != "" {
				mcapObj["scene_id"] = mf.SceneID
			}
			if mf.EnvironmentID != "" {
				mcapObj["environment_id"] = mf.EnvironmentID
			}
			if mf.TaskID != "" {
				mcapObj["task_id"] = mf.TaskID
			}
			if mf.FileDurationMs > 0 {
				mcapObj["file_duration_ms"] = mf.FileDurationMs
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

	// CYB-3268: actions[] nested projection removed. Actions are now first-class
	// assets (asset_type='action') with their own top-level ES docs, so the seg
	// no longer carries a nested actions[] array. The Builder.Actions field is
	// retained (unused) pending cleanup; stale actions[] on old docs are wiped by
	// a one-off _update_by_query at deploy time.

	return doc, true, nil
}

func addTypedMetadataProjection(doc map[string]any, assetType string, meta map[string]any) {
	switch assetType {
	case "dataset":
		projection := map[string]any{}
		copyIfPresent(projection, meta, "format")
		copyIfPresent(projection, meta, "record_count")
		copyIfPresent(projection, meta, "size_bytes")
		copyIfPresent(projection, meta, "annotation_status")
		if len(projection) > 0 {
			doc["dataset"] = projection
		}
	case "annotation_result":
		projection := map[string]any{}
		copyIfPresent(projection, meta, "tool")
		copyIfPresent(projection, meta, "quality_score")
		copyIfPresent(projection, meta, "coverage")
		if len(projection) > 0 {
			doc["annotation_result"] = projection
		}
	case "ml_model":
		projection := map[string]any{}
		copyIfPresent(projection, meta, "framework")
		copyIfPresent(projection, meta, "architecture")
		copyIfPresent(projection, meta, "metrics")
		copyIfPresent(projection, meta, "quantization")
		copyIfPresent(projection, meta, "artifact_uri")
		if len(projection) > 0 {
			doc["ml_model"] = projection
		}
	case "evaluation_report":
		projection := map[string]any{}
		copyIfPresent(projection, meta, "model_id")
		copyIfPresent(projection, meta, "dataset_id")
		copyIfPresent(projection, meta, "metrics")
		copyIfPresent(projection, meta, "tool")
		if len(projection) > 0 {
			doc["evaluation_report"] = projection
		}
	}
}

func copyIfPresent(dst map[string]any, src map[string]any, key string) {
	if value, ok := src[key]; ok {
		dst[key] = value
	}
}
