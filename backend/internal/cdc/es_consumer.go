package cdc

import (
	"context"
	"fmt"

	"data-platform/internal/elasticsearch"
	"data-platform/internal/repository"
)

// AssetDocumentBuilder is the minimal interface needed to rebuild a complete
// Elasticsearch document for an asset from current-state PostgreSQL tables.
type AssetDocumentBuilder interface {
	Build(ctx context.Context, assetID string) (doc map[string]any, ok bool, err error)
}

// ESConsumer handles current-state CDC events and rebuilds the affected ES
// documents. It deliberately does not consume asset_events; long-term ES sync
// is driven by current-state table changes.
type ESConsumer struct {
	Assets  repository.AssetRepository
	Builder AssetDocumentBuilder
	ES      *elasticsearch.Client
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
		return nil
	}
	result, err := c.ES.BulkIndex(ctx, docs)
	if err != nil {
		return fmt.Errorf("cdc es consumer bulk index: %w", err)
	}
	if len(result.Failed) > 0 {
		return fmt.Errorf("cdc es consumer: %d bulk item(s) failed", len(result.Failed))
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
	case "assets", "asset_tags", "asset_algo_latest":
		assetID := event.StringField("asset_id")
		if assetID == "" {
			return nil, nil
		}
		return []string{assetID}, nil
	case "mcap_files":
		mcapFileID := event.StringField("mcap_file_id")
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
		return nil, nil
	}
}
