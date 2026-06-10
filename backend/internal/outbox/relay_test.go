package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/metrics"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	dto "github.com/prometheus/client_model/go"
)

type relayEventRepo struct {
	pending       []*models.AssetEvent
	pendingErr    error
	markFailed    []int64
	markPublished []int64
}

func (m *relayEventRepo) Append(context.Context, repository.AssetEventAppendInput) error { return nil }
func (m *relayEventRepo) ListPending(context.Context, int) ([]*models.AssetEvent, error) {
	return nil, nil
}
func (m *relayEventRepo) ListPendingSafe(_ context.Context, _ time.Duration, _ int) ([]*models.AssetEvent, error) {
	if m.pendingErr != nil {
		return nil, m.pendingErr
	}
	return m.pending, nil
}
func (m *relayEventRepo) ListByAsset(context.Context, string, repository.AssetEventListOptions) ([]*models.AssetEvent, error) {
	return nil, nil
}
func (m *relayEventRepo) ListVersionPromotedByLogical(context.Context, string) ([]*models.AssetEvent, error) {
	return nil, nil
}

func (m *relayEventRepo) ListGlobal(_ context.Context, opts repository.AssetEventListOptions) ([]*models.AssetEvent, error) {
	return nil, nil
}

func (m *relayEventRepo) MarkPublished(_ context.Context, seqs []int64) error {
	m.markPublished = append(m.markPublished, seqs...)
	return nil
}
func (m *relayEventRepo) MarkFailed(_ context.Context, seq int64, _ string) error {
	m.markFailed = append(m.markFailed, seq)
	return nil
}
func (m *relayEventRepo) CountPending(context.Context) (int64, error) { return 0, nil }
func (m *relayEventRepo) CountPendingClaimable(context.Context, time.Duration) (int64, error) {
	return 0, nil
}
func (m *relayEventRepo) CountProcessing(context.Context) (int64, error)    { return 0, nil }
func (m *relayEventRepo) OldestPendingAge(context.Context) (float64, error) { return 0, nil }
func (m *relayEventRepo) ListBetweenSeq(context.Context, int64, int64, int) ([]*models.AssetEvent, error) {
	return nil, nil
}

func (m *relayEventRepo) PublishStateCounts(context.Context) (map[string]int64, error) {
	return map[string]int64{}, nil
}

func TestOrderingKeyFor(t *testing.T) {
	if got := orderingKeyFor(&models.AssetEvent{AssetID: "a1"}); got != "a1" {
		t.Fatalf("asset ordering key = %q", got)
	}
	if got := orderingKeyFor(&models.AssetEvent{McapFileID: "m1"}); got != "mcap:m1" {
		t.Fatalf("mcap ordering key = %q", got)
	}
	if got := orderingKeyFor(&models.AssetEvent{}); got != "_na" {
		t.Fatalf("fallback ordering key = %q", got)
	}
}

func TestRelayPublishOne_MarkFailedOnPublishError(t *testing.T) {
	evRepo := &relayEventRepo{}
	relay := &Relay{
		Events:    evRepo,
		Publisher: &Publisher{},
		Config:    DefaultRelayConfig(),
	}
	err := relay.publishOne(context.Background(), DefaultRelayConfig(), &models.AssetEvent{
		EventSeq: 10,
		AssetID:  "a1",
	})
	if err != nil {
		t.Fatalf("expected mark-failed path to return nil, got %v", err)
	}
	if len(evRepo.markFailed) != 1 || evRepo.markFailed[0] != 10 {
		t.Fatalf("mark failed mismatch: %#v", evRepo.markFailed)
	}
}

func TestRelayPublishOne_SkipWhenRetryExceeded(t *testing.T) {
	evRepo := &relayEventRepo{}
	relay := &Relay{
		Events:    evRepo,
		Publisher: &Publisher{},
	}
	cfg := DefaultRelayConfig()
	cfg.MaxRetries = 2
	if err := relay.publishOne(context.Background(), cfg, &models.AssetEvent{
		EventSeq:    11,
		AssetID:     "a1",
		RetryCount:  3,
		EventSource: "backend",
	}); err != nil {
		t.Fatalf("expected nil error when skipped, got %v", err)
	}
	if len(evRepo.markFailed) != 0 || len(evRepo.markPublished) != 0 {
		t.Fatalf("skip should not mark state: failed=%v published=%v", evRepo.markFailed, evRepo.markPublished)
	}
}

