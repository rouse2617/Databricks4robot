package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	btpkg "data-platform/internal/bigtable"
	"data-platform/internal/config"
	assetH "data-platform/internal/handlers/asset"
	deliveryH "data-platform/internal/handlers/delivery"
	mcapH "data-platform/internal/handlers/mcap"
	"data-platform/internal/postgres"
	assetUC "data-platform/internal/usecase/asset"
	"data-platform/routes"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file, using environment variables")
	}

	cfg := config.Load()
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	ctx := context.Background()

	var (
		assetHandler    *assetH.Handler
		mcapHandler     *mcapH.Handler
		deliveryHandler *deliveryH.Handler
	)

	switch cfg.StorageBackend {
	case "bigtable":
		btClient, err := btpkg.New(ctx, cfg.BigtableProject, cfg.BigtableInstance)
		if err != nil {
			slog.Error("bigtable connect failed", "err", err)
			os.Exit(1)
		}
		defer btClient.Close()

		assetRepo := btpkg.NewAssetRepo(btClient)
		assetHandler = assetH.New(assetUC.New(assetRepo))
		mcapHandler = mcapH.New(btpkg.NewMcapFileRepo(btClient))
		deliveryHandler = deliveryH.New(btpkg.NewDeliveryRepo(btClient), btpkg.NewIdempotencyRepo(btClient))

	case "postgres":
		pgClient, err := postgres.New(ctx, cfg)
		if err != nil {
			slog.Error("postgres connect failed", "err", err)
			os.Exit(1)
		}
		defer pgClient.Close()

		assetRepo := postgres.NewAssetRepo(pgClient)
		assetHandler = assetH.New(assetUC.New(assetRepo))
		mcapHandler = mcapH.New(postgres.NewMcapFileRepo(pgClient))
		deliveryHandler = deliveryH.New(postgres.NewDeliveryRepo(pgClient), postgres.NewIdempotencyRepo(pgClient))

	default:
		slog.Error("invalid STORAGE_BACKEND", "value", cfg.StorageBackend, "allowed", "bigtable|postgres")
		os.Exit(1)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	routes.RegisterAll(r, cfg, assetHandler, mcapHandler, deliveryHandler)

	addr := ":" + cfg.Port
	slog.Info("backend server starting", "addr", addr, "storage_backend", cfg.StorageBackend)
	if err := r.Run(addr); err != nil {
		slog.Error("server exited", "err", fmt.Errorf("listen %s: %w", addr, err))
		os.Exit(1)
	}
}

