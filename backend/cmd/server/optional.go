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
	"github.com/CyberOrigin2077/cyber-databrew/internal/deliveryrules"
	espkg "github.com/CyberOrigin2077/cyber-databrew/internal/elasticsearch"
	adminH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/admin"
	searchH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/search"
	"github.com/CyberOrigin2077/cyber-databrew/internal/lifecycle"
	"github.com/CyberOrigin2077/cyber-databrew/internal/openlineage"
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
			// C3 fix: each subscriber gets its own channel on the fan-out bus
			// (InMemoryBus.Subscribe). Lifecycle tied to outboxCtx; unsub
			// closure is kept to prevent GC but never called explicitly.
			busCh, _ := getInMemoryBus().Subscribe()
			var err error
			subscriber, err = outbox.NewInternalSubscriber(busCh, wWorkers)
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
				Lineage: postgres.NewAssetRepo(pg),
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

	// AlgoRun ES subscriber — syncs algo_runs to a separate ES index.
	// Runs alongside the asset ESSubscriber when outbox is enabled.
	if pg != nil && es != nil && cfg.OutboxESSubscriberEnabled == "true" {
		algoRunES := espkg.New(cfg.ElasticsearchURL, "algo_runs", cfg.ElasticsearchUsername, cfg.ElasticsearchPassword)
		var algoRunSub outbox.EventSubscriber
		switch outboxTransport {
		case "internal":
			w, _ := strconv.Atoi(cfg.OutboxInternalSubscriberWorkers)
			algoCh, _ := getInMemoryBus().Subscribe()
			var err error
			algoRunSub, err = outbox.NewInternalSubscriber(algoCh, w)
			if err != nil {
				slog.Error("algo_run internal subscriber init failed", "err", err)
				os.Exit(1)
			}
		case "pubsub":
			algoRunSubID := strings.TrimSpace(cfg.OutboxESAlgoRunSubscription)
			if algoRunSubID == "" {
				algoRunSubID = strings.TrimSpace(cfg.OutboxESSubscription) + "-algorun"
			}
			var err error
			algoRunSub, err = outbox.NewPubSubSubscriber(ctx, cfg.PubSubProject, algoRunSubID)
			if err != nil {
				slog.Error("algo_run pubsub subscriber init failed", "err", err)
				os.Exit(1)
			}
			slog.Info("algo_run es subscriber using dedicated subscription", "subscription", algoRunSubID)
		case "kafka":
			brokers := splitCSV(cfg.OutboxKafkaBrokers)
			topic := resolveKafkaTopic()
			groupID := strings.TrimSpace(cfg.OutboxKafkaAlgoRunGroupID)
			var err error
			algoRunSub, err = outbox.NewKafkaSubscriber(brokers, topic, groupID)
			if err != nil {
				slog.Error("algo_run kafka subscriber init failed", "err", err, "topic", topic, "group_id", groupID)
				os.Exit(1)
			}
		}
		algoRunESSub := &outbox.AlgoRunESSubscriber{
			Subscriber: algoRunSub,
			ES:         algoRunES,
			Builder: &searchindex.AlgoRunBuilder{
				Runs: postgres.NewAlgoRunRepo(pg),
			},
		}
		slog.Info("algo_run es subscriber starting", "transport", outboxTransport, "index", "algo_runs")
		go func() {
			if err := algoRunESSub.Run(outboxCtx); err != nil && !errors.Is(err, context.Canceled) {
				slog.Error("algo_run es subscriber exited", "err", err)
			}
		}()
	}

	if cfg.OutboxRelayEnabled == "true" || cfg.OutboxESSubscriberEnabled == "true" {
		go startOutboxPendingMetrics(outboxCtx, outboxTransport, assetEventRepo)
	}

	if cfg.OpenLineageEmitterEnabled == "true" {
		if strings.TrimSpace(cfg.PubSubProject) == "" || strings.TrimSpace(cfg.OpenLineageSubscription) == "" {
			slog.Error("openlineage emitter requires PUBSUB_PROJECT and OPENLINEAGE_SUBSCRIPTION")
			os.Exit(1)
		}
		timeoutMs, _ := strconv.Atoi(cfg.OpenLineageTimeoutMs)
		emitter, err := openlineage.NewEmitter(cfg.OpenLineageEndpoint, time.Duration(timeoutMs)*time.Millisecond)
		if err != nil {
			slog.Error("openlineage emitter init failed", "err", err)
			os.Exit(1)
		}
		lineageSub, err := outbox.NewPubSubSubscriber(ctx, cfg.PubSubProject, cfg.OpenLineageSubscription)
		if err != nil {
			slog.Error("openlineage pubsub subscriber init failed", "err", err)
			os.Exit(1)
		}
		subscriber := &openlineage.Subscriber{
			Source: lineageSub,
			Builder: openlineage.Builder{
				Namespace: cfg.OpenLineageNamespace,
				Producer:  cfg.OpenLineageProducer,
			},
			Emitter: emitter,
		}
		slog.Info("openlineage emitter starting", "subscription", cfg.OpenLineageSubscription, "endpoint", cfg.OpenLineageEndpoint)
		go func() {
			defer func() { _ = lineageSub.Close() }()
			if err := subscriber.Run(outboxCtx); err != nil && !errors.Is(err, context.Canceled) {
				slog.Error("openlineage emitter exited", "err", err)
			}
		}()
	}

	// ── DeliveryEligibilityProjector — automatic delivery readiness tagging ──
	if pg != nil && cfg.DeliveryEligibilityProjectorEnabled == "true" {
		// Build projector components.
		engine := deliveryrules.NewEngine(
			postgres.NewDeliveryRuleRepo(pg),
			postgres.NewAssetRepo(pg),
			postgres.NewAssetTagRepo(pg),
			postgres.NewCustomerRepo(pg),
		)
		var subscriber outbox.EventSubscriber
		switch outboxTransport {
		case "internal":
			w, _ := strconv.Atoi(cfg.OutboxInternalSubscriberWorkers)
			// C3 fix: own channel on the fan-out bus.
			delivCh, _ := getInMemoryBus().Subscribe()
			var err error
			subscriber, err = outbox.NewInternalSubscriber(delivCh, w)
			if err != nil {
				slog.Error("delivery eligibility projector internal subscriber init failed", "err", err)
				os.Exit(1)
			}
		case "pubsub":
			// C3 fix: dedicated subscription.
			delivSubID := strings.TrimSpace(cfg.OutboxESDeliverySubscription)
			if delivSubID == "" {
				delivSubID = strings.TrimSpace(cfg.OutboxESSubscription) + "-delivery"
			}
			if strings.TrimSpace(cfg.PubSubProject) == "" || delivSubID == "" {
				slog.Error("delivery eligibility projector pubsub transport requires PUBSUB_PROJECT and OUTBOX_ES_DELIVERY_SUBSCRIPTION (or fallback OUTBOX_ES_SUBSCRIPTION)")
				os.Exit(1)
			}
			var err error
			subscriber, err = outbox.NewPubSubSubscriber(ctx, cfg.PubSubProject, delivSubID)
			if err != nil {
				slog.Error("delivery eligibility projector pubsub subscriber init failed", "err", err)
				os.Exit(1)
			}
			slog.Info("delivery eligibility projector using dedicated subscription", "subscription", delivSubID)
		case "kafka":
			brokers := splitCSV(cfg.OutboxKafkaBrokers)
			topic := resolveKafkaTopic()
			// C3 fix: dedicated consumer group.
			groupID := strings.TrimSpace(cfg.OutboxKafkaDeliveryGroupID)
			var err error
			subscriber, err = outbox.NewKafkaSubscriber(brokers, topic, groupID)
			if err != nil {
				slog.Error("delivery eligibility projector kafka subscriber init failed", "err", err, "topic", topic, "group_id", groupID)
				os.Exit(1)
			}
		default:
			slog.Error("invalid OUTBOX_TRANSPORT for delivery eligibility projector", "value", outboxTransport, "allowed", "internal|pubsub|kafka")
			os.Exit(1)
		}
		projector := outbox.NewDeliveryEligibilityProjector(
			subscriber,
			engine,
			postgres.NewAssetTagRepo(pg),
			postgres.NewAssetRepo(pg),
			postgres.NewCustomerRepo(pg),
		)
		slog.Info("delivery eligibility projector starting", "transport", outboxTransport)
		go func() {
			defer func() { _ = subscriber.Close() }()
			if err := projector.Run(outboxCtx); err != nil && !errors.Is(err, context.Canceled) {
				slog.Error("delivery eligibility projector exited", "err", err)
			}
		}()
	}

	// ── Retention job: archive expired assets ──
	if pg != nil && strings.EqualFold(strings.TrimSpace(os.Getenv("RETENTION_ENABLED")), "true") {
		retentionJob := &lifecycle.RetentionJob{
			Assets:   postgres.NewAssetRepo(pg),
			Events:   assetEventRepo,
			TxRunner: pg,
			Config:   lifecycle.DefaultRetentionConfig(),
		}
		slog.Info("retention job starting", "interval", retentionJob.Config.Interval, "batch_size", retentionJob.Config.BatchSize)
		go func() {
			if err := retentionJob.Run(outboxCtx); err != nil && !errors.Is(err, context.Canceled) {
				slog.Error("retention job exited", "err", err)
			}
		}()
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
