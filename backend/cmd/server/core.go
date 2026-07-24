package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/deliveryrules"
	"github.com/CyberOrigin2077/cyber-databrew/internal/grace"
	actionH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/action"
	algorunH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/algorun"
	assetH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/asset"
	backfillH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/backfill"
	customerH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/customer"
	deliveryH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/delivery"
	deliveryruleH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/deliveryrule"
	evalH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/eval"
	mcapH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/mcap"
	pipelineH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/pipeline"
	pipelineComponentH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/pipeline_component"
	pipelineConfigH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/pipeline_config"
	queryH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/query"
	storageH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/storage" // NEW
	subtaskH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/subtask"
	workflowH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/workflow"
	"github.com/CyberOrigin2077/cyber-databrew/internal/k8s"
	"github.com/CyberOrigin2077/cyber-databrew/internal/metrics/cloudmonitoring"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/notify/feishu"
	"github.com/CyberOrigin2077/cyber-databrew/internal/postgres"
	runtimeArgo "github.com/CyberOrigin2077/cyber-databrew/internal/runtimeos/adapter/argo"
	"github.com/CyberOrigin2077/cyber-databrew/internal/subtask"
	actionUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/action"
	algorunUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/algorun"
	assetUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/asset"
	backfillUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/backfill"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
	pipelineComponentUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline_component"
	pipelineConfigUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline_config"
)

// ── Layer 2: Core business layer (repos + usecases + handlers) ──

