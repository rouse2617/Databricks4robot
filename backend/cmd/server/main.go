package main

import (
	"context"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	espkg "github.com/CyberOrigin2077/cyber-databrew/internal/elasticsearch"
	actionH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/action"
	adminH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/admin"
	assetH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/asset"
	deliveryH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/delivery"
	evalH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/eval"
	mcapH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/mcap"
	queryH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/query"
	searchH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/search"
	"github.com/CyberOrigin2077/cyber-databrew/internal/lakehouse"
	"github.com/CyberOrigin2077/cyber-databrew/internal/postgres"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	assetUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/asset"
)

// ── Application wiring types ──

// infra holds everything needed before any business logic is constructed.
type infra struct {
	cfg             *config.Config
	pg              *postgres.Client
	es              *espkg.Client
	lake            lakehouse.Querier
	mcapBytesSource mcapH.BytesSource // optional

	algoRegistry   *config.AlgoRegistry
	tagRegistry    *config.TagRegistry
	metricRegistry *config.MetricRegistry
	queryFieldReg  *config.QueryFieldRegistry
	actionLabelReg *config.ActionLabelRegistry
}

func (inf *infra) close() {
	if inf.pg != nil {
		inf.pg.Close()
	}
	if inf.lake != nil {
		inf.lake.Close()
	}
}

// coreRepos groups all repository implementations constructed from PG.
type coreRepos struct {
	asset       repository.AssetRepository
	assetTag    repository.AssetTagRepository
	algoLatest  repository.AssetAlgoLatestRepository
	assetEvent  repository.AssetEventRepository
	mcap        repository.McapFileRepository
	delivery    repository.DeliveryRepository
	action      repository.ActionRepository
	savedQuery  *postgres.SavedQueryRepo
	idempotency repository.IdempotencyRepository
}

// coreHandlers groups all HTTP handlers that form the main API surface.
type coreHandlers struct {
	asset    *assetH.Handler
	algo     *assetH.AlgoHandler
	mcap     *mcapH.Handler
	delivery *deliveryH.Handler
	eval     *evalH.Handler
	action   *actionH.Handler
	query    *queryH.Handler
	assetUC  *assetUC.Usecase
}

// optional holds components that are not required for the core API to function.
type optional struct {
	admin            *adminH.Handler
	purge            *adminH.PurgeHandler
	outboxCancel     context.CancelFunc
	searchSyncFn     func() searchH.SyncInfo
	searchProgressFn func(context.Context) (searchH.SyncProgress, error)
	configWatcher    *config.ConfigWatcher

	outboxRelayStarted        bool
	outboxESSubscriberStarted bool
}

func (o *optional) close() {
	if o.outboxCancel != nil {
		o.outboxCancel()
	}
	if o.configWatcher != nil {
		o.configWatcher.Stop()
	}
}

// ── main: wire everything together ──

func main() {
	inf := setupInfra()
	defer inf.close()

	core := setupCore(inf)
	opt := setupOptional(inf, core)
	defer opt.close()

	runServer(inf, core, opt)
}