func TestRelayFlushOnce_PropagatesListError(t *testing.T) {
	evRepo := &relayEventRepo{pendingErr: errors.New("boom")}
	relay := &Relay{
		Events:    evRepo,
		Publisher: &Publisher{},
	}
	err := relay.flushOnce(context.Background(), DefaultRelayConfig())
	if err == nil {
		t.Fatalf("expected list error")
	}
}

func TestRoutingKeyFromPayloadMatchesOrderingKeyFor(t *testing.T) {
	ev := &models.AssetEvent{EventSeq: 1, AssetID: "asset-1", EventSource: "test"}
	raw, err := json.Marshal(ev)
	if err != nil {
		t.Fatal(err)
	}
	if routingKeyFromPayload(raw) != orderingKeyFor(ev) {
		t.Fatalf("asset key mismatch: payload=%q ordering=%q", routingKeyFromPayload(raw), orderingKeyFor(ev))
	}
	ev2 := &models.AssetEvent{EventSeq: 2, McapFileID: "mf1", EventSource: "test"}
	raw2, _ := json.Marshal(ev2)
	if routingKeyFromPayload(raw2) != orderingKeyFor(ev2) {
		t.Fatalf("mcap key mismatch")
	}
}

type noopPublisher struct{}

func (noopPublisher) Publish(context.Context, string, []byte) (PublishReceipt, error) {
	return immediateReceipt{}, nil
}

func (noopPublisher) ResumePublishAfterError(string) {}

func (noopPublisher) Close() error { return nil }

func TestRelayFlushOnce_ParallelKeysMarksPublished(t *testing.T) {
	evRepo := &relayEventRepo{
		pending: []*models.AssetEvent{
			{EventSeq: 1, AssetID: "a", EventSource: "test"},
			{EventSeq: 2, AssetID: "b", EventSource: "test"},
		},
	}
	cfg := DefaultRelayConfig()
	cfg.ParallelOrderingKeys = 8
	relay := &Relay{
		Events:    evRepo,
		Publisher: noopPublisher{},
		Config:    cfg,
	}
	if err := relay.flushOnce(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	if len(evRepo.markPublished) != 2 {
		t.Fatalf("expected 2 published marks, got %v", evRepo.markPublished)
	}
}

func TestRelayPublishOne_EmitsPublishedAndLagMetrics(t *testing.T) {
	evRepo := &relayEventRepo{}
	relay := &Relay{
		Events:    evRepo,
		Publisher: noopPublisher{},
	}
	cfg := DefaultRelayConfig()
	ev := &models.AssetEvent{
		EventSeq:    99,
		AssetID:     "asset-metric",
		EventSource: "test",
		CreatedAt:   time.Now().Add(-2 * time.Second),
	}
	beforePublished := readCounterValue(t, metrics.OutboxRelayPublishedTotal.WithLabelValues("asset"))
	beforeLagCount := readHistogramCount(t, metrics.OutboxEventLagSeconds)

	if err := relay.publishOne(context.Background(), cfg, ev); err != nil {
		t.Fatalf("publishOne returned error: %v", err)
	}

	afterPublished := readCounterValue(t, metrics.OutboxRelayPublishedTotal.WithLabelValues("asset"))
	afterLagCount := readHistogramCount(t, metrics.OutboxEventLagSeconds)
	if afterPublished-beforePublished != 1 {
		t.Fatalf("expected published counter +1, got before=%v after=%v", beforePublished, afterPublished)
	}
	if afterLagCount-beforeLagCount != 1 {
		t.Fatalf("expected lag histogram count +1, got before=%v after=%v", beforeLagCount, afterLagCount)
	}
}

func readCounterValue(t *testing.T, c interface{ Write(*dto.Metric) error }) float64 {
	t.Helper()
	m := &dto.Metric{}
	if err := c.Write(m); err != nil {
		t.Fatalf("read counter metric: %v", err)
	}
	return m.GetCounter().GetValue()
}

func readHistogramCount(t *testing.T, h interface{ Write(*dto.Metric) error }) uint64 {
	t.Helper()
	m := &dto.Metric{}
	if err := h.Write(m); err != nil {
		t.Fatalf("read histogram metric: %v", err)
	}
	return m.GetHistogram().GetSampleCount()
}

func readGaugeValue(t *testing.T, g interface{ Write(*dto.Metric) error }) float64 {
	t.Helper()
	m := &dto.Metric{}
	if err := g.Write(m); err != nil {
		t.Fatalf("read gauge metric: %v", err)
	}
	return m.GetGauge().GetValue()
}
