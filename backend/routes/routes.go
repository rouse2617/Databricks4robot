package routes

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	promhttp "github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"data-platform/internal/config"
	adminH "data-platform/internal/handlers/admin"
	assetH "data-platform/internal/handlers/asset"
	deliveryH "data-platform/internal/handlers/delivery"
	lakehouseH "data-platform/internal/handlers/lakehouse"
	mcapH "data-platform/internal/handlers/mcap"
	registryH "data-platform/internal/handlers/registry"
	searchH "data-platform/internal/handlers/search"
	"data-platform/internal/middleware"
	"data-platform/internal/outbox"

	_ "data-platform/docs/swagger" // swagger docs
)

// RegisterAll wires up all API domains in a single process.
// This is the default local/prod runtime mode for the current project.
func RegisterAll(
	r *gin.Engine,
	cfg *config.Config,
	assetHandler *assetH.Handler,
	mcapHandler *mcapH.Handler,
	deliveryHandler *deliveryH.Handler,
	algoHandler *assetH.AlgoHandler,
	lakehouseHandler *lakehouseH.Handler,
	registryHandler *registryH.Handler,
	searchHandler *searchH.Handler,
	adminHandler *adminH.Handler,
	outboxHealth *outbox.HealthStatus,
) {
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestGuard(2048))
	r.Use(middleware.StructuredLogger())

	// Rate limiting (disabled by default, set RATE_LIMIT_RPS to enable).
	if rl := middleware.RateLimitFromConfig(cfg.RateLimitRPS, cfg.RateLimitBurst); rl != nil {
		r.Use(rl.Middleware())
	}

	// Circuit breaker (disabled by default, set CB_ENABLED=true to enable).
	if cb := middleware.NewCircuitBreaker(
		cfg.CBEnabled == "true",
		atoi(cfg.CBWindowSec, 60),
		atoi(cfg.CBThreshold, 10),
		atoi(cfg.CBCooldownSec, 30),
	); cb != nil {
		r.Use(cb.Middleware())
	}

	r.GET("/healthz", healthz("backend"))
	if outboxHealth != nil {
		r.GET("/healthz/outbox", gin.WrapF(outboxHealth.Handler()))
	}
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	auth := middleware.StaticTokenAuth(cfg.GraceToken)

	api := r.Group("/api/v1", auth)
	{
		assets := api.Group("/assets")
		assets.GET("", assetHandler.List)
		assets.POST("", assetHandler.Create)
		assets.GET("/:id", assetHandler.Get)
		assets.PATCH("/:id", assetHandler.Update)
		assets.DELETE("/:id", assetHandler.Delete)
		assets.GET("/:id/deliveries", assetHandler.ListDeliveries)
		assets.GET("/:id/events", assetHandler.ListEvents)
		assets.POST("/:id/tags", assetHandler.UpsertTag)
		assets.DELETE("/:id/tags/:key", assetHandler.DeleteTag)
		assets.GET("/:id/tags/history", assetHandler.ListTagHistory)

		// Batch operations (custom method syntax: POST /assets:batch_get)
		api.POST("/assets:batch_get", assetHandler.BatchGet)

		// Algorithm lifecycle routes
		if algoHandler != nil {
			assets.GET("/:id/algo", algoHandler.ListCurrent)
			assets.POST("/:id/algo/:algo_key/start", algoHandler.Start)
			assets.POST("/:id/algo/:algo_key/finish", algoHandler.Finish)
			assets.POST("/:id/algo/:algo_key/reset", algoHandler.Reset)
		}

		api.POST("/mcap/upload/finalize", mcapHandler.FinalizeUpload)
		api.GET("/mcap/:id/messages", mcapHandler.IterMessages)

		mcapFiles := api.Group("/mcap-files")
		mcapFiles.GET("", mcapHandler.ListFiles)
		mcapFiles.GET("/:id", mcapHandler.GetFile)

		api.POST("/deliveries", deliveryHandler.Commit)
		api.GET("/deliveries", deliveryHandler.List)
		api.GET("/deliveries/:id", deliveryHandler.Get)
		api.GET("/deliveries/:id/items", deliveryHandler.ListItems)
		api.GET("/customers/:customer_id/deliveries", deliveryHandler.ListByCustomer)

		// Registry endpoints (read-only, from YAML config)
		api.GET("/algo-registry", registryHandler.AlgoRegistry)
		api.GET("/tag-registry", registryHandler.TagRegistry)

		// Search endpoints (Elasticsearch-backed)
		if searchHandler != nil {
			api.GET("/search/assets", searchHandler.SearchAssets)
		}

		if lakehouseHandler != nil {
			api.GET("/lakehouse/report", lakehouseHandler.Report)
			api.GET("/lakehouse/status", lakehouseHandler.Status)
			api.GET("/lakehouse/tables", lakehouseHandler.Tables)
			api.GET("/lakehouse/sync-status", lakehouseHandler.SyncStatus)
			api.GET("/lakehouse/training-assets", lakehouseHandler.TrainingAssets)
			api.GET("/lakehouse/recompute-candidates", lakehouseHandler.RecomputeCandidates)
			api.GET("/lakehouse/tag-timeline", lakehouseHandler.TagTimeline)
			api.GET("/lakehouse/quality-distribution", lakehouseHandler.QualityDistribution)
			api.GET("/lakehouse/customer-replay", lakehouseHandler.CustomerReplay)
		}
	}

	// Internal (service-to-service, no external auth required in Phase 0)
	r.POST("/internal/commit-segments", assetHandler.CommitSegments)

	if adminHandler != nil && cfg.AdminToken != "" {
		adm := r.Group("/api/v1/admin", adminH.AdminTokenAuth(cfg.AdminToken))
		adm.POST("/search/reindex", adminHandler.SearchReindex)
	}
}

func healthz(service string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": service})
	}
}

func atoi(s string, fallback int) int {
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}
