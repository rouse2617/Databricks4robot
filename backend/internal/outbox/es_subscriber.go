package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"net/http"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/elasticsearch"
	"github.com/CyberOrigin2077/cyber-databrew/internal/metrics"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// ESCheckpointWriter is the minimal interface the subscriber needs to record
// per-shard high-water marks. Implemented by *postgres.ESSyncCheckpointRepo.
// Optional — when nil, checkpoint updates are skipped (consumer_lag stays at 0).
type ESCheckpointWriter interface {
	Upsert(ctx context.Context, shardID int, appliedSeq int64) error
}

// SearchDocBuilder rebuilds the Elasticsearch document for an asset from PostgreSQL.
type SearchDocBuilder interface {
	Build(ctx context.Context, assetID string) (doc map[string]any, ok bool, err error)
}

// ESSubscriber consumes outbox bus events and updates Elasticsearch.
//
// When the underlying Subscriber implements BatchEventSubscriber AND
// BatchSize > 1, ESSubscriber switches to a short-window batched path that
// coalesces same-asset events to a single Build + composes one ES _bulk
// request per worker batch (see handleBatch). Otherwise it falls back to the
// legacy one-event-per-BulkIndex path (handleData) so PubSub / Kafka transports
// keep working unchanged.
type ESSubscriber struct {
	Subscriber EventSubscriber
	ES         *elasticsearch.Client
	Builder    SearchDocBuilder

	// BatchSize controls the maximum number of events coalesced into a single
	// _bulk request. <=1 disables batching (legacy behavior).
	BatchSize int
	// BatchWaitMs is how long a worker is allowed to wait after the first
	// message arrives before flushing a partial batch. 0 means "drain whatever
	// is already buffered without waiting; flush as soon as the worker would
	// block on an empty queue".
	BatchWaitMs int

	// Checkpoint, when non-nil, receives an Upsert(shardID, maxSeq) after
	// every successful ES action. This powers the cross-shard MIN watermark
	// exposed as consumer_lag in /api/v1/search/sync-progress. Failures are
	// logged but never propagate — checkpoint is best-effort observability
	// and must not stall the data path.
	Checkpoint ESCheckpointWriter
	// CheckpointShards is the modulus used to fan event_seq updates across
	// es_sync_checkpoint rows. 0 falls back to 1 (a single global row), which
	// degrades to "any one event was applied" granularity and is suitable for
	// dev. Production should match OUTBOX_ES_CHECKPOINT_SHARDS (default 16).
	CheckpointShards int
}

