package main

import (
	"context"
	"log/slog"
	"os"

	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/CyberOrigin2077/cyber-databrew/internal/audit"
	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	espkg "github.com/CyberOrigin2077/cyber-databrew/internal/elasticsearch"
	mcapH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/mcap"
	"github.com/CyberOrigin2077/cyber-databrew/internal/metrics"
	"github.com/CyberOrigin2077/cyber-databrew/internal/middleware"
	"github.com/CyberOrigin2077/cyber-databrew/internal/postgres"
	"github.com/CyberOrigin2077/cyber-databrew/internal/validate"
)

// ── Layer 1: Infrastructure ──

func setupInfra() *infra {
	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file, using environment variables")
	}

	cfg := config.Load()

	validate.RegisterCustomValidators()

	closeLog := middleware.SetupLogger(cfg.LogLevel, cfg.LogFormat, cfg.LogFile)
	_ = closeLog // kept alive for process lifetime

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	ctx := context.Background()
	for _, dep := range []string{"postgres", "lakehouse", "elasticsearch"} {
		metrics.BackendDependencyUp.WithLabelValues(dep).Set(0)
	}

	// Load registries.
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
	queryFieldReg, err := config.LoadQueryFieldRegistry("config/query_field_registry.yaml")
	if err != nil {
		slog.Warn("failed to load query field registry; query field capabilities will fall back to postgres-only defaults", "err", err)
		queryFieldReg = nil
	}
	actionLabelReg, err := config.LoadActionLabelRegistry("config/action_label_registry.yaml")
	if err != nil {
		slog.Warn("failed to load action label registry; action label validation disabled", "err", err)
		actionLabelReg = nil
	}

	// ── Storage backend (Postgres only) ──
	var pgClient *postgres.Client
	switch cfg.StorageBackend {
	case "bigtable":
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
		audit.Init(postgres.NewAuditSink(pgClient))
	default:
		slog.Error("invalid STORAGE_BACKEND", "value", cfg.StorageBackend, "allowed", "postgres")
		os.Exit(1)
	}

	// ── Optional: GCS bytes source for MCAP proxy ──
	var mcapBytesSource mcapH.BytesSource
	{
		gcsClient, err := storage.NewClient(ctx)
		if err != nil {
			slog.Warn("gcs client unavailable; mcap bytes proxy disabled", "err", err)
		} else {
			defer func() { _ = gcsClient.Close() }()
			src, srcErr := mcapH.NewGCSBytesSource(gcsClient)
			if srcErr != nil {
				slog.Warn("gcs bytes source init failed; mcap bytes proxy disabled", "err", srcErr)
			} else {
				mcapBytesSource = src
			}
		}
	}

	// ── Optional: Lakehouse query layer ──
	lakeClient := newLakehouseQuerier(ctx, cfg)
	if st := lakeClient.Status(ctx); st.Enabled && st.Healthy {
		metrics.BackendDependencyUp.WithLabelValues("lakehouse").Set(1)
		slog.Info("lakehouse query layer connected",
			"backend", st.Backend, "project", st.Project, "dataset", st.Dataset)
	} else {
		slog.Warn("lakehouse query layer disabled or unhealthy",
			"backend", st.Backend, "error", st.Error)
	}

	// ── Optional: Elasticsearch ──
	var esClient *espkg.Client
	if cfg.ElasticsearchURL != "" {
		esClient = espkg.New(cfg.ElasticsearchURL, "assets", cfg.ElasticsearchUsername, cfg.ElasticsearchPassword)
		if err := esClient.Ping(ctx); err != nil {
			slog.Warn("elasticsearch unavailable, search will return 503", "err", err)
			esClient = nil
		} else {
			metrics.BackendDependencyUp.WithLabelValues("elasticsearch").Set(1)
			slog.Info("elasticsearch connected", "url", cfg.ElasticsearchURL)
		}
	}

	return &infra{
		cfg:             cfg,
		pg:              pgClient,
		es:              esClient,
		lake:            lakeClient,
		mcapBytesSource: mcapBytesSource,
		algoRegistry:    algoRegistry,
		tagRegistry:     tagRegistry,
		metricRegistry:  metricRegistry,
		queryFieldReg:   queryFieldReg,
		actionLabelReg:  actionLabelReg,
	}
}
