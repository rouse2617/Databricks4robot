package main

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	espkg "github.com/CyberOrigin2077/cyber-databrew/internal/elasticsearch"
	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	searchH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/search"
	"github.com/CyberOrigin2077/cyber-databrew/internal/lakehouse"
	lakehousebq "github.com/CyberOrigin2077/cyber-databrew/internal/lakehouse/bigquery"
	"github.com/CyberOrigin2077/cyber-databrew/internal/metrics"
	"github.com/CyberOrigin2077/cyber-databrew/internal/postgres"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	"github.com/CyberOrigin2077/cyber-databrew/internal/searchindex"
	assetUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/asset"
)

// ── Helpers (unchanged) ──

func buildSearchSyncInfo(cfg *config.Config, esClient *espkg.Client, outboxRelayStarted, outboxESSubscriberStarted bool) searchH.SyncInfo {
	info := searchH.SyncInfo{
		ElasticsearchOK:           esClient != nil,
		OutboxRelayEnabled:        outboxRelayStarted,
		OutboxESSubscriberEnabled: outboxESSubscriberStarted,
		Env:                       cfg.Env,
	}
	return info
}

func buildSearchProgress(ctx context.Context, cfg *config.Config, pgClient *postgres.Client, esClient *espkg.Client, assetRepo repository.AssetRepository, assetEventRepo repository.AssetEventRepository) (searchH.SyncProgress, error) {
	progress := searchH.SyncProgress{CheckedAt: time.Now().UTC()}
	lagSec, _ := strconv.Atoi(cfg.OutboxRelaySafetyLagSec)
	if lagSec < 0 {
		lagSec = 0
	}
	safetyLag := time.Duration(lagSec) * time.Second
	progress.OutboxRelaySafetyLagSec = float64(lagSec)

	if assetEventRepo != nil {
		pending, err := assetEventRepo.CountPending(ctx)
		if err != nil {
			return progress, err
		}
		progress.OutboxPendingEvents = pending
		claimable, err := assetEventRepo.CountPendingClaimable(ctx, safetyLag)
		if err != nil {
			return progress, err
		}
		progress.OutboxPendingClaimable = claimable
		proc, err := assetEventRepo.CountProcessing(ctx)
		if err != nil {
			return progress, err
		}
		progress.OutboxProcessingEvents = proc
		oldestAgeSec, err := assetEventRepo.OldestPendingAge(ctx)
		if err != nil {
			return progress, err
		}
		progress.OldestPendingAgeSec = oldestAgeSec
		if seqRepo, ok := assetEventRepo.(interface {
			MaxEventSeq(context.Context) (int64, error)
			MaxPublishedSeq(context.Context) (int64, error)
		}); ok {
			pgMax, err := seqRepo.MaxEventSeq(ctx)
			if err != nil {
				return progress, err
			}
			pubMax, err := seqRepo.MaxPublishedSeq(ctx)
			if err != nil {
				return progress, err
			}
			progress.PGMaxEventSeq = pgMax
			progress.OutboxPublishedMaxSeq = pubMax
			if pgMax > pubMax {
				progress.SeqLag = pgMax - pubMax
			}
		}
	}
	if pgClient != nil {
		cpRepo := postgres.NewESSyncCheckpointRepo(pgClient)
		shards, _ := strconv.Atoi(cfg.OutboxESCheckpointShards)
		if shards < 1 {
			shards = 1
		}
		idleAfter, _ := strconv.Atoi(cfg.OutboxESCheckpointIdleAfterSec)
		if idleAfter < 0 {
			idleAfter = 0
		}
		appliedMin, err := cpRepo.MinAppliedSeq(ctx, shards, idleAfter, progress.OutboxPublishedMaxSeq)
		if err != nil {
			return progress, err
		}
		progress.ESAppliedMinSeq = appliedMin
		if appliedMin > 0 && progress.OutboxPublishedMaxSeq > appliedMin {
			progress.ConsumerLag = progress.OutboxPublishedMaxSeq - appliedMin
		}
	}
	if assetRepo != nil {
		_, total, err := assetRepo.ListWithFilters(ctx, "", nil, 1, 1, filter.OrderByClause{SQL: "asset_id ASC"})
		if err != nil {
			return progress, err
		}
		progress.PostgresAssetsTotal = total
	}
	if esClient != nil {
		total, err := esClient.Count(ctx)
		if err != nil {
			return progress, err
		}
		progress.ElasticsearchDocsTotal = total
	}
	progress.PGESGap = progress.PostgresAssetsTotal - progress.ElasticsearchDocsTotal
	if progress.PostgresAssetsTotal > 0 {
		progress.PGESSyncRatio = float64(progress.ElasticsearchDocsTotal) / float64(progress.PostgresAssetsTotal)
	}
	metrics.OutboxPGMaxEventSeq.Set(float64(progress.PGMaxEventSeq))
	metrics.OutboxPublishedMaxSeq.Set(float64(progress.OutboxPublishedMaxSeq))
	metrics.OutboxSeqLag.Set(float64(progress.SeqLag))
	metrics.OutboxESAppliedMinSeq.Set(float64(progress.ESAppliedMinSeq))
	metrics.OutboxConsumerLag.Set(float64(progress.ConsumerLag))
	return progress, nil
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func startLocalSearchReconciler(
	ctx context.Context,
	assetRepo repository.AssetRepository,
	assetTagRepo repository.AssetTagRepository,
	algoLatestRepo repository.AssetAlgoLatestRepository,
	mcapRepo repository.McapFileRepository,
	actionRepo repository.ActionRepository,
	esClient *espkg.Client,
) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	indexer := &searchindex.Builder{
		Assets:  assetRepo,
		Tags:    assetTagRepo,
		Algos:   algoLatestRepo,
		Mcap:    mcapRepo,
		Actions: actionRepo,
	}

	reconcileOnce := func() {
		const pageSize = 200
		page := 1
		var docs []espkg.BulkIndexDoc
		for {
			assets, total, err := assetRepo.ListWithFilters(ctx, "", nil, page, pageSize, filter.OrderByClause{SQL: "asset_id ASC"})
			if err != nil {
				slog.Warn("local search reconciler: list assets failed", "err", err)
				return
			}
			if len(assets) == 0 {
				break
			}
			for _, a := range assets {
				if a == nil || a.AssetID == "" {
					continue
				}
				doc, ok, err := indexer.Build(ctx, a.AssetID)
				if err != nil {
					slog.Warn("local search reconciler: build failed", "asset_id", a.AssetID, "err", err)
					continue
				}
				if !ok {
					continue
				}
				docs = append(docs, espkg.BulkIndexDoc{ID: a.AssetID, Doc: doc})
				if len(docs) >= 200 {
					if _, err := esClient.BulkIndex(ctx, docs); err != nil {
						slog.Warn("local search reconciler: bulk index failed", "err", err)
						return
					}
					docs = docs[:0]
				}
			}
			if int64(page*pageSize) >= total {
				break
			}
			page++
		}
		if len(docs) > 0 {
			if _, err := esClient.BulkIndex(ctx, docs); err != nil {
				slog.Warn("local search reconciler: bulk index failed", "err", err)
				return
			}
		}
	}

	time.Sleep(5 * time.Second)
	reconcileOnce()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reconcileOnce()
		}
	}
}

