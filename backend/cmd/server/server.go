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
	dashboardH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/dashboard"
	lakehouseH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/lakehouse"
	registryH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/registry"
	searchH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/search"
	"github.com/CyberOrigin2077/cyber-databrew/internal/postgres"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	dashboardUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/dashboard"
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

	// CYB-3425 Phase B PR 1: cluster registry. Reads exposed to any authed
	// user so the frontend can render cluster names; writes gated by admin.
	//
	// CYB-3486 PR 4a wired per-cluster k8s + argo factories on infra so admin
	// CRUD can invalidate the per-cluster cache. PR 4b flipped elastic_quota
	// / resource_quota consumers over — the factories now do real routing.
	var clusterHandler *adminH.ClusterHandler
	if inf.clusterRepo != nil {
		clusterHandler = adminH.NewClusterHandler(inf.clusterRepo, inf.k8sFactory, inf.argoFactory)
	}

	// CYB-4303: dashboard aggregations. Handler is nil-safe (nil pg → no
	// registration), matching the lakehouse pattern above.
	var dashboardHandler *dashboardH.Handler
	if inf.pg != nil && core.assetRepo != nil {
		dashboardHandler = dashboardH.New(dashboardUC.New(core.assetRepo))
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
		dashboardHandler,
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
		clusterHandler,
		core.subscriptionTask,
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
