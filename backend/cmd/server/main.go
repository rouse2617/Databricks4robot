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
	"fmt"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"data-platform/internal/audit"
	"data-platform/internal/config"
	espkg "data-platform/internal/elasticsearch"
	assetH "data-platform/internal/handlers/asset"
	deliveryH "data-platform/internal/handlers/delivery"
	lakehouseH "data-platform/internal/handlers/lakehouse"
	mcapH "data-platform/internal/handlers/mcap"
	registryH "data-platform/internal/handlers/registry"
	searchH "data-platform/internal/handlers/search"
	"data-platform/internal/middleware"
	"data-platform/internal/postgres"
	"data-platform/internal/repository"
	trinopkg "data-platform/internal/trino"
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

	var (
		assetHandler    *assetH.Handler
		algoHandler     *assetH.AlgoHandler
		mcapHandler     *mcapH.Handler
		deliveryHandler *deliveryH.Handler
		pgClient        *postgres.Client
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
		defer pgClient.Close()

		audit.Init(postgres.NewAuditSink(pgClient))

		assetRepo := postgres.NewAssetRepo(pgClient)
		algoEventRepo := postgres.NewAlgoEventRepo(pgClient)
		deliveryRepo := postgres.NewDeliveryRepo(pgClient)
		algoUC := assetUC.NewAlgoUsecase(assetRepo, algoEventRepo, algoRegistry)
		algoHandler = assetH.NewAlgoHandler(algoUC)
		assetHandler = assetH.New(newAssetUsecase(assetRepo, tagRegistry, algoRegistry), deliveryRepo)
		mcapHandler = mcapH.New(postgres.NewMcapFileRepo(pgClient))
		deliveryHandler = deliveryH.New(deliveryRepo, postgres.NewIdempotencyRepo(pgClient))

	default:
		slog.Error("invalid STORAGE_BACKEND", "value", cfg.StorageBackend, "allowed", "postgres")
		os.Exit(1)
	}

	trinoClient, err := trinopkg.New(ctx, cfg)
	if err != nil {
		slog.Warn("trino query layer unavailable", "err", err)
	} else if trinoClient != nil {
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
			slog.Info("elasticsearch connected", "url", cfg.ElasticsearchURL)
		}
	}

	r := gin.New()
	r.Use(gin.Recovery())
	routes.RegisterAll(
		r,
		cfg,
		assetHandler,
		mcapHandler,
		deliveryHandler,
		algoHandler,
		lakehouseH.New(cfg.LakehouseReportPath, trinoClient, pgClient),
		registryH.New(algoRegistry, tagRegistry),
		searchH.New(esClient),
	)

	// Start config watcher for hot-reload of registries.
	configWatcher, err := config.NewConfigWatcher("config", tagRegistry, algoRegistry)
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

func newAssetUsecase(
	repo repository.AssetRepository,
	tagRegistry *config.TagRegistry,
	algoRegistry *config.AlgoRegistry,
) *assetUC.Usecase {
	return assetUC.NewFull(repo, tagRegistry, algoRegistry)
}