func newAssetUsecase(
	repo repository.AssetRepository,
	tagRegistry *config.TagRegistry,
	algoRegistry *config.AlgoRegistry,
) *assetUC.Usecase {
	return assetUC.NewFull(repo, tagRegistry, algoRegistry)
}

func startOutboxPendingMetrics(ctx context.Context, transport string, repo repository.AssetEventRepository) {
	labelTransport := strings.TrimSpace(strings.ToLower(transport))
	if labelTransport == "" {
		labelTransport = "internal"
	}
	observe := func() {
		pending, err := repo.CountPending(ctx)
		if err != nil {
			slog.Warn("outbox pending metrics update failed", "err", err)
			return
		}
		metrics.OutboxPendingEvents.WithLabelValues(labelTransport).Set(float64(pending))
	}

	observe()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			observe()
		}
	}
}

func newLakehouseQuerier(ctx context.Context, cfg *config.Config) lakehouse.Querier {
	switch strings.ToLower(strings.TrimSpace(cfg.LakehouseBackend)) {
	case "", "none", "disabled":
		return lakehouse.Nop()
	case "bigquery":
		c, err := lakehousebq.New(ctx, lakehousebq.Config{
			Project: cfg.LakehouseBQProject,
			Dataset: cfg.LakehouseBQDataset,
		})
		if err != nil {
			slog.Warn("lakehouse bigquery init failed; degrading to nop",
				"err", err,
				"project", cfg.LakehouseBQProject,
				"dataset", cfg.LakehouseBQDataset)
			return lakehouse.Nop()
		}
		return c
	default:
		slog.Warn("unknown lakehouse backend; using nop",
			"backend", cfg.LakehouseBackend, "supported", []string{"bigquery", "none"})
		return lakehouse.Nop()
	}
}
