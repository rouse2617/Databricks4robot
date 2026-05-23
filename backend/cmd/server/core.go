package main

import (
	"github.com/CyberOrigin2077/cyber-databrew/internal/deliveryrules"
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
	actionUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/action"
	algorunUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/algorun"
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
	assetUsecase.SetCustomerRepo(customerRepo)     // CYB-1070: customer.* namespace lint
	assetUsecase.SetUsageStatsRepo(usageStatsRepo) // CYB-1095/1096: usage stats

	assetHandler := assetH.New(assetUsecase, deliveryRepo)
	assetHandler.SetMcapRepo(mcapRepo)
	assetHandler.SetPG(pg)

	algoHandler := assetH.NewAlgoHandler(algoUC)

	mcapHandler := mcapH.New(mcapRepo)
	mcapHandler.SetTxRunner(pg)
	mcapHandler.SetEventRepo(assetEventRepo)
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

	return &coreHandlers{
		asset:         assetHandler,
		algo:          algoHandler,
		mcap:          mcapHandler,
		delivery:      deliveryHandler,
		customer:      customerHandler,
		deliveryRule:  deliveryRuleHandler,
		algoRun:  algoRunHandler,
		eval:     evalHandler,
		action:   actionHandler,
		query:    queryHandler,
		assetUC:  assetUsecase,
	}
}
