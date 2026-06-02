package main

import (
	"context"
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
	queryH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/query"
	workflowH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/workflow"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/postgres"
	actionUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/action"
	algorunUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/algorun"
	assetUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/asset"
	backfillUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/backfill"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
	pipelineComponentUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline_component"
	"log/slog"
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
	puc := pipelineUC.New(pipelineTemplateRepo, pipelineDeploymentRepo, assetRepo, inf.workflowClient, inf.cfg.ArgoWorkflowsNamespace)
	puc.SetRunRepositories(executionTargetRepo, pipelineRunRepo, pipelineRunNodeRepo)
	puc.SetAssetEventRepo(assetEventRepo)
	puc.SetRelationWriter(assetRepo)
	puc.SetLogicalAssetRepo(postgres.NewLogicalAssetRepo(pg))
	if inf.cfg.PricingConfigPath != "" {
		priceCfg, err := pipelineUC.LoadPricing(inf.cfg.PricingConfigPath)
		if err != nil {
			slog.Warn("load pricing config", "err", err)
		}
		puc.SetPricing(priceCfg)
	}
	pipelineHandler := pipelineH.New(puc, inf.cfg.PricingConfigPath)

	// Pipeline component registry
	pipelineComponentRepo := postgres.NewPipelineComponentRepo(pg)
	pipelineComponentUC := pipelineComponentUC.New(pipelineComponentRepo)
	if err := pipelineComponentUC.SeedSystemComponents(context.Background()); err != nil {
		slog.Warn("seed system components", "err", err)
	}
	pipelineComponentHandler := pipelineComponentH.New(pipelineComponentUC)

	backfillRepo := postgres.NewBackfillRepo(pg)
	backfillUC := backfillUC.New(backfillRepo, puc)
	backfillHandler := backfillH.New(backfillUC)

	// ── Workflow monitoring ──
	workflowHandler := workflowH.New(inf.workflowClient, inf.cfg.ArgoWorkflowsNamespace)
	workflowHandler.SetPodClient(inf.podClient)

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
		pipelineComponent: pipelineComponentHandler,
		backfill:          backfillHandler,
		query:             queryHandler,
		workflow:          workflowHandler,
		assetUC:           assetUsecase,
	}
}