func setupCore(inf *infra) *coreHandlers {
	pg := inf.pg

	assetRepo := postgres.NewAssetRepo(pg)
	assetTagRepo := postgres.NewAssetTagRepo(pg)
	algoLatestRepo := postgres.NewAssetAlgoLatestRepo(pg)
	assetEventRepo := postgres.NewAssetEventRepo(pg)
	mcapRepo := postgres.NewMcapFileRepo(pg)
	deliveryRepo := postgres.NewDeliveryRepo(pg)
	customerRepo := postgres.NewCustomerRepo(pg)
	deliveryRuleRepo := postgres.NewDeliveryRuleRepo(pg)
	algoRunRepo := postgres.NewAlgoRunRepo(pg)
	evalRepo := postgres.NewEvalRepo(pg)
	actionRepo := postgres.NewActionRepo(pg)
	savedQueryRepo := postgres.NewSavedQueryRepo(pg)
	idempotencyRepo := postgres.NewIdempotencyRepo(pg)

	usageStatsRepo := postgres.NewUsageStatsRepo(pg)

	algoUC := assetUC.NewAlgoUsecase(pg, assetRepo, algoLatestRepo, assetEventRepo, inf.algoRegistry)
	algoUC.SetAlgoRunRepo(algoRunRepo)
	assetUsecase := assetUC.NewWithProjections(pg, assetRepo, assetTagRepo, algoLatestRepo, assetEventRepo, inf.tagRegistry, inf.algoRegistry)
	assetTypeSchemas := models.NewSchemaRegistry()
	assetUsecase.SetSchemaRegistry(assetTypeSchemas)
	assetUsecase.SetLogicalAssetRepo(postgres.NewLogicalAssetRepo(pg))
	assetUsecase.SetCustomerRepo(customerRepo) // CYB-1070: customer.* namespace lint
	// CYB-1164: asset hierarchy validator.
	assetUsecase.SetValidator(deliveryrules.NewAssetWriteValidatorWithSchemas(
		deliveryrules.NewAssetRepoParentGetter(assetRepo),
		assetTypeSchemas,
	))
	assetUsecase.SetUsageStatsRepo(usageStatsRepo)          // CYB-1095/1096: usage stats
	assetUsecase.SetActionLabelRegistry(inf.actionLabelReg) // CYB-3268: action label vocabulary

	assetHandler := assetH.New(assetUsecase, deliveryRepo)
	assetHandler.SetMcapRepo(mcapRepo)
	assetHandler.SetPG(pg)

	algoHandler := assetH.NewAlgoHandler(algoUC)

	mcapHandler := mcapH.New(mcapRepo)
	mcapHandler.SetTxRunner(pg)
	mcapHandler.SetEventRepo(assetEventRepo)
	mcapHandler.SetAssetRepo(assetRepo)
	// CYB-3797: enable auto-extract of vibecap_tasks / source_platform /
	// location.address into task / source / city tags on every mcap POST.
	mcapHandler.SetAssetTagRepo(assetTagRepo)
	if inf.mcapBytesSource != nil {
		mcapHandler.SetBytesSource(inf.mcapBytesSource)
	}

	deliveryHandler := deliveryH.New(deliveryRepo, idempotencyRepo, customerRepo, assetEventRepo)
	deliveryHandler.SetRuleEngine(deliveryrules.NewEngine(deliveryRuleRepo, assetRepo, assetTagRepo, customerRepo))
	customerHandler := customerH.New(customerRepo)
	deliveryRuleHandler := deliveryruleH.New(deliveryRuleRepo, customerRepo)
	algoRunUC := algorunUC.New(algoRunRepo)
	algoRunUC.SetAssetRepo(assetRepo)
	algoRunUC.SetEventRepo(assetEventRepo)
	algoRunHandler := algorunH.New(algoRunUC)
	evalHandler := evalH.New(evalRepo, inf.metricRegistry, assetEventRepo)
	actionHandler := actionH.New(actionUC.NewWithLabelRegistry(pg, actionRepo, assetRepo, assetEventRepo, inf.actionLabelReg))

	var queryHandler *queryH.Handler
	if assetUsecase != nil {
		queryHandler = queryH.New(assetUsecase, inf.queryFieldReg, inf.es, savedQueryRepo)
	}

	// ── Pipeline (Argo Workflows) ──
	pipelineTemplateRepo := postgres.NewPipelineTemplateRepo(pg)
	pipelineDeploymentRepo := postgres.NewPipelineDeploymentRepo(pg)
	executionTargetRepo := postgres.NewExecutionTargetRepo(pg)
	pipelineRunRepo := postgres.NewPipelineRunRepo(pg)
	pipelineRunNodeRepo := postgres.NewPipelineRunNodeRepo(pg)
	pipelineRunEventRepo := postgres.NewPipelineRunEventRepo(pg)
	runRelationRepo := postgres.NewRunRelationRepo(pg)
	runInputRepo := postgres.NewRunInputRepo(pg)
	pipelineRunAssetNodeRepo := postgres.NewPipelineRunAssetNodeRepo(pg)
	pipelineRunNotificationRepo := postgres.NewPipelineRunNotificationRepo(pg)
	pipelineRunWatcherStateRepo := postgres.NewPipelineRunWatcherStateRepo(pg)
	pipelineConfigRepo := postgres.NewPipelineConfigRepo(pg)
	puc := pipelineUC.New(pipelineTemplateRepo, pipelineDeploymentRepo, assetRepo, inf.workflowClient, inf.cfg.ArgoWorkflowsNamespace)
	if inf.workflowClient != nil {
		puc.SetRuntimeAdapter(runtimeArgo.New(inf.workflowClient, inf.cfg.ArgoWorkflowsNamespace))
	}
	// CYB-3486 PR 4c: wire per-cluster argo factory so pipeline submits route
	// by target.cluster_id. Nil-safe (no PG → nil factory → legacy adapter path).
	puc.SetArgoFactory(inf.argoFactory)
	puc.SetArgoWorkflowTTLSecondsAfterCompletion(inf.cfg.ArgoWorkflowTTLSecondsAfterCompletion)
	// CYB-3681: exit-hook injection is opt-in (ARGO_EXIT_HOOK_ENABLED=true).
	// The bulk-pull watcher is the writeback path; an exit-notify pod per
	// workflow is pure cost at batch scale. The inbound webhook endpoint stays
	// registered regardless, so re-enabling is a config flip, not a deploy.
	if inf.cfg.ArgoExitHookEnabled {
		puc.SetArgoRunWebhook(
			inf.cfg.ArgoRunWebhookURL,
			inf.cfg.ArgoRunWebhookTokenSecretName,
			inf.cfg.ArgoRunWebhookTokenSecretKey,
			inf.cfg.ArgoRunWebhookImage,
		)
	}
	puc.SetResourceGuardConfig(pipelineUC.ResourceGuardConfig{
		MaxCPU:                        inf.cfg.PipelineResourceMaxCPU,
		MaxMemory:                     inf.cfg.PipelineResourceMaxMemory,
		MaxDisk:                       inf.cfg.PipelineResourceMaxDisk,
		MaxGPU:                        inf.cfg.PipelineResourceMaxGPU,
		UnschedulablePendingThreshold: inf.cfg.PipelineUnschedulablePendingThresholdDuration(),
	})
	puc.SetRunRepositories(executionTargetRepo, pipelineRunRepo, pipelineRunNodeRepo)
	puc.SetRunEventRepo(pipelineRunEventRepo)
	puc.SetRunFactRepositories(runRelationRepo, runInputRepo)
	puc.SetObservabilityRepositories(pipelineRunAssetNodeRepo, pipelineRunNotificationRepo, pipelineRunWatcherStateRepo)
	videoDurationRepo := postgres.NewVideoDurationRepo(pg)
	puc.SetVideoDurationRepo(videoDurationRepo)
	puc.SetAssetEventRepo(assetEventRepo)
	puc.SetRelationWriter(assetRepo)
	puc.SetLogicalAssetRepo(postgres.NewLogicalAssetRepo(pg))
	puc.SetPipelineConfigRepo(pipelineConfigRepo)
	if clientset, err := k8s.NewClientset(""); err != nil {
		slog.Warn("runtime config projection store disabled", "err", err)
	} else {
		puc.SetRuntimeConfigStore(k8s.NewRuntimeConfigStore(clientset))
		// Price pipeline step costs by the node's real machine type (CYB-3073).
		puc.SetNodeInstanceResolver(k8s.NewNodeInstanceResolver(clientset))
		// CYB-3680: content-addressed runtime-config CMs are shared and
		// owner-less; this janitor reclaims those whose sliding-reference
		// annotation aged past the TTL (default cluster; per-cluster sweeps
		// ride CYB-3678/3681's channel loops).
		k8s.StartRuntimeConfigJanitor(context.Background(), clientset,
			time.Duration(inf.cfg.RuntimeConfigTTLDays)*24*time.Hour, time.Hour)
	}
	// CYB-3680: DB blob = rebuildable source of truth for content-addressed
	// runtime configs.
	puc.SetRuntimeConfigBlobStore(postgres.NewRuntimeConfigBlobRepo(pg))
	// CYB-3486: route the runtime-config ConfigMap to the TARGET's cluster.
	// Without this, a delivery-clust run creates its ConfigMap through the
	// default (cyber-clust) clientset and fails with `namespaces
	// "cyber-delivery-prod" not found`. Nil-safe (no PG → nil factory → the
	// default-cluster singleton set above stays in effect).
	if inf.k8sFactory != nil {
		puc.SetRuntimeConfigStoreFactory(k8s.NewRuntimeConfigStoreFactory(inf.k8sFactory))
	}
	if inf.cfg.PricingConfigPath != "" {
		priceCfg, err := pipelineUC.LoadPricing(inf.cfg.PricingConfigPath)
		if err != nil {
			slog.Warn("load pricing config", "err", err)
		}
		puc.SetPricing(priceCfg)
	}
	// Push (Argo exit hook) is the primary status signal (CYB-3058); the watcher
	// is a low-frequency reconcile backstop. Interval/scan-limit are configurable.
	watcherInterval := time.Duration(inf.cfg.PipelineRunWatcherIntervalSec) * time.Second
	if watcherInterval <= 0 {
		watcherInterval = 30 * time.Second
	}
	watcherScanLimit := int(inf.cfg.PipelineRunWatcherScanLimit)
	if watcherScanLimit <= 0 {
		watcherScanLimit = 100
	}
	puc.StartRunEventWatcher(context.Background(), watcherInterval, watcherScanLimit)

	// Grace video-duration sync (CYB-3072): DataBrew actively pulls durations
	// from Grace and upserts video_durations on a background loop. Best-effort;
	// disabled when GRACE_* is unconfigured. Reusable grace.Client can grow other
	// Grace data needs later.
	if graceClient := grace.NewClient(grace.ConfigFromEnv()); graceClient.Enabled() {
		syncInterval := time.Duration(inf.cfg.VideoDurationSyncIntervalSec) * time.Second
		grace.NewSyncer(graceClient, videoDurationRepo).StartSyncLoop(context.Background(), syncInterval)
		slog.Info("grace video duration sync enabled", "intervalSec", inf.cfg.VideoDurationSyncIntervalSec)
	} else {
		slog.Info("grace video duration sync disabled (GRACE_API_URL/USERNAME/PASSWORD not set)")
	}

	backfillRepo := postgres.NewBackfillRepo(pg)
	puc.SetBackfillRepo(backfillRepo)
	backfillResultRepo := postgres.NewBackfillResultRepo(pg)
	backfillUC := backfillUC.New(backfillRepo, puc)
	backfillUC.SetResultRepositories(backfillResultRepo, assetRepo)

	// CYB-3691: write stale-items gauge to GCP Cloud Monitoring (best-effort).
	// projectID is passed empty; the writer falls back to the metadata server.
	if mw, mwErr := cloudmonitoring.NewWriter(context.Background(), ""); mwErr == nil {
		backfillUC.SetMonWriter(mw)
		slog.Info("cloud monitoring writer initialized")
	} else {
		slog.Warn("cloud monitoring writer unavailable", "err", mwErr)
	}
	// Batch job completion Feishu notification (CYB-3071). Empty webhook URL
	// disables it; feishu.Client.SendText becomes a no-op in that case.
	backfillUC.SetNotifier(
		feishu.NewClient(feishu.Config{WebhookURL: inf.cfg.BackfillNotifyFeishuWebhookURL}),
		inf.cfg.FrontendBaseURL,
	)
	// CYB-3491 (P2): the submitter replaced the claim/reaper/worker-pool
	// execution queue. It is periodic AND kicked by materialize/resume/rerun,
	// so dispatch survives redeploys by construction — the reaper, the
	// boot-time ResumeIncompleteBatches, and the P0 pool-recovery stopgap are
	// all gone. Argo owns queueing/parallelism/execution from here.
	backfillUC.SetSubmitQueue(backfillRepo)
	// CYB-3679: per-cluster online tuning (concurrency/rate/batch/paused) —
	// re-read every cycle, so a PUT bites within one tick.
	backfillUC.SetDispatcherConfigRepo(postgres.NewDispatcherConfigRepo(pg))
	backfillUC.StartSubmitter()
	puc.SetBatchSubmitKicker(backfillUC.KickSubmitter)
	// Reconcile backstop (CYB-3078): finalize + notify batch jobs whose children
	// finished, without depending on the exit hook or a user opening the page.
	backfillUC.StartJobReconciler(
		time.Duration(inf.cfg.BackfillReconcileIntervalSec)*time.Second,
		int(inf.cfg.BackfillReconcileScanLimit),
	)
	backfillHandler := backfillH.New(backfillUC)

	pipelineHandler := pipelineH.New(puc, inf.cfg.PricingConfigPath, backfillUC)

	// Standalone pipeline config library
	pipelineConfigHandler := pipelineConfigH.New(pipelineConfigUC.New(pipelineConfigRepo))

	// Pipeline component registry
	pipelineComponentRepo := postgres.NewPipelineComponentRepo(pg)
	pipelineComponentUC := pipelineComponentUC.New(pipelineComponentRepo)
	if err := pipelineComponentUC.SeedSystemComponents(context.Background()); err != nil {
		slog.Warn("seed system components", "err", err)
	}
	pipelineComponentHandler := pipelineComponentH.New(pipelineComponentUC)

	// ── Workflow monitoring ──
	workflowHandler := workflowH.New(inf.workflowClient, inf.cfg.ArgoWorkflowsNamespace)
	workflowHandler.SetPodClient(inf.podClient)
	workflowHandler.SetExecClient(inf.execClient)
	workflowHandler.SetRunRepositories(pipelineRunRepo, pipelineRunEventRepo, pipelineRunNodeRepo)
	// CYB-3486 PR 4b: elastic_quota / resource_quota now route by cluster_id
	// through the factory. Nil-safe: if inf.k8sFactory is nil (no PG), the
	// handlers fall back to the pre-3486 env singleton.
	workflowHandler.SetK8sFactory(inf.k8sFactory)
	// CYB-3486: /workflows/:name resolves the workflow's owning cluster (by name
	// → run → target → cluster_id) and routes its Argo calls there, so
	// non-default-cluster runs (e.g. delivery-clust) return logs/detail instead
	// of "workflow not found" from the default argo-server. Nil-safe: no PG →
	// nil factory → env singleton, unchanged behavior.
	workflowHandler.SetArgoFactory(inf.argoFactory)
	workflowHandler.SetExecutionTargetRepo(executionTargetRepo)

	// ── Storage (GCS signed URL proxy + Grace resolver) ──
	var storageHandler *storageH.Handler
	if inf.gcsClient != nil {
		storageHandler = storageH.NewHandler(inf.gcsClient)
	}

	// ── Subscription tasks (CYB-3778): Pub/Sub consumer auto-dispatch. ──
	subTaskRepo := postgres.NewSubscriptionTaskRepo(pg)
	subTaskHandler := subtaskH.New(subTaskRepo)
	subTaskUC := subtask.New(
		subTaskRepo,
		subtask.NewPubSubClient(),
		subtask.BatchCreatorFn(func(ctx context.Context, tmpl, name string, ids []string, target string, ver int, owner string) (string, error) {
			job, err := puc.CreateBatchJob(ctx, tmpl, name, ids, target, ver, owner)
			if err != nil {
				return "", err
			}
			if job == nil {
				return "", nil
			}
			return job.ID, nil
		}),
		subtask.RunCreatorFn(func(ctx context.Context, tmpl, name, assetID, target string, ver int, owner string) (string, error) {
			run, err := puc.CreateRunByTemplateID(ctx, tmpl, name, []string{assetID}, pipelineUC.DeployOptions{
				TargetID:        target,
				Owner:           owner,
				TemplateVersion: ver,
			})
			if err != nil {
				return "", err
			}
			if run == nil {
				return "", nil
			}
			return run.ID, nil
		}),
		subtask.NewFeishuNotifierFromEnv(),
		subtask.Options{},
	)
	subTaskUC.StartLoop(context.Background())
	slog.Info("subscription-task scheduler started (cyb-3778)")

	return &coreHandlers{
		asset:             assetHandler,
		algo:              algoHandler,
		mcap:              mcapHandler,
		delivery:          deliveryHandler,
		customer:          customerHandler,
		deliveryRule:      deliveryRuleHandler,
		algoRun:           algoRunHandler,
		eval:              evalHandler,
		action:            actionHandler,
		pipeline:          pipelineHandler,
		pipelineConfig:    pipelineConfigHandler,
		pipelineComponent: pipelineComponentHandler,
		backfill:          backfillHandler,
		subscriptionTask:  subTaskHandler,
		query:             queryHandler,
		workflow:          workflowHandler,
		storage:           storageHandler,
		assetUC:           assetUsecase,
		assetRepo:         assetRepo,
	}
}
