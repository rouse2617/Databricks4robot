package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/deliveryrules"
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
	workflowH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/workflow"
	"github.com/CyberOrigin2077/cyber-databrew/internal/k8s"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/postgres"
	runtimeArgo "github.com/CyberOrigin2077/cyber-databrew/internal/runtimeos/adapter/argo"
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
	assetUsecase.SetUsageStatsRepo(usageStatsRepo) // CYB-1095/1096: usage stats

	assetHandler := assetH.New(assetUsecase, deliveryRepo)
	assetHandler.SetMcapRepo(mcapRepo)
	assetHandler.SetPG(pg)

	algoHandler := assetH.NewAlgoHandler(algoUC)

	mcapHandler := mcapH.New(mcapRepo)
	mcapHandler.SetTxRunner(pg)
	mcapHandler.SetEventRepo(assetEventRepo)
	mcapHandler.SetAssetRepo(assetRepo)
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
	puc.SetArgoWorkflowTTLSecondsAfterCompletion(inf.cfg.ArgoWorkflowTTLSecondsAfterCompletion)
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
	puc.SetAssetEventRepo(assetEventRepo)
	puc.SetRelationWriter(assetRepo)
	puc.SetLogicalAssetRepo(postgres.NewLogicalAssetRepo(pg))
	puc.SetPipelineConfigRepo(pipelineConfigRepo)
	if clientset, err := k8s.NewClientset(""); err != nil {
		slog.Warn("runtime config projection store disabled", "err", err)
	} else {
		puc.SetRuntimeConfigStore(k8s.NewRuntimeConfigStore(clientset))
	}
	if inf.cfg.PricingConfigPath != "" {
		priceCfg, err := pipelineUC.LoadPricing(inf.cfg.PricingConfigPath)
		if err != nil {
			slog.Warn("load pricing config", "err", err)
		}
		puc.SetPricing(priceCfg)
	}
	puc.StartRunEventWatcher(context.Background(), 3*time.Second, 100)

	backfillRepo := postgres.NewBackfillRepo(pg)
	puc.SetBackfillRepo(backfillRepo)
	backfillResultRepo := postgres.NewBackfillResultRepo(pg)
	// Use NewWithPostgres so backfill.CreateBackfill and other lifecycle paths
	// run inside an atomic WithTx boundary (see usecase CreateBackfill ~:242).
	// pg is already in scope at line 79; with pgClient nil the constructor
	// would have run the legacy non-atomic fallback.
	backfillUC := backfillUC.NewWithPostgres(backfillRepo, puc, pg)
	backfillUC.SetResultRepositories(backfillResultRepo, assetRepo)

	// Phase 4 Commit C1: dispatcher is now DEFAULT-ON. The legacy
	// pool is still reachable as a fallback via
	// BACKFILL_DISPATCH_MODE=legacy (committed Commit C2 will delete
	// that path entirely once we've validated the dispatcher in
	// dev). Both dispatcher and legacy cannot run on the same data
	// in the same process — see Commit B's BACKFILL_DISPATCH_MODE
	// documentation.
	if os.Getenv("BACKFILL_DISPATCH_MODE") == "legacy" {
		// EMERGENCY FALLBACK. To re-enable, set
		// BACKFILL_DISPATCH_MODE=legacy. Will be removed in Commit C2.
		backfillUC.StartReaper()
		backfillUC.ResumeIncompleteBatches(context.Background())
		slog.Info("backfill dispatcher SKIPPED (mode=legacy legacy path active)")
	} else {
		dispatcher := backfillUC.NewDispatcher(backfillRepo, puc, backfillUC.DispatcherConfig{
			Tick:          5 * time.Second,
			LeaseSec:      60,
			MaxAttempts:   3,
			WorkerCount:   5,
			JobBufferSize: 64,
			BackoffBase:   1 * time.Second,
			BackoffMax:    30 * time.Second,
		})
		dispatcher.Start(context.Background())
		slog.Info("backfill dispatcher started (mode=outbox default)")
	}
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
	workflowHandler.SetRunRepositories(pipelineRunRepo, pipelineRunEventRepo)

	// ── Storage (GCS signed URL proxy + Grace resolver) ──
	var storageHandler *storageH.Handler
	if inf.gcsClient != nil {
		storageHandler = storageH.NewHandler(inf.gcsClient)
	}

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
		query:             queryHandler,
		workflow:          workflowHandler,
		storage:           storageHandler,
		assetUC:           assetUsecase,
	}
}
