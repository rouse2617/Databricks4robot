package cdc

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/elasticsearch"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

var esConsumerEmptyAssetIDDebugCount atomic.Int32
var esConsumerUnknownTableDebugCount atomic.Int32

// AssetDocumentBuilder is the minimal interface needed to rebuild a complete
// Elasticsearch document for an asset from current-state PostgreSQL tables.
type AssetDocumentBuilder interface {
	Build(ctx context.Context, assetID string) (doc map[string]any, ok bool, err error)
}

// ESConsumer handles current-state CDC events and rebuilds the affected ES
// documents. It deliberately does not consume asset_events; long-term ES sync
// is driven by current-state table changes.
type ESConsumer struct {
	Assets             repository.AssetRepository
	Builder            AssetDocumentBuilder
	ES                 *elasticsearch.Client
	BatchSize          int
	RateLimitPerSecond float64
}

// Handle processes one normalized change event. Unknown tables are ignored.
func (c *ESConsumer) Handle(ctx context.Context, event ChangeEvent) error {
	return c.HandleBatch(ctx, []ChangeEvent{event})
}

// HandleBatch processes a batch of normalized CDC change events. Asset IDs are
// deduplicated before rebuilding documents so multiple row changes affecting the
// same asset only trigger one ES rebuild.
func (c *ESConsumer) HandleBatch(ctx context.Context, events []ChangeEvent) error {
	assetIDs, err := c.affectedAssetIDsForBatch(ctx, events)
	if err != nil {
		return err
	}
	if len(assetIDs) == 0 {
		return nil
	}

	CDCBatchProcessedTotal.WithLabelValues("search_projection").Add(float64(len(events)))

	rebuildStart := time.Now()

	var docs []elasticsearch.BulkIndexDoc
	for _, assetID := range assetIDs {
		doc, ok, err := c.Builder.Build(ctx, assetID)
		if err != nil {
			return fmt.Errorf("cdc es consumer build %s: %w", assetID, err)
		}
		if !ok {
			if err := c.ES.DeleteDocument(ctx, assetID); err != nil {
				return fmt.Errorf("cdc es consumer delete %s: %w", assetID, err)
			}
			continue
		}
		docs = append(docs, elasticsearch.BulkIndexDoc{ID: assetID, Doc: doc})
	}

	if len(docs) == 0 {
		CDCESRebuildDurationMs.Observe(float64(time.Since(rebuildStart).Milliseconds()))
		return nil
	}
	if err := c.bulkIndexInChunks(ctx, docs); err != nil {
		return err
	}
	CDCESRebuildDurationMs.Observe(float64(time.Since(rebuildStart).Milliseconds()))
	return nil
}

