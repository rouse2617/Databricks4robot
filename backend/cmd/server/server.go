package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"

	adminH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/admin"
	apikeyH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/apikey"
	auditH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/audit"
	lakehouseH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/lakehouse"
	registryH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/registry"
	searchH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/search"
	"github.com/CyberOrigin2077/cyber-databrew/internal/postgres"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	"github.com/CyberOrigin2077/cyber-databrew/routes"
)

// ── Layer 4: HTTP server ──

func runServer(inf *infra, core *coreHandlers, opt *optional) {
	cfg := inf.cfg

	r := gin.New()
	r.Use(gin.Recovery())

	var pgPingFn func(context.Context) error
	if inf.pg != nil {
		pgPingFn = inf.pg.Ping
	}

	// Unified auth: API keys for SDK/API callers (postgres-backed).
	var apiKeyRepo repository.APIKeyRepository
	var apiKeyHandler *apikeyH.Handler
	if inf.pg != nil {
		akr := postgres.NewAPIKeyRepo(inf.pg)
		apiKeyRepo = akr                 // pragma: allowlist secret
		apiKeyHandler = apikeyH.New(akr) // pragma: allowlist secret
	}

	// CYB-3246 Phase 2: managed tag registry (DB-backed overlay). Load DB
	// definitions into the in-memory managed overlay on top of the YAML
	// baseline; the validation hot path stays DB-free. On error (e.g. table not
	// yet migrated) the YAML baseline still validates — no seeding, so startup
	// never races the migration job and admin writes never erase the baseline.
	var tagRegistryHandler *adminH.TagRegistryHandler
	if inf.pg != nil {
		tagRegistryRepo := postgres.NewTagRegistryRepo(inf.pg)
		if err := adminH.LoadManagedTags(context.Background(), tagRegistryRepo, inf.tagRegistry); err != nil {
			slog.Error("tag registry load failed; using YAML baseline only", "err", err)
		}
		tagRegistryHandler = adminH.NewTagRegistryHandler(tagRegistryRepo, inf.tagRegistry)
	}

	routes.RegisterAll(
		r,
		cfg,
		pgPingFn,
		core.asset,
		core.mcap,
		core.delivery,
		core.customer,
		core.deliveryRule,
		core.algoRun,
		core.algo,
		auditH.New(inf.pg),
		lakehouseH.New(cfg.LakehouseReportPath, inf.lake, inf.pg).
			WithBronzeCheckpoint(postgres.NewLakehouseBronzeCheckpointRepo(inf.pg)),
		registryH.New(inf.algoRegistry, inf.tagRegistry, inf.metricRegistry, inf.actionLabelReg),
		searchH.New(inf.es, opt.searchSyncFn, opt.searchProgressFn),
		opt.admin,
		opt.purge,
		core.eval,
		core.action,
		core.pipeline,
		core.pipelineConfig,
		core.pipelineComponent,
		core.query,
		core.workflow,
		core.backfill,
		core.storage,
		apiKeyRepo,
		apiKeyHandler,
		tagRegistryHandler,
	)

	// Config watcher is created and managed by setupOptional (optional.go).
	// No duplicate watcher needed here.

	addr := ":" + cfg.Port
	slog.Info("backend server starting", "addr", addr, "storage_backend", cfg.StorageBackend)
	if err := r.Run(addr); err != nil {
		slog.Error("server exited", "err", fmt.Errorf("listen %s: %w", addr, err))
		os.Exit(1)
	}
}