// shardForEvent computes the checkpoint shard for an event using FNV(asset_id)
// (or "mcap:<id>" / "_na") % shards, mirroring the relay's ordering-key hash so
// events for the same asset always update the same shard row.
func shardForEvent(ev models.AssetEvent, shards int) int {
	if shards <= 1 {
		return 0
	}
	var key string
	switch {
	case ev.AssetID != "":
		key = ev.AssetID
	case ev.McapFileID != "":
		key = "mcap:" + ev.McapFileID
	default:
		key = "_na"
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return int(h.Sum32() % uint32(shards))
}

// advanceCheckpoint upserts per-shard max(event_seq). Errors are logged but
// not returned; the data path stays unaffected.
func (s *ESSubscriber) advanceCheckpoint(ctx context.Context, perShardMax map[int]int64) {
	if s.Checkpoint == nil || len(perShardMax) == 0 {
		return
	}
	for shardID, seq := range perShardMax {
		if err := s.Checkpoint.Upsert(ctx, shardID, seq); err != nil {
			slog.Warn("outbox es subscriber: checkpoint upsert failed (consumer_lag may stall)",
				"shard_id", shardID, "applied_seq", seq, "err", err)
		}
	}
}

// Run blocks until ctx is cancelled.
func (s *ESSubscriber) Run(ctx context.Context) error {
	if s == nil || s.Subscriber == nil || s.ES == nil || s.Builder == nil {
		return errors.New("outbox es subscriber: incomplete wiring")
	}
	if s.BatchSize > 1 {
		if bs, ok := s.Subscriber.(BatchEventSubscriber); ok {
			wait := time.Duration(s.BatchWaitMs) * time.Millisecond
			return bs.ReceiveBatch(ctx, s.BatchSize, wait, s.handleBatch)
		}
	}
	return s.Subscriber.Receive(ctx, func(handlerCtx context.Context, data []byte) error {
		return s.handleData(handlerCtx, data)
	})
}

func (s *ESSubscriber) handleData(ctx context.Context, data []byte) error {
	metrics.OutboxSubscriberBatchSize.Observe(1)
	var ev models.AssetEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		return err
	}
	if ev.AssetID == "" {
		return nil
	}
	doc, ok, err := s.Builder.Build(ctx, ev.AssetID)
	if err != nil {
		return err
	}
	if !ok {
		startDelete := time.Now()
		if err := s.ES.DeleteDocument(ctx, ev.AssetID); err != nil {
			metrics.OutboxSubscriberHandleDurationSeconds.WithLabelValues("delete").Observe(time.Since(startDelete).Seconds())
			metrics.OutboxSubscriberESErrorsTotal.WithLabelValues("other").Inc()
			return err
		}
		metrics.OutboxSubscriberHandleDurationSeconds.WithLabelValues("delete").Observe(time.Since(startDelete).Seconds())
		s.advanceCheckpoint(ctx, map[int]int64{shardForEvent(ev, s.CheckpointShards): ev.EventSeq})
		return nil
	}
	seq := ev.EventSeq
	startIndex := time.Now()
	res, err := s.ES.BulkIndex(ctx, []elasticsearch.BulkIndexDoc{{
		ID:              ev.AssetID,
		Doc:             doc,
		ExternalVersion: &seq,
	}})
	metrics.OutboxSubscriberHandleDurationSeconds.WithLabelValues("index").Observe(time.Since(startIndex).Seconds())
	if err != nil {
		metrics.OutboxSubscriberESErrorsTotal.WithLabelValues("other").Inc()
		return err
	}
	for _, okItem := range res.Succeeded {
		if okItem.Status == http.StatusConflict {
			metrics.OutboxSubscriberESErrorsTotal.WithLabelValues("conflict").Inc()
		}
	}
	for _, f := range res.Failed {
		metrics.OutboxSubscriberESErrorsTotal.WithLabelValues("other").Inc()
		return fmt.Errorf("elasticsearch bulk index %s: %s", f.ID, f.Error)
	}
	s.advanceCheckpoint(ctx, map[int]int64{shardForEvent(ev, s.CheckpointShards): seq})
	return nil
}

