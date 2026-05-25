package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"

	auditH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/audit"
	lakehouseH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/lakehouse"
	registryH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/registry"
	searchH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/search"
	"github.com/CyberOrigin2077/cyber-databrew/internal/postgres"
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
		core.query,
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
