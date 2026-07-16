package main

import (
	"context"

	"cloud.google.com/go/storage"

	"github.com/CyberOrigin2077/cyber-databrew/internal/argo"
	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	espkg "github.com/CyberOrigin2077/cyber-databrew/internal/elasticsearch"
	actionH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/action"
	adminH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/admin"
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
	searchH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/search"
	storageH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/storage"
	workflowH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/workflow"
	"github.com/CyberOrigin2077/cyber-databrew/internal/k8s"
	"github.com/CyberOrigin2077/cyber-databrew/internal/lakehouse"
	"github.com/CyberOrigin2077/cyber-databrew/internal/postgres"
	"github.com/CyberOrigin2077/cyber-databrew/internal/queryplan"
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
	gcsClient       *storage.Client   // optional; owns GCS client for mcapBytesSource

	algoRegistry   *config.AlgoRegistry
	tagRegistry    *config.TagRegistry
	metricRegistry *config.MetricRegistry
	queryFieldReg  *config.QueryFieldRegistry
	actionLabelReg *config.ActionLabelRegistry
	workflowClient argo.WorkflowClient
	podClient      k8s.PodClient
	execClient     k8s.ExecClient

	// CYB-3486 multi-cluster: shared repo + per-cluster factories.
	// nil when inf.pg is nil (tests without a DB).
	clusterRepo repository.ClusterRepository
	k8sFactory  k8s.ClientFactory
	argoFactory argo.ClientFactory
}

func (inf *infra) close() {
	if inf.gcsClient != nil {
		_ = inf.gcsClient.Close()
	}
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
	asset             *assetH.Handler
	algo              *assetH.AlgoHandler
	mcap              *mcapH.Handler
	delivery          *deliveryH.Handler
	customer          *customerH.Handler
	deliveryRule      *deliveryruleH.Handler
	algoRun           *algorunH.Handler
	eval              *evalH.Handler
	action            *actionH.Handler
	query             *queryH.Handler
	workflow          *workflowH.Handler
	pipeline          *pipelineH.Handler
	pipelineConfig    *pipelineConfigH.Handler
	pipelineComponent *pipelineComponentH.Handler
	backfill          *backfillH.Handler
		storage           *storageH.Handler
	assetUC           *assetUC.Usecase
	// CYB-3384: assetRepo is retained so setupOptional can hand a facet
	// source to the query handler after the sync-health cache is ready.
	assetRepo *postgres.AssetRepo
}

// optional holds components that are not required for the core API to function.
type optional struct {
	admin            *adminH.Handler
	purge            *adminH.PurgeHandler
	outboxCancel     context.CancelFunc
	searchSyncFn     func() searchH.SyncInfo
	searchProgressFn func(context.Context) (searchH.SyncProgress, error)
	configWatcher    *config.ConfigWatcher

	// CYB-3384: syncHealth is the PG↔ES gap cache the query planner reads to
	// decide whether to route facet aggregations to PG (drift-safe) or ES
	// (fast). Nil when neither pg nor es are wired.
	syncHealth       *queryplan.SyncHealthCache
	syncHealthCancel context.CancelFunc

	outboxRelayStarted        bool
	outboxESSubscriberStarted bool
}

func (o *optional) close() {
	if o.outboxCancel != nil {
		o.outboxCancel()
	}
	if o.syncHealthCancel != nil {
		o.syncHealthCancel()
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
