package main

import (
	"context"
	"github.com/CyberOrigin2077/cyber-databrew/internal/k8s"
	"log/slog"
	"github.com/CyberOrigin2077/cyber-databrew/internal/deliveryrules"
	pipelineH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/pipeline"
	pipelineComponentH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/pipeline_component"
	backfillH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/backfill"
	actionH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/action"
	algorunH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/algorun"
	assetH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/asset"
	customerH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/customer"
	deliveryH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/delivery"
	deliveryruleH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/deliveryrule"
	evalH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/eval"
	mcapH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/mcap"
	queryH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/query"
	"github.com/CyberOrigin2077/cyber-databrew/internal/postgres"
	workflowH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/workflow"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
	pipelineComponentUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline_component"
	actionUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/action"
	algorunUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/algorun"
	backfillUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/backfill"
	assetUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/asset"
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
	assetUsecase.SetLogicalAssetRepo(postgres.NewLogicalAssetRepo(pg))
	assetUsecase.SetCustomerRepo(customerRepo) // CYB-1070: customer.* namespace lint
	// CYB-1164: asset hierarchy validator.
	assetUsecase.SetValidator(deliveryrules.NewAssetWriteValidator(
		deliveryrules.NewAssetRepoParentGetter(assetRepo),
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
	var (
		pipelineHandler *pipelineH.Handler
		puc             *pipelineUC.Usecase
	)
	if kc := inf.k8sClient; kc != nil {
		wfClient := k8s.NewArgoClient(kc.ArgoClientset, kc.KubeClientset)
		puc = pipelineUC.New(pipelineTemplateRepo, pipelineDeploymentRepo, assetRepo, wfClient, inf.cfg.ArgoWorkflowsNamespace)
		puc.SetAssetEventRepo(assetEventRepo)
		puc.SetRelationWriter(assetRepo)
		puc.SetMetricsClient(k8s.NewMetricsClient(kc.KubeClientset, kc.MetricsClientset))
		pipelineHandler = pipelineH.New(puc)
	}

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
	var workflowHandler *workflowH.Handler
	if kc := inf.k8sClient; kc != nil {
		wfClient := k8s.NewArgoClient(kc.ArgoClientset, kc.KubeClientset)
		workflowHandler = workflowH.New(wfClient, inf.cfg.ArgoWorkflowsNamespace)
	}

	return &coreHandlers{
		asset:        assetHandler,
		algo:         algoHandler,
		mcap:         mcapHandler,
		delivery:     deliveryHandler,
		customer:     customerHandler,
		deliveryRule: deliveryRuleHandler,
		algoRun:      algoRunHandler,
		eval:         evalHandler,
		action:       actionHandler,
		pipeline:           pipelineHandler,
			pipelineComponent:  pipelineComponentHandler,
			backfill:           backfillHandler,
		query:        queryHandler,
		workflow:     workflowHandler,
		assetUC:      assetUsecase,
	}
}
