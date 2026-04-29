// Package outbox implements the in-process ES sink worker for asset_events.
package outbox

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"data-platform/internal/elasticsearch"
	"data-platform/internal/metrics"
	"data-platform/internal/repository"
	"data-platform/internal/searchindex"
)

// ESWorker polls pending asset_events and projects affected assets into Elasticsearch.
type ESWorker struct {
	Events    repository.AssetEventRepository
	Indexer   *searchindex.Builder
	ES        *elasticsearch.Client
	BatchSize int
}

func (w *ESWorker) processBatch(ctx context.Context) error {
	if w.BatchSize <= 0 {
		w.BatchSize = 100
	}
	pending, err := w.Events.ListPending(ctx, w.BatchSize)
	if err != nil {
		return err
	}
	if len(pending) == 0 {
		return nil
	}

	seqsByAsset := make(map[string][]int64)
	var skipSeqs []int64
	for _, e := range pending {
		if e.AggregateType != "" && e.AggregateType != "asset" {
			skipSeqs = append(skipSeqs, e.EventSeq)
			continue
		}
		if e.AssetID == "" {
			skipSeqs = append(skipSeqs, e.EventSeq)
			continue
		}
		seqsByAsset[e.AssetID] = append(seqsByAsset[e.AssetID], e.EventSeq)
	}

	if len(skipSeqs) > 0 {
		if err := w.Events.MarkPublished(ctx, skipSeqs); err != nil {
			return fmt.Errorf("mark non-asset skipped events: %w", err)
		}
		metrics.OutboxEventsPublished.Add(float64(len(skipSeqs)))
	}

	var delSeqs []int64
	var docs []elasticsearch.BulkIndexDoc
	indexedAssetIDs := make([]string, 0, len(seqsByAsset))

	for assetID, seqs := range seqsByAsset {
		doc, ok, err := w.Indexer.Build(ctx, assetID)
		if err != nil {
			for _, seq := range seqs {
				_ = w.Events.MarkFailed(ctx, seq, err.Error())
			}
			metrics.OutboxEventsFailedMark.Add(float64(len(seqs)))
			continue
		}
		if !ok {
			if err := w.ES.DeleteDocument(ctx, assetID); err != nil {
				for _, seq := range seqs {
					_ = w.Events.MarkFailed(ctx, seq, err.Error())
				}
				metrics.OutboxEventsFailedMark.Add(float64(len(seqs)))
				continue
			}
			delSeqs = append(delSeqs, seqs...)
			continue
		}
		docs = append(docs, elasticsearch.BulkIndexDoc{ID: assetID, Doc: doc})
		indexedAssetIDs = append(indexedAssetIDs, assetID)
	}

	var markSeqs []int64
	markSeqs = append(markSeqs, delSeqs...)

	if len(docs) > 0 {
		_, err := w.ES.BulkIndex(ctx, docs)
		if err != nil {
			var fail int64
			for _, id := range indexedAssetIDs {
				fail += int64(len(seqsByAsset[id]))
				for _, seq := range seqsByAsset[id] {
					_ = w.Events.MarkFailed(ctx, seq, err.Error())
				}
			}
			metrics.OutboxEventsFailedMark.Add(float64(fail))
			return err
		}
		metrics.OutboxBulkDocs.Add(float64(len(docs)))
		for _, id := range indexedAssetIDs {
			markSeqs = append(markSeqs, seqsByAsset[id]...)
		}
	}

	if len(markSeqs) > 0 {
		if err := w.Events.MarkPublished(ctx, markSeqs); err != nil {
			return err
		}
		metrics.OutboxEventsPublished.Add(float64(len(markSeqs)))
	}
	return nil
}

// Run starts a polling loop until ctx is cancelled.
func (w *ESWorker) Run(ctx context.Context, tick time.Duration) {
	if tick <= 0 {
		tick = 30 * time.Second
	}
	t := time.NewTicker(tick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			n, err := w.Events.CountPending(ctx)
			if err == nil {
				metrics.OutboxPendingGauge.Set(float64(n))
			}
			if err := w.processBatch(ctx); err != nil {
				slog.Error("outbox worker batch failed", "err", err)
				continue
			}
			metrics.OutboxWorkerBatches.Inc()
		}
	}
}
