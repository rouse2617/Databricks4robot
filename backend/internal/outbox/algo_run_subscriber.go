package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/elasticsearch"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// AlgoRunDocBuilder rebuilds the Elasticsearch document for an algo run
// from PostgreSQL.
type AlgoRunDocBuilder interface {
	Build(ctx context.Context, runID string) (doc map[string]any, ok bool, err error)
}

// AlgoRunESSubscriber consumes outbox events of aggregate_type="algo_run"
// and updates the "algo_runs" Elasticsearch index.
//
// It shares the same EventSubscriber (InMemoryBus) as the asset ESSubscriber
// but filters for algo_run events only. Asset events are silently skipped.
type AlgoRunESSubscriber struct {
	Subscriber EventSubscriber
	ES         *elasticsearch.Client
	Builder    AlgoRunDocBuilder
}

// Run blocks until ctx is cancelled.
func (s *AlgoRunESSubscriber) Run(ctx context.Context) error {
	if s == nil || s.Subscriber == nil || s.ES == nil || s.Builder == nil {
		return errors.New("outbox algo_run es subscriber: incomplete wiring")
	}
	return s.Subscriber.Receive(ctx, func(handlerCtx context.Context, data []byte) error {
		return s.handleData(handlerCtx, data)
	})
}

// handleData processes a single outbox event. Non-algo_run events are skipped.
func (s *AlgoRunESSubscriber) handleData(ctx context.Context, data []byte) error {
	var ev models.AssetEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		return err
	}

	// Only process algo_run aggregate events.
	if ev.AggregateType != "algo_run" {
		return nil
	}

	// The run_id is carried in the AssetID field for algo_run events
	// (the outbox table's aggregate identity column).
	runID := ev.AssetID
	if runID == "" {
		return nil
	}

	doc, ok, err := s.Builder.Build(ctx, runID)
	if err != nil {
		return err
	}
	if !ok {
		// Run was deleted — remove from ES.
		startDelete := time.Now()
		err := s.ES.DeleteDocument(ctx, runID)
		slog.Debug("algo_run es subscriber: delete", "run_id", runID, "duration_ms", time.Since(startDelete).Milliseconds(), "err", err)
		return err
	}

	startIndex := time.Now()
	res, err := s.ES.BulkIndex(ctx, []elasticsearch.BulkIndexDoc{{
		ID:  runID,
		Doc: doc,
	}})
	if err != nil {
		slog.Warn("algo_run es subscriber: bulk index failed", "run_id", runID, "err", err)
		return err
	}
	for _, okItem := range res.Succeeded {
		if okItem.Status == http.StatusConflict {
			slog.Debug("algo_run es subscriber: version conflict (ignored)", "run_id", runID)
		}
	}
	for _, f := range res.Failed {
		slog.Error("algo_run es subscriber: index failed", "run_id", f.ID, "error", f.Error)
		return errors.New("algo_run es subscriber: bulk index " + f.ID + ": " + f.Error)
	}

	slog.Debug("algo_run es subscriber: indexed", "run_id", runID, "duration_ms", time.Since(startIndex).Milliseconds())
	return nil
}

// Close releases subscriber resources.
func (s *AlgoRunESSubscriber) Close() error {
	if s == nil || s.Subscriber == nil {
		return nil
	}
	return s.Subscriber.Close()
}
