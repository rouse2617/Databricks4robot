// @title           Data Platform API
// @version         1.0
// @description     Backend API for the Data Platform (asset, algo, delivery, mcap management).
// @host            localhost:8080
// @BasePath        /api/v1
// @securityDefinitions.apikey GraceToken
// @in header
// @name X-Grace-Token

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"data-platform/internal/audit"
	"data-platform/internal/cdc"
	"data-platform/internal/config"
	espkg "data-platform/internal/elasticsearch"
	actionH "data-platform/internal/handlers/action"
	adminH "data-platform/internal/handlers/admin"
	assetH "data-platform/internal/handlers/asset"
	deliveryH "data-platform/internal/handlers/delivery"
	evalH "data-platform/internal/handlers/eval"
	lakehouseH "data-platform/internal/handlers/lakehouse"
	mcapH "data-platform/internal/handlers/mcap"
	queryH "data-platform/internal/handlers/query"
	registryH "data-platform/internal/handlers/registry"
	searchH "data-platform/internal/handlers/search"
	"data-platform/internal/metrics"
	"data-platform/internal/middleware"
	"data-platform/internal/postgres"
	"data-platform/internal/repository"
	"data-platform/internal/searchindex"
	trinopkg "data-platform/internal/trino"
	actionUC "data-platform/internal/usecase/action"
	assetUC "data-platform/internal/usecase/asset"
	"data-platform/internal/validate"
	"data-platform/routes"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file, using environment variables")
	}

	cfg := config.Load()

	// Register custom validation rules before any handler uses Gin binding.
	validate.RegisterCustomValidators()

	// Setup structured logging (must be before any slog calls).
	closeLog := middleware.SetupLogger(cfg.LogLevel, cfg.LogFormat, cfg.LogFile)
	defer closeLog()

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	ctx := context.Background()
	for _, dep := range []string{"postgres", "trino", "elasticsearch"} {
		metrics.BackendDependencyUp.WithLabelValues(dep).Set(0)
	}

	// Load registries (required for all backends).
	algoRegistry, err := config.LoadAlgoRegistry("config/algo_registry.yaml")
	if err != nil {
		slog.Error("failed to load algo registry", "err", err)
		os.Exit(1)
	}
	tagRegistry, err := config.LoadTagRegistry("config/tag_registry.yaml")
	if err != nil {
		slog.Error("failed to load tag registry", "err", err)
		os.Exit(1)
	}
	metricRegistry, err := config.LoadMetricRegistry("config/metric_registry.yaml")
	if err != nil {
		slog.Error("failed to load metric registry", "err", err)
		os.Exit(1)
	}
	queryFieldRegistry, err := config.LoadQueryFieldRegistry("config/query_field_registry.yaml")
	if err != nil {
		slog.Warn("failed to load query field registry; query field capabilities will fall back to postgres-only defaults", "err", err)
		queryFieldRegistry = nil
	}
	actionLabelRegistry, err := config.LoadActionLabelRegistry("config/action_label_registry.yaml")
	if err != nil {
		// Keep startup tolerant: action labels are used for input validation only.
		// If the file is missing in a minimal deployment, we can still run.
		slog.Warn("failed to load action label registry; action label validation disabled", "err", err)
		actionLabelRegistry = nil
	}

	var (
		assetHandler    *assetH.Handler
		algoHandler     *assetH.AlgoHandler
		mcapHandler     *mcapH.Handler
		deliveryHandler *deliveryH.Handler
		evalHandler     *evalH.Handler
		actionHandler   *actionH.Handler
		queryHandler    *queryH.Handler
		assetUsecase    *assetUC.Usecase
		pgClient        *postgres.Client
		assetRepo       repository.AssetRepository
		assetTagRepo    repository.AssetTagRepository
		algoLatestRepo  repository.AssetAlgoLatestRepository
		assetEventRepo  repository.AssetEventRepository
		mcapRepo        repository.McapFileRepository
		actionRepo      repository.ActionRepository
		savedQueryRepo  *postgres.SavedQueryRepo
	)

	switch cfg.StorageBackend {
	case "bigtable":
		// Bigtable backend is DEPRECATED and no longer supported as a runtime
		// target. The package and tests in `internal/bigtable` are retained
		// only as historical reference and may be removed in a future cut.
		// Use `STORAGE_BACKEND=postgres` (the default).
		slog.Error("STORAGE_BACKEND=bigtable is deprecated and no longer supported; set STORAGE_BACKEND=postgres")
		os.Exit(1)

	case "postgres":
		var pgErr error
		pgClient, pgErr = postgres.New(ctx, cfg)
		if pgErr != nil {
			slog.Error("postgres connect failed", "err", pgErr)
			os.Exit(1)
		}
		metrics.BackendDependencyUp.WithLabelValues("postgres").Set(1)
		defer pgClient.Close()

		audit.Init(postgres.NewAuditSink(pgClient))

		assetRepo = postgres.NewAssetRepo(pgClient)
		assetTagRepo = postgres.NewAssetTagRepo(pgClient)
		algoLatestRepo = postgres.NewAssetAlgoLatestRepo(pgClient)
		assetEventRepo = postgres.NewAssetEventRepo(pgClient)
		mcapRepo = postgres.NewMcapFileRepo(pgClient)
		deliveryRepo := postgres.NewDeliveryRepo(pgClient)
		algoUC := assetUC.NewAlgoUsecase(pgClient, assetRepo, algoLatestRepo, assetEventRepo, algoRegistry)
		algoHandler = assetH.NewAlgoHandler(algoUC)
		assetUsecase = assetUC.NewWithProjections(pgClient, assetRepo, assetTagRepo, algoLatestRepo, assetEventRepo, tagRegistry, algoRegistry)
		assetHandler = assetH.New(assetUsecase, deliveryRepo)
		mcapHandler = mcapH.New(mcapRepo)
		deliveryHandler = deliveryH.New(deliveryRepo, postgres.NewIdempotencyRepo(pgClient))
		evalRepo := postgres.NewEvalRepo(pgClient)
		evalHandler = evalH.New(evalRepo, metricRegistry, assetEventRepo)
		actionRepo = postgres.NewActionRepo(pgClient)
		actionHandler = actionH.New(actionUC.NewWithLabelRegistry(pgClient, actionRepo, assetRepo, assetEventRepo, actionLabelRegistry))
		savedQueryRepo = postgres.NewSavedQueryRepo(pgClient)

	default:
		slog.Error("invalid STORAGE_BACKEND", "value", cfg.StorageBackend, "allowed", "postgres")
		os.Exit(1)
	}

	trinoClient, err := trinopkg.New(ctx, cfg)
	if err != nil {
		slog.Warn("trino query layer unavailable", "err", err)
	} else if trinoClient != nil {
		metrics.BackendDependencyUp.WithLabelValues("trino").Set(1)
		defer trinoClient.Close()
		slog.Info("trino query layer connected", "catalog", cfg.TrinoCatalog, "schema", cfg.TrinoSchema)
	}

	// Elasticsearch client (optional — search degrades gracefully if unavailable).
	var esClient *espkg.Client
	if cfg.ElasticsearchURL != "" {
		esClient = espkg.New(cfg.ElasticsearchURL, "assets")
		if err := esClient.Ping(ctx); err != nil {
			slog.Warn("elasticsearch unavailable, search will return 503", "err", err)
			esClient = nil
		} else {
			metrics.BackendDependencyUp.WithLabelValues("elasticsearch").Set(1)
			slog.Info("elasticsearch connected", "url", cfg.ElasticsearchURL)
		}
	}

	var adminHandler *adminH.Handler
	if pgClient != nil && esClient != nil {
		adminHandler = adminH.New(
			assetRepo,
			assetTagRepo,
			algoLatestRepo,
			mcapRepo,
			esClient,
		)
	}
	if assetUsecase != nil {
		queryHandler = queryH.New(assetUsecase, queryFieldRegistry, esClient, savedQueryRepo)
	}

	cdcCtx, cdcCancel := context.WithCancel(context.Background())
	defer cdcCancel()
	cdcRuntimeStarted := false
	if pgClient != nil && cfg.CDCEnabled == "true" {
		runtimeCfg := cdc.BuildRuntimeConfig(cfg)
		handlers := map[string]cdc.BatchHandler{}
		if esClient != nil {
			esBatchSize, _ := strconv.Atoi(cfg.CDCESBatchSize)
			esRateLimitPerSec, _ := strconv.ParseFloat(cfg.CDCESRateLimitPerSec, 64)
			esConsumer := &cdc.ESConsumer{
				Assets: assetRepo,
				Builder: &searchindex.Builder{
					Assets:  assetRepo,
					Tags:    assetTagRepo,
					Algos:   algoLatestRepo,
					Mcap:    mcapRepo,
					Actions: actionRepo,
				},
				ES:                 esClient,
				BatchSize:          esBatchSize,
				RateLimitPerSecond: esRateLimitPerSec,
			}
			for _, topic := range runtimeCfg.TopicsFor(cdc.ConsumerKindSearchProjection) {
				handlers[topic] = esConsumer
			}
		}
		bronzeSink := cdc.NewBronzeSink(cdc.BronzeSinkConfig{
			Enabled:    true,
			StagingDir: cfg.CDCBronzeStagingDir,
		})
		bronzeConsumer := &cdc.BronzeConsumer{Sink: bronzeSink}
		for _, topic := range runtimeCfg.TopicsFor(cdc.ConsumerKindBronzeEvents) {
			handlers[topic] = bronzeConsumer
		}
		cdcRuntime := &cdc.Runtime{
			Config:   runtimeCfg,
			Handlers: handlers,
			Source:   nil,
		}
		if err := cdc.ValidateRuntimeHealth(cfg, runtimeCfg); err != nil {
			slog.Error("cdc startup validation failed", "err", err)
			os.Exit(1)
		}
		switch cfg.CDCSourceDriver {
		case "":
			// Leave Source nil so Validate reports a configuration error below.
		case "debezium-kafka":
			cdcRuntime.Source = cdc.NewKafkaSourceFromConfig(cfg, runtimeCfg)
		case "in-memory":
			cdcRuntime.Source = &cdc.InMemorySource{}
		default:
			slog.Error("unknown CDC_SOURCE_DRIVER", "driver", cfg.CDCSourceDriver)
		}
		if err := cdcRuntime.Validate(); err != nil {
			slog.Error("cdc runtime failed validation", "err", err, "source_driver", cfg.CDCSourceDriver)
			os.Exit(1)
		}
		cdcRuntimeStarted = true
		slog.Info("cdc runtime started", "source_driver", cfg.CDCSourceDriver)
		go func() {
			if err := cdcRuntime.Run(cdcCtx); err != nil && !errors.Is(err, context.Canceled) {
				slog.Error("cdc runtime exited", "err", err)
			}
		}()
	}

	// Legacy dev safety net: when CDC is disabled locally, periodically
	// rebuild ES from PostgreSQL current-state tables. Once CDC is enabled,
	// we keep ES writes on the single CDC path only.
	if pgClient != nil && esClient != nil && cfg.Env != "production" && !cdcRuntimeStarted {
		go startLocalSearchReconciler(context.Background(), assetRepo, assetTagRepo, algoLatestRepo, mcapRepo, actionRepo, esClient)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	searchSyncFn := func() searchH.SyncInfo {
		return buildSearchSyncInfo(cfg, esClient, cdcRuntimeStarted)
	}
	routes.RegisterAll(
		r,
		cfg,
		assetHandler,
		mcapHandler,
		deliveryHandler,
		algoHandler,
		lakehouseH.New(cfg.LakehouseReportPath, trinoClient, pgClient),
		registryH.New(algoRegistry, tagRegistry, metricRegistry, actionLabelRegistry),
		searchH.New(esClient, searchSyncFn),
		adminHandler,
		evalHandler,
		actionHandler,
		queryHandler,
	)

	// Start config watcher for hot-reload of registries.
	configWatcher, err := config.NewConfigWatcher("config", tagRegistry, algoRegistry, actionLabelRegistry)
	if err != nil {
		slog.Warn("config watcher failed to start, hot-reload disabled", "err", err)
	} else {
		defer configWatcher.Stop()
		slog.Info("config watcher started for hot-reload")
	}

	addr := ":" + cfg.Port
	slog.Info("backend server starting", "addr", addr, "storage_backend", cfg.StorageBackend)
	if err := r.Run(addr); err != nil {
		slog.Error("server exited", "err", fmt.Errorf("listen %s: %w", addr, err))
		os.Exit(1)
	}
}

func buildSearchSyncInfo(cfg *config.Config, esClient *espkg.Client, cdcRuntimeStarted bool) searchH.SyncInfo {
	info := searchH.SyncInfo{
		ElasticsearchOK: esClient != nil,
		CDCEnabled:      cfg.CDCEnabled == "true",
		CDCSourceDriver: cfg.CDCSourceDriver,
		Env:             cfg.Env,
	}
	if !info.ElasticsearchOK {
		info.SearchIndexMode = "unavailable"
		return info
	}
	if cdcRuntimeStarted {
		info.SearchIndexMode = "cdc"
		return info
	}
	if cfg.Env != "production" {
		info.SearchIndexMode = "local_reconcile"
		return info
	}
	info.SearchIndexMode = "manual"
	return info
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
			assets, total, err := assetRepo.ListWithFilters(ctx, "", nil, page, pageSize, "asset_id ASC")
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

	// Run once soon after startup, then continue periodically.
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