// handleBatch processes a batch of payloads delivered by a BatchEventSubscriber.
//
// Per-asset ordering is guaranteed by the upstream routing key (FNV(asset_id)
// % workers), so every event for one asset arrives in event_seq order on the
// same worker. We dedupe events for the same asset to a single Build call
// (latest event_seq wins) — Build re-reads the asset's full state from
// PostgreSQL, so collapsing N events into one rebuild is semantically
// equivalent to processing them sequentially as long as we send the resulting
// doc with ExternalVersion = max(event_seq); that prevents an older seq from
// overwriting a newer one if the relay ever retries.
//
// Build is intentionally serial per unique asset for v1 — the throughput win
// comes from a single _bulk HTTP round-trip rather than parallel rebuilds.
// TODO: optionally parallelize Build via golang.org/x/sync/errgroup with a
// small concurrency cap if PG read latency becomes the bottleneck.
//
// BulkIndex doesn't currently support delete actions; we use a hybrid path:
// one BulkIndex call for upserts, then sequential DeleteDocument calls for
// any assets where Build returned ok=false (rare — soft-deleted / missing).
//
// Errors fail the entire batch (see BatchEventSubscriber doc): the relay
// retries each event_seq independently. 409 conflicts on a doc are tolerated
// the same way the single-message path does it.
func (s *ESSubscriber) handleBatch(ctx context.Context, batch [][]byte) error {
	if len(batch) == 0 {
		return nil
	}
	metrics.OutboxSubscriberBatchSize.Observe(float64(len(batch)))

	type assetEntry struct {
		latestSeq int64
		ev        models.AssetEvent
	}
	assets := make(map[string]*assetEntry, len(batch))
	order := make([]string, 0, len(batch))

	for _, data := range batch {
		var ev models.AssetEvent
		if err := json.Unmarshal(data, &ev); err != nil {
			return err
		}
		if ev.AssetID == "" {
			continue
		}
		if existing, ok := assets[ev.AssetID]; ok {
			if ev.EventSeq > existing.latestSeq {
				existing.latestSeq = ev.EventSeq
				existing.ev = ev
			}
			continue
		}
		assets[ev.AssetID] = &assetEntry{latestSeq: ev.EventSeq, ev: ev}
		order = append(order, ev.AssetID)
	}

	if len(order) == 0 {
		return nil
	}

	docs := make([]elasticsearch.BulkIndexDoc, 0, len(order))
	deletes := make([]string, 0)
	// perShardMax accumulates max(event_seq) per checkpoint shard across this
	// batch. We only apply it after all bulk+delete actions succeed so a
	// failure mid-batch never advances a watermark past unapplied work.
	perShardMax := make(map[int]int64, len(order))
	noteShard := func(ev models.AssetEvent, seq int64) {
		shardID := shardForEvent(ev, s.CheckpointShards)
		if cur, ok := perShardMax[shardID]; !ok || seq > cur {
			perShardMax[shardID] = seq
		}
	}

	for _, assetID := range order {
		entry := assets[assetID]
		doc, ok, err := s.Builder.Build(ctx, assetID)
		if err != nil {
			return err
		}
		if !ok {
			deletes = append(deletes, assetID)
			noteShard(entry.ev, entry.latestSeq)
			continue
		}
		seq := entry.latestSeq
		docs = append(docs, elasticsearch.BulkIndexDoc{
			ID:              assetID,
			Doc:             doc,
			ExternalVersion: &seq,
		})
		noteShard(entry.ev, seq)
	}

	if len(docs) > 0 {
		startIndex := time.Now()
		res, err := s.ES.BulkIndex(ctx, docs)
		metrics.OutboxSubscriberHandleDurationSeconds.WithLabelValues("index").Observe(time.Since(startIndex).Seconds())
		if err != nil {
			metrics.OutboxSubscriberESErrorsTotal.WithLabelValues("other").Inc()
			return err
		}
		for _, okItem := range res.Succeeded {
			if okItem.Status == http.StatusConflict {
				metrics.OutboxSubscriberESErrorsTotal.WithLabelValues("conflict").Inc()
			}
		}
		for _, f := range res.Failed {
			metrics.OutboxSubscriberESErrorsTotal.WithLabelValues("other").Inc()
			return fmt.Errorf("elasticsearch bulk index %s: %s", f.ID, f.Error)
		}
	}

	for _, id := range deletes {
		startDelete := time.Now()
		if err := s.ES.DeleteDocument(ctx, id); err != nil {
			metrics.OutboxSubscriberHandleDurationSeconds.WithLabelValues("delete").Observe(time.Since(startDelete).Seconds())
			metrics.OutboxSubscriberESErrorsTotal.WithLabelValues("other").Inc()
			return err
		}
		metrics.OutboxSubscriberHandleDurationSeconds.WithLabelValues("delete").Observe(time.Since(startDelete).Seconds())
	}

	s.advanceCheckpoint(ctx, perShardMax)
	return nil
}

// Close releases subscriber resources.
func (s *ESSubscriber) Close() error {
	if s == nil || s.Subscriber == nil {
		return nil
	}
	return s.Subscriber.Close()
}
