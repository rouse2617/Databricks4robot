package routes

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	promhttp "github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	actionH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/action"
	adminH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/admin"
	algorunH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/algorun"
	assetH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/asset"
	auditH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/audit"
	customerH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/customer"
	deliveryH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/delivery"
	backfillH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/backfill"
	deliveryruleH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/deliveryrule"
	evalH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/eval"
	lakehouseH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/lakehouse"
	mcapH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/mcap"
	pipelineH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/pipeline"
	pipelineComponentH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/pipeline_component"
	queryH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/query"
	workflowH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/workflow"
	registryH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/registry"
	searchH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/search"
	"github.com/CyberOrigin2077/cyber-databrew/internal/middleware"

	_ "github.com/CyberOrigin2077/cyber-databrew/docs/swagger" // swagger docs
)

// RegisterAll wires up all API domains in a single process.
// This is the default local/prod runtime mode for the current project.
// pgPing is a function that pings the PostgreSQL database (e.g. pgClient.Ping).
// When nil, the readyz endpoint reports PG as unhealthy.
func RegisterAll(
	r *gin.Engine,
	cfg *config.Config,
	pgPing func(context.Context) error,
	assetHandler *assetH.Handler,
	mcapHandler *mcapH.Handler,
	deliveryHandler *deliveryH.Handler,
	customerHandler *customerH.Handler,
	deliveryRuleHandler *deliveryruleH.Handler,
	algoRunHandler *algorunH.Handler,
	algoHandler *assetH.AlgoHandler,
	auditHandler *auditH.Handler,
	lakehouseHandler *lakehouseH.Handler,
	registryHandler *registryH.Handler,
	searchHandler *searchH.Handler,
	adminHandler *adminH.Handler,
	purgeHandler *adminH.PurgeHandler,
	evalHandler *evalH.Handler,
	actionHandler *actionH.Handler,
	pipelineHandler *pipelineH.Handler,
	pipelineComponentHandler *pipelineComponentH.Handler,
	queryHandler *queryH.Handler,
	workflowHandler *workflowH.Handler,
	backfillHandler *backfillH.Handler,
) {
	r.Use(middleware.RequestID())
	r.Use(middleware.HTTPMetrics())
	r.Use(middleware.RequestGuard(2048))
	r.Use(middleware.StructuredLogger())
	r.Use(middleware.UserEmail())

	// Rate limiting (disabled by default, set RATE_LIMIT_RPS to enable).
	if rl := middleware.RateLimitFromConfig(cfg.RateLimitRPS, cfg.RateLimitBurst); rl != nil {
		r.Use(rl.Middleware())
	}

	// Circuit breaker scoped to API routes only — infrastructure endpoints
	// (/healthz, /readyz, /metrics) must remain reachable when CB is open.
	var cbMiddleware gin.HandlerFunc
	if cb := middleware.NewCircuitBreaker(
		cfg.CBEnabled == "true",
		atoi(cfg.CBWindowSec, 60),
		atoi(cfg.CBThreshold, 10),
		atoi(cfg.CBCooldownSec, 30),
	); cb != nil {
		cbMiddleware = cb.Middleware()
	}

	r.GET("/healthz", healthz("backend"))

	r.GET("/readyz", readyz(pgPing, cfg))
	r.GET("/version", version())
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	auth := middleware.StaticTokenAuth(cfg.DatabrewToken)
	adminAuth := middleware.AdminTokenAuth(cfg.AdminToken, cfg.DatabrewToken, cfg.Env)
	adminRoutesEnabled := cfg.AdminRoutesEnabled()
	secureSessionCookie := cfg.Env == "production"

	authPublic := r.Group("/api/v1/auth")
	{
		authPublic.POST("/login", func(c *gin.Context) {
			var req struct {
				Token string `json:"token"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
				return
			}
			token := strings.TrimSpace(req.Token)
			if token == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
				return
			}
			if token != cfg.DatabrewToken {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
				return
			}
			c.SetCookie("databrew_session", token, 86400, "/", "", secureSessionCookie, true)
			c.JSON(http.StatusOK, gin.H{"authenticated": true})
		})

		authProtected := authPublic.Group("", auth)
		authProtected.GET("/me", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"authenticated": true})
		})
		authProtected.POST("/logout", func(c *gin.Context) {
			c.SetCookie("databrew_session", "", -1, "/", "", secureSessionCookie, true)
			c.JSON(http.StatusOK, gin.H{"authenticated": false})
		})
	}

	api := r.Group("/api/v1", auth)
	if cbMiddleware != nil {
		api.Use(cbMiddleware)
	}
	{
		assets := api.Group("/assets")
		assets.POST("", assetHandler.Create)
		assets.GET("/:id", assetHandler.Get)
		assets.PATCH("/:id", assetHandler.Update)
		assets.DELETE("/:id", assetHandler.Delete)
		assets.GET("/:id/deliveries", assetHandler.ListDeliveries)
		assets.GET("/:id/mcap-locator", assetHandler.McapLocator)
		assets.GET("/:id/foxglove-source", assetHandler.FoxgloveSource)
		assets.GET("/:id/events", assetHandler.ListEvents)
		assets.GET("/:id/events/stream", assetHandler.HandleEventsStream)
		assets.GET("/:id/lineage", assetHandler.GetLineage)
		assets.GET("/:id/provenance", assetHandler.GetProvenance)
		assets.GET("/:id/timeline", assetHandler.Timeline)
		assets.POST("/:id/tags", assetHandler.UpsertTag)
		assets.DELETE("/:id/tags/:key", assetHandler.DeleteTag)
		assets.GET("/:id/tags/history", assetHandler.ListTagHistory)
		assets.POST("/:id/revisions", assetHandler.PromoteRevision)

		// Layered child-asset creation (CYB-1222, CYB-1228)
		assets.POST("/:id/clips", assetHandler.CreateClip)
		assets.POST("/:id/frames", assetHandler.CreateFrame)
		assets.POST("/:id/tasks", assetHandler.CreateTask)
		assets.POST("/:id/actions", assetHandler.CreateAction)

		// Logical asset endpoints
		api.GET("/logical-assets/:id/current", assetHandler.GetCurrentForLogical)
		api.GET("/logical-assets/:id/ratings-history", assetHandler.HandleRatingsHistory)

		// Usage stats (CYB-1095/1096)
		assets.POST("/:id/view", assetHandler.RecordView)
		assets.POST("/:id/favorite", assetHandler.ToggleFavorite)

		// Batch operations (custom method syntax: POST /assets:batch_get)
		api.POST("/assets:batch_get", assetHandler.BatchGet)
		api.GET("/asset-types/:type/schema", assetHandler.GetAssetTypeSchema)

		// Global event stream — no asset_id required.
		api.GET("/events", assetHandler.ListGlobalEvents)

		// Audit / discovery layer (CYB-1097/1098)
		if auditHandler != nil {
			api.GET("/audit/search", auditHandler.HandleAuditSearch)
			api.GET("/audit/lineage-search", auditHandler.HandleLineageSearch)
		}

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
		mcapFiles.POST("", mcapHandler.CreateFile)
		mcapFiles.POST("/:id/finalize", mcapHandler.FinalizeUpload)
		mcapFiles.GET("", mcapHandler.ListFiles)
		mcapFiles.GET("/:id", mcapHandler.GetFile)
		mcapFiles.GET("/:id/bytes", mcapHandler.Bytes)
		mcapFiles.HEAD("/:id/bytes", mcapHandler.Bytes)

		if customerHandler != nil {
			api.POST("/customers", customerHandler.Create)
			api.GET("/customers", customerHandler.List)
			api.GET("/customers/:customer_id", customerHandler.Get)
			api.PATCH("/customers/:customer_id", customerHandler.Update)
		}

		if deliveryRuleHandler != nil {
			api.POST("/delivery-rules", deliveryRuleHandler.Create)
			api.GET("/delivery-rules", deliveryRuleHandler.List)
		}

		if algoRunHandler != nil {
			api.POST("/algo-runs", algoRunHandler.Create)
			api.GET("/algo-runs", algoRunHandler.List)
			api.GET("/algo-runs/:run_id", algoRunHandler.Get)
			api.POST("/algo-runs/:run_id/start", algoRunHandler.Start)
			api.POST("/algo-runs/:run_id/finish", algoRunHandler.Finish)
			api.POST("/algo-runs/:run_id/cancel", algoRunHandler.Cancel)
			api.GET("/algo-runs/:run_id/affected-assets", algoRunHandler.GetAffectedAssets)
		}

		api.POST("/deliveries", deliveryHandler.Commit)
		api.GET("/deliveries", deliveryHandler.List)
		api.GET("/deliveries/:id", deliveryHandler.Get)
		api.GET("/deliveries/:id/items", deliveryHandler.ListItems)
		api.GET("/customers/:customer_id/deliveries", deliveryHandler.ListByCustomer)

		// C2 (two-step) delivery workflow: draft → add items → commit
		api.POST("/deliveries/draft", deliveryHandler.HandleDraft)
		api.POST("/deliveries/:id/items", deliveryHandler.HandleAddItems)
		api.POST("/deliveries/:id/commit", deliveryHandler.HandleCommitC2)

		// Delivery operations (CYB-1104~1106)
		api.POST("/deliveries/:id/cancel", deliveryHandler.HandleCancel)
		api.POST("/deliveries/:id/retry", deliveryHandler.HandleRetry)
		api.POST("/deliveries/:id/ack", deliveryHandler.HandleAck)

		// Registry endpoints (read-only, from YAML config)
		api.GET("/algo-registry", registryHandler.AlgoRegistry)
		api.GET("/tag-registry", registryHandler.TagRegistry)
		api.GET("/metric-registry", registryHandler.MetricRegistry)
		api.GET("/action-label-registry", registryHandler.ActionLabelRegistry)
		api.GET("/lifecycle-states", registryHandler.LifecycleStates)

		// Search endpoints (Elasticsearch-backed)
		if searchHandler != nil {
			api.GET("/search/sync-status", searchHandler.SyncStatus)
			api.GET("/search/sync-progress", searchHandler.SyncProgress)
		}

		if lakehouseHandler != nil {
			api.GET("/lakehouse/report", lakehouseHandler.Report)
			api.GET("/lakehouse/status", lakehouseHandler.Status)
			api.GET("/lakehouse/sync-status", lakehouseHandler.SyncStatus)
			api.GET("/lakehouse/sync-progress", lakehouseHandler.BronzeSyncProgress)
			api.GET("/lakehouse/failure-clusters", lakehouseHandler.FailureClusters)
			api.GET("/lakehouse/overview", lakehouseHandler.Overview)
			api.GET("/lakehouse/asset-growth", lakehouseHandler.AssetGrowth)
			api.GET("/lakehouse/tables", lakehouseHandler.Tables)
			api.GET("/lakehouse/event-daily", lakehouseHandler.EventDaily)
			api.GET("/lakehouse/event-type-share", lakehouseHandler.EventTypeShare)
			api.GET("/lakehouse/quality-distribution", lakehouseHandler.QualityDistribution)
			api.GET("/lakehouse/customer-replay", lakehouseHandler.CustomerReplay)
		}

		if adminRoutesEnabled && adminHandler != nil {
			admin := api.Group("/admin", adminAuth)
			admin.POST("/search/reindex", adminHandler.SearchReindex)
			admin.POST("/search/reindex-jobs", adminHandler.SearchReindexCreateJob)
			admin.GET("/search/reindex-jobs", adminHandler.SearchReindexListJobs)
			admin.GET("/search/reindex-jobs/:id", adminHandler.SearchReindexGetJob)
			admin.POST("/search/reindex-jobs/:id/stop", adminHandler.SearchReindexStopJob)
			admin.POST("/search/reindex-jobs/:id/resume", adminHandler.SearchReindexResumeJob)
			admin.POST("/search/reindex-jobs/:id/abandon", adminHandler.SearchReindexAbandonJob)
			admin.GET("/search/outbox-stats", adminHandler.SearchOutboxStats)
			admin.GET("/search/audit", adminHandler.SearchAudit)
		}

		// Internal admin (hard delete). Requires ADMIN_TOKEN in production.
		if adminRoutesEnabled && purgeHandler != nil {
			internal := api.Group("/internal", adminAuth)
			internal.DELETE("/assets/:id", purgeHandler.DeleteAssetHard)
			internal.POST("/assets:batch_delete", purgeHandler.BatchDeleteAssets)
		}

		// Action annotations (mcap → seg → action 第三层; docs/review/data-platform-design.md §5.2.15)
		// NOTE: moved from /:id/actions to /:id/action-annotations — the layered
		// asset creation API now owns /:id/actions (CYB-1228).
		if actionHandler != nil {
			assets.POST("/:id/action-annotations", actionHandler.Create)
			assets.GET("/:id/action-annotations", actionHandler.List)
			assets.PATCH("/:id/action-annotations/:action_id", actionHandler.Patch)
			assets.DELETE("/:id/action-annotations/:action_id", actionHandler.Delete)
		}

		// Eval / Metrics (Phase 1.5)
		if evalHandler != nil {
			assets.POST("/:id/eval-results", evalHandler.ReportEvalResult)
			assets.GET("/:id/eval-results", evalHandler.ListEvalResults)
			assets.GET("/:id/metrics", evalHandler.ListMetrics)
			api.GET("/metrics/registry", evalHandler.GetRegistry)
			api.POST("/metrics:search", evalHandler.SearchByMetrics)
		}

		// Pipeline (Argo Workflows) — templates, deploy, deployments
		if pipelineHandler != nil {
			api.POST("/pipelines", pipelineHandler.SaveTemplate)
			api.GET("/pipelines", pipelineHandler.ListTemplates)
			api.GET("/pipelines/:id", pipelineHandler.GetTemplate)
			api.DELETE("/pipelines/:id", pipelineHandler.DeleteTemplate)
			api.GET("/pipelines/:id/versions", pipelineHandler.ListVersions)
			api.GET("/pipelines/:id/diff/:id2", pipelineHandler.DiffTemplates)
			api.POST("/deploy", pipelineHandler.Deploy)
			api.POST("/deploy/template/:id", pipelineHandler.DeployByTemplate)
			api.GET("/deployments", pipelineHandler.ListDeployments)
			api.GET("/deployments/:id", pipelineHandler.GetDeployment)
			api.GET("/deployments/:id/resources", pipelineHandler.GetResourceUsage)
			api.POST("/deployments/:id/retry", pipelineHandler.RetryDeployment)
			api.POST("/deployments/:id/stop", pipelineHandler.StopDeployment)
			api.POST("/deployments/:id/save-template", pipelineHandler.SaveFromDeployment)
			api.DELETE("/deployments/:id", pipelineHandler.DeleteDeployment)
			api.POST("/pipeline-assets", pipelineHandler.RegisterOutput)
			api.GET("/assets/:id/pipeline-lineage", pipelineHandler.GetLineage)
		}

		// Pipeline component registry
		if pipelineComponentHandler != nil {
			api.POST("/components", pipelineComponentHandler.CreateComponent)
			api.GET("/components", pipelineComponentHandler.ListComponents)
			api.GET("/components/:id", pipelineComponentHandler.GetComponent)
			api.PUT("/components/:id", pipelineComponentHandler.UpdateComponent)
			api.DELETE("/components/:id", pipelineComponentHandler.DeleteComponent)
		}

		// Workflow monitoring
		if workflowHandler != nil {
			api.GET("/workflows", workflowHandler.ListWorkflows)
			api.GET("/workflows/:name/logs", workflowHandler.GetWorkflowLogs)
			api.GET("/workflows/:name", workflowHandler.GetWorkflow)
		}

		// Backfill jobs
		if backfillHandler != nil {
			api.POST("/backfill", backfillHandler.CreateJob)
			api.GET("/backfill", backfillHandler.ListJobs)
			api.GET("/backfill/:id", backfillHandler.GetJob)
			api.POST("/backfill/:id/pause", backfillHandler.PauseJob)
			api.POST("/backfill/:id/resume", backfillHandler.ResumeJob)
			api.POST("/backfill/:id/retry-failed", backfillHandler.RetryFailed)
		}

		if queryHandler != nil {
			api.POST("/queries/validate", queryHandler.Validate)
			api.POST("/queries/run", queryHandler.Run)
			api.GET("/saved-queries", queryHandler.ListSavedQueries)
			api.POST("/saved-queries", queryHandler.CreateSavedQuery)
			api.GET("/saved-queries/:id", queryHandler.GetSavedQuery)
			api.PATCH("/saved-queries/:id", queryHandler.UpdateSavedQuery)
			api.DELETE("/saved-queries/:id", queryHandler.DeleteSavedQuery)
		}
	}

	// Internal (service-to-service). Disabled in production when ADMIN_TOKEN is unset.
	if adminRoutesEnabled {
		r.POST("/internal/commit-segments", adminAuth, assetHandler.CommitSegments)
	}
}

func healthz(service string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": service})
	}
}

func readyz(pgPing func(context.Context) error, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		checks := gin.H{}
		healthy := true

		// PG readiness — reuse the existing connection pool.
		if pgPing != nil {
			if err := pgPing(ctx); err != nil {
				checks["pg"] = gin.H{"status": "unhealthy", "error": err.Error()}
				healthy = false
			} else {
				checks["pg"] = gin.H{"status": "healthy"}
			}
		} else {
			checks["pg"] = gin.H{"status": "unhealthy", "error": "not configured"}
			healthy = false
		}

		if cfg.LakehouseBackend != "" && cfg.LakehouseBackend != "none" {
			checks["lakehouse"] = gin.H{
				"status":  "configured",
				"backend": cfg.LakehouseBackend,
				"project": cfg.LakehouseBQProject,
				"dataset": cfg.LakehouseBQDataset,
			}
		}

		status := http.StatusOK
		if !healthy {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{"status": gin.H{"healthy": healthy}, "checks": checks})
	}
}

var (
	buildVersion = "dev"
	buildCommit  = "unknown"
	buildTime    = "unknown"
)

func version() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"version": buildVersion,
			"commit":  buildCommit,
			"time":    buildTime,
			"service": "cyber-databrew-backend",
		})
	}
}

func atoi(s string, fallback int) int {
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}
