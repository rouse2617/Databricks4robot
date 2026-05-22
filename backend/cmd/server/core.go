package main

import (
	actionH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/action"
	assetH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/asset"
	deliveryH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/delivery"
	evalH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/eval"
	mcapH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/mcap"
	queryH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/query"
	"github.com/CyberOrigin2077/cyber-databrew/internal/postgres"
	actionUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/action"
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
	evalRepo := postgres.NewEvalRepo(pg)
	actionRepo := postgres.NewActionRepo(pg)
	savedQueryRepo := postgres.NewSavedQueryRepo(pg)
	idempotencyRepo := postgres.NewIdempotencyRepo(pg)

	algoUC := assetUC.NewAlgoUsecase(pg, assetRepo, algoLatestRepo, assetEventRepo, inf.algoRegistry)
	assetUsecase := assetUC.NewWithProjections(pg, assetRepo, assetTagRepo, algoLatestRepo, assetEventRepo, inf.tagRegistry, inf.algoRegistry)
	assetUsecase.SetLogicalAssetRepo(postgres.NewLogicalAssetRepo(pg))

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

	deliveryHandler := deliveryH.New(deliveryRepo, idempotencyRepo, assetEventRepo)
	evalHandler := evalH.New(evalRepo, inf.metricRegistry, assetEventRepo)
	actionHandler := actionH.New(actionUC.NewWithLabelRegistry(pg, actionRepo, assetRepo, assetEventRepo, inf.actionLabelReg))

	var queryHandler *queryH.Handler
	if assetUsecase != nil {
		queryHandler = queryH.New(assetUsecase, inf.queryFieldReg, inf.es, savedQueryRepo)
	}

	return &coreHandlers{
		asset:    assetHandler,
		algo:     algoHandler,
		mcap:     mcapHandler,
		delivery: deliveryHandler,
		eval:     evalHandler,
		action:   actionHandler,
		query:    queryHandler,
		assetUC:  assetUsecase,
	}
}
