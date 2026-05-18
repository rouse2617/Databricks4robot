package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	adminH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/admin"
	searchH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/search"
	"github.com/CyberOrigin2077/cyber-databrew/internal/outbox"
	"github.com/CyberOrigin2077/cyber-databrew/internal/postgres"
	"github.com/CyberOrigin2077/cyber-databrew/internal/searchindex"
	adminUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/admin"
)

// ── Layer 3: Optional / admin / outbox ──

func setupOptional(inf *infra, core *coreHandlers) *optional {
	ctx := context.Background()
	pg := inf.pg
	es := inf.es
	cfg := inf.cfg

	// Admin handler (requires both PG and ES).
	var adminHandler *adminH.Handler
	if pg != nil && es != nil {
		adminHandler = adminH.New(
			postgres.NewAssetRepo(pg),
			postgres.NewAssetTagRepo(pg),
			postgres.NewAssetAlgoLatestRepo(pg),
			postgres.NewMcapFileRepo(pg),
			postgres.NewActionRepo(pg),
			es,
			postgres.NewAssetEventRepo(pg),
			postgres.NewOutboxDLQRepo(pg),
			postgres.NewSearchReindexJobRepo(pg),
		)
	}
	var purgeHandler *adminH.PurgeHandler
	if pg != nil {
		batchOpsRepo := postgres.NewBatchOpsRepo(pg)
		purgeHandler = adminH.NewPurgeHandler(adminUC.New(batchOpsRepo).WithES(es))
	}

	// ── Outbox ──
	outboxCtx, outboxCancel := context.WithCancel(context.Background())
	outboxRelayStarted := false
	outboxESSubscriberStarted := false

	outboxTransport := strings.ToLower(strings.TrimSpace(cfg.OutboxTransport))
	if outboxTransport == "" {
		outboxTransport = "internal"
	}
	var inMemoryBus *outbox.InMemoryBus
	internalBusBuffer, _ := strconv.Atoi(cfg.OutboxInternalBusBuffer)
	if internalBusBuffer <= 0 {
		internalBusBuffer = 1024
	}
	getInMemoryBus := func() *outbox.InMemoryBus {
		if inMemoryBus == nil {
			inMemoryBus = outbox.NewInMemoryBus(internalBusBuffer)
		}
		return inMemoryBus
	}
	resolveKafkaTopic := func() string {
		if t := strings.TrimSpace(cfg.OutboxKafkaTopic); t != "" {
			return t
		}
		return strings.TrimSpace(cfg.TopicAssetEvents)
	}

	assetEventRepo := postgres.NewAssetEventRepo(pg)

	// Outbox relay.
	if pg != nil && cfg.OutboxRelayEnabled == "true" {
		var pub outbox.EventPublisher
		switch outboxTransport {
		case "internal":
			var err error
			pub, err = outbox.NewInternalPublisher(getInMemoryBus())
			if err != nil {
				slog.Error("outbox internal publisher init failed", "err", err)
				os.Exit(1)
			}
		case "pubsub":
			if strings.TrimSpace(cfg.PubSubProject) == "" || strings.TrimSpace(cfg.TopicAssetEvents) == "" {
				slog.Error("outbox relay pubsub transport requires PUBSUB_PROJECT and TOPIC_ASSET_EVENTS")
				os.Exit(1)
			}
			var err error
			pub, err = outbox.NewPublisher(ctx, cfg.PubSubProject, cfg.TopicAssetEvents)
			if err != nil {
				slog.Error("outbox publisher init failed", "err", err)
				os.Exit(1)
			}
		case "kafka":
			brokers := splitCSV(cfg.OutboxKafkaBrokers)
			topic := resolveKafkaTopic()
			var err error
			pub, err = outbox.NewKafkaPublisher(brokers, topic)
			if err != nil {
				slog.Error("outbox kafka publisher init failed", "err", err, "topic", topic)
				os.Exit(1)
			}
		default:
			slog.Error("invalid OUTBOX_TRANSPORT", "value", outboxTransport, "allowed", "internal|pubsub|kafka")
			os.Exit(1)
		}
		defer func() { _ = pub.Close() }()

		batch, _ := strconv.Atoi(cfg.OutboxRelayBatchSize)
		intervalMs, _ := strconv.Atoi(cfg.OutboxRelayIntervalMs)
		lagSec, _ := strconv.Atoi(cfg.OutboxRelaySafetyLagSec)
		leaseSec, _ := strconv.Atoi(cfg.OutboxRelayLeaseSec)
		maxRetries, _ := strconv.Atoi(cfg.OutboxRelayMaxRetries)
		progressLogSec, _ := strconv.Atoi(cfg.OutboxProgressLogInterval)
		parallelKeys, _ := strconv.Atoi(cfg.OutboxRelayParallelKeys)
		relayCfg := outbox.RelayConfig{
			BatchSize:            batch,
			Interval:             time.Duration(intervalMs) * time.Millisecond,
			SafetyLag:            time.Duration(lagSec) * time.Second,
			ProcessingLease:      time.Duration(leaseSec) * time.Second,
			MaxRetries:           maxRetries,
			ProgressLogInterval:  time.Duration(progressLogSec) * time.Second,
			DLQEveryNBatches:     20,
			ParallelOrderingKeys: parallelKeys,
		}
		relay := &outbox.Relay{
			Events:    assetEventRepo,
			DLQ:       postgres.NewOutboxDLQRepo(pg),
			Publisher: pub,
			Config:    relayCfg,
		}
		outboxRelayStarted = true
		slog.Info("outbox relay starting", "transport", outboxTransport, "topic", cfg.TopicAssetEvents, "parallel_ordering_keys", relayCfg.ParallelOrderingKeys)
		go func() {
			if err := relay.Run(outboxCtx); err != nil && !errors.Is(err, context.Canceled) {
				slog.Error("outbox relay exited", "err", err)
			}
		}()
	}

	// Outbox ES subscriber.
	if pg != nil && es != nil && cfg.OutboxESSubscriberEnabled == "true" {
		var subscriber outbox.EventSubscriber
		switch outboxTransport {
		case "internal":
			wWorkers, _ := strconv.Atoi(cfg.OutboxInternalSubscriberWorkers)
			var err error
			subscriber, err = outbox.NewInternalSubscriber(getInMemoryBus(), wWorkers)
			if err != nil {
				slog.Error("outbox internal subscriber init failed", "err", err)
				os.Exit(1)
			}
		case "pubsub":
			if strings.TrimSpace(cfg.PubSubProject) == "" || strings.TrimSpace(cfg.OutboxESSubscription) == "" {
				slog.Error("outbox es subscriber pubsub transport requires PUBSUB_PROJECT and OUTBOX_ES_SUBSCRIPTION")
				os.Exit(1)
			}
			var err error
			subscriber, err = outbox.NewPubSubSubscriber(ctx, cfg.PubSubProject, cfg.OutboxESSubscription)
			if err != nil {
				slog.Error("outbox pubsub subscriber init failed", "err", err)
				os.Exit(1)
			}
		case "kafka":
			brokers := splitCSV(cfg.OutboxKafkaBrokers)
			topic := resolveKafkaTopic()
			groupID := strings.TrimSpace(cfg.OutboxKafkaGroupID)
			var err error
			subscriber, err = outbox.NewKafkaSubscriber(brokers, topic, groupID)
			if err != nil {
				slog.Error("outbox kafka subscriber init failed", "err", err, "topic", topic, "group_id", groupID)
				os.Exit(1)
			}
		default:
			slog.Error("invalid OUTBOX_TRANSPORT", "value", outboxTransport, "allowed", "internal|pubsub|kafka")
			os.Exit(1)
		}
		defer func() { _ = subscriber.Close() }()
		esBatchSize, _ := strconv.Atoi(cfg.OutboxInternalSubscriberBatchSize)
		if esBatchSize < 1 {
			esBatchSize = 1
		}
		esBatchWaitMs, _ := strconv.Atoi(cfg.OutboxInternalSubscriberBatchWaitMs)
		if esBatchWaitMs < 0 {
			esBatchWaitMs = 0
		}
		checkpointShards, _ := strconv.Atoi(cfg.OutboxESCheckpointShards)
		if checkpointShards < 1 {
			checkpointShards = 1
		} else if checkpointShards > 1024 {
			checkpointShards = 1024
		}
		var esCheckpoint outbox.ESCheckpointWriter
		if pg != nil {
			esCheckpoint = postgres.NewESSyncCheckpointRepo(pg)
		}
		esSub := &outbox.ESSubscriber{
			Subscriber: subscriber,
			ES:         es,
			Builder: &searchindex.Builder{
				Assets:  postgres.NewAssetRepo(pg),
				Tags:    postgres.NewAssetTagRepo(pg),
				Algos:   postgres.NewAssetAlgoLatestRepo(pg),
				Mcap:    postgres.NewMcapFileRepo(pg),
				Actions: postgres.NewActionRepo(pg),
			},
			BatchSize:        esBatchSize,
			BatchWaitMs:      esBatchWaitMs,
			Checkpoint:       esCheckpoint,
			CheckpointShards: checkpointShards,
		}
		outboxESSubscriberStarted = true
		esWorkers := 0
		if outboxTransport == "internal" {
			esWorkers, _ = strconv.Atoi(cfg.OutboxInternalSubscriberWorkers)
		}
		slog.Info(
			"outbox es subscriber starting",
			"transport", outboxTransport,
			"subscription", cfg.OutboxESSubscription,
			"internal_workers", esWorkers,
			"internal_bus_buffer", internalBusBuffer,
			"batch_size", esBatchSize,
			"batch_wait_ms", esBatchWaitMs,
		)
		go func() {
			if err := esSub.Run(outboxCtx); err != nil && !errors.Is(err, context.Canceled) {
				slog.Error("outbox es subscriber exited", "err", err)
			}
		}()
	}

	if cfg.OutboxRelayEnabled == "true" || cfg.OutboxESSubscriberEnabled == "true" {
		go startOutboxPendingMetrics(outboxCtx, outboxTransport, assetEventRepo)
	}

	// Dev safety net: periodic ES reconciliation when outbox subscriber is off.
	if pg != nil && es != nil && cfg.Env != "production" && !outboxESSubscriberStarted {
		go startLocalSearchReconciler(outboxCtx,
			postgres.NewAssetRepo(pg),
			postgres.NewAssetTagRepo(pg),
			postgres.NewAssetAlgoLatestRepo(pg),
			postgres.NewMcapFileRepo(pg),
			postgres.NewActionRepo(pg),
			es,
		)
	}

	// ── Config watcher for hot-reload of registries ──
	configWatcher, err := config.NewConfigWatcher("config", inf.tagRegistry, inf.algoRegistry, inf.actionLabelReg)
	if err != nil {
		slog.Warn("config watcher failed to start, hot-reload disabled", "err", err)
	} else {
		slog.Info("config watcher started for hot-reload")
	}

	// ── Search sync helpers ──
	searchSyncFn := func() searchH.SyncInfo {
		return buildSearchSyncInfo(cfg, es, outboxRelayStarted, outboxESSubscriberStarted)
	}
	searchProgressFn := func(ctx context.Context) (searchH.SyncProgress, error) {
		return buildSearchProgress(ctx, cfg, pg, es,
			postgres.NewAssetRepo(pg),
			postgres.NewAssetEventRepo(pg),
		)
	}

	return &optional{
		admin:                     adminHandler,
		purge:                     purgeHandler,
		outboxCancel:              outboxCancel,
		configWatcher:             configWatcher,
		outboxRelayStarted:        outboxRelayStarted,
		outboxESSubscriberStarted: outboxESSubscriberStarted,
		searchSyncFn:              searchSyncFn,
		searchProgressFn:          searchProgressFn,
	}
}
