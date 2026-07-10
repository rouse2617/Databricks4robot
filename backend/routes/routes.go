// Package routes wires HTTP handlers and middleware for the DataBrew API server.
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

	"github.com/CyberOrigin2077/cyber-databrew/internal/auth"
	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	actionH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/action"
	adminH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/admin"
	algoRunH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/algorun"
	apikeyH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/apikey"
	assetH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/asset"
	auditH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/audit"
	backfillH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/backfill"
	customerH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/customer"
	deliveryH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/delivery"
	deliveryRuleH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/deliveryrule"
	evalH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/eval"
	lakehouseH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/lakehouse"
	mcapH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/mcap"
	pipelineH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/pipeline"
	pipelineComponentH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/pipeline_component"
	pipelineConfigH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/pipeline_config"
	queryH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/query"
	registryH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/registry"
	searchH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/search"
	storageH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/storage"
	workflowH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/workflow"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/middleware"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"

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
	deliveryRuleHandler *deliveryRuleH.Handler,
	algoRunHandler *algoRunH.Handler,
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
	pipelineConfigHandler *pipelineConfigH.Handler,
	pipelineComponentHandler *pipelineComponentH.Handler,
	queryHandler *queryH.Handler,
	workflowHandler *workflowH.Handler,
	backfillHandler *backfillH.Handler,
	storageHandler *storageH.Handler,
	apiKeyRepo repository.APIKeyRepository,
	apiKeyHandler *apikeyH.Handler,
	tagRegistryHandler *adminH.TagRegistryHandler,
) {
	// Suppress unused warnings for handler params that don't have route
	// registrations wired yet (routes are registered in follow-up PRs).
	_, _, _, _ = algoRunHandler, pipelineHandler, pipelineConfigHandler, pipelineComponentHandler
	_ = workflowHandler
	_ = storageHandler

	r.Use(middleware.RequestID())
	r.Use(middleware.HTTPMetrics())
	r.Use(middleware.RequestGuard(2048))
	r.Use(middleware.StructuredLogger())

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

	adminAuth := middleware.AdminTokenAuth(cfg.AdminToken, cfg.DatabrewToken, cfg.Env)
	adminRoutesEnabled := cfg.AdminRoutesEnabled()
	secureSessionCookie := cfg.Env == "production"

	// ── Auth routes (public) ──
	authPublic := r.Group("/api/v1/auth")
	{
		authPublic.POST("/email-login", func(c *gin.Context) {
			var req struct {
				Email string `json:"email"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "email is required", map[string]any{"error": err.Error()})
				return
			}
			email := strings.TrimSpace(strings.ToLower(req.Email))
			if email == "" || !strings.Contains(email, "@") {
				httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "valid email is required", nil)
				return
			}

			domain := email[strings.LastIndex(email, "@")+1:]
			if cfg.AllowedDomain == "" || domain != cfg.AllowedDomain {
				httpresp.Unauthorized(c, httpresp.CodeUnauthorized, "email domain not allowed")
				return
			}

			role := "user"
			if cfg.IsAdminEmail(email) {
				role = "admin"
			}
			jwtToken, err := auth.SignToken(cfg.JWTSecret, email, role, 24*time.Hour)
			if err != nil {
				httpresp.Internal(c, "failed to sign token")
				return
			}
			c.SetSameSite(http.SameSiteLaxMode)
			c.SetCookie("databrew_session", jwtToken, 86400, "/", "", secureSessionCookie, true)
			c.JSON(http.StatusOK, gin.H{
				"authenticated": true,
				"token":         jwtToken,
				"email":         email,
				"role":          role,
			})
		})

		authProtected := authPublic.Group("", middleware.JWTAuth(cfg.DatabrewToken, cfg.JWTSecret))
		authProtected.GET("/me", func(c *gin.Context) {
			email, _ := c.Get(middleware.CtxKeyEmail)
			role, _ := c.Get(middleware.CtxKeyRole)
			c.JSON(http.StatusOK, gin.H{
				"authenticated": true,
				"email":         email,
				"role":          role,
			})
		})
		authProtected.POST("/logout", func(c *gin.Context) {
			c.SetSameSite(http.SameSiteLaxMode)
			c.SetCookie("databrew_session", "", -1, "/", "", secureSessionCookie, true)
			c.JSON(http.StatusOK, gin.H{"authenticated": false})
		})
	}

	// Legacy static-token login for SDK backward compat.
	authPublic.POST("/login", func(c *gin.Context) {
		var req struct {
			Token string `json:"token"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "token is required", map[string]any{"error": err.Error()})
			return
		}
		token := strings.TrimSpace(req.Token)
		if token == "" {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "token is required", nil)
			return
		}
		if token != cfg.DatabrewToken {
			httpresp.Unauthorized(c, httpresp.CodeUnauthorized, "invalid token")
			return
		}
		jwtToken, err := auth.SignToken(cfg.JWTSecret, "legacy", "admin", 24*time.Hour)
		if err != nil {
			httpresp.Internal(c, "failed to sign token")
			return
		}
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("databrew_session", jwtToken, 86400, "/", "", secureSessionCookie, true)
		c.JSON(http.StatusOK, gin.H{"authenticated": true})
	})

	if pipelineComponentHandler != nil {
		releaseIngest := r.Group("/api/v1", pipelineComponentH.ReleaseIngestAuth(cfg.ComponentReleaseIngestToken, cfg.DatabrewToken, cfg.JWTSecret))
		if cbMiddleware != nil {
			releaseIngest.Use(cbMiddleware)
		}
		releaseIngest.POST("/pipeline-component-releases/sync", pipelineComponentHandler.SyncReleases)
	}

	// Argo workflow run status push webhook (CYB-3058). Dedicated machine-token
	// auth (not user/JWT): the caller is the Argo controller exit hook. Empty
	// token disables the route (poll-only fallback).
	if pipelineHandler != nil && cfg.ArgoRunWebhookToken != "" {
		argoWebhook := r.Group("/api/v1", middleware.ArgoWebhookAuth(cfg.ArgoRunWebhookToken))
		if cbMiddleware != nil {
			argoWebhook.Use(cbMiddleware)
		}
		argoWebhook.POST("/pipeline-runs/webhook", pipelineHandler.HandleRunWebhook)
	}

	if workflowHandler != nil {
		terminalAttach := r.Group("/api/v1")
		if cbMiddleware != nil {
			terminalAttach.Use(cbMiddleware)
		}
		terminalAttach.GET("/pod-terminal/sessions/:id/attach", workflowHandler.AttachTerminalSession)
	}

	api := r.Group("/api/v1", middleware.Authenticate(cfg.DatabrewToken, cfg.JWTSecret, apiKeyRepo))
	if cbMiddleware != nil {
		api.Use(cbMiddleware)
	}
	{
		assets := api.Group("/assets")
		assets.POST("", middleware.RequireScope("assets:write"), assetHandler.Create)
		assets.GET("/:id", assetHandler.Get)
		assets.GET("/:id/metadata", assetHandler.GetMetadata)
		assets.PATCH("/:id", middleware.RequireScope("assets:write"), assetHandler.Update)
		assets.DELETE("/:id", middleware.RequireScope("assets:write"), assetHandler.Delete)
		assets.GET("/:id/deliveries", assetHandler.ListDeliveries)
		assets.GET("/:id/mcap-locator", assetHandler.McapLocator)
		assets.GET("/:id/foxglove-source", assetHandler.FoxgloveSource)
		assets.GET("/:id/events", assetHandler.ListEvents)
		assets.GET("/:id/events/stream", assetHandler.HandleEventsStream)
		assets.GET("/:id/lineage", assetHandler.GetLineage)
		assets.GET("/:id/timeline", assetHandler.Timeline)
		assets.POST("/:id/tags", middleware.RequireScope("assets:write"), assetHandler.UpsertTag)
		assets.DELETE("/:id/tags/:key", middleware.RequireScope("assets:write"), assetHandler.DeleteTag)
		assets.GET("/:id/tags/history", assetHandler.ListTagHistory)

		// Batch operations (custom method syntax: POST /assets:batch_get)
		api.POST("/assets:batch_get", assetHandler.BatchGet)

		// Global event stream — no asset_id required.
		api.GET("/events", assetHandler.ListGlobalEvents)
		api.GET("/events/stream", assetHandler.HandleGlobalEventsStream)

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
		mcapFiles.GET("", mcapHandler.ListFiles)
		mcapFiles.GET("/:id", mcapHandler.GetFile)
		mcapFiles.GET("/:id/bytes", mcapHandler.Bytes)
		mcapFiles.HEAD("/:id/bytes", mcapHandler.Bytes)

		api.POST("/deliveries", deliveryHandler.Commit)
		api.GET("/deliveries", deliveryHandler.List)
		api.GET("/deliveries/:id", deliveryHandler.Get)
		api.GET("/deliveries/:id/items", deliveryHandler.ListItems)
		api.GET("/customers/:customer_id/deliveries", deliveryHandler.ListByCustomer)
		api.POST("/deliveries/draft", deliveryHandler.HandleDraft)
		api.POST("/deliveries/:id/items", deliveryHandler.HandleAddItems)
		api.POST("/deliveries/:id/commit", deliveryHandler.HandleCommitC2)
		api.POST("/deliveries/:id/cancel", deliveryHandler.HandleCancel)
		api.POST("/deliveries/:id/retry", deliveryHandler.HandleRetry)
		api.POST("/deliveries/:id/ack", deliveryHandler.HandleAck)

		if deliveryRuleHandler != nil {
			api.POST("/delivery-rules", deliveryRuleHandler.Create)
			api.GET("/delivery-rules", deliveryRuleHandler.List)
		}

		// Customer CRUD
		if customerHandler != nil {
			api.POST("/customers", customerHandler.Create)
			api.GET("/customers", customerHandler.List)
			api.GET("/customers/:customer_id", customerHandler.Get)
			api.PATCH("/customers/:customer_id", customerHandler.Update)
		}

		// Registry endpoints (read-only, from YAML config)
		api.GET("/algo-registry", registryHandler.AlgoRegistry)
		api.GET("/tag-registry", registryHandler.TagRegistry)
		api.GET("/metric-registry", registryHandler.MetricRegistry)
		api.GET("/action-label-registry", registryHandler.ActionLabelRegistry)
		api.GET("/lifecycle-states", registryHandler.LifecycleStates)

		// Search endpoints (Elasticsearch-backed)
		if searchHandler != nil {
			api.GET("/search/assets", searchHandler.SearchAssets)
			api.GET("/search/sync-status", searchHandler.SyncStatus)
			api.GET("/search/sync-progress", searchHandler.SyncProgress)
		}

		if auditHandler != nil {
			api.GET("/audit/search", auditHandler.HandleAuditSearch)
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
			// Destructive admin ops: static admin/databrew token only.
			admin := api.Group("/admin", adminAuth)
			admin.POST("/search/reindex", adminHandler.SearchReindex)
			admin.POST("/search/reindex-jobs", adminHandler.SearchReindexCreateJob)
			admin.POST("/search/reindex-jobs/:id/stop", adminHandler.SearchReindexStopJob)
			admin.POST("/search/reindex-jobs/:id/resume", adminHandler.SearchReindexResumeJob)
			admin.POST("/search/reindex-jobs/:id/abandon", adminHandler.SearchReindexAbandonJob)

			// CYB-3229: read-only admin search views are also reachable by an
			// admin-role web session (ADMIN_EMAILS) via Authenticate, not just the
			// static token. No destructive capability here.
			adminRO := api.Group("/admin", middleware.AdminTokenOrAdminRole(cfg.AdminToken, cfg.DatabrewToken, cfg.Env))
			adminRO.GET("/search/reindex-jobs", adminHandler.SearchReindexListJobs)
			adminRO.GET("/search/reindex-jobs/:id", adminHandler.SearchReindexGetJob)
			adminRO.GET("/search/outbox-stats", adminHandler.SearchOutboxStats)
			adminRO.GET("/search/audit", adminHandler.SearchAudit)

			// CYB-3246 Phase 2: managed tag-registry CRUD. Reachable by static
			// admin token OR an admin-role web session (ADMIN_EMAILS) so the
			// Settings UI can manage tags without a restart.
			if tagRegistryHandler != nil {
				adminRO.GET("/tag-registry", tagRegistryHandler.List)
				adminRO.POST("/tag-registry", tagRegistryHandler.Create)
				adminRO.PATCH("/tag-registry/:key", tagRegistryHandler.Update)
				adminRO.DELETE("/tag-registry/:key", tagRegistryHandler.Delete)
			}
		}

		// Internal admin (hard delete). Requires ADMIN_TOKEN in production.
		if adminRoutesEnabled && purgeHandler != nil {
			internal := api.Group("/internal", adminAuth)
			internal.DELETE("/assets/:id", purgeHandler.DeleteAssetHard)
			internal.POST("/assets:batch_delete", purgeHandler.BatchDeleteAssets)
		}

		// API key management (issue/list/revoke keys for SDK/API callers).
		// Under admin auth; keys themselves carry scopes for least-privilege.
		if apiKeyHandler != nil { // pragma: allowlist secret
			// Admin-scoped: admin-role web sessions (ADMIN_EMAILS) and the
			// legacy static token (both carry "*") pass; regular users get 403.
			keys := api.Group("/admin/api-keys", middleware.RequireScope("apikeys:manage"))
			keys.POST("", apiKeyHandler.Create)
			keys.GET("", apiKeyHandler.List)
			keys.DELETE("/:id", apiKeyHandler.Revoke)
		}

		// Actions (mcap → seg → action 第三层) — CYB-3268: the 4 methods are now
		// served as first-class assets (asset_type='action') by assetHandler on
		// the assets table, unifying read+write. The legacy actionHandler is kept
		// constructed for a future backfill issue but no longer routes here; the
		// guard stays so routing tracks the action feature being wired.
		if actionHandler != nil {
			// CYB-3296: these mutate child (action) assets and MUST require the
			// same assets:write scope as their siblings (Create/Update/Delete
			// above) — otherwise a read-only API key can create/patch/delete
			// action assets (confirmed exploitable on dev).
			assets.POST("/:id/actions", middleware.RequireScope("assets:write"), assetHandler.CreateAction)
			assets.GET("/:id/actions", assetHandler.ListActions)
			assets.PATCH("/:id/actions/:action_id", middleware.RequireScope("assets:write"), assetHandler.UpdateAction)
			assets.DELETE("/:id/actions/:action_id", middleware.RequireScope("assets:write"), assetHandler.DeleteAction)
		}

		// Algo-runs (CYB-1018)
		if algoRunHandler != nil {
			api.POST("/algo-runs", algoRunHandler.Create)
			api.GET("/algo-runs", algoRunHandler.List)
			api.GET("/algo-runs/:run_id", algoRunHandler.Get)
			api.POST("/algo-runs/:run_id/start", algoRunHandler.Start)
			api.POST("/algo-runs/:run_id/finish", algoRunHandler.Finish)
			api.POST("/algo-runs/:run_id/cancel", algoRunHandler.Cancel)
			api.GET("/algo-runs/:run_id/affected-assets", algoRunHandler.GetAffectedAssets)
		}

		// Eval / Metrics (Phase 1.5)
		if evalHandler != nil {
			// CYB-3296: writing eval results mutates asset data — require assets:write.
			assets.POST("/:id/eval-results", middleware.RequireScope("assets:write"), evalHandler.ReportEvalResult)
			assets.GET("/:id/eval-results", evalHandler.ListEvalResults)
			assets.GET("/:id/metrics", evalHandler.ListMetrics)
			api.GET("/metrics/registry", evalHandler.GetRegistry)
			api.POST("/metrics:search", evalHandler.SearchByMetrics)
		}

		// Pipeline (Argo Workflows) — templates, deploy, deployments
		api.POST("/pipelines", pipelineHandler.SaveTemplate)
		api.GET("/pipelines", pipelineHandler.ListTemplates)
		api.GET("/pipelines/stats", pipelineHandler.GetStats)
		api.GET("/pipelines/:id", pipelineHandler.GetTemplate)
		api.PUT("/pipelines/:id", pipelineHandler.UpdatePipeline)
		api.DELETE("/pipelines/:id", pipelineHandler.DeleteTemplate)
		api.GET("/pipelines/:id/versions", pipelineHandler.ListVersions)
		api.PATCH("/pipelines/:id/active-version", pipelineHandler.SetActiveVersion)
		api.POST("/pipelines/:id/promote", pipelineHandler.Promote)
		api.GET("/pipelines/:id/diff/:id2", pipelineHandler.DiffTemplates)
		api.POST("/deploy", pipelineHandler.Deploy)
		api.POST("/deploy/template/:id", pipelineHandler.DeployByTemplate)
		api.GET("/execution-targets", pipelineHandler.ListExecutionTargets)
		api.POST("/execution-targets", pipelineHandler.CreateExecutionTarget)
		api.PUT("/execution-targets/:id", pipelineHandler.UpdateExecutionTarget)
		api.DELETE("/execution-targets/:id", pipelineHandler.DeleteExecutionTarget)
		api.GET("/resource-quotas", workflowHandler.ListResourceQuotas)
		api.GET("/pipeline/runtime-mounts", pipelineHandler.ListRuntimeMounts)
		api.POST("/runs", pipelineHandler.CreateRun)
		api.POST("/runs/template/:id", pipelineHandler.CreateRunByTemplate)
		api.GET("/runs", pipelineHandler.ListRuns)
		api.POST("/runs/batch", pipelineHandler.CreateBatchRun)
		api.POST("/runs/batch/:batchId/stop", pipelineHandler.StopBatchRun)
		api.GET("/runs/batch/:batchId", pipelineHandler.GetBatchStatus)
		api.GET("/runs/watcher/status", pipelineHandler.GetRunWatcherStatus)
		api.GET("/runs/by-workflow/:workflowName", pipelineHandler.GetRunByWorkflowName)
		api.GET("/runs/:id", pipelineHandler.GetRun)
		api.GET("/runs/:id/events", pipelineHandler.ListRunEvents)
		api.GET("/runs/:id/nodes", pipelineHandler.ListRunNodes)
		api.GET("/runs/:id/asset-nodes", pipelineHandler.ListRunAssetNodes)
		api.GET("/runs/:id/cost-summary", pipelineHandler.GetRunCostSummary)
		api.GET("/runs/:id/inputs", pipelineHandler.ListRunInputs)
		api.GET("/runs/:id/outputs", pipelineHandler.ListRunOutputs)
		api.GET("/runs/:id/children", pipelineHandler.ListRunChildren)
		api.GET("/runs/:id/runtime", pipelineHandler.GetRunRuntime)
		api.POST("/runs/:id/retry", pipelineHandler.RetryRunRuntime)
		api.POST("/runs/:id/resubmit", pipelineHandler.ResubmitRun)
		api.POST("/runs/:id/rerun", pipelineHandler.RerunRun)
		api.POST("/runs/:id/stop", pipelineHandler.StopRun)
		api.POST("/runs/:id/suspend", pipelineHandler.SuspendRun)
		api.POST("/runs/:id/resume", pipelineHandler.ResumeRun)
		api.POST("/runs/:id/terminate", pipelineHandler.TerminateRun)
		api.DELETE("/runs/:id", pipelineHandler.DeleteRun)
		api.POST("/pipeline-runs", pipelineHandler.CreateRun)
		api.POST("/pipeline-runs/template/:id", pipelineHandler.CreateRunByTemplate)
		api.GET("/pipeline-runs", pipelineHandler.ListRuns)
		api.GET("/pipeline-runs/watcher/status", pipelineHandler.GetRunWatcherStatus)
		api.GET("/pipeline-runs/by-workflow/:workflowName", pipelineHandler.GetRunByWorkflowName)
		api.GET("/pipeline-runs/:id", pipelineHandler.GetRun)
		api.GET("/pipeline-runs/:id/events", pipelineHandler.ListRunEvents)
		api.GET("/pipeline-runs/:id/asset-nodes", pipelineHandler.ListRunAssetNodes)
		api.GET("/pipeline-runs/:id/cost-summary", pipelineHandler.GetRunCostSummary)
		api.POST("/pipeline-runs/:id/retry", pipelineHandler.RetryRun)
		api.POST("/pipeline-runs/:id/stop", pipelineHandler.StopRun)
		api.DELETE("/pipeline-runs/:id", pipelineHandler.DeleteRun)
		api.GET("/deployments", pipelineHandler.ListDeployments)
		api.GET("/deployments/:id", pipelineHandler.GetDeployment)
		api.GET("/deployments/:id/resources", pipelineHandler.GetResourceUsage)
		api.POST("/deployments/:id/retry", pipelineHandler.RetryDeployment)
		api.POST("/deployments/:id/stop", pipelineHandler.StopDeployment)
		api.POST("/deployments/:id/save-template", pipelineHandler.SaveFromDeployment)
		api.DELETE("/deployments/:id", pipelineHandler.DeleteDeployment)
		api.POST("/pipeline-assets", pipelineHandler.RegisterOutput)
		api.GET("/assets/:id/pipeline-lineage", pipelineHandler.GetLineage)

		// Pipeline component registry
		if pipelineConfigHandler != nil {
			api.POST("/pipeline-configs", pipelineConfigHandler.Create)
			api.GET("/pipeline-configs", pipelineConfigHandler.List)
			api.GET("/pipeline-configs/:id", pipelineConfigHandler.Get)
			api.PUT("/pipeline-configs/:id", pipelineConfigHandler.Update)
			api.POST("/pipeline-configs/:id/versions", pipelineConfigHandler.CreateVersion)
			api.GET("/pipeline-configs/:id/versions/:version", pipelineConfigHandler.GetVersion)
			api.PUT("/pipeline-configs/:id/versions/:version/status", pipelineConfigHandler.UpdateVersionStatus)
			api.PUT("/pipeline-configs/:id/versions/:version", pipelineConfigHandler.UpdateVersionContent)
			api.POST("/pipeline-configs/:id/deprecate", pipelineConfigHandler.Deprecate)
		}

		if pipelineComponentHandler != nil {
			api.POST("/pipeline-components", pipelineComponentHandler.CreateComponent)
			api.GET("/pipeline-components", pipelineComponentHandler.ListComponents)
			api.GET("/pipeline-components/:id", pipelineComponentHandler.GetComponent)
			api.PUT("/pipeline-components/:id", pipelineComponentHandler.UpdateComponent)
			api.DELETE("/pipeline-components/:id", pipelineComponentHandler.DeleteComponent)
			api.GET("/pipeline-component-releases", pipelineComponentHandler.ListReleases)
			api.GET("/pipeline-component-releases/:id", pipelineComponentHandler.GetRelease)
		}

		// Workflow monitoring
		api.GET("/workflows", workflowHandler.ListWorkflows)
		api.GET("/workflows/:name/logs", workflowHandler.GetWorkflowLogs)
		api.GET("/workflows/:name/logs/stream", workflowHandler.StreamWorkflowLogs)
		api.GET("/workflows/:name/log/stream", workflowHandler.StreamWorkflowLogs)
		api.GET("/workflows/:name/resources", pipelineHandler.GetWorkflowResourceUsage)
		api.GET("/workflows/:name/nodes/:nodeId/resources", pipelineHandler.GetWorkflowNodeResourceUsage)
		api.GET("/workflows/:name/nodes/:nodeId/pod", workflowHandler.GetNodePodDiagnostics)
		api.POST("/workflows/:name/nodes/:nodeId/terminal-sessions", workflowHandler.CreateTerminalSession)
		api.GET("/pod-terminal/sessions/:id", workflowHandler.GetTerminalSession)
		api.POST("/pod-terminal/sessions/:id/terminate", workflowHandler.TerminateTerminalSession)
		api.GET("/workflows/:name", workflowHandler.GetWorkflow)
		api.POST("/workflows/:name/retry", workflowHandler.RetryWorkflow)
		api.POST("/workflows/:name/resubmit", workflowHandler.ResubmitWorkflow)
		api.POST("/workflows/:name/suspend", workflowHandler.SuspendWorkflow)
		api.POST("/workflows/:name/stop", workflowHandler.StopWorkflow)
		api.POST("/workflows/:name/resume", workflowHandler.ResumeWorkflow)
		api.POST("/workflows/:name/terminate", workflowHandler.TerminateWorkflow)
		api.DELETE("/workflows/:name", workflowHandler.DeleteWorkflow)

		// Backfill jobs
		if backfillHandler != nil {
			api.POST("/backfill", backfillHandler.CreateJob)
			api.POST("/backfill/validate-assets", backfillHandler.ValidateAssets)
			api.GET("/backfill", backfillHandler.ListJobs)
			api.GET("/backfill/:id", backfillHandler.GetJob)
			api.GET("/backfill/:id/node-summary", backfillHandler.GetNodeSummary)
			api.GET("/backfill/:id/node-failures", backfillHandler.ListNodeFailures)
			api.GET("/backfill/:id/attempts", backfillHandler.GetItemAttempts)
			api.POST("/backfill/:id/pause", backfillHandler.PauseJob)
			api.POST("/backfill/:id/resume", backfillHandler.ResumeJob)
			api.POST("/backfill/:id/rerun", backfillHandler.Rerun)
			api.POST("/backfill/:id/retry-failed", backfillHandler.RetryFailed)
			api.POST("/backfill/:id/continue-full", backfillHandler.ContinueFull)
			api.POST("/backfill/results", backfillHandler.UploadResult)
		}

		// Storage (GCS signed URL proxy + source resolver)
		if storageHandler != nil {
			api.POST("/storage/resolve", storageHandler.Resolve)
			api.POST("/storage/sign-url", storageHandler.SignURL)
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