func (c *ESConsumer) bulkIndexInChunks(ctx context.Context, docs []elasticsearch.BulkIndexDoc) error {
	batchSize := c.BatchSize
	if batchSize <= 0 {
		batchSize = 200
	}
	delay := time.Duration(0)
	if c.RateLimitPerSecond > 0 {
		delay = time.Duration(float64(time.Second) / c.RateLimitPerSecond)
	}

	for i := 0; i < len(docs); i += batchSize {
		end := i + batchSize
		if end > len(docs) {
			end = len(docs)
		}
		result, err := c.ES.BulkIndex(ctx, docs[i:end])
		if err != nil {
			return fmt.Errorf("cdc es consumer bulk index: %w", err)
		}
		if len(result.Failed) > 0 {
			return fmt.Errorf("cdc es consumer: %d bulk item(s) failed", len(result.Failed))
		}
		if delay > 0 && end < len(docs) {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}
	return nil
}

func (c *ESConsumer) affectedAssetIDsForBatch(ctx context.Context, events []ChangeEvent) ([]string, error) {
	seen := make(map[string]struct{})
	assetIDs := make([]string, 0, len(events))
	for _, event := range events {
		ids, err := c.affectedAssetIDs(ctx, event)
		if err != nil {
			return nil, err
		}
		for _, assetID := range ids {
			if assetID == "" {
				continue
			}
			if _, exists := seen[assetID]; exists {
				continue
			}
			seen[assetID] = struct{}{}
			assetIDs = append(assetIDs, assetID)
		}
	}
	return assetIDs, nil
}

func (c *ESConsumer) affectedAssetIDs(ctx context.Context, event ChangeEvent) ([]string, error) {
	switch event.Table {
	case "assets", "asset_tags", "asset_algo_latest", "actions":
		assetID := assetIDFromChangeEvent(event, "asset_id")
		if assetID == "" {
			// Helps pinpoint why ES projection can't extract asset_id from the
			// normalized ChangeEvent payload (Debezium envelopes vary between
			// topics/operations).
			if esConsumerEmptyAssetIDDebugCount.Add(1) <= 20 {
				afterAssetID := ""
				if event.After != nil {
					if v, ok := event.After["asset_id"]; ok && v != nil {
						afterAssetID = fmt.Sprint(v)
					}
				}
				keyAssetID := ""
				if event.Key != nil {
					if payloadRaw, ok := event.Key["payload"]; ok && payloadRaw != nil {
						if payloadMap, ok := payloadRaw.(map[string]any); ok {
							if v, ok := payloadMap["asset_id"]; ok && v != nil {
								keyAssetID = fmt.Sprint(v)
							}
						}
					}
				}
				slog.Warn("cdc es consumer: empty asset_id extracted",
					"table", event.Table,
					"op", event.Op,
					"after_has_asset_id", afterAssetID != "",
					"key_has_asset_id", keyAssetID != "",
					"after_asset_id", afterAssetID,
					"key_asset_id", keyAssetID,
				)
			}
			return nil, nil
		}
		return []string{assetID}, nil
	case "mcap_files":
		mcapFileID := assetIDFromChangeEvent(event, "mcap_file_id")
		if mcapFileID == "" {
			return nil, nil
		}
		items, err := c.Assets.ListByMcapFile(ctx, mcapFileID)
		if err != nil {
			return nil, fmt.Errorf("cdc es consumer list assets by mcap %s: %w", mcapFileID, err)
		}
		assetIDs := make([]string, 0, len(items))
		seen := make(map[string]struct{}, len(items))
		for _, item := range items {
			if item == nil || item.AssetID == "" {
				continue
			}
			if _, exists := seen[item.AssetID]; exists {
				continue
			}
			seen[item.AssetID] = struct{}{}
			assetIDs = append(assetIDs, item.AssetID)
		}
		return assetIDs, nil
	default:
		if esConsumerUnknownTableDebugCount.Add(1) <= 20 {
			slog.Warn("cdc es consumer: unknown/ignored table",
				"table", event.Table,
				"op", event.Op,
			)
		}
		return nil, nil
	}
}

func assetIDFromChangeEvent(event ChangeEvent, key string) string {
	// 1) Prefer After/Before maps, because these are the most direct.
	if event.After != nil {
		if raw, ok := event.After[key]; ok && raw != nil {
			return fmt.Sprint(raw)
		}
	}
	if event.Before != nil {
		if raw, ok := event.Before[key]; ok && raw != nil {
			return fmt.Sprint(raw)
		}
	}

	// 2) Then check Key at the top level.
	if event.Key != nil {
		if raw, ok := event.Key[key]; ok && raw != nil {
			return fmt.Sprint(raw)
		}

		// 3) Debezium Kafka keys are sometimes wrapped:
		//    {"payload":{"asset_id":"..."}}
		if payloadRaw, ok := event.Key["payload"]; ok && payloadRaw != nil {
			// Common case: map[string]any
			if payloadMap, ok := payloadRaw.(map[string]any); ok {
				if raw, ok := payloadMap[key]; ok && raw != nil {
					return fmt.Sprint(raw)
				}
			}

			// Fallback: map[any]any (or other map-ish structures)
			rv := reflect.ValueOf(payloadRaw)
			if rv.IsValid() && rv.Kind() == reflect.Map {
				iter := rv.MapRange()
				for iter.Next() {
					k := iter.Key()
					v := iter.Value()
					if k.IsValid() && k.Kind() == reflect.String && k.String() == key && v.IsValid() && !v.IsZero() {
						return fmt.Sprint(v.Interface())
					}
				}
			}
		}
	}

	return ""
}
