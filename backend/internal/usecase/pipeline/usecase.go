package pipeline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"errors"

	"log/slog"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	"github.com/CyberOrigin2077/cyber-databrew/internal/argo"
	"github.com/CyberOrigin2077/cyber-databrew/internal/batchprogress"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	runtimeadapter "github.com/CyberOrigin2077/cyber-databrew/internal/runtimeos/adapter"
	runstate "github.com/CyberOrigin2077/cyber-databrew/internal/runtimeos/state"
	"github.com/CyberOrigin2077/cyber-databrew/internal/transpiler"
	"github.com/CyberOrigin2077/cyber-databrew/internal/usecase/assetvalidation"
	configUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline_config"
	sigsyaml "sigs.k8s.io/yaml"
)

// Sentinel errors.
var (
	ErrTemplateNotFound        = errors.New("template not found")
	ErrDeploymentNotFound      = errors.New("deployment not found")
	ErrAssetNotFound           = errors.New("asset not found")
	ErrInvalidArgument         = errors.New("invalid argument")
	ErrExecutionTargetNotFound = errors.New("execution target not found")
	ErrWorkflowUnavailable     = errors.New("workflow service unavailable: argo server not configured")
	// ErrWorkflowSubmitIncomplete means the runtime accepted the submit call but
	// no Argo UID ever materialized (e.g. a client rate-limited "phantom
	// success"): the workflow CR isn't actually live. It is a RETRYABLE submit
	// failure — the batch submitter leaves the item pending and re-submits next
	// cycle — as opposed to a deterministic failure (bad template/asset) that
	// fails the item. Guards invariant ②: a uid-less run must never be surfaced
	// as submitted, or the submitter (which only re-lists pending items) strands
	// it forever. CYB-3491 follow-up.
	ErrWorkflowSubmitIncomplete = errors.New("workflow submitted without an argo uid")
)

// Usecase orchestrates pipeline template management and deployment.
type Usecase struct {
	templateRepo    repository.PipelineTemplateRepository
	deploymentRepo  repository.PipelineDeploymentRepository
	targetRepo      repository.ExecutionTargetRepository
	runRepo         repository.PipelineRunRepository
	runNodeRepo     repository.PipelineRunNodeRepository
	runEventRepo    repository.PipelineRunEventRepository
	runRelationRepo repository.RunRelationRepository
	runInputRepo    repository.RunInputRepository
	assetNodeRepo   repository.PipelineRunAssetNodeRepository
	notifyRepo      repository.PipelineRunNotificationRepository
	watcherRepo     repository.PipelineRunWatcherStateRepository
	// watcherActiveCursor rotates the active-run refresh window across watcher
	// cycles so runs beyond the per-cycle cap don't starve (CYB-3490). Only the
	// single watcher goroutine touches it; resets on restart, which is fine —
	// coverage is eventual, the webhook remains the primary signal.
	watcherActiveCursor int
	// watcherAppliedRV gates repeat applies of unchanged workflows in the
	// bulk-pull path (CYB-3681): runID → last applied resourceVersion.
	// In-memory by design — restart means one full re-apply (recalibration).
	watcherRVMu      sync.Mutex
	watcherAppliedRV map[string]string
	// watcherLoadCursor paginates loadRunsForWatcherSync's active-run fetch
	// across scans so a total active set larger than watcherActiveRunLoadCap
	// still gets covered eventually (CYB-3746). The cursor is
	// (createdAt, id) DESC: each scan resumes strictly older than the last
	// row returned; wraps back to newest when the page returns short. The
	// recent-N union in loadRunsForWatcherSync still guarantees a fresh
	// batch is refreshed every tick, so pagination only affects the older
	// rotating tail — no live-batch starvation. In-memory (restart resets
	// to newest, same eventual-coverage semantic as watcherAppliedRV).
	watcherLoadCursorMu        sync.Mutex
	watcherLoadCursorCreatedAt time.Time
	watcherLoadCursorID        string
	// targetClusterCache memoizes execution_target_id → cluster_id so the
	// watcher / status-refresh paths can resolve a run's cluster without an
	// ExecutionTarget object populated on the run (ListSummaries doesn't load
	// the nested target). CYB-3486 4d.5 fix: without this, non-default-cluster
	// runs resolved to cluster-default and their status read back from the
	// wrong argo endpoint. Bindings are effectively immutable (a target's
	// cluster_id doesn't change in practice); a process restart clears it.
	targetClusterCache sync.Map
	backfillRepo       repository.BackfillRepository
	assetRepo          repository.AssetRepository
	assetEventRepo     repository.AssetEventRepository
	relationWriter     repository.AssetRelationWriter
	logicalRepo        repository.LogicalAssetRepository
	pipelineConfigRepo repository.PipelineConfigRepository
	runtimeConfigStore RuntimeConfigStore
	// CYB-3486: per-cluster runtime-config store factory. When set, Deploy
	// resolves the store via ForTarget(target) so the runtime-config ConfigMap
	// is created on the TARGET's cluster, not the default-cluster singleton.
	// Nil is tolerated — falls back to runtimeConfigStore (default cluster).
	runtimeConfigStoreFactory RuntimeConfigStoreFactory

	// runtimeConfigBlobs persists content-addressed config bytes as the
	// rebuildable source of truth (CYB-3680). Nil disables blob persistence;
	// blob write failures never block dispatch (the CM itself was ensured).
	runtimeConfigBlobs RuntimeConfigBlobStore
	wfClient           argo.WorkflowClient
	// CYB-3486 PR 4c: per-cluster Argo client factory. When set, Deploy
	// resolves argo client via argoFactory.ForTarget(target) instead of the
	// wfClient singleton so multi-cluster submit routing works. Nil is
	// tolerated — falls through to the existing adapter/singleton path.
	argoFactory             argo.ClientFactory
	runtimeAdapter          runtimeadapter.RuntimeAdapter
	namespace               string
	pricing                 *PricingConfig
	resourceGuard           ResourceGuardConfig
	runtimeMountCatalog     RuntimeMountCatalog
	now                     func() time.Time
	workflowTTLSecondsAfter int32

	// Argo run status push webhook (CYB-3058). When argoRunWebhookURL is empty,
	// no exit hook is injected into transpiled workflows (poll-only fallback).
	argoRunWebhookURL             string
	argoRunWebhookTokenSecretName string
	argoRunWebhookTokenSecretKey  string
	argoRunWebhookImage           string

	// videoDurations looks up source-video durations by asset_id (CYB-3059).
	videoDurations videoDurationLookup

	// nodeResolver resolves a node's real machine type for accurate per-pod cost
	// pricing (CYB-3073); nil falls back to the default GPU-pool rate.
	nodeResolver nodeInstanceResolver

	// batchCancels holds cancel funcs for in-flight batch submission
	// goroutines so StopBatchRuns can halt further run creation. Legacy
	// dispatch mode only — in submitter mode cancellation is DB-state driven
	// (job status 'cancelled' is never selected by the submitter).
	batchCancelMu sync.Mutex
	batchCancels  map[string]context.CancelFunc

	// batchDispatchLegacy selects the pre-CYB-3677 in-memory dispatch
	// goroutine instead of the durable backfill submitter. Rollback-only;
	// removed one release after G4 (see CYB-3677).
	batchDispatchLegacy bool
	// batchSubmitKick pokes the backfill submitter after a batch lands so
	// dispatch starts immediately instead of on the next 15s tick. Nil is
	// fine — the ticker picks the job up regardless (durability never
	// depends on the kick).
	batchSubmitKick func()

	watcherLedgerMu     sync.RWMutex
	watcherLedgerHealth models.LedgerHealth
}

// SetBatchDispatchMode selects the batch dispatch path (CYB-3677).
// "legacy" restores the in-memory goroutine; anything else (default
// "submitter") persists the job as 'running' and lets the durable backfill
// submitter own submission.
func (uc *Usecase) SetBatchDispatchMode(mode string) {
	uc.batchDispatchLegacy = strings.EqualFold(strings.TrimSpace(mode), "legacy")
}

// SetBatchSubmitKicker wires the backfill submitter's kick so newly created
// batches dispatch without waiting for the next ticker cycle.
func (uc *Usecase) SetBatchSubmitKicker(kick func()) {
	uc.batchSubmitKick = kick
}

type DeployOptions struct {
	DryRun             bool
	TemplateID         string
	TemplateVersion    int
	TargetID           string
	Owner              string
	BatchJobID         string
	PreallocatedRunID  string
	AllowUnknownAssets bool
	ConfigSelection    *RuntimeConfigSelection
}

type RuntimeConfigSelection struct {
	Mode           string
	ConfigID       string
	Version        int
	FileName       string
	Content        string
	MountPath      string
	TargetFilename string
}

type resolvedRuntimeConfig struct {
	Mode           string
	ConfigID       string
	Version        int
	FileName       string
	Content        string
	MountPath      string
	TargetFilename string
	VolumeName     string
	ProjectionKey  string
}

type RuntimeConfigProjection struct {
	FileName   string
	Content    string
	Files      map[string]string
	VolumeName string
	// ContentHash is the canonical sha256 (hex) of the projection's files.
	// Non-empty marks the projection content-addressed (CYB-3680): the
	// ConfigMap name derives from content, is shared across runs, is created
	// BEFORE the Workflow, carries no owner, and is reclaimed by the
	// sliding-reference TTL janitor.
	ContentHash string
}

// runtimeConfigContentHash canonically hashes a projection: the single-file
// form and the multi-file form that carry identical bytes hash identically,
// and map iteration order never leaks into the digest.
func runtimeConfigProjectionHash(p RuntimeConfigProjection) string {
	files := make(map[string]string, len(p.Files)+1)
	for k, v := range p.Files {
		files[k] = v
	}
	if name := strings.TrimSpace(p.FileName); name != "" {
		files[name] = p.Content
	}
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		// Length-prefixed fields prevent boundary ambiguity between
		// name/content pairs.
		fmt.Fprintf(h, "%d:%s%d:%s", len(k), k, len(files[k]), files[k])
	}
	return hex.EncodeToString(h.Sum(nil))
}

// runtimeConfigVolumeNameForContent derives the shared, content-addressed
// ConfigMap/volume name. 32 hex chars = 128 bits — collision-safe at any
// realistic distinct-config count, and "runtime-config-" + 32 stays well
// under the 63-char DNS label limit.
func runtimeConfigVolumeNameForContent(hash string) string {
	if len(hash) > 32 {
		hash = hash[:32]
	}
	return "runtime-config-" + hash
}

// RuntimeConfigBlobStore persists the content-addressed config bytes in the
// database as the source of truth (CYB-3680): the in-cluster ConfigMap is a
// disposable projection that self-heal layers can rebuild from the blob.
type RuntimeConfigBlobStore interface {
	Upsert(ctx context.Context, hash string, files map[string]string) error
}

type RuntimeConfigOwnerReference struct {
	APIVersion string
	Kind       string
	Name       string
	UID        string
}

type RuntimeConfigStore interface {
	Create(ctx context.Context, namespace, deploymentID string, config RuntimeConfigProjection, owner *RuntimeConfigOwnerReference) (string, error)
}

// RuntimeConfigStoreFactory resolves a RuntimeConfigStore bound to the target's
// own cluster. The default-cluster singleton (runtimeConfigStore) writes the
// runtime-config ConfigMap through one clientset; for a run dispatched to a
// non-default cluster that lands on the wrong cluster (the target namespace
// doesn't exist there → "namespaces ... not found"). This mirrors the
// argo.ClientFactory seam (CYB-3486).
type RuntimeConfigStoreFactory interface {
	ForTarget(ctx context.Context, target *models.ExecutionTarget) (RuntimeConfigStore, error)
}

type resolvedNodeRuntimeConfig struct {
	NodeID string
	Config *resolvedRuntimeConfig
}

const runConfigInputsPipelineJSONKey = "_run_config_inputs"

// mergePoolPodLabels overlays the pool's scheduling podLabels (data-driven,
// from resource_defaults.scheduling) onto a base label set (e.g. cost-tracking
// labels). Pool labels win on conflict. CYB-3486 pool.3: no hardcoded label
// key — the pool config supplies whatever labels it wants (a Koordinator pool
// carries its own EQ label key/value).
func mergePoolPodLabels(base map[string]string, target *models.ExecutionTarget) map[string]string {
	poolLabels := executionTargetPodLabels(target)
	if len(poolLabels) == 0 {
		return base
	}
	if base == nil {
		base = map[string]string{}
	}
	for k, v := range poolLabels {
		base[k] = v
	}
	return base
}

// SetAssetEventRepo sets the asset event repository (optional, for F4.3+).
func (uc *Usecase) SetAssetEventRepo(r repository.AssetEventRepository) {
	uc.assetEventRepo = r
}

// SetRelationWriter sets the asset relation writer (optional, for F4.7).
func (uc *Usecase) SetRelationWriter(r repository.AssetRelationWriter) {
	uc.relationWriter = r
}

// SetLogicalAssetRepo wires logical_assets persistence (CYB-1013 pipeline outputs).
func (uc *Usecase) SetLogicalAssetRepo(r repository.LogicalAssetRepository) {
	uc.logicalRepo = r
}

// SetPipelineConfigRepo wires standalone config persistence for deploy-time
// resolution of saved config references.
func (uc *Usecase) SetPipelineConfigRepo(r repository.PipelineConfigRepository) {
	uc.pipelineConfigRepo = r
}

func (uc *Usecase) SetRuntimeConfigStore(store RuntimeConfigStore) {
	uc.runtimeConfigStore = store
}

// SetRuntimeConfigStoreFactory wires the per-cluster runtime-config store
// factory (CYB-3486). When set, Deploy creates the runtime-config ConfigMap on
// the target's own cluster instead of the default-cluster singleton.
// SetRuntimeConfigBlobStore wires the DB-backed blob source of truth for
// content-addressed runtime configs (CYB-3680).
func (uc *Usecase) SetRuntimeConfigBlobStore(s RuntimeConfigBlobStore) {
	uc.runtimeConfigBlobs = s
}

// persistRuntimeConfigBlob best-effort persists the projection's bytes keyed
// by content hash. Dispatch never blocks on it: the ConfigMap was already
// ensured; the blob only powers rebuild/self-heal.
func (uc *Usecase) persistRuntimeConfigBlob(ctx context.Context, p *RuntimeConfigProjection) {
	if uc.runtimeConfigBlobs == nil || p == nil || p.ContentHash == "" {
		return
	}
	files := make(map[string]string, len(p.Files)+1)
	for k, v := range p.Files {
		files[k] = v
	}
	if name := strings.TrimSpace(p.FileName); name != "" {
		files[name] = p.Content
	}
	if err := uc.runtimeConfigBlobs.Upsert(ctx, p.ContentHash, files); err != nil {
		slog.Warn("runtime config blob upsert failed (self-heal source degraded)",
			"hash", p.ContentHash, "err", err)
	}
}

// ResolveTargetClusterID maps a target id to its cluster id — the sharding
// key for per-cluster dispatch channels (CYB-3678). Unresolvable targets and
// empty cluster ids collapse to "default".
func (uc *Usecase) ResolveTargetClusterID(ctx context.Context, targetID string) string {
	target, err := uc.resolveExecutionTarget(ctx, targetID)
	if err != nil || target == nil || strings.TrimSpace(target.ClusterID) == "" {
		return "default"
	}
	return strings.TrimSpace(target.ClusterID)
}

// ListBatchDLQ returns a batch's dead-lettered items (status=failed) with
// their recorded failure reasons (CYB-3678).
func (uc *Usecase) ListBatchDLQ(ctx context.Context, jobID string) ([]models.BackfillItem, error) {
	if uc.backfillRepo == nil {
		return nil, fmt.Errorf("%w: backfill repository is not configured", ErrInvalidArgument)
	}
	items, err := uc.backfillRepo.FindItemsByJobID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	failed := make([]models.BackfillItem, 0, 8)
	for _, it := range items {
		if it.Status == "failed" {
			failed = append(failed, it)
		}
	}
	return failed, nil
}

// RetryBatchDLQ re-queues a batch's dead-lettered items: failed → pending
// with a fresh attempt cap, job back to 'running' so the submitter selects
// it, and an immediate kick (CYB-3678).
func (uc *Usecase) RetryBatchDLQ(ctx context.Context, jobID string) (int64, error) {
	if uc.backfillRepo == nil {
		return 0, fmt.Errorf("%w: backfill repository is not configured", ErrInvalidArgument)
	}
	n, err := uc.backfillRepo.ResetFailedItems(ctx, jobID)
	if err != nil {
		return 0, err
	}
	if n > 0 {
		if err := uc.backfillRepo.UpdateJobStatus(ctx, jobID, "running"); err != nil {
			return n, fmt.Errorf("dlq retry: job status: %w", err)
		}
		if uc.batchSubmitKick != nil {
			uc.batchSubmitKick()
		}
	}
	return n, nil
}

func (uc *Usecase) SetRuntimeConfigStoreFactory(f RuntimeConfigStoreFactory) {
	uc.runtimeConfigStoreFactory = f
}

// resolveRuntimeConfigStore returns the store to use for a Deploy: the
// per-target store when a factory is wired, otherwise the default-cluster
// singleton. Keeping the singleton fallback preserves the no-PG/test path.
func (uc *Usecase) resolveRuntimeConfigStore(
	ctx context.Context,
	target *models.ExecutionTarget,
) (RuntimeConfigStore, error) {
	if uc.runtimeConfigStoreFactory != nil {
		return uc.runtimeConfigStoreFactory.ForTarget(ctx, target)
	}
	return uc.runtimeConfigStore, nil
}

// SetRuntimeAdapter wires the Run Kernel runtime boundary. When unset, legacy
// workflow-client operations remain available for compatibility and tests.
func (uc *Usecase) SetRuntimeAdapter(adapter runtimeadapter.RuntimeAdapter) {
	uc.runtimeAdapter = adapter
}

// SetArgoFactory wires the per-cluster Argo client factory (CYB-3486 PR 4c).
// When set, submit paths route by ExecutionTarget.ClusterID via
// factory.ForTarget instead of the wfClient singleton.
func (uc *Usecase) SetArgoFactory(f argo.ClientFactory) {
	uc.argoFactory = f
}

// SetRunRepositories wires first-class pipeline run persistence. The legacy
// deployment repo remains required for compatibility endpoints.
func (uc *Usecase) SetRunRepositories(
	targetRepo repository.ExecutionTargetRepository,
	runRepo repository.PipelineRunRepository,
	runNodeRepo repository.PipelineRunNodeRepository,
) {
	uc.targetRepo = targetRepo
	uc.runRepo = runRepo
	uc.runNodeRepo = runNodeRepo
}

// SetRunEventRepo wires durable pipeline run event persistence.
func (uc *Usecase) SetRunEventRepo(r repository.PipelineRunEventRepository) {
	uc.runEventRepo = r
}

// SetRunFactRepositories wires durable Run Kernel facts for lineage and inputs.
func (uc *Usecase) SetRunFactRepositories(
	relationRepo repository.RunRelationRepository,
	inputRepo repository.RunInputRepository,
) {
	uc.runRelationRepo = relationRepo
	uc.runInputRepo = inputRepo
}

func (uc *Usecase) SetBackfillRepo(r repository.BackfillRepository) {
	uc.backfillRepo = r
}

// SetObservabilityRepositories wires optional pipeline observability
// persistence for asset-node snapshots, notification candidates, and watcher state.
func (uc *Usecase) SetObservabilityRepositories(
	assetNodeRepo repository.PipelineRunAssetNodeRepository,
	notifyRepo repository.PipelineRunNotificationRepository,
	watcherRepo repository.PipelineRunWatcherStateRepository,
) {
	uc.assetNodeRepo = assetNodeRepo
	uc.notifyRepo = notifyRepo
	uc.watcherRepo = watcherRepo
}

// SetPricing wires the GCP pricing config for cost estimation.
func (uc *Usecase) SetPricing(p *PricingConfig) {
	uc.pricing = p
}

// SetArgoWorkflowTTLSecondsAfterCompletion configures Argo workflow CR TTL
// after completion. Zero keeps transpiler.DefaultTTLSecondsAfterCompletion.
func (uc *Usecase) SetArgoWorkflowTTLSecondsAfterCompletion(seconds int32) {
	uc.workflowTTLSecondsAfter = seconds
}

func (uc *Usecase) argoWorkflowTTLSecondsAfter() int32 {
	if uc.workflowTTLSecondsAfter > 0 {
		return uc.workflowTTLSecondsAfter
	}
	return transpiler.DefaultTTLSecondsAfterCompletion
}

// SetArgoRunWebhook configures the exit-hook that pushes run status to DataBrew.
// url empty disables hook injection (poll-only). secretName/secretKey reference
// the K8s Secret (in the workflow namespace) holding the webhook auth token.
func (uc *Usecase) SetArgoRunWebhook(url, secretName, secretKey, image string) {
	uc.argoRunWebhookURL = strings.TrimSpace(url)
	uc.argoRunWebhookTokenSecretName = strings.TrimSpace(secretName)
	uc.argoRunWebhookTokenSecretKey = strings.TrimSpace(secretKey)
	uc.argoRunWebhookImage = strings.TrimSpace(image)
}

// videoDurationLookup returns source-video durations (seconds) by video_id.
type videoDurationLookup interface {
	GetByVideoIDs(ctx context.Context, videoIDs []string) (map[string]float64, error)
}

// SetVideoDurationRepo wires the video_durations lookup used to enrich the
// batch subtask runs list with the source video's duration (CYB-3059).
func (uc *Usecase) SetVideoDurationRepo(r videoDurationLookup) {
	uc.videoDurations = r
}

// nodeInstanceResolver resolves a node name to its machine type / accelerator /
// provisioning for cost pricing (CYB-3073).
type nodeInstanceResolver interface {
	ResolveNodeInstance(ctx context.Context, nodeName string) (instanceType, accelerator, provisioning string)
}

// SetNodeInstanceResolver wires per-node instance-type resolution so pipeline
// step costs are priced by the node's real machine type instead of the default
// GPU-pool rate. Nil keeps the previous default behavior.
func (uc *Usecase) SetNodeInstanceResolver(r nodeInstanceResolver) {
	uc.nodeResolver = r
}

// enrichResourcesDurationWithNode annotates a node's resourcesDuration with the
// real instance_type / gpu_type / provisioning (looked up by host node) so
// resourcesDurationToCost prices it correctly. Best-effort: an unresolved node
// is left unchanged and keeps the default pricing.
func (uc *Usecase) enrichResourcesDurationWithNode(ctx context.Context, node *models.PipelineRunNode) {
	if uc.nodeResolver == nil || node == nil || strings.TrimSpace(node.HostNodeName) == "" {
		return
	}
	it, acc, prov := uc.nodeResolver.ResolveNodeInstance(ctx, node.HostNodeName)
	if it == "" {
		return
	}
	rd := node.ResourcesDuration
	if rd == nil {
		rd = map[string]interface{}{}
	}
	rd["instance_type"] = it
	if acc != "" {
		rd["gpu_type"] = acc
	} else {
		rd["gpu_type"] = "none"
	}
	if prov != "" {
		rd["provisioning"] = prov
	}
	node.ResourcesDuration = rd
}

// attachVideoDurations enriches each run with its source video's duration,
// looked up by asset_id. Best-effort: on error or missing rows the field stays
// nil (some videos simply have no recorded duration).
func (uc *Usecase) attachVideoDurations(ctx context.Context, items []models.PipelineRun) {
	if uc.videoDurations == nil || len(items) == 0 {
		return
	}
	seen := make(map[string]struct{})
	for i := range items {
		for _, a := range items[i].AssetIDs {
			if a != "" {
				seen[a] = struct{}{}
			}
		}
	}
	if len(seen) == 0 {
		return
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	durs, err := uc.videoDurations.GetByVideoIDs(ctx, ids)
	if err != nil {
		slog.Warn("attachVideoDurations failed", "err", err)
		return
	}
	if len(durs) == 0 {
		return
	}
	for i := range items {
		for _, a := range items[i].AssetIDs {
			if d, ok := durs[a]; ok {
				dd := d
				items[i].VideoDurationSec = &dd
				break
			}
		}
	}
}

func defaultExecutionTargetServiceAccount() string {
	return strings.TrimSpace(os.Getenv("PIPELINE_DEFAULT_SERVICE_ACCOUNT"))
}

func logPipelineSideEffect(op string, err error) {
	if err != nil {
		slog.Warn("pipeline side effect failed", "op", op, "err", err)
	}
}

// resolveArgoClient returns the argo client for the target's cluster.
// CYB-3486 PR 4c.1: factory-first, singleton fallback, no adapter awareness.
//   - argoFactory wired → factory.ForTarget(target) — the cluster-routed path
//   - factory nil → uc.wfClient (may itself be nil — that's fine, caller must
//     handle nil for the adapter-only test path where SetRuntimeAdapter was
//     used without a wfClient)
//
// A nil return value means "no direct client available; caller should fall
// through to the runtime adapter or fail". A non-nil error means "resolution
// was attempted (factory) and failed" (misconfigured cluster row, etc.).
func (uc *Usecase) resolveArgoClient(ctx context.Context, target *models.ExecutionTarget) (argo.WorkflowClient, error) {
	if uc.argoFactory != nil {
		c, err := uc.argoFactory.ForTarget(ctx, target)
		if err != nil {
			return nil, fmt.Errorf("%w: resolve argo client for cluster %q: %v",
				ErrWorkflowUnavailable, targetClusterID(target), err)
		}
		return c, nil
	}
	return uc.wfClient, nil
}

// resolveArgoClientForRun is the run-facing sibling of resolveArgoClient:
// the run watcher and per-run lifecycle ops (Stop/Retry/Suspend/Resume/
// Terminate/Delete) already carry the resolved cluster on the run row, so
// they don't need to re-derive it from an ExecutionTarget. Factory-first,
// singleton fallback, same semantics as resolveArgoClient — CYB-3486 PR 4d.5.
//
// This is the correctness half of "per-cluster watcher": before this
// refactor, active runs on delivery-clust were being polled through the
// cyber-clust singleton (they simply didn't exist there, so the watcher
// silently no-op'd). Goroutine-level parallelism per cluster (the scaling
// half) is deferred until we have >1 non-default cluster with meaningful
// active-run volume.
func (uc *Usecase) resolveArgoClientForRun(ctx context.Context, run *models.PipelineRun) (argo.WorkflowClient, error) {
	if uc.argoFactory != nil {
		id := uc.resolveRunClusterID(ctx, run)
		c, err := uc.argoFactory.ForCluster(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("%w: resolve argo client for run cluster %q: %v",
				ErrWorkflowUnavailable, id, err)
		}
		return c, nil
	}
	return uc.wfClient, nil
}

// resolveRunClusterID determines which cluster a run's workflow lives on.
//
// PipelineRun has no direct cluster_id column; the truth is
// execution_targets.cluster_id, reachable two ways:
//  1. run.ExecutionTarget populated (GetRun/Deploy paths) → read it directly.
//  2. only run.ExecutionTargetID present (watcher's ListSummaries, the status
//     webhook, reconcile) → look the target up by id (memoized).
//
// Case 2 is the CYB-3486 4d.5 fix: resolveArgoClientForRun used to depend on
// callers pre-populating run.ExecutionTarget, but the watcher/webhook paths
// don't, so every non-default-cluster run fell back to cluster-default and
// its status was read from cyber-clust's argo-server (→ "workflow not found",
// run stuck Running). Resolving via ExecutionTargetID here removes that
// dependency on the caller.
func (uc *Usecase) resolveRunClusterID(ctx context.Context, run *models.PipelineRun) string {
	const fallback = "cluster-default"
	if run == nil {
		return fallback
	}
	if run.ExecutionTarget != nil {
		if id := strings.TrimSpace(run.ExecutionTarget.ClusterID); id != "" {
			return id
		}
	}
	tid := strings.TrimSpace(run.ExecutionTargetID)
	if tid == "" || uc.targetRepo == nil {
		return fallback
	}
	if cached, ok := uc.targetClusterCache.Load(tid); ok {
		if cid, _ := cached.(string); cid != "" {
			return cid
		}
	}
	t, err := uc.targetRepo.FindByID(ctx, tid)
	if err != nil || t == nil {
		// Don't cache misses — a transient DB error shouldn't pin this run to
		// cluster-default for the process lifetime.
		return fallback
	}
	cid := strings.TrimSpace(t.ClusterID)
	if cid == "" {
		cid = fallback
	}
	uc.targetClusterCache.Store(tid, cid)
	return cid
}

// groupRunIndicesByCluster partitions run indices by cluster id using the
// provided resolver. Used by SyncActiveRunEvents to fan out per-cluster
// goroutines (CYB-3486 4d.5.b). Takes clusterOf as a param so it stays a pure,
// unit-testable function while still honoring the ExecutionTargetID lookup
// that resolveRunClusterID does.
func groupRunIndicesByCluster(runs []models.PipelineRun, indices []int, clusterOf func(*models.PipelineRun) string) map[string][]int {
	out := map[string][]int{}
	for _, i := range indices {
		cid := clusterOf(&runs[i])
		out[cid] = append(out[cid], i)
	}
	return out
}

func (uc *Usecase) getWorkflowWithUID(ctx context.Context, client argo.WorkflowClient, name, namespace string) (*wfv1.Workflow, error) {
	if client == nil {
		return nil, ErrWorkflowUnavailable
	}
	var lastErr error
	for attempt := 0; attempt < 6; attempt++ {
		wf, err := client.GetWorkflow(ctx, name, namespace)
		if err == nil && wf != nil && wf.UID != "" {
			return wf, nil
		}
		if err != nil {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("workflow %s/%s has no UID", namespace, name)
}

func (uc *Usecase) resolveRuntimeConfig(
	ctx context.Context,
	selection *RuntimeConfigSelection,
) (*resolvedRuntimeConfig, error) {
	if selection == nil {
		return nil, nil
	}
	mode := strings.TrimSpace(selection.Mode)
	mountPath := strings.TrimSpace(selection.MountPath)
	targetFilename := strings.TrimSpace(selection.TargetFilename)
	if mode == "" {
		return nil, fmt.Errorf("%w: config mode is required", ErrInvalidArgument)
	}
	if mountPath == "" || targetFilename == "" {
		return nil, fmt.Errorf("%w: config mountPath and targetFilename are required", ErrInvalidArgument)
	}
	switch mode {
	case "saved":
		if uc.pipelineConfigRepo == nil {
			return nil, fmt.Errorf("%w: pipeline config repository is not configured", ErrInvalidArgument)
		}
		configID := strings.TrimSpace(selection.ConfigID)
		if configID == "" || selection.Version <= 0 {
			return nil, fmt.Errorf("%w: saved config requires configId and version", ErrInvalidArgument)
		}
		version, err := uc.pipelineConfigRepo.FindVersion(ctx, configID, selection.Version)
		if err != nil {
			if errors.Is(err, repository.ErrPipelineConfigNotFound) {
				return nil, fmt.Errorf("%w: saved config version not found", ErrInvalidArgument)
			}
			return nil, err
		}
		if version == nil {
			return nil, fmt.Errorf("%w: saved config version not found", ErrInvalidArgument)
		}
		fileName := strings.TrimSpace(selection.FileName)
		if fileName == "" {
			cfg, err := uc.pipelineConfigRepo.FindByID(ctx, configID)
			if err == nil && cfg != nil {
				fileName = cfg.Name
			}
		}
		if fileName == "" {
			fileName = targetFilename
		}
		return &resolvedRuntimeConfig{
			Mode:           mode,
			ConfigID:       configID,
			Version:        selection.Version,
			FileName:       fileName,
			Content:        version.Content,
			MountPath:      mountPath,
			TargetFilename: targetFilename,
		}, nil
	case "upload", "inline":
		content := selection.Content
		if strings.TrimSpace(content) == "" {
			return nil, fmt.Errorf("%w: config content is required", ErrInvalidArgument)
		}
		if len(content) > configUC.MaxConfigFileBytes {
			return nil, fmt.Errorf("%w: config content exceeds %d bytes", ErrInvalidArgument, configUC.MaxConfigFileBytes)
		}
		fileName := strings.TrimSpace(selection.FileName)
		if fileName == "" {
			fileName = targetFilename
		}
		return &resolvedRuntimeConfig{
			Mode:           mode,
			FileName:       fileName,
			Content:        content,
			MountPath:      mountPath,
			TargetFilename: targetFilename,
		}, nil
	default:
		return nil, fmt.Errorf("%w: unsupported config mode %q", ErrInvalidArgument, mode)
	}
}

const defaultRuntimeConfigMountPath = "/workspace/configs"

var invalidRuntimeConfigProjectionKeyChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)
var invalidRuntimeConfigVolumeNameChars = regexp.MustCompile(`[^a-z0-9-]+`)
var compatibleExternalVideoIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func (uc *Usecase) resolveNodeRuntimeConfig(
	ctx context.Context,
	nodeID string,
	binding *transpiler.RuntimeConfigBinding,
) (*resolvedRuntimeConfig, error) {
	if binding == nil {
		return nil, nil
	}
	mode := strings.TrimSpace(binding.Mode)
	if mode == "" {
		mode = "saved"
	}
	if mode != "saved" {
		return nil, fmt.Errorf("%w: node %s runtime config only supports saved mode", ErrInvalidArgument, nodeID)
	}
	if uc.pipelineConfigRepo == nil {
		return nil, fmt.Errorf("%w: node %s runtime config repository is not configured", ErrInvalidArgument, nodeID)
	}
	configID := strings.TrimSpace(binding.ConfigID)
	if configID == "" || binding.Version <= 0 {
		return nil, fmt.Errorf("%w: node %s saved config requires configId and version", ErrInvalidArgument, nodeID)
	}
	cfg, err := uc.pipelineConfigRepo.FindByID(ctx, configID)
	if err != nil {
		if errors.Is(err, repository.ErrPipelineConfigNotFound) {
			return nil, fmt.Errorf("%w: node %s saved config not found", ErrInvalidArgument, nodeID)
		}
		return nil, err
	}
	if cfg == nil {
		return nil, fmt.Errorf("%w: node %s saved config not found", ErrInvalidArgument, nodeID)
	}
	if cfg.Lifecycle != "ready" {
		return nil, fmt.Errorf("%w: node %s saved config %s is not ready", ErrInvalidArgument, nodeID, configID)
	}
	version, err := uc.pipelineConfigRepo.FindVersion(ctx, configID, binding.Version)
	if err != nil {
		if errors.Is(err, repository.ErrPipelineConfigNotFound) {
			return nil, fmt.Errorf("%w: node %s saved config version not found", ErrInvalidArgument, nodeID)
		}
		return nil, err
	}
	if version == nil {
		return nil, fmt.Errorf("%w: node %s saved config version not found", ErrInvalidArgument, nodeID)
	}
	if version.Status != "" && version.Status != "ready" {
		return nil, fmt.Errorf("%w: node %s saved config version %d is not ready", ErrInvalidArgument, nodeID, binding.Version)
	}
	fileName := strings.TrimSpace(binding.FileName)
	if fileName == "" {
		fileName = strings.TrimSpace(cfg.Name)
	}
	mountPath := strings.TrimSpace(binding.MountPath)
	if mountPath == "" {
		mountPath = defaultRuntimeConfigMountPath
	}
	targetFilename := strings.TrimSpace(binding.TargetFilename)
	if targetFilename == "" {
		targetFilename = fileName
	}
	if targetFilename == "" {
		targetFilename = "runtime-config.yaml"
	}
	if fileName == "" {
		fileName = targetFilename
	}
	return &resolvedRuntimeConfig{
		Mode:           "saved",
		ConfigID:       configID,
		Version:        binding.Version,
		FileName:       fileName,
		Content:        version.Content,
		MountPath:      mountPath,
		TargetFilename: targetFilename,
	}, nil
}

func (uc *Usecase) resolveNodeRuntimeConfigs(ctx context.Context, pipe *transpiler.Pipeline) ([]resolvedNodeRuntimeConfig, error) {
	if pipe == nil {
		return nil, nil
	}
	var out []resolvedNodeRuntimeConfig
	var walk func(nodes []transpiler.Node) error
	walk = func(nodes []transpiler.Node) error {
		for i := range nodes {
			node := nodes[i]
			if node.RuntimeConfig != nil {
				config, err := uc.resolveNodeRuntimeConfig(ctx, node.ID, node.RuntimeConfig)
				if err != nil {
					return err
				}
				if config != nil {
					out = append(out, resolvedNodeRuntimeConfig{NodeID: node.ID, Config: config})
				}
			}
			if len(node.SubNodes) > 0 {
				if err := walk(node.SubNodes); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(pipe.Nodes); err != nil {
		return nil, err
	}
	return out, nil
}

func runtimeConfigEnvVars(config *resolvedRuntimeConfig) []transpiler.EnvVar {
	if config == nil {
		return nil
	}
	env := []transpiler.EnvVar{
		{Name: "PIPELINE_CONFIG_PATH", Value: path.Join(config.MountPath, config.TargetFilename)},
		{Name: "PIPELINE_CONFIG_FILENAME", Value: config.TargetFilename},
		{Name: "PIPELINE_CONFIG_SOURCE", Value: config.Mode},
	}
	if config.ConfigID != "" {
		env = append(env, transpiler.EnvVar{Name: "PIPELINE_CONFIG_ID", Value: config.ConfigID})
	}
	if config.Version > 0 {
		env = append(env, transpiler.EnvVar{Name: "PIPELINE_CONFIG_VERSION", Value: fmt.Sprintf("%d", config.Version)})
	}
	return env
}

func runtimeConfigSubPath(config *resolvedRuntimeConfig) string {
	if config == nil {
		return ""
	}
	if strings.TrimSpace(config.ProjectionKey) != "" {
		return strings.TrimSpace(config.ProjectionKey)
	}
	return config.TargetFilename
}

func runtimeConfigVolumeMount(config *resolvedRuntimeConfig) transpiler.VolumeMount {
	return transpiler.VolumeMount{
		Name:      config.VolumeName,
		MountPath: path.Join(config.MountPath, config.TargetFilename),
		SubPath:   runtimeConfigSubPath(config),
		ReadOnly:  true,
	}
}

func appendRuntimeConfigToNode(node *transpiler.Node, config *resolvedRuntimeConfig) {
	if node == nil || config == nil {
		return
	}
	node.Component.Env = append(node.Component.Env, runtimeConfigEnvVars(config)...)
	node.VolumeMounts = append(node.VolumeMounts, runtimeConfigVolumeMount(config))
}

func runtimeConfigProjectionKey(nodeID, targetFilename string, index int) string {
	base := strings.Trim(strings.ToLower(strings.TrimSpace(nodeID)+"-"+strings.TrimSpace(targetFilename)), "-._")
	base = invalidRuntimeConfigProjectionKeyChars.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-._")
	if base == "" {
		base = fmt.Sprintf("runtime-config-%d", index)
	}
	if len(base) > 180 {
		base = strings.Trim(base[:180], "-._")
	}
	if base == "" {
		base = fmt.Sprintf("runtime-config-%d", index)
	}
	return fmt.Sprintf("%02d-%s", index, base)
}

func buildRuntimeConfigProjection(
	runtimeConfig *resolvedRuntimeConfig,
	nodeConfigs []resolvedNodeRuntimeConfig,
) RuntimeConfigProjection {
	if runtimeConfig != nil && len(nodeConfigs) == 0 {
		runtimeConfig.ProjectionKey = runtimeConfig.TargetFilename
		return RuntimeConfigProjection{
			FileName: runtimeConfig.TargetFilename,
			Content:  runtimeConfig.Content,
		}
	}
	files := make(map[string]string)
	index := 1
	if runtimeConfig != nil {
		runtimeConfig.ProjectionKey = runtimeConfigProjectionKey("global", runtimeConfig.TargetFilename, index)
		files[runtimeConfig.ProjectionKey] = runtimeConfig.Content
		index++
	}
	for i := range nodeConfigs {
		config := nodeConfigs[i].Config
		if config == nil {
			continue
		}
		config.ProjectionKey = runtimeConfigProjectionKey(nodeConfigs[i].NodeID, config.TargetFilename, index)
		files[config.ProjectionKey] = config.Content
		index++
	}
	return RuntimeConfigProjection{Files: files}
}

func runtimeConfigVolumeNameForDeployment(deploymentID string) string {
	value := strings.ToLower(strings.TrimSpace("runtime-config-" + strings.TrimSpace(deploymentID)))
	value = invalidRuntimeConfigVolumeNameChars.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "runtime-config"
	}
	if len(value) > 63 {
		value = strings.Trim(value[:63], "-")
	}
	if value == "" {
		return "runtime-config"
	}
	return value
}

func materializedRuntimeConfigRunInputs(
	runtimeConfig *resolvedRuntimeConfig,
	nodeConfigs []resolvedNodeRuntimeConfig,
) []interface{} {
	total := len(nodeConfigs)
	if runtimeConfig != nil {
		total++
	}
	if total == 0 {
		return nil
	}
	items := make([]interface{}, 0, total)
	if runtimeConfig != nil {
		items = append(items, runtimeConfigRunInputMetadata("global", "", runtimeConfig))
	}
	for _, nodeConfig := range nodeConfigs {
		if nodeConfig.Config == nil {
			continue
		}
		items = append(items, runtimeConfigRunInputMetadata("node", nodeConfig.NodeID, nodeConfig.Config))
	}
	return items
}

func runtimeConfigRunInputMetadata(scope, nodeID string, config *resolvedRuntimeConfig) map[string]interface{} {
	if config == nil {
		return nil
	}
	item := map[string]interface{}{
		"scope":          scope,
		"mode":           config.Mode,
		"fileName":       config.FileName,
		"mountPath":      config.MountPath,
		"targetFilename": config.TargetFilename,
		"contentHash":    runtimeConfigContentHash(config.Content),
	}
	if strings.TrimSpace(nodeID) != "" {
		item["nodeId"] = strings.TrimSpace(nodeID)
	}
	if strings.TrimSpace(config.ConfigID) != "" {
		item["configId"] = strings.TrimSpace(config.ConfigID)
	}
	if config.Version > 0 {
		item["version"] = config.Version
	}
	if strings.TrimSpace(config.ProjectionKey) != "" {
		item["projectionKey"] = strings.TrimSpace(config.ProjectionKey)
	}
	return item
}

func runtimeConfigContentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func boundNodeIDs(nodeConfigs []resolvedNodeRuntimeConfig) map[string]struct{} {
	out := make(map[string]struct{}, len(nodeConfigs))
	for _, item := range nodeConfigs {
		out[item.NodeID] = struct{}{}
	}
	return out
}

func applyNodeRuntimeConfigs(pipe *transpiler.Pipeline, nodeConfigs []resolvedNodeRuntimeConfig, volumeName string) {
	if pipe == nil || len(nodeConfigs) == 0 {
		return
	}
	byNode := make(map[string]*resolvedRuntimeConfig, len(nodeConfigs))
	for i := range nodeConfigs {
		config := nodeConfigs[i].Config
		if config == nil {
			continue
		}
		config.VolumeName = volumeName
		byNode[nodeConfigs[i].NodeID] = config
	}
	var walk func(nodes []transpiler.Node)
	walk = func(nodes []transpiler.Node) {
		for i := range nodes {
			if config := byNode[nodes[i].ID]; config != nil {
				appendRuntimeConfigToNode(&nodes[i], config)
			}
			if len(nodes[i].SubNodes) > 0 {
				walk(nodes[i].SubNodes)
			}
		}
	}
	walk(pipe.Nodes)
}

func applyRuntimeConfigToUnboundNodes(
	pipe *transpiler.Pipeline,
	config *resolvedRuntimeConfig,
	bound map[string]struct{},
) {
	if pipe == nil || config == nil {
		return
	}
	var walk func(nodes []transpiler.Node)
	walk = func(nodes []transpiler.Node) {
		for i := range nodes {
			if _, ok := bound[nodes[i].ID]; !ok {
				appendRuntimeConfigToNode(&nodes[i], config)
			}
			if len(nodes[i].SubNodes) > 0 {
				walk(nodes[i].SubNodes)
			}
		}
	}
	walk(pipe.Nodes)
}

func applyRuntimeConfigToPipeline(
	pipe *transpiler.Pipeline,
	config *resolvedRuntimeConfig,
	globalEnv *[]transpiler.EnvVar,
	extraVolumes *[]transpiler.Volume,
) {
	if pipe == nil || config == nil {
		return
	}
	*globalEnv = append(*globalEnv, runtimeConfigEnvVars(config)...)
	*extraVolumes = append(*extraVolumes, transpiler.Volume{
		Name:          config.VolumeName,
		ConfigMapName: config.VolumeName,
	})
	var applyNodeMounts func(nodes []transpiler.Node)
	applyNodeMounts = func(nodes []transpiler.Node) {
		for i := range nodes {
			nodes[i].VolumeMounts = append(nodes[i].VolumeMounts, runtimeConfigVolumeMount(config))
			if len(nodes[i].SubNodes) > 0 {
				applyNodeMounts(nodes[i].SubNodes)
			}
		}
	}
	applyNodeMounts(pipe.Nodes)
}

const (
	runEventSubmitted             = "run_submitted"
	runEventScheduled             = "run_scheduled"
	runEventWorkflowCreated       = "workflow_created"
	runEventWorkflowObserved      = "workflow_observed"
	runEventWorkflowPhaseChanged  = "workflow_phase_changed"
	runEventNodeStarted           = "node_started"
	runEventNodeSucceeded         = "node_succeeded"
	runEventNodeFailed            = "node_failed"
	runEventNodeError             = "node_error"
	runEventPodCreated            = "pod_created"
	runEventPodPhaseChanged       = "pod_phase_changed"
	runEventCompleted             = "run_completed"
	runEventFailed                = "run_failed"
	runEventRetryRequested        = "run_retry_requested"
	runEventRetryFailed           = "run_retry_failed"
	runEventRuntimeRetryRequested = "run_runtime_retry_requested"
	runEventRuntimeRetryFailed    = "run_runtime_retry_failed"
	runEventRuntimeRetrySucceeded = "run_runtime_retry_succeeded"
	runEventResubmitRequested     = "run_resubmit_requested"
	runEventResubmitFailed        = "run_resubmit_failed"
	runEventResubmitted           = "run_resubmitted"
	runEventRerunRequested        = "run_rerun_requested"
	runEventRerunFailed           = "run_rerun_failed"
	runEventRerunCreated          = "run_rerun_created"
	runEventStopRequested         = "run_stop_requested"
	runEventStopFailed            = "run_stop_failed"
	runEventStopSucceeded         = "run_stopped"
	runEventSuspendRequested      = "run_suspend_requested"
	runEventSuspendFailed         = "run_suspend_failed"
	runEventSuspendSucceeded      = "run_suspended"
	runEventResumeRequested       = "run_resume_requested"
	runEventResumeFailed          = "run_resume_failed"
	runEventResumeSucceeded       = "run_resumed"
	runEventTerminateRequested    = "run_terminate_requested"
	runEventTerminateFailed       = "run_terminate_failed"
	runEventTerminateSucceeded    = "run_terminated"
	runEventDeleteRequested       = "run_delete_requested"
	runEventDeleted               = "run_deleted"
	runEventDeleteFailed          = "run_delete_failed"
)

// New creates a Usecase.
func New(
	templateRepo repository.PipelineTemplateRepository,
	deploymentRepo repository.PipelineDeploymentRepository,
	assetRepo repository.AssetRepository,
	wfClient argo.WorkflowClient,
	namespace string,
) *Usecase {
	return &Usecase{
		templateRepo:   templateRepo,
		deploymentRepo: deploymentRepo,
		assetRepo:      assetRepo,
		wfClient:       wfClient,
		namespace:      namespace,
	}
}

// ListExecutionTargets returns currently available runtime destinations.
func (uc *Usecase) ListExecutionTargets(ctx context.Context) ([]models.ExecutionTarget, error) {
	if uc.targetRepo == nil {
		return []models.ExecutionTarget{uc.defaultExecutionTarget()}, nil
	}
	if err := uc.ensureDefaultExecutionTarget(ctx); err != nil {
		return nil, err
	}
	targets, err := uc.targetRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	for i := range targets {
		uc.normalizeExecutionTarget(&targets[i])
	}
	return targets, nil
}

func (uc *Usecase) CreateExecutionTarget(ctx context.Context, t *models.ExecutionTarget) error {
	return uc.targetRepo.Save(ctx, t)
}

func (uc *Usecase) UpdateExecutionTarget(ctx context.Context, t *models.ExecutionTarget) error {
	_, err := uc.targetRepo.FindByID(ctx, t.ID)
	if err != nil {
		return err
	}
	if err := uc.targetRepo.Save(ctx, t); err != nil {
		return err
	}
	// The target's cluster_id may have just changed; drop the memoized
	// execution_target_id → cluster_id entry so resolveRunClusterID re-reads it.
	// A stale entry would keep routing this target's runs (status reads, stop /
	// retry, log fetch) to the OLD cluster for the process lifetime.
	uc.targetClusterCache.Delete(strings.TrimSpace(t.ID))
	return nil
}

func (uc *Usecase) DeleteExecutionTarget(ctx context.Context, id string) error {
	_, err := uc.targetRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if err := uc.targetRepo.Delete(ctx, id); err != nil {
		return err
	}
	uc.targetClusterCache.Delete(strings.TrimSpace(id))
	return nil
}

func (uc *Usecase) StopBatchRuns(ctx context.Context, batchJobID, owner string) (stopped, failed int, _ error) {
	items, err := uc.backfillRepo.FindItemsByJobID(ctx, batchJobID)
	if err != nil {
		return 0, 0, fmt.Errorf("list batch items: %w", err)
	}
	for _, item := range items {
		runID := ""
		if item.PipelineRunID != nil {
			runID = *item.PipelineRunID
		}
		if runID == "" {
			continue
		}
		if err := uc.StopRun(ctx, runID); err != nil {
			failed++
			continue
		}
		stopped++
	}
	// Halt the background submission goroutine so it stops creating new runs
	// for any items that have not been submitted yet.
	uc.cancelBatch(batchJobID)
	_ = uc.backfillRepo.UpdateJobStatus(ctx, batchJobID, "cancelled")
	return stopped, failed, nil
}

func (uc *Usecase) registerBatchCancel(jobID string, cancel context.CancelFunc) {
	uc.batchCancelMu.Lock()
	if uc.batchCancels == nil {
		uc.batchCancels = make(map[string]context.CancelFunc)
	}
	uc.batchCancels[jobID] = cancel
	uc.batchCancelMu.Unlock()
}

func (uc *Usecase) unregisterBatchCancel(jobID string) {
	uc.batchCancelMu.Lock()
	delete(uc.batchCancels, jobID)
	uc.batchCancelMu.Unlock()
}

// cancelBatch cancels the in-flight submission goroutine for a batch job.
// Returns false when no submission is currently tracked (already finished).
func (uc *Usecase) cancelBatch(jobID string) bool {
	uc.batchCancelMu.Lock()
	cancel := uc.batchCancels[jobID]
	uc.batchCancelMu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

func (uc *Usecase) defaultExecutionTarget() models.ExecutionTarget {
	status := "available"
	if uc.wfClient == nil || strings.TrimSpace(uc.namespace) == "" {
		status = "unavailable"
	}
	return models.ExecutionTarget{
		ID:                   "default",
		Name:                 "Default Argo target",
		Cluster:              "default",
		Namespace:            uc.namespace,
		ServiceAccount:       defaultExecutionTargetServiceAccount(),
		ArgoServerConfigured: uc.wfClient != nil,
		Status:               status,
		Enabled:              true,
		IsDefault:            true,
		Description:          "Current backend-configured Argo workflow namespace.",
		ResourceDefaults:     map[string]interface{}{},
		QuotaPolicy:          map[string]interface{}{},
		Labels:               map[string]interface{}{"source": "backend-default"},
	}
}

func (uc *Usecase) ensureDefaultExecutionTarget(ctx context.Context) error {
	if uc.targetRepo == nil {
		return nil
	}
	existing, err := uc.targetRepo.FindDefault(ctx)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}
	target := uc.defaultExecutionTarget()
	return uc.targetRepo.Save(ctx, &target)
}

func (uc *Usecase) normalizeExecutionTarget(target *models.ExecutionTarget) {
	if target == nil {
		return
	}
	if target.Status == "" {
		target.Status = "available"
	}
	if target.ID == "default" && uc.wfClient == nil {
		target.Status = "unavailable"
	}
	if target.ID == "default" && target.Namespace == "" {
		target.Namespace = uc.namespace
	}
	if (target.ID == "default" || target.IsDefault) && strings.TrimSpace(target.ServiceAccount) == "" {
		target.ServiceAccount = defaultExecutionTargetServiceAccount()
	}
	if target.ResourceDefaults == nil {
		target.ResourceDefaults = map[string]interface{}{}
	}
	if target.QuotaPolicy == nil {
		target.QuotaPolicy = map[string]interface{}{}
	}
	if target.Labels == nil {
		target.Labels = map[string]interface{}{}
	}
	target.ArgoServerConfigured = target.ArgoServerURL != "" || uc.wfClient != nil
}

func (uc *Usecase) resolveExecutionTarget(ctx context.Context, targetID string) (*models.ExecutionTarget, error) {
	targetID = strings.TrimSpace(targetID)
	if uc.targetRepo == nil {
		if targetID == "" || targetID == "default" {
			target := uc.defaultExecutionTarget()
			return &target, nil
		}
		return nil, fmt.Errorf("%w: target_id=%q", ErrExecutionTargetNotFound, targetID)
	}
	if err := uc.ensureDefaultExecutionTarget(ctx); err != nil {
		return nil, err
	}
	var (
		target *models.ExecutionTarget
		err    error
	)
	if targetID == "" || targetID == "default" {
		target, err = uc.targetRepo.FindDefault(ctx)
	} else {
		target, err = uc.targetRepo.FindByID(ctx, targetID)
	}
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, fmt.Errorf("%w: target_id=%q", ErrExecutionTargetNotFound, targetID)
	}
	uc.normalizeExecutionTarget(target)
	if !target.Enabled {
		return nil, fmt.Errorf("%w: target_id=%q is disabled", ErrExecutionTargetNotFound, targetID)
	}
	return target, nil
}

func executionTargetSnapshot(target *models.ExecutionTarget) map[string]interface{} {
	if target == nil {
		return map[string]interface{}{}
	}
	return map[string]interface{}{
		"id":                     target.ID,
		"name":                   target.Name,
		"cluster":                target.Cluster,
		"namespace":              target.Namespace,
		"serviceAccount":         target.ServiceAccount,
		"argoServerConfigured":   target.ArgoServerConfigured,
		"status":                 target.Status,
		"isDefault":              target.IsDefault,
		"resourceDefaults":       target.ResourceDefaults,
		"quotaPolicy":            target.QuotaPolicy,
		"labels":                 target.Labels,
		"argoAuthSecretRef":      target.ArgoAuthSecretRef,
		"argoInsecureSkipTls":    target.ArgoInsecureSkipTLS,
		"argoCaCertRef":          target.ArgoCACertRef,
		"argoServerUrlRedacted":  target.ArgoServerURL != "",
		"argoAuthSecretRedacted": target.ArgoAuthSecretRef != "",
	}
}

func (uc *Usecase) deploymentToRun(dep *models.PipelineDeployment) *models.PipelineRun {
	if dep == nil {
		return nil
	}
	target := uc.defaultExecutionTarget()
	if dep.ExecutionTarget != nil {
		target = *dep.ExecutionTarget
	}
	run := &models.PipelineRun{
		ID:                dep.ID,
		TemplateID:        dep.TemplateID,
		TemplateVersion:   dep.TemplateVersion,
		PipelineName:      dep.PipelineName,
		WorkflowName:      dep.WorkflowName,
		ExecutionTargetID: target.ID,
		TargetSnapshot:    executionTargetSnapshot(&target),
		Status:            dep.Status,
		NodeCount:         dep.NodeCount,
		AssetIDs:          dep.AssetIDs,
		AssetCount:        dep.AssetCount,
		NoAssetRun:        len(dep.AssetIDs) == 0,
		Manifest:          dep.Manifest,
		PipelineJSON:      dep.PipelineJSON,
		ArgoNamespace:     target.Namespace,
		ExecutionTarget:   &target,
		Scope:             dep.Scope,
		Owner:             dep.Owner,
		CreatedAt:         dep.CreatedAt,
		UpdatedAt:         dep.UpdatedAt,
		FinishedAt:        dep.FinishedAt,
	}
	if run.ArgoNamespace == "" {
		run.ArgoNamespace = uc.namespace
	}
	return run
}

func runToDeployment(run *models.PipelineRun) *models.PipelineDeployment {
	if run == nil {
		return nil
	}
	dep := &models.PipelineDeployment{
		ID:              run.ID,
		TemplateID:      run.TemplateID,
		TemplateVersion: run.TemplateVersion,
		PipelineName:    run.PipelineName,
		WorkflowName:    run.WorkflowName,
		Status:          run.Status,
		NodeCount:       run.NodeCount,
		AssetIDs:        run.AssetIDs,
		AssetCount:      run.AssetCount,
		ExecutionTarget: run.ExecutionTarget,
		Scope:           run.Scope,
		Owner:           run.Owner,
		Manifest:        run.Manifest,
		PipelineJSON:    run.PipelineJSON,
		CreatedAt:       run.CreatedAt,
		UpdatedAt:       run.UpdatedAt,
		FinishedAt:      run.FinishedAt,
	}
	return dep
}

func (uc *Usecase) savePipelineRun(ctx context.Context, dep *models.PipelineDeployment, templateVersion int, wfUID string, opts ...DeployOptions) error {
	if uc.runRepo == nil || dep == nil {
		return nil
	}
	run := uc.deploymentToRun(dep)
	if templateVersion > 0 {
		run.TemplateVersion = &templateVersion
	}
	run.ArgoWorkflowUID = wfUID
	if len(opts) > 0 && opts[0].BatchJobID != "" {
		run.BatchJobID = &opts[0].BatchJobID
	}
	if err := uc.runRepo.Save(ctx, run); err != nil {
		return err
	}
	logPipelineSideEffect("persist run inputs", uc.persistRunInputs(ctx, run))
	if len(opts) > 0 && strings.TrimSpace(opts[0].BatchJobID) != "" {
		logPipelineSideEffect("persist batch child run relation", uc.persistRunRelation(ctx, &models.RunRelation{
			ParentRunID:  strings.TrimSpace(opts[0].BatchJobID),
			ChildRunID:   run.ID,
			RelationType: "batch_child",
			Source:       "run_kernel",
			Snapshot: map[string]interface{}{
				"batchJobId": strings.TrimSpace(opts[0].BatchJobID),
				"workflow":   run.WorkflowName,
			},
		}))
	}
	uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
		EventType:      runEventSubmitted,
		SubjectType:    "run",
		SubjectID:      run.ID,
		Status:         run.Status,
		Message:        "pipeline run submitted",
		OccurredAt:     run.CreatedAt,
		IdempotencyKey: fmt.Sprintf("run_submitted:%s", run.ID),
		Payload: map[string]interface{}{
			"pipelineName": run.PipelineName,
			"templateId":   run.TemplateID,
			"assetCount":   run.AssetCount,
			"targetId":     run.ExecutionTargetID,
		},
	})
	uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
		EventType:      runEventScheduled,
		SubjectType:    "run",
		SubjectID:      run.ID,
		Status:         run.Status,
		Message:        "pipeline run scheduled",
		OccurredAt:     run.CreatedAt,
		IdempotencyKey: fmt.Sprintf("run_scheduled:%s", run.ID),
		Payload: map[string]interface{}{
			"workflowName": run.WorkflowName,
			"namespace":    run.ArgoNamespace,
			"targetId":     run.ExecutionTargetID,
		},
	})
	if run.WorkflowName != "" {
		uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
			EventType:      runEventWorkflowCreated,
			SubjectType:    "workflow",
			SubjectID:      run.WorkflowName,
			Status:         run.Status,
			Message:        "argo workflow created",
			OccurredAt:     run.CreatedAt,
			IdempotencyKey: fmt.Sprintf("workflow_created:%s:%s", run.ID, run.WorkflowName),
			Payload: map[string]interface{}{
				"workflowUid": run.ArgoWorkflowUID,
				"namespace":   run.ArgoNamespace,
			},
		})
	}
	return nil
}

func (uc *Usecase) persistRunInputs(ctx context.Context, run *models.PipelineRun) error {
	if uc.runInputRepo == nil || run == nil {
		return nil
	}
	inputs := materializeRunInputs(run)
	for i := range inputs {
		id := strings.TrimSpace(inputs[i].ID)
		if id == "" {
			continue
		}
		if _, err := uuid.Parse(id); err != nil {
			inputs[i].ID = ""
		}
	}
	if len(inputs) == 0 {
		return nil
	}
	return uc.runInputRepo.UpsertMany(ctx, inputs)
}

func materializeRunInputs(run *models.PipelineRun) []models.RunInput {
	if run == nil {
		return nil
	}
	items := make([]models.RunInput, 0, len(run.AssetIDs)+4)
	for i, assetID := range run.AssetIDs {
		assetID = strings.TrimSpace(assetID)
		if assetID == "" {
			continue
		}
		items = append(items, models.RunInput{
			ID:     fmt.Sprintf("%s:asset:%d", run.ID, i),
			RunID:  run.ID,
			Type:   "asset",
			RefID:  assetID,
			Source: "asset_ids",
		})
	}
	if run.ExecutionTargetID != "" || len(run.TargetSnapshot) > 0 {
		items = append(items, models.RunInput{
			ID:       fmt.Sprintf("%s:runtime-target", run.ID),
			RunID:    run.ID,
			Type:     "runtime_target",
			RefID:    run.ExecutionTargetID,
			Source:   "execution_target",
			Snapshot: copyStringAnyMap(run.TargetSnapshot),
		})
	}
	collectRunInputsFromPipelineJSON(run.ID, run.PipelineJSON, &items)
	return items
}

func (uc *Usecase) persistRunRelation(ctx context.Context, relation *models.RunRelation) error {
	if uc.runRelationRepo == nil || relation == nil {
		return nil
	}
	if relation.Source == "" {
		relation.Source = "run_kernel"
	}
	return uc.runRelationRepo.Upsert(ctx, relation)
}

func (uc *Usecase) enrichRun(ctx context.Context, run *models.PipelineRun, refreshAssets bool) {
	if run == nil {
		return
	}
	if len(run.AssetIDs) == 0 {
		if assetIDs := assetIDsFromPipelineJSON(run.PipelineJSON); len(assetIDs) > 0 {
			run.AssetIDs = assetIDs
		}
	}
	if len(run.AssetIDs) > 0 {
		run.AssetCount = len(run.AssetIDs)
	}
	if run.ExecutionTarget == nil {
		if uc.targetRepo != nil && run.ExecutionTargetID != "" {
			if target, err := uc.targetRepo.FindByID(ctx, run.ExecutionTargetID); err == nil && target != nil {
				uc.normalizeExecutionTarget(target)
				run.ExecutionTarget = target
			}
		}
		if run.ExecutionTarget == nil {
			target := uc.defaultExecutionTarget()
			run.ExecutionTarget = &target
		}
	}
	if uc.runNodeRepo != nil {
		if nodes, err := uc.runNodeRepo.FindByRunID(ctx, run.ID); err == nil {
			run.Nodes = nodes
		}
	}
	// asset_node projection is a WRITE (ReplaceByRunID = DELETE+INSERT). Keep it
	// OFF the list read path: ListRuns fans enrichRun over every row, turning a
	// read into an O(N) write storm + row-lock churn. The run-status write path
	// already refreshes asset_nodes, so list reads don't need to. Single-run
	// reads (GetRun / create) still pass true for the lazy refresh.
	if refreshAssets {
		uc.refreshAssetNodes(ctx, run)
	}
	runstate.AnnotateRunDiagnostics(run)
}

func timePtrFromMeta(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	out := t.UTC()
	return &out
}

func structToMap(v interface{}) map[string]interface{} {
	raw, err := json.Marshal(v)
	if err != nil || len(raw) == 0 || string(raw) == "null" {
		return map[string]interface{}{}
	}
	out := map[string]interface{}{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]interface{}{}
	}
	return out
}

func phaseRunTerminalEvent(phase wfv1.WorkflowPhase) string {
	switch phase {
	case wfv1.WorkflowSucceeded:
		return runEventCompleted
	case wfv1.WorkflowFailed:
		return runEventFailed
	case wfv1.WorkflowError:
		return runEventFailed
	default:
		return ""
	}
}

func phaseNodeEvent(phase wfv1.NodePhase) string {
	switch phase {
	case wfv1.NodeSucceeded:
		return runEventNodeSucceeded
	case wfv1.NodeFailed:
		return runEventNodeFailed
	case wfv1.NodeError:
		return runEventNodeError
	default:
		return ""
	}
}

func ptrTimeOrNow(t *time.Time) time.Time {
	if t != nil && !t.IsZero() {
		return t.UTC()
	}
	return time.Now().UTC()
}

func argoTimeOrZero(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	out := t.UTC()
	return &out
}

func (uc *Usecase) appendRunEvent(ctx context.Context, run *models.PipelineRun, event models.PipelineRunEvent) {
	if uc.runEventRepo == nil || run == nil {
		return
	}
	event.RunID = run.ID
	if event.WorkflowName == "" {
		event.WorkflowName = run.WorkflowName
	}
	if event.SubjectType == "" {
		event.SubjectType = "run"
	}
	if event.SubjectID == "" {
		event.SubjectID = run.ID
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	if event.ObservedAt.IsZero() {
		event.ObservedAt = time.Now().UTC()
	}
	if event.IdempotencyKey == "" {
		event.IdempotencyKey = strings.Join([]string{
			event.EventType,
			event.SubjectType,
			event.SubjectID,
			event.Status,
			event.OccurredAt.UTC().Format(time.RFC3339Nano),
		}, ":")
	}
	logPipelineSideEffect("append pipeline run event", uc.runEventRepo.Append(ctx, &event))
	uc.createNotificationCandidate(ctx, run, &event)
}

func isFailureEvent(eventType, status string) bool {
	normalized := strings.ToLower(eventType + " " + status)
	return strings.Contains(normalized, "failed") || strings.Contains(normalized, "error")
}

func (uc *Usecase) createNotificationCandidate(ctx context.Context, run *models.PipelineRun, event *models.PipelineRunEvent) {
	if uc.notifyRepo == nil || run == nil || event == nil || !isFailureEvent(event.EventType, event.Status) {
		return
	}
	eventID := event.IdempotencyKey
	if eventID == "" {
		eventID = event.ID
	}
	logPipelineSideEffect("append pipeline run notification candidate", uc.notifyRepo.AppendCandidate(ctx, &models.PipelineRunNotificationCandidate{
		RunID:          run.ID,
		EventID:        eventID,
		EventType:      event.EventType,
		SubjectType:    event.SubjectType,
		SubjectID:      event.SubjectID,
		Status:         event.Status,
		Message:        event.Message,
		SinkType:       "candidate",
		DeliveryStatus: "pending",
		IdempotencyKey: fmt.Sprintf("notify:%s:%s", run.ID, eventID),
	}))
}

func watcherStateWithHealth(state *models.PipelineRunWatcherState) *models.PipelineRunWatcherState {
	if state == nil {
		return nil
	}
	state.Healthy = state.ConsecutiveFailures == 0 && state.LastError == ""
	if state.LastSuccessAt != nil {
		lag := int64(time.Since(*state.LastSuccessAt).Seconds())
		if lag < 0 {
			lag = 0
		}
		state.ScanLagSeconds = &lag
		state.Stale = lag > int64(3*time.Minute/time.Second)
	} else {
		state.Stale = true
	}
	return state
}

func cloneWatcherState(state *models.PipelineRunWatcherState, limit int) models.PipelineRunWatcherState {
	if limit <= 0 {
		limit = 50
	}
	out := models.PipelineRunWatcherState{ID: "default", ActiveScanLimit: limit}
	if state != nil {
		out = *state
	}
	if out.ID == "" {
		out.ID = "default"
	}
	if out.ActiveScanLimit <= 0 {
		out.ActiveScanLimit = limit
	}
	if out.ActiveScanLimit <= 0 {
		out.ActiveScanLimit = 100
	}
	return out
}

func deriveAssetNodeRows(run *models.PipelineRun, nodes []models.PipelineRunNode) []models.PipelineRunAssetNode {
	if run == nil || len(nodes) == 0 {
		return nil
	}
	assetIDs := run.AssetIDs
	if len(assetIDs) == 0 {
		assetIDs = []string{"no-asset"}
	}
	rows := make([]models.PipelineRunAssetNode, 0, len(assetIDs)*len(nodes))
	rowIndexes := make(map[string]int, len(assetIDs)*len(nodes))
	now := time.Now().UTC()
	for _, assetID := range assetIDs {
		if strings.TrimSpace(assetID) == "" {
			continue
		}
		for _, node := range nodes {
			if isWorkflowControlRunNode(node) {
				continue
			}
			nodeID := node.PipelineNodeID
			if nodeID == "" {
				nodeID = node.ArgoNodeID
			}
			if nodeID == "" {
				nodeID = node.DisplayName
			}
			if nodeID == "" {
				continue
			}
			costSource := "not_available"
			if node.EstimatedCostUSD != nil {
				costSource = "estimated_resource_duration"
			}
			row := models.PipelineRunAssetNode{
				RunID:            run.ID,
				AssetID:          assetID,
				PipelineNodeID:   nodeID,
				ArgoNodeID:       node.ArgoNodeID,
				DisplayName:      node.DisplayName,
				Status:           node.Phase,
				Message:          node.Message,
				PodName:          node.PodName,
				LogRef:           node.LogRef,
				EstimatedCostUSD: node.EstimatedCostUSD,
				CostSource:       costSource,
				StartedAt:        node.StartedAt,
				FinishedAt:       node.FinishedAt,
				UpdatedAt:        now,
			}
			key := assetID + "\x00" + nodeID
			if existingIndex, ok := rowIndexes[key]; ok {
				if assetNodeRowScore(row) > assetNodeRowScore(rows[existingIndex]) {
					rows[existingIndex] = row
				}
				continue
			}
			rowIndexes[key] = len(rows)
			rows = append(rows, row)
		}
	}
	return rows
}

func isWorkflowControlRunNode(node models.PipelineRunNode) bool {
	if node.Type == string(wfv1.NodeTypeDAG) || node.Type == string(wfv1.NodeTypeSteps) {
		return true
	}
	id := strings.TrimSpace(node.PipelineNodeID)
	return id == "dag"
}

func assetNodeRowScore(row models.PipelineRunAssetNode) int {
	score := 0
	if row.PodName != "" {
		score += 10
	}
	if row.EstimatedCostUSD != nil {
		score += 8
	}
	if row.StartedAt != nil {
		score += 4
	}
	if row.FinishedAt != nil {
		score += 4
	}
	switch row.Status {
	case string(wfv1.NodeSucceeded), string(wfv1.NodeFailed), string(wfv1.NodeError):
		score += 3
	case string(wfv1.NodeRunning):
		score += 2
	case string(wfv1.NodePending), "":
		score--
	}
	if row.ArgoNodeID != "" && !strings.HasPrefix(row.ArgoNodeID, "static:") {
		score++
	}
	return score
}

func (uc *Usecase) refreshAssetNodes(ctx context.Context, run *models.PipelineRun) {
	if uc.assetNodeRepo == nil || run == nil {
		return
	}
	nodes := run.Nodes
	if len(nodes) == 0 && uc.runNodeRepo != nil {
		if stored, err := uc.runNodeRepo.FindByRunID(ctx, run.ID); err == nil {
			nodes = stored
		}
	}
	rows := deriveAssetNodeRows(run, nodes)
	logPipelineSideEffect("replace pipeline run asset nodes", uc.assetNodeRepo.ReplaceByRunID(ctx, run.ID, rows))
}

func (uc *Usecase) appendWorkflowEvents(ctx context.Context, run *models.PipelineRun, wf *wfv1.Workflow) {
	if run == nil || wf == nil {
		return
	}
	workflowUID := string(wf.UID)
	workflowName := wf.Name
	if workflowName == "" {
		workflowName = run.WorkflowName
	}
	uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
		EventType:      runEventWorkflowObserved,
		SubjectType:    "workflow",
		SubjectID:      workflowName,
		Status:         string(wf.Status.Phase),
		Message:        wf.Status.Message,
		OccurredAt:     ptrTimeOrNow(argoTimeOrZero(wf.CreationTimestamp.Time)),
		IdempotencyKey: fmt.Sprintf("workflow_observed:%s:%s", workflowName, workflowUID),
		Payload: map[string]interface{}{
			"workflowUid": workflowUID,
			"namespace":   wf.Namespace,
		},
	})
	if wf.Status.Phase != "" {
		occurredAt := ptrTimeOrNow(argoTimeOrZero(wf.Status.FinishedAt.Time))
		uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
			EventType:      runEventWorkflowPhaseChanged,
			SubjectType:    "workflow",
			SubjectID:      workflowName,
			Status:         string(wf.Status.Phase),
			Message:        wf.Status.Message,
			OccurredAt:     occurredAt,
			IdempotencyKey: fmt.Sprintf("workflow_phase:%s:%s", workflowName, wf.Status.Phase),
			Payload: map[string]interface{}{
				"workflowUid": workflowUID,
				"namespace":   wf.Namespace,
			},
		})
	}
	if terminalEvent := phaseRunTerminalEvent(wf.Status.Phase); terminalEvent != "" {
		uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
			EventType:      terminalEvent,
			SubjectType:    "run",
			SubjectID:      run.ID,
			Status:         string(wf.Status.Phase),
			Message:        wf.Status.Message,
			OccurredAt:     ptrTimeOrNow(argoTimeOrZero(wf.Status.FinishedAt.Time)),
			IdempotencyKey: fmt.Sprintf("run_terminal:%s:%s", run.ID, wf.Status.Phase),
		})
	}
}

func (uc *Usecase) appendNodeEvents(ctx context.Context, run *models.PipelineRun, nodes map[string]wfv1.NodeStatus) {
	for id, node := range nodes {
		// Skip the DataBrew exit-notify hook node (CYB-3058): it is infrastructure,
		// not a business step, and must not surface as an extra node/pod in the run
		// timeline or pods view (it is already excluded from runNodesFromWorkflow).
		if node.TemplateName == transpiler.ExitNotifyTemplateName {
			continue
		}
		displayName := node.DisplayName
		if displayName == "" {
			displayName = node.Name
		}
		payload := map[string]interface{}{
			"nodeId":       id,
			"name":         node.Name,
			"displayName":  displayName,
			"templateName": node.TemplateName,
			"type":         string(node.Type),
		}
		if !node.StartedAt.Time.IsZero() {
			uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
				EventType:      runEventNodeStarted,
				SubjectType:    "node",
				SubjectID:      id,
				Status:         string(node.Phase),
				Message:        node.Message,
				OccurredAt:     node.StartedAt.Time.UTC(),
				IdempotencyKey: fmt.Sprintf("node_started:%s:%s", id, node.StartedAt.Time.UTC().Format(time.RFC3339Nano)),
				Payload:        payload,
			})
		}
		if eventType := phaseNodeEvent(node.Phase); eventType != "" {
			occurredAt := ptrTimeOrNow(argoTimeOrZero(node.FinishedAt.Time))
			uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
				EventType:      eventType,
				SubjectType:    "node",
				SubjectID:      id,
				Status:         string(node.Phase),
				Message:        node.Message,
				OccurredAt:     occurredAt,
				IdempotencyKey: fmt.Sprintf("node_phase:%s:%s", id, node.Phase),
				Payload:        payload,
			})
		}
		if node.Type == wfv1.NodeTypePod && node.Name != "" {
			uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
				EventType:      runEventPodCreated,
				SubjectType:    "pod",
				SubjectID:      node.Name,
				Status:         string(node.Phase),
				Message:        node.Message,
				OccurredAt:     ptrTimeOrNow(argoTimeOrZero(node.StartedAt.Time)),
				IdempotencyKey: fmt.Sprintf("pod_created:%s", node.Name),
				Payload:        payload,
			})
			if node.Phase != "" {
				uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
					EventType:      runEventPodPhaseChanged,
					SubjectType:    "pod",
					SubjectID:      node.Name,
					Status:         string(node.Phase),
					Message:        node.Message,
					OccurredAt:     ptrTimeOrNow(argoTimeOrZero(node.FinishedAt.Time)),
					IdempotencyKey: fmt.Sprintf("pod_phase:%s:%s", node.Name, node.Phase),
					Payload:        payload,
				})
			}
		}
	}
}

func runNodesFromWorkflow(runID string, wfName string, nodes map[string]wfv1.NodeStatus) []models.PipelineRunNode {
	out := make([]models.PipelineRunNode, 0, len(nodes))
	for id, node := range nodes {
		// Exclude the DataBrew exit-notify hook node (CYB-3058): it is
		// infrastructure, not a business step, and must not appear in the step
		// list or influence run status derivation.
		if node.TemplateName == transpiler.ExitNotifyTemplateName {
			continue
		}
		now := time.Now().UTC()
		podName := ""
		if node.Type == wfv1.NodeTypePod {
			podName = node.Name
		}
		displayName := node.DisplayName
		if displayName == "" {
			displayName = node.Name
		}
		pipelineNodeID := node.TemplateName
		if pipelineNodeID == "" {
			pipelineNodeID = displayName
		}
		logRef := ""
		if podName != "" {
			logRef = fmt.Sprintf("/api/v1/workflows/%s/logs?podName=%s", wfName, podName)
		}
		phase := pipelineRunNodePhase(node)
		finishedAt := timePtrFromMeta(node.FinishedAt.Time)
		if finishedAt == nil && isTerminalWorkflowNodePhase(phase) {
			finishedAt = &now
		}
		out = append(out, models.PipelineRunNode{
			ID:                uuid.New().String(),
			RunID:             runID,
			PipelineNodeID:    pipelineNodeID,
			ArgoNodeID:        id,
			ArgoNodeName:      node.Name,
			DisplayName:       displayName,
			TemplateName:      node.TemplateName,
			Type:              string(node.Type),
			Phase:             phase,
			Message:           node.Message,
			PodName:           podName,
			HostNodeName:      node.HostNodeName,
			Children:          node.Children,
			Inputs:            structToMap(node.Inputs),
			Outputs:           structToMap(node.Outputs),
			ResourcesDuration: structToMap(node.ResourcesDuration),
			ResourceSummary:   map[string]interface{}{},
			LogRef:            logRef,
			StartedAt:         timePtrFromMeta(node.StartedAt.Time),
			FinishedAt:        finishedAt,
			CreatedAt:         now,
			UpdatedAt:         now,
		})
	}
	return out
}

func pipelineRunNodePhase(node wfv1.NodeStatus) string {
	if node.Phase == wfv1.NodePending && isImageStartupFailureMessage(node.Message) {
		return string(wfv1.NodeError)
	}
	return string(node.Phase)
}

func isTerminalWorkflowNodePhase(phase string) bool {
	switch phase {
	case string(wfv1.NodeSucceeded), string(wfv1.NodeFailed), string(wfv1.NodeError):
		return true
	default:
		return false
	}
}

func workflowTaskNameCandidates(node models.PipelineRunNode) []string {
	candidates := []string{
		strings.TrimSpace(node.PipelineNodeID),
		strings.TrimSpace(node.TemplateName),
		strings.TrimSpace(node.DisplayName),
	}
	if node.ArgoNodeName != "" {
		parts := strings.Split(node.ArgoNodeName, ".")
		candidates = append(candidates, strings.TrimSpace(parts[len(parts)-1]))
	}
	out := make([]string, 0, len(candidates))
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		out = append(out, candidate)
	}
	return out
}

func templateNodeType(tmpl *wfv1.Template) string {
	if tmpl == nil {
		return string(wfv1.NodeTypePod)
	}
	switch {
	case tmpl.DAG != nil:
		return string(wfv1.NodeTypeDAG)
	case tmpl.Steps != nil:
		return string(wfv1.NodeTypeSteps)
	case tmpl.Container != nil || tmpl.Script != nil:
		return string(wfv1.NodeTypePod)
	case tmpl.Suspend != nil:
		return string(wfv1.NodeTypeSuspend)
	default:
		return string(wfv1.NodeTypePod)
	}
}

func appendMissingStaticRunNodesFromWorkflow(runID string, wf *wfv1.Workflow, nodes []models.PipelineRunNode) []models.PipelineRunNode {
	if wf == nil || len(wf.Spec.Templates) == 0 {
		return nodes
	}
	templatesByName := make(map[string]*wfv1.Template, len(wf.Spec.Templates))
	for i := range wf.Spec.Templates {
		tmpl := &wf.Spec.Templates[i]
		templatesByName[tmpl.Name] = tmpl
	}
	seen := map[string]struct{}{}
	for _, node := range nodes {
		for _, candidate := range workflowTaskNameCandidates(node) {
			seen[candidate] = struct{}{}
		}
	}
	now := time.Now().UTC()
	for i := range wf.Spec.Templates {
		tmpl := &wf.Spec.Templates[i]
		if tmpl.DAG == nil {
			continue
		}
		for _, task := range tmpl.DAG.Tasks {
			taskName := strings.TrimSpace(task.Name)
			if taskName == "" {
				continue
			}
			if _, ok := seen[taskName]; ok {
				continue
			}
			templateName := strings.TrimSpace(task.Template)
			phase := "Pending"
			if wf.Status.Phase == wfv1.WorkflowSucceeded {
				phase = string(wfv1.NodeSucceeded)
			}
			nodes = append(nodes, models.PipelineRunNode{
				ID:              uuid.New().String(),
				RunID:           runID,
				PipelineNodeID:  taskName,
				ArgoNodeID:      fmt.Sprintf("static:%s", taskName),
				DisplayName:     taskName,
				TemplateName:    templateName,
				Type:            templateNodeType(templatesByName[templateName]),
				Phase:           phase,
				Children:        []string{},
				ResourceSummary: map[string]interface{}{},
				CreatedAt:       now,
				UpdatedAt:       now,
			})
			seen[taskName] = struct{}{}
		}
	}
	return nodes
}

func (uc *Usecase) replaceRunNodesFromWorkflow(ctx context.Context, run *models.PipelineRun, wf *wfv1.Workflow) {
	if uc.runNodeRepo == nil || run == nil || wf == nil {
		return
	}
	uc.appendNodeEvents(ctx, run, wf.Status.Nodes)
	nodes := runNodesFromWorkflow(run.ID, run.WorkflowName, wf.Status.Nodes)
	nodes = appendMissingStaticRunNodesFromWorkflow(run.ID, wf, nodes)
	if len(nodes) == 0 {
		return
	}
	for i := range nodes {
		if uc.pricing != nil {
			uc.enrichResourcesDurationWithNode(ctx, &nodes[i])
			nodes[i].EstimatedCostUSD = resourcesDurationToCost(nodes[i].ResourcesDuration, uc.pricing)
		}
	}
	logPipelineSideEffect("replace pipeline run nodes", uc.runNodeRepo.ReplaceByRunID(ctx, run.ID, nodes))
	run.Nodes = nodes
	uc.refreshAssetNodes(ctx, run)
}

func (uc *Usecase) runCostSnapshotMissing(run *models.PipelineRun) bool {
	if run == nil {
		return false
	}
	if len(run.Nodes) == 0 {
		return true
	}
	businessNodeCount := 0
	for _, node := range run.Nodes {
		if node.PipelineNodeID != "dag" {
			businessNodeCount++
		}
		if uc.pricing != nil && len(node.ResourcesDuration) > 0 && node.EstimatedCostUSD == nil {
			return true
		}
	}
	if run.NodeCount > 0 && businessNodeCount < run.NodeCount {
		return true
	}
	return false
}

func (uc *Usecase) applyWorkflowToRun(ctx context.Context, run *models.PipelineRun, wf *wfv1.Workflow, nodeMode nodeProjectionMode) {
	if uc.runRepo == nil || run == nil || wf == nil {
		return
	}
	prevStatus := run.Status
	uc.appendWorkflowEvents(ctx, run, wf)
	status := run.Status
	if wf.Status.Phase != "" {
		status = string(wf.Status.Phase)
	}
	message := wf.Status.Message
	finishedAt := argoTimeOrZero(wf.Status.FinishedAt.Time)
	derivedFailureReason := ""
	if derivedStatus, derivedMessage, derivedFinishedAt, ok := deriveTerminalRunFromWorkflowNodes(wf); ok {
		status = derivedStatus
		message = derivedMessage
		if derivedFinishedAt != nil {
			finishedAt = derivedFinishedAt
		}
	} else if derivedStatus, derivedMessage, derivedFinishedAt, ok := uc.deriveImageStartupRunFromWorkflow(run, wf); ok {
		status = derivedStatus
		message = derivedMessage
		finishedAt = derivedFinishedAt
		derivedFailureReason = "image_startup"
	} else if derivedStatus, ok := deriveActiveRunFromWorkflowNodes(wf); ok {
		// CYB-3491(语义): scheduling starvation is WAITING, not failure. A pod
		// that cannot schedule today can schedule when capacity frees — the
		// run stays in its Argo-truth active phase, and the read side
		// classifies the scheduler message as BlockingReason "unschedulable"
		// (runtimeos state machine). Only the workload's own errors (or
		// deterministic config errors like a bad image) are terminal.
		status = derivedStatus
		message = ""
		finishedAt = nil
	}
	// CYB-3491: finalize a run the moment its business steps succeed, without
	// waiting for the injected exit-notify hook (CYB-3058). That hook fires the
	// status webhook FROM INSIDE the workflow, so at poke time wf.Status.Phase
	// is still "Running" (the hook is part of the workflow). Relying on the
	// phase left runs stuck "Running" until the slow watcher re-observed them
	// after the hook finished — ~1h of projection lag under load, where the
	// batch completed-count trailed Argo reality by hundreds. The hook is infra
	// (already excluded from progress + the node list); exclude it from terminal
	// derivation too. Conservative: only SUCCEEDED is derived — any non-succeeded
	// business pod yields false, so failure/retry paths are untouched.
	if isActiveDeploymentStatus(status) {
		// CYB-3707: only finalize off the business pods when the node set is
		// complete. An in-flight argo-server /retry deletes the failed step's
		// node from wf.Status.Nodes so the controller can re-create it; during
		// that window the remaining pods can all be "succeeded" and
		// allBusinessPodsSucceeded would spuriously report done — promoting a
		// still-failed / mid-retry run to Succeeded and locking further retries.
		// Require the step-pod count to match the pipeline's step count first.
		if businessFinishedAt, ok := allBusinessPodsSucceeded(wf); ok &&
			(run.NodeCount <= 0 || businessStepPodCount(wf) >= run.NodeCount) {
			status = string(wfv1.WorkflowSucceeded)
			message = ""
			if businessFinishedAt != nil {
				finishedAt = businessFinishedAt
			}
		}
	}
	run.Status = status
	if startedAt := argoTimeOrZero(wf.Status.StartedAt.Time); startedAt != nil {
		run.StartedAt = startedAt
	}
	if string(wf.UID) != "" {
		run.ArgoWorkflowUID = string(wf.UID)
	}
	if isActiveDeploymentStatus(run.Status) {
		run.FinishedAt = nil
		run.Message = ""
	}
	if wf.Status.Phase == wfv1.WorkflowSucceeded || wf.Status.Phase == wfv1.WorkflowFailed || wf.Status.Phase == wfv1.WorkflowError {
		if finishedAt != nil {
			run.FinishedAt = finishedAt
		} else if run.FinishedAt == nil || run.FinishedAt.IsZero() {
			now := time.Now().UTC()
			run.FinishedAt = &now
		}
	}
	if !isActiveDeploymentStatus(run.Status) {
		run.Message = strings.TrimSpace(message)
		if finishedAt != nil {
			run.FinishedAt = finishedAt
		} else if run.FinishedAt == nil || run.FinishedAt.IsZero() {
			now := time.Now().UTC()
			run.FinishedAt = &now
		}
	}
	if derivedFailureReason != "" {
		uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
			EventType:      runEventFailed,
			SubjectType:    "run",
			SubjectID:      run.ID,
			Status:         run.Status,
			Message:        run.Message,
			Reason:         derivedFailureReason,
			OccurredAt:     ptrTimeOrNow(finishedAt),
			IdempotencyKey: fmt.Sprintf("run_%s:%s", derivedFailureReason, run.ID),
			Payload: map[string]interface{}{
				"workflowName": run.WorkflowName,
				"namespace":    run.ArgoNamespace,
			},
		})
	}
	// CYB-3490/CYB-3491: workflow-level progress is the continuous projection —
	// one column per run. Recomputed from the step pods rather than copied from
	// Argo's status.progress, because Argo counts the injected databrew-exit-notify
	// onExit hook (CYB-3058) as a pod: a single-step pipeline would otherwise
	// report "1/2" (step done, hook still running) or "2/2" instead of the
	// truthful "1/1". Fall back to Argo's raw value only before any step pod is
	// observed.
	if p := workflowStepProgress(wf); p != "" {
		run.Progress = p
	} else if p := strings.TrimSpace(string(wf.Status.Progress)); p != "" {
		run.Progress = p
	}
	uc.persistRunObservation(ctx, run)
	uc.projectRunNodes(ctx, run, wf, nodeMode, prevStatus)
}

// workflowStepProgress recomputes the "done/total" progress string from the
// workflow's step pods, counting only real pipeline steps. Argo's own
// status.progress counts the injected databrew-exit-notify onExit hook
// (CYB-3058) as a pod, which inflates the denominator — a single-step pipeline
// reports "1/2" (step done, hook running) or "2/2" instead of the truthful
// "1/1". DAG/Steps container nodes are not pods and are already excluded.
// Returns "" when no step pod has been observed yet so the caller can fall
// back to Argo's raw value during the brief pre-pod window.
func workflowStepProgress(wf *wfv1.Workflow) string {
	if wf == nil {
		return ""
	}
	total, done := 0, 0
	for _, node := range wf.Status.Nodes {
		if node.Type != wfv1.NodeTypePod {
			continue
		}
		if node.TemplateName == transpiler.ExitNotifyTemplateName {
			continue
		}
		total++
		switch node.Phase {
		case wfv1.NodeSucceeded, wfv1.NodeSkipped, wfv1.NodeOmitted:
			done++
		}
	}
	if total == 0 {
		return ""
	}
	return fmt.Sprintf("%d/%d", done, total)
}

// nodeProjectionMode controls whether an Argo observation projects per-node
// rows (CYB-3490). Continuous observation is workflow-level only; the full
// node snapshot is archived exactly once when the workflow reaches terminal.
type nodeProjectionMode int

const (
	// nodeProjectTerminalArchive skips node writes while the workflow is
	// active and archives the final snapshot once at terminal. Default for
	// the watcher, webhook, and reconcile paths — at 1000 workflows x 100
	// steps the per-cycle delete+insert of node rows was the write amplifier.
	nodeProjectTerminalArchive nodeProjectionMode = iota
	// nodeProjectLive always projects nodes while the workflow is active —
	// reserved for bounded single-run drill-in (GetRun), where the user is
	// looking at one workflow's step detail.
	nodeProjectLive
)

// projectRunNodes applies the node-projection policy for one observation.
// prevStatus is the run's status BEFORE this observation was applied, so the
// active->terminal transition archives exactly once; re-observations of an
// already-terminal run only backfill a missing archive (lost webhook, or a
// restart between the status persist and the archive write).
func (uc *Usecase) projectRunNodes(ctx context.Context, run *models.PipelineRun, wf *wfv1.Workflow, mode nodeProjectionMode, prevStatus string) {
	if isActiveDeploymentStatus(run.Status) {
		if mode == nodeProjectLive {
			uc.replaceRunNodesFromWorkflow(ctx, run, wf)
		}
		return
	}
	if isActiveDeploymentStatus(prevStatus) {
		uc.replaceRunNodesFromWorkflow(ctx, run, wf)
		return
	}
	if uc.runNodeRepo == nil {
		return
	}
	rows, err := uc.runNodeRepo.FindByRunID(ctx, run.ID)
	if err != nil || len(rows) > 0 {
		return
	}
	uc.replaceRunNodesFromWorkflow(ctx, run, wf)
}

func deriveActiveRunFromWorkflowNodes(wf *wfv1.Workflow) (string, bool) {
	if wf == nil || wf.Status.Phase != "" {
		return "", false
	}
	hasPending := false
	for _, node := range wf.Status.Nodes {
		switch node.Phase {
		case wfv1.NodeRunning:
			return string(wfv1.WorkflowRunning), true
		case wfv1.NodePending:
			hasPending = true
		}
	}
	if hasPending {
		return string(wfv1.WorkflowPending), true
	}
	return "", false
}

func deriveTerminalRunFromWorkflowNodes(wf *wfv1.Workflow) (string, string, *time.Time, bool) {
	if wf == nil || !isActiveDeploymentStatus(string(wf.Status.Phase)) {
		return "", "", nil, false
	}
	var latestFinishedAt *time.Time
	derivedMessage := ""
	for _, node := range wf.Status.Nodes {
		phase := node.Phase
		if phase != wfv1.NodeFailed && phase != wfv1.NodeError {
			continue
		}
		message := strings.TrimSpace(node.Message)
		if !isWorkflowShutdownMessage(message) {
			continue
		}
		if finishedAt := argoTimeOrZero(node.FinishedAt.Time); finishedAt != nil {
			if latestFinishedAt == nil || finishedAt.After(*latestFinishedAt) {
				latestFinishedAt = finishedAt
				derivedMessage = message
			}
		} else if derivedMessage == "" {
			derivedMessage = message
		}
	}
	if derivedMessage == "" {
		return "", "", nil, false
	}
	return string(wfv1.WorkflowFailed), derivedMessage, latestFinishedAt, true
}

func isWorkflowShutdownMessage(message string) bool {
	normalized := strings.ToLower(strings.TrimSpace(message))
	return strings.Contains(normalized, "workflow shutdown with strategy:") ||
		strings.Contains(normalized, "stopped with strategy")
}

// allBusinessPodsSucceeded reports whether every real step pod in the workflow
// has finished successfully, ignoring the injected databrew-exit-notify hook
// (CYB-3058) and non-pod DAG/Steps containers. It returns the latest business
// finish time. Used to finalize a run as Succeeded while the workflow phase is
// still "Running" only because the exit hook (which itself fires the status
// webhook) has not completed yet — see the caller in applyWorkflowToRun.
// Conservative by design: a single non-succeeded business pod (running, pending,
// failed, or errored) returns false, leaving failure and retry handling to the
// existing phase/derivation logic. Returns false when no business pod exists.
func allBusinessPodsSucceeded(wf *wfv1.Workflow) (*time.Time, bool) {
	if wf == nil {
		return nil, false
	}
	sawBusinessPod := false
	var latestFinishedAt *time.Time
	for _, node := range wf.Status.Nodes {
		if node.Type != wfv1.NodeTypePod || node.TemplateName == transpiler.ExitNotifyTemplateName {
			continue
		}
		sawBusinessPod = true
		switch node.Phase {
		case wfv1.NodeSucceeded, wfv1.NodeSkipped, wfv1.NodeOmitted:
			if fa := argoTimeOrZero(node.FinishedAt.Time); fa != nil &&
				(latestFinishedAt == nil || fa.After(*latestFinishedAt)) {
				latestFinishedAt = fa
			}
		default:
			return nil, false
		}
	}
	return latestFinishedAt, sawBusinessPod
}

// businessStepPodCount returns how many real step pods (excluding the injected
// databrew-exit-notify hook) exist in the workflow, regardless of phase. Used
// by applyWorkflowToRun to detect a retry that has deleted a failed step's node
// from wf.Status.Nodes (CYB-3707): a count below the pipeline's step count means
// the node set is incomplete, so allBusinessPodsSucceeded must not be trusted to
// finalize the run as Succeeded.
func businessStepPodCount(wf *wfv1.Workflow) int {
	if wf == nil {
		return 0
	}
	count := 0
	for _, node := range wf.Status.Nodes {
		if node.Type != wfv1.NodeTypePod || node.TemplateName == transpiler.ExitNotifyTemplateName {
			continue
		}
		count++
	}
	return count
}

func (uc *Usecase) persistRunObservation(ctx context.Context, run *models.PipelineRun) {
	if uc.runRepo == nil || run == nil || strings.TrimSpace(run.ID) == "" {
		return
	}
	existing, err := uc.runRepo.FindByID(ctx, run.ID)
	if err != nil || existing == nil {
		saveErr := uc.runRepo.Save(ctx, run)
		logPipelineSideEffect("save pipeline run observation", saveErr)
		if saveErr == nil {
			uc.syncBackfillItemStatusFromRun(ctx, run)
		}
		return
	}
	existingWasActive := isActiveDeploymentStatus(existing.Status)
	// Monotonicity guard (CYB-3058): a run that has already succeeded must not be
	// regressed to an active phase by a late or out-of-order observation (e.g. a
	// delayed poll snapshot landing after a push-triggered terminal apply). Only
	// success is guarded here — Argo never un-succeeds a workflow. Error/Failed
	// remain revivable ONLY as misclassification recovery (TTL-cleanup false
	// positives, which carry a stale-unavailable message); a definitive failure
	// is guarded separately below.
	if isSucceededRunStatus(existing.Status) && isActiveDeploymentStatus(run.Status) {
		slog.Warn("persistRunObservation: ignoring active-status regression on succeeded run",
			"runID", run.ID,
			"workflowName", run.WorkflowName,
			"succeededStatus", existing.Status,
			"incomingStatus", run.Status,
		)
		*run = *existing
		return
	}
	// Monotonicity guard (CYB-3080): a run rejected before submission (e.g. by the
	// resource guard) is Failed/Error with a real, non-transient message and will
	// never have an Argo workflow. The "waiting for workflow creation" heuristic
	// (isPendingBatchWorkflowCreation matches any placeholder batch name) would
	// otherwise revive it to Pending with the message wiped, leaving it to poll a
	// workflow that cannot exist. Unlike a misclassification (stale-unavailable
	// message), a definitive failure is final and must not be regressed to active.
	if isDefinitiveTerminalFailure(existing) && isActiveDeploymentStatus(run.Status) {
		slog.Warn("persistRunObservation: ignoring active-status regression on definitively-failed run",
			"runID", run.ID,
			"workflowName", run.WorkflowName,
			"failedStatus", existing.Status,
			"failedMessage", existing.Message,
			"incomingStatus", run.Status,
		)
		*run = *existing
		return
	}
	if existing.Status != run.Status || existing.Message != run.Message {
		slog.Info("persistRunObservation status change",
			"runID", run.ID,
			"workflowName", run.WorkflowName,
			"oldStatus", existing.Status,
			"newStatus", run.Status,
			"oldMessage", existing.Message,
			"newMessage", run.Message,
		)
	}
	existing.Status = run.Status
	if isActiveDeploymentStatus(run.Status) {
		existing.FinishedAt = nil
	} else if run.FinishedAt != nil && !run.FinishedAt.IsZero() {
		if existingWasActive || existing.FinishedAt == nil || existing.FinishedAt.IsZero() || run.FinishedAt.Before(*existing.FinishedAt) {
			existing.FinishedAt = run.FinishedAt
		}
	}
	existing.Message = run.Message
	// CYB-3490: carry the workflow-level progress onto the persisted row —
	// only when the observation actually saw one, so a poll that raced an
	// empty status.progress does not wipe a previously stored value.
	if p := strings.TrimSpace(run.Progress); p != "" {
		existing.Progress = p
	}
	if uid := strings.TrimSpace(run.ArgoWorkflowUID); uid != "" {
		existing.ArgoWorkflowUID = uid
	}
	if name := strings.TrimSpace(run.WorkflowName); name != "" {
		existing.WorkflowName = name
	}
	if run.StartedAt != nil {
		existing.StartedAt = run.StartedAt
	}
	saveErr := uc.runRepo.Save(ctx, existing)
	logPipelineSideEffect("save pipeline run observation", saveErr)
	if saveErr == nil {
		uc.syncBackfillItemStatusFromRun(ctx, existing)
	}
	*run = *existing
}

func (uc *Usecase) syncBackfillItemStatusFromRun(ctx context.Context, run *models.PipelineRun) {
	if uc.backfillRepo == nil || run == nil || strings.TrimSpace(run.ID) == "" {
		return
	}
	if run.BatchJobID == nil || strings.TrimSpace(*run.BatchJobID) == "" {
		return
	}
	item, err := uc.backfillRepo.FindItemByPipelineRunID(ctx, run.ID)
	if err != nil || item == nil {
		if err != nil {
			slog.Warn("syncBackfillItemStatusFromRun: find item failed", "runID", run.ID, "err", err)
		}
		return
	}
	nextStatus := mapRunStatusToBackfillItem(run.Status)
	// CYB-3491: an in-flight projection is only legitimate when the run has a
	// persisted Argo UID. A uid-less run is an unsubmitted placeholder — the
	// item must stay "pending" for the submitter (invariant ②: unsubmitted is
	// never failed, never in-flight).
	if nextStatus == "submitted" && strings.TrimSpace(run.ArgoWorkflowUID) == "" {
		return
	}
	if nextStatus == "" || strings.EqualFold(strings.TrimSpace(item.Status), nextStatus) {
		return
	}
	errMsg := ""
	if nextStatus == "failed" {
		errMsg = strings.TrimSpace(run.Message)
	}
	logPipelineSideEffect("sync backfill item status from run",
		uc.backfillRepo.UpdateItemStatus(ctx, item.ID, nextStatus, run.WorkflowName, errMsg))
}

// mapRunStatusToBackfillItem projects a run status onto its backfill item.
// CYB-3491: in-flight (queued or executing) maps to "submitted" — the item
// state machine is pending → submitted → completed|failed, forward-only.
func mapRunStatusToBackfillItem(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "succeeded", "success":
		return "completed"
	case "failed", "error", "expired":
		return "failed"
	case "pending", "running", "unknown":
		return "submitted"
	default:
		return ""
	}
}

func isStaleWorkflowUnavailableMessage(message string) bool {
	switch strings.TrimSpace(message) {
	case staleWorkflowTTLCleanupMessage, messageWorkflowAwaitingDeploy, messageWorkflowUnavailable:
		return true
	default:
		return false
	}
}

// isNoisyMisclassifiedMessage reports whether the run's message carries no
// actionable diagnostic — i.e., we'd learn nothing by asking Argo again.
// Empty message + the known "stale/unavailable" markers count as noisy;
// anything else (real scheduler / image error text) counts as real and
// deserves a re-check. Used by the CYB-3672 age-based short-circuit.
func isNoisyMisclassifiedMessage(message string) bool {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return true
	}
	return isStaleWorkflowUnavailableMessage(trimmed)
}

// runHasCreatedTimestamp reports whether the run carries a real CreatedAt or
// StartedAt. Zero timestamps only occur in test fixtures; production DB rows
// always populate CreatedAt (NOT NULL). Used by CYB-3672 to avoid gating
// tests that leave timestamps unset.
func runHasCreatedTimestamp(run *models.PipelineRun) bool {
	if run == nil {
		return false
	}
	if !run.CreatedAt.IsZero() {
		return true
	}
	if run.StartedAt != nil && !run.StartedAt.IsZero() {
		return true
	}
	return false
}

func workflowUnavailableMessage(run *models.PipelineRun) string {
	if run == nil {
		return messageWorkflowUnavailable
	}
	if strings.TrimSpace(run.ArgoWorkflowUID) == "" && isBatchSubtaskPlaceholderWorkflowName(run.WorkflowName) {
		return messageWorkflowAwaitingDeploy
	}
	return messageWorkflowUnavailable
}

func (uc *Usecase) markRunWorkflowNotFound(ctx context.Context, run *models.PipelineRun) {
	slog.Warn("markRunWorkflowNotFound called",
		"runID", run.ID,
		"workflowName", run.WorkflowName,
		"currentStatus", run.Status,
		"currentMessage", run.Message,
	)
	shouldPreserve := shouldPreserveActiveWorkflowNotFound(run, uc.nowUTC())
	slog.Info("markRunWorkflowNotFound preserve check",
		"runID", run.ID,
		"workflowName", run.WorkflowName,
		"status", run.Status,
		"shouldPreserve", shouldPreserve,
		"isActiveDeployment", isActiveDeploymentStatus(run.Status),
	)
	if shouldPreserve {
		slog.Info("markRunWorkflowNotFound preserved active",
			"runID", run.ID,
			"newStatus", run.Status,
		)
		if isStaleWorkflowUnavailableMessage(run.Message) {
			run.Message = ""
		}
		run.FinishedAt = nil
		uc.persistRunObservation(ctx, run)
		return
	}
	if uc.reconcileTerminalRunFromLedger(ctx, run) {
		return
	}
	if isPendingBatchWorkflowCreation(run) {
		if isStaleWorkflowUnavailableMessage(run.Message) {
			run.Message = ""
			uc.persistRunObservation(ctx, run)
		}
		return
	}
	if run.FinishedAt == nil || run.FinishedAt.IsZero() {
		now := time.Now().UTC()
		run.FinishedAt = &now
	}
	run.Status = deploymentStatusExpired
	run.Message = workflowUnavailableMessage(run)
	slog.Warn("markRunWorkflowNotFound writing terminal status (non-preserve path)",
		"runID", run.ID,
		"workflowName", run.WorkflowName,
		"newStatus", run.Status,
		"newMessage", run.Message,
		"createdAt", run.CreatedAt,
		"ageHours", time.Now().UTC().Sub(run.CreatedAt).Hours(),
	)
	uc.persistRunObservation(ctx, run)
}

func shouldPreserveActiveWorkflowNotFound(run *models.PipelineRun, now time.Time) bool {
	if run == nil || !isActiveDeploymentStatus(run.Status) {
		return false
	}
	return runAgeWithinStaleLimit(run, now)
}

func shouldReviveMisclassifiedWorkflowNotFound(run *models.PipelineRun, now time.Time) bool {
	if run == nil || !isMisclassifiedTerminalRunStatus(run.Status) || !isStaleWorkflowUnavailableMessage(run.Message) {
		return false
	}
	return runAgeWithinStaleLimit(run, now)
}

func runAgeWithinStaleLimit(run *models.PipelineRun, now time.Time) bool {
	if run == nil {
		return false
	}
	ref := run.CreatedAt
	if run.StartedAt != nil && !run.StartedAt.IsZero() {
		ref = *run.StartedAt
	}
	if ref.IsZero() {
		return false
	}
	return now.Sub(ref) < staleActiveRunMaxAge
}

func (uc *Usecase) reconcileMisclassifiedRunFromArgo(ctx context.Context, run *models.PipelineRun, nodeMode nodeProjectionMode) {
	if uc.wfClient == nil || run == nil {
		return
	}
	if strings.TrimSpace(run.WorkflowName) == "" {
		return
	}
	if !isMisclassifiedTerminalRunStatus(run.Status) && !isStaleWorkflowUnavailableMessage(run.Message) {
		return
	}
	// CYB-3672: past-revival-age terminal runs whose message carries no real
	// diagnostic will never come back from Argo — the workflow was TTL-GCed
	// and every GetWorkflow will 404. Skip the lookup (and its ERROR log).
	// Observed on shared argo-server: ~1 req/s steady load per legacy stranded
	// run across all namespaces.
	//
	// Runs with a real diagnostic message (e.g., "Kubernetes 调度失败：..."
	// or scheduler / image-pull error text) still hit Argo — the operator
	// might benefit from a live re-check even beyond the revival window.
	//
	// runHasCreatedTimestamp gates the age check to real DB rows; test
	// fixtures that leave CreatedAt zero still exercise the full flow.
	if isMisclassifiedTerminalRunStatus(run.Status) &&
		runHasCreatedTimestamp(run) &&
		!runAgeWithinStaleLimit(run, uc.nowUTC()) &&
		isNoisyMisclassifiedMessage(run.Message) {
		return
	}
	if isPendingBatchWorkflowCreation(run) && shouldWaitForWorkflowCreation(run, time.Now().UTC()) {
		run.Status = "Pending"
		run.Message = ""
		run.FinishedAt = nil
		uc.persistRunObservation(ctx, run)
		return
	}
	namespace := run.ArgoNamespace
	if namespace == "" {
		namespace = uc.namespace
	}
	client, resolveErr := uc.resolveArgoClientForRun(ctx, run)
	if resolveErr != nil || client == nil {
		slog.Warn("reconcileMisclassifiedRunFromArgo: no argo client for run cluster",
			"runID", run.ID, "clusterID", uc.resolveRunClusterID(ctx, run), "err", resolveErr)
		return
	}
	wf, err := client.GetWorkflow(ctx, run.WorkflowName, namespace)
	if err != nil || wf == nil {
		slog.Warn("reconcileMisclassifiedRunFromArgo GetWorkflow failed",
			"runID", run.ID,
			"workflowName", run.WorkflowName,
			"err", err,
		)
		if errors.Is(err, argo.ErrUnexpectedNotFound) {
			slog.Warn("reconcileMisclassifiedRunFromArgo: unexpected 404 (config error) — not Argo, skip",
				"runID", run.ID, "workflowName", run.WorkflowName)
			return
		}
		if errors.Is(err, argo.ErrNotFound) {
			if uc.reconcileTerminalRunFromLedger(ctx, run) {
				return
			}
			if shouldReviveMisclassifiedWorkflowNotFound(run, uc.nowUTC()) {
				slog.Info("reconcileMisclassifiedRunFromArgo reviving run",
					"runID", run.ID, "workflowName", run.WorkflowName)
				run.Status = string(wfv1.WorkflowRunning)
				run.Message = ""
				run.FinishedAt = nil
				uc.persistRunObservation(ctx, run)
			} else {
				slog.Info("reconcileMisclassifiedRunFromArgo not reviving",
					"runID", run.ID, "workflowName", run.WorkflowName)
			}
		}
		return
	}
	uc.applyWorkflowToRun(ctx, run, wf, nodeMode)
}

// RefreshRunForList performs a bounded status refresh for batch list views.
// It avoids full run enrichment and is safe to call per page item.
func (uc *Usecase) RefreshRunForList(ctx context.Context, run *models.PipelineRun) {
	if uc.runRepo == nil || run == nil || strings.TrimSpace(run.ID) == "" {
		return
	}
	if !needsRunListRefresh(run) && !needsMisclassifiedReconcile(run) {
		runstate.AnnotateRunDiagnostics(run)
		return
	}
	if isActiveDeploymentStatus(run.Status) {
		uc.refreshRunStatus(ctx, run, nodeProjectTerminalArchive)
	}
	uc.reconcileMisclassifiedRunFromArgo(ctx, run, nodeProjectTerminalArchive)
	if isActiveDeploymentStatus(run.Status) && !isStaleWorkflowUnavailableMessage(run.Message) {
		if fresh, err := uc.runRepo.FindByID(ctx, run.ID); err == nil && fresh != nil {
			*run = *fresh
		}
		runstate.AnnotateRunDiagnostics(run)
		return
	}
	uc.reconcileTerminalRunFromLedger(ctx, run)
	if fresh, err := uc.runRepo.FindByID(ctx, run.ID); err == nil && fresh != nil {
		*run = *fresh
	}
	runstate.AnnotateRunDiagnostics(run)
}

func needsMisclassifiedReconcile(run *models.PipelineRun) bool {
	if run == nil {
		return false
	}
	// Only reconcile truly terminal misclassified statuses (Failed/Error/Expired).
	// Running + stale message is not misclassified — normalizeActiveRunRuntimeField
	// handles cleaning up the message locally without an Argo call.
	//
	// NOTE (CYB-3672): the age-based early-exit is enforced INSIDE
	// reconcileMisclassifiedRunFromArgo, not here. `now` is not available in
	// this signature and callers of needsMisclassifiedReconcile also drive the
	// non-reconcile display-annotation path — we still want to route past-age
	// runs through that path.
	return isMisclassifiedTerminalRunStatus(run.Status)
}

func needsRunListRefresh(run *models.PipelineRun) bool {
	if run == nil {
		return false
	}
	if isActiveDeploymentStatus(run.Status) {
		return true
	}
	if needsLedgerReconcile(run) {
		return true
	}
	return needsMisclassifiedReconcile(run)
}

func needsLedgerReconcile(run *models.PipelineRun) bool {
	if run == nil || run.FinishedAt == nil || run.FinishedAt.IsZero() {
		return false
	}
	return isActiveDeploymentStatus(run.Status)
}

// reconcileTerminalRunFromLedger infers a terminal run status from durable
// asset-node rows when Argo has already TTL'd the workflow CR.
func (uc *Usecase) reconcileTerminalRunFromLedger(ctx context.Context, run *models.PipelineRun) bool {
	if uc.runRepo == nil || run == nil {
		return false
	}
	if uc.assetNodeRepo == nil {
		return false
	}
	needsReconcile := needsLedgerReconcile(run)
	isActive := isActiveDeploymentStatus(run.Status)
	isStale := isStaleWorkflowUnavailableMessage(run.Message)
	if !needsReconcile && !isActive && !isStale {
		return false
	}
	result, err := uc.assetNodeRepo.ListByRunID(ctx, run.ID, models.PipelineRunAssetNodeListOptions{Limit: 500})
	if err != nil || result == nil || len(result.Items) == 0 {
		slog.Info("reconcileTerminalRunFromLedger no asset nodes",
			"runID", run.ID,
			"workflowName", run.WorkflowName,
			"currentStatus", run.Status,
			"needsReconcile", needsReconcile,
			"isActive", isActive,
		)
		return false
	}
	status, message, ok := inferTerminalRunFromAssetNodes(result.Items)
	if !ok {
		slog.Info("reconcileTerminalRunFromLedger cannot infer",
			"runID", run.ID,
			"workflowName", run.WorkflowName,
			"currentStatus", run.Status,
			"assetNodeCount", len(result.Items),
		)
		return false
	}
	slog.Warn("reconcileTerminalRunFromLedger overriding status",
		"runID", run.ID,
		"workflowName", run.WorkflowName,
		"oldStatus", run.Status,
		"newStatus", status,
		"newMessage", message,
		"assetNodeCount", len(result.Items),
	)
	run.Status = status
	if trimmed := strings.TrimSpace(message); trimmed != "" && (strings.TrimSpace(run.Message) == "" || isStaleWorkflowUnavailableMessage(run.Message)) {
		run.Message = trimmed
	}
	if run.FinishedAt == nil || run.FinishedAt.IsZero() {
		now := uc.nowUTC()
		run.FinishedAt = &now
	}
	uc.persistRunObservation(ctx, run)
	return true
}

// recoverStuckSucceededRun corrects a run that a prior revival race (CYB-3707)
// left persisted as Succeeded while its durable asset-node ledger still shows a
// failed/errored leaf. Bounded to the single-run detail read path. The resulting
// Succeeded→Failed correction is terminal→terminal and is permitted by
// persistRunObservation (only →active regressions are guarded). A run whose
// ledger agrees it succeeded (or cannot be inferred) is left untouched.
func (uc *Usecase) recoverStuckSucceededRun(ctx context.Context, run *models.PipelineRun) {
	if run == nil || uc.assetNodeRepo == nil || !isSucceededRunStatus(run.Status) {
		return
	}
	result, err := uc.assetNodeRepo.ListByRunID(ctx, run.ID, models.PipelineRunAssetNodeListOptions{Limit: 500})
	if err != nil || result == nil || len(result.Items) == 0 {
		return
	}
	status, message, ok := inferTerminalRunFromAssetNodes(result.Items)
	if !ok || !isTerminalFailureRunStatus(status) {
		return
	}
	slog.Warn("recoverStuckSucceededRun correcting mislabeled succeeded run",
		"runID", run.ID,
		"workflowName", run.WorkflowName,
		"oldStatus", run.Status,
		"newStatus", status,
	)
	run.Status = status
	if trimmed := strings.TrimSpace(message); trimmed != "" {
		run.Message = trimmed
	}
	if run.FinishedAt == nil || run.FinishedAt.IsZero() {
		now := uc.nowUTC()
		run.FinishedAt = &now
	}
	uc.persistRunObservation(ctx, run)
}

func inferRunStatusFromAssetNodes(nodes []models.PipelineRunAssetNode) (string, bool) {
	status, _, ok := inferTerminalRunFromAssetNodes(nodes)
	return status, ok
}

func inferTerminalRunFromAssetNodes(nodes []models.PipelineRunAssetNode) (string, string, bool) {
	if len(nodes) == 0 {
		return "", "", false
	}
	hasRunning := false
	hasPending := false
	failureStatus := ""
	failureMessage := ""
	for _, node := range nodes {
		switch strings.ToLower(strings.TrimSpace(node.Status)) {
		case "error":
			if failureStatus == "" || failureStatus == string(wfv1.WorkflowFailed) {
				failureStatus = string(wfv1.WorkflowError)
				failureMessage = strings.TrimSpace(node.Message)
			}
		case "failed":
			if failureStatus == "" {
				failureStatus = string(wfv1.WorkflowFailed)
				failureMessage = strings.TrimSpace(node.Message)
			}
		case "running":
			hasRunning = true
		case "pending":
			if isImageStartupFailureMessage(node.Message) {
				if failureStatus == "" {
					failureStatus = string(wfv1.WorkflowError)
					failureMessage = strings.TrimSpace(node.Message)
				}
				continue
			}
			hasPending = true
		case "succeeded", "success", "skipped", "omitted", "completed":
			// terminal success path
		default:
			hasPending = true
		}
	}
	if failureStatus != "" && !hasRunning {
		return failureStatus, failureMessage, true
	}
	if hasRunning || hasPending {
		return "", "", false
	}
	return string(wfv1.WorkflowSucceeded), "", true
}

func (uc *Usecase) refreshRunStatus(ctx context.Context, run *models.PipelineRun, nodeMode nodeProjectionMode) {
	if run == nil || !isActiveDeploymentStatus(run.Status) {
		return
	}
	if strings.TrimSpace(run.WorkflowName) == "" {
		return
	}
	namespace := run.ArgoNamespace
	if namespace == "" {
		namespace = uc.namespace
	}
	// CYB-3486d5: resolve client from run.ClusterID so runs on non-default
	// clusters (delivery-clust) hit the right K8s API. Factory-less test
	// deployments still fall back to uc.wfClient (may itself be nil).
	client, resolveErr := uc.resolveArgoClientForRun(ctx, run)
	if resolveErr != nil {
		slog.Warn("refreshRunStatus: resolve argo client failed",
			"runID", run.ID, "clusterID", uc.resolveRunClusterID(ctx, run), "err", resolveErr)
		return
	}
	if client == nil {
		return
	}
	wf, err := client.GetWorkflow(ctx, run.WorkflowName, namespace)
	if err != nil {
		// Argo TTL-cleans finished workflows, so refresh sees plain 404 on
		// most runs after a while. That's normal — mark run as done or wait,
		// but don't spam Warn on it. Only surface the truly unexpected paths.
		if errors.Is(err, argo.ErrUnexpectedNotFound) {
			slog.Warn("refreshRunStatus: unexpected 404 (config error) -- skip markRunWorkflowNotFound",
				"runID", run.ID, "workflowName", run.WorkflowName)
			return
		}
		if errors.Is(err, argo.ErrNotFound) {
			if shouldWaitForWorkflowCreation(run, time.Now().UTC()) {
				if isPendingBatchWorkflowCreation(run) && isStaleWorkflowUnavailableMessage(run.Message) {
					run.Message = ""
					uc.persistRunObservation(ctx, run)
				}
				return
			}
			uc.markRunWorkflowNotFound(ctx, run)
			return
		}
		slog.Warn("refreshRunStatus GetWorkflow failed",
			"runID", run.ID,
			"err", err,
		)
		return
	}
	if wf == nil {
		return
	}
	uc.applyWorkflowToRun(ctx, run, wf, nodeMode)
}

func (uc *Usecase) refreshPipelineRunStatus(ctx context.Context, run *models.PipelineRun) {
	if uc.runRepo == nil {
		return
	}
	uc.refreshRunStatus(ctx, run, nodeProjectTerminalArchive)
}

// refreshPipelineRunStatusLive is the single-run drill-in variant: it also
// projects live per-node rows so the run detail view shows fresh step state
// while the workflow is running (bounded: one workflow per call).
func (uc *Usecase) refreshPipelineRunStatusLive(ctx context.Context, run *models.PipelineRun) {
	if uc.runRepo == nil {
		return
	}
	uc.refreshRunStatus(ctx, run, nodeProjectLive)
}

// backfillRunStatus refreshes a run from Argo regardless of its current status.
// Unlike refreshRunStatus, this does NOT skip completed runs — it's used by the
// watcher backfill to sync events for runs that completed before the watcher scanned them.
func (uc *Usecase) backfillRunStatus(ctx context.Context, run *models.PipelineRun) {
	if run == nil {
		return
	}
	namespace := run.ArgoNamespace
	if namespace == "" {
		namespace = uc.namespace
	}
	// CYB-3486d5: resolve per-run cluster so backfill on non-default cluster
	// runs polls the right K8s API.
	client, resolveErr := uc.resolveArgoClientForRun(ctx, run)
	if resolveErr != nil {
		slog.Warn("backfillRunStatus: resolve argo client failed",
			"runID", run.ID, "clusterID", uc.resolveRunClusterID(ctx, run), "err", resolveErr)
		return
	}
	if client == nil {
		return
	}
	wf, err := client.GetWorkflow(ctx, run.WorkflowName, namespace)
	if err != nil {
		if errors.Is(err, argo.ErrUnexpectedNotFound) {
			slog.Warn("backfillRunStatus: unexpected 404 (config error) -- skip ledger update",
				"runID", run.ID, "workflowName", run.WorkflowName)
			return
		}
		if errors.Is(err, argo.ErrNotFound) {
			run.LedgerState = "no_ledger"
			logPipelineSideEffect("update pipeline run ledger state",
				uc.runRepo.UpdateLedgerState(ctx, run.ID, "no_ledger"))
		}
		return
	}
	if wf == nil {
		return
	}
	uc.appendWorkflowEvents(ctx, run, wf)
	uc.replaceRunNodesFromWorkflow(ctx, run, wf)
	run.LedgerState = "has_ledger"
	logPipelineSideEffect("update pipeline run ledger state + backfill completed",
		uc.runRepo.UpdateLedgerState(ctx, run.ID, "has_ledger"))
}

// RefreshRunFromWorkflowByName is the entry point for the run status push
// webhook (CYB-3058). The webhook payload is a trigger ("poke") only: DataBrew
// resolves the run by workflow name, cross-checks the workflow UID, fetches the
// authoritative workflow state from Argo, and applies it (status + events +
// nodes) via applyWorkflowToRun. The payload phase is never trusted, so a
// forged/replayed call cannot inject false status. Returns (nil, nil) when no
// run matches the workflow name (handler maps to 404).
func (uc *Usecase) RefreshRunFromWorkflowByName(ctx context.Context, workflowName, workflowUID string) (*models.PipelineRun, error) {
	if uc.runRepo == nil {
		return nil, nil
	}
	workflowName = strings.TrimSpace(workflowName)
	if workflowName == "" {
		return nil, nil
	}
	run, err := uc.runRepo.FindByWorkflowName(ctx, workflowName)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, nil
	}
	// Guard against workflow-name reuse across generations: if we know the UID
	// and the payload carries a different one, skip (idempotent no-op).
	if uid := strings.TrimSpace(workflowUID); uid != "" {
		if known := strings.TrimSpace(run.ArgoWorkflowUID); known != "" && known != uid {
			slog.Warn("RefreshRunFromWorkflowByName: workflow UID mismatch, ignoring poke",
				"runID", run.ID, "workflowName", workflowName,
				"knownUID", known, "payloadUID", uid)
			return run, nil
		}
	}
	namespace := run.ArgoNamespace
	if namespace == "" {
		namespace = uc.namespace
	}
	// CYB-3486d5: resolve per-run cluster so the run-status push webhook
	// routes to the right K8s API for non-default cluster runs.
	client, resolveErr := uc.resolveArgoClientForRun(ctx, run)
	if resolveErr != nil {
		return run, resolveErr
	}
	if client == nil {
		return run, nil
	}
	wf, err := client.GetWorkflow(ctx, run.WorkflowName, namespace)
	if err != nil {
		if errors.Is(err, argo.ErrNotFound) || errors.Is(err, argo.ErrUnexpectedNotFound) {
			slog.Warn("RefreshRunFromWorkflowByName: GetWorkflow not found, skipping",
				"runID", run.ID, "workflowName", workflowName, "err", err)
			return run, nil
		}
		return run, err
	}
	if wf == nil {
		return run, nil
	}
	uc.applyWorkflowToRun(ctx, run, wf, nodeProjectTerminalArchive)
	return run, nil
}

// SyncActiveRunEvents refreshes active runs from Argo and records durable
// workflow/node/pod transition events. It is safe to call repeatedly because
// event writes are idempotent.
func (uc *Usecase) SyncActiveRunEvents(ctx context.Context, limit int) (int, error) {
	if uc.runRepo == nil || uc.wfClient == nil || uc.runEventRepo == nil {
		return 0, nil
	}
	if limit <= 0 {
		limit = 50
	}
	var prior *models.PipelineRunWatcherState
	if uc.watcherRepo != nil {
		if state, err := uc.watcherRepo.FindByID(ctx, "default"); err == nil && state != nil && state.ActiveScanLimit > 0 {
			limit = state.ActiveScanLimit
			if limit > 50 {
				limit = 50
			}
			prior = state
		}
	}
	scanStartedAt := time.Now().UTC()
	nextState := cloneWatcherState(prior, limit)
	nextState.LastScanStartedAt = &scanStartedAt
	nextState.ActiveScanLimit = limit
	nextState.TotalScans++
	runs, err := uc.loadRunsForWatcherSync(ctx)
	if err != nil {
		if uc.watcherRepo != nil {
			now := time.Now().UTC()
			nextState.LastScanFinishedAt = &now
			nextState.LastErrorAt = &now
			nextState.LastError = err.Error()
			nextState.ConsecutiveFailures++
			nextState.TotalErrors++
			logPipelineSideEffect("save pipeline watcher state", uc.watcherRepo.Save(ctx, &nextState))
		}
		return 0, err
	}
	// CYB-3490: rotate the refresh window over ALL active runs instead of
	// always taking the first `limit` — with more actives than the cap, the
	// tail would otherwise never be refreshed (starvation). The cursor makes
	// coverage round-robin: every active run is visited within
	// ceil(actives/limit) cycles. This rotation is the drift sweep.
	activeIdx := make([]int, 0, len(runs))
	for i := range runs {
		if isActiveDeploymentStatus(runs[i].Status) {
			activeIdx = append(activeIdx, i)
		}
	}
	synced := 0
	if watcherBulkModeEnabled() {
		// CYB-3681: one LIST per (cluster, namespace) covers ALL active runs
		// per tick — no rotating window, no starvation. Every 10th scan
		// recalibrates (bypasses the resourceVersion change gate).
		//
		// The residual GET budget is watcherResidualGetBudget, NOT the scan
		// limit: short tasks finish and leave the active-workflow LIST almost
		// immediately, so their runs are resolved via a targeted GET, not the
		// LIST. Tying that budget to the 50-cap throttled writeback to ~50/scan
		// and a burst of 1000 short tasks drained for minutes. These GETs are
		// cheap single-workflow reads (cpu-throttling is off), so the budget is
		// generous.
		synced = uc.bulkSyncActiveRuns(ctx, runs, activeIdx, watcherResidualGetBudget, nextState.TotalScans%recalibrateEvery == 0)
	} else if n := len(activeIdx); n > 0 {
		start := uc.watcherActiveCursor % n
		// Pick the round-robin window FIRST — cursor semantics unchanged from
		// the pre-4d.5.b sequential loop; fairness across active runs is still
		// managed by uc.watcherActiveCursor.
		picked := make([]int, 0, limit)
		for k := 0; k < n && len(picked) < limit; k++ {
			picked = append(picked, activeIdx[(start+k)%n])
		}
		// CYB-3486 4d.5.b: fan out by cluster so one slow cluster's K8s API
		// doesn't stall refreshes for other clusters. Runs within the SAME
		// cluster stay sequential — the same K8s API server would be hit
		// anyway, and serializing keeps the client-go rate limiter (QPS=5
		// default) from becoming a per-cluster stampede.
		byCluster := groupRunIndicesByCluster(runs, picked, func(r *models.PipelineRun) string {
			return uc.resolveRunClusterID(ctx, r)
		})
		var wg sync.WaitGroup
		for _, group := range byCluster {
			wg.Add(1)
			go func(g []int) {
				defer wg.Done()
				for _, i := range g {
					uc.refreshPipelineRunStatus(ctx, &runs[i])
				}
			}(group)
		}
		wg.Wait()
		synced = len(picked)
		uc.watcherActiveCursor = (start + synced) % n
	}
	anomalyLimit := watcherAnomalyReconcileLimit(limit)
	anomalyReconciled := 0
	now := time.Now().UTC()
	for i := range runs {
		if anomalyReconciled >= anomalyLimit {
			break
		}
		if !needsWatcherAnomalyReconcile(&runs[i], now) {
			continue
		}
		uc.RefreshRunForList(ctx, &runs[i])
		anomalyReconciled++
	}
	synced += anomalyReconciled
	// Backfill: scan completed runs (within the last 7 days) that may be
	// missing their ledger events (e.g. they completed before watcher scan).
	backfillLimit := limit / 2
	if backfillLimit < 5 {
		backfillLimit = 5
	}
	backfilled := 0
	since := time.Now().UTC().AddDate(0, 0, -7)
	for i := range runs {
		if backfilled >= backfillLimit {
			break
		}
		if isActiveDeploymentStatus(runs[i].Status) {
			continue
		}
		if runs[i].FinishedAt == nil || runs[i].FinishedAt.Before(since) {
			continue
		}
		if runs[i].LedgerState == "has_ledger" {
			continue
		}
		uc.backfillRunStatus(ctx, &runs[i])
		backfilled++
	}
	synced += backfilled
	if uc.watcherRepo != nil {
		now := time.Now().UTC()
		lag := int64(0)
		if nextState.LastSuccessAt != nil {
			lag = int64(now.Sub(*nextState.LastSuccessAt).Seconds())
			if lag < 0 {
				lag = 0
			}
		}
		nextState.LastSyncedAt = &now
		nextState.LastScanFinishedAt = &now
		nextState.LastSuccessAt = &now
		nextState.LastSyncedRunCount = synced
		nextState.ConsecutiveFailures = 0
		nextState.LastError = ""
		nextState.ScanLagSeconds = &lag
		uc.setWatcherLedgerHealth(computeLedgerHealth(runs, &now))
		logPipelineSideEffect("save pipeline watcher state", uc.watcherRepo.Save(ctx, &nextState))
	}
	return synced, nil
}

func (uc *Usecase) loadRunsForWatcherSync(ctx context.Context) ([]models.PipelineRun, error) {
	if uc.runRepo == nil {
		return nil, nil
	}
	// The candidate set has three consumers: (1) active-run refresh, (2)
	// terminal-anomaly reconcile (Error/Expired runs re-verified against Argo),
	// (3) ledger backfill for recently-completed runs. It is the UNION of:
	//
	//   • ALL active-status runs (CYB-3681 load-test fix) — the old "most
	//     recent 500" window silently broke on bursts: 1000 short tasks finish
	//     near-simultaneously, the 500 newest rows fill with just-completed
	//     runs, and older still-active runs fall outside the window → never
	//     refreshed → stuck "Running" forever though Argo finished them.
	//     Filtering by active status makes this the true in-flight universe,
	//     bounded by concurrent work not history, so no active run is invisible.
	//   • a recent window — feeds the terminal-repair paths (2)/(3), which
	//     operate on non-active recently-finished runs the active query omits.
	//
	// The CYB-3491 concern (unbounded 39s scan) stays addressed: both halves
	// are status-indexed and hard-capped.
	// Load the active-run rotating page from the current cursor. CYB-3746:
	// a single-page cap of watcherActiveRunLoadCap is smaller than the total
	// active set under multi-batch load (3 × 9999-item batches ≈ 30k active),
	// so paging AFTER the last-seen (createdAt, id) is how the loader covers
	// everything eventually instead of pinning the newest window forever.
	uc.watcherLoadCursorMu.Lock()
	cursorCreatedAt := uc.watcherLoadCursorCreatedAt
	cursorID := uc.watcherLoadCursorID
	uc.watcherLoadCursorMu.Unlock()
	active, err := uc.runRepo.FindActiveRunSummariesAfter(ctx, cursorCreatedAt, cursorID, watcherActiveRunLoadCap)
	if err != nil {
		return nil, err
	}
	// Advance cursor on a full page; wrap to newest when the page returns
	// short (either the tail was reached, or the active set shrank below the
	// cursor). The wrap is what makes rotation cyclic instead of one-way.
	uc.watcherLoadCursorMu.Lock()
	if len(active) == watcherActiveRunLoadCap && len(active) > 0 {
		last := active[len(active)-1]
		uc.watcherLoadCursorCreatedAt = last.CreatedAt
		uc.watcherLoadCursorID = last.ID
	} else {
		uc.watcherLoadCursorCreatedAt = time.Time{}
		uc.watcherLoadCursorID = ""
	}
	uc.watcherLoadCursorMu.Unlock()
	// Recent-500 union is unchanged: it always includes the freshest activity
	// (any status), so a just-created run in an in-flight batch is refreshed
	// every tick even while the rotating page is elsewhere.
	recent, _, err := uc.runRepo.ListSummaries(ctx, models.PipelineRunListFilter{PageSize: 500})
	if err != nil {
		return nil, err
	}
	return mergeRunsByID(active, recent), nil
}

// mergeRunsByID concatenates two run slices, dropping the second slice's
// duplicates (matched by ID). Order: all of a, then b's new entries.
func mergeRunsByID(a, b []models.PipelineRun) []models.PipelineRun {
	seen := make(map[string]struct{}, len(a)+len(b))
	out := make([]models.PipelineRun, 0, len(a)+len(b))
	for _, r := range a {
		if _, dup := seen[r.ID]; dup {
			continue
		}
		seen[r.ID] = struct{}{}
		out = append(out, r)
	}
	for _, r := range b {
		if _, dup := seen[r.ID]; dup {
			continue
		}
		seen[r.ID] = struct{}{}
		out = append(out, r)
	}
	return out
}

// watcherActiveRunLoadCap bounds the active-run candidate set per scan. Sized
// well above steady-state in-flight volume so a large burst is fully covered
// in one scan; a status-indexed query keeps it cheap even against a big
// pipeline_runs table.
const watcherActiveRunLoadCap = 3000

// watcherResidualGetBudget caps targeted GetWorkflow calls per scan for runs
// whose workflow is absent from the active-workflow LIST (finished short
// tasks, orphans). Decoupled from the scan limit so bursty short-task
// writeback isn't throttled to ~50/scan.
const watcherResidualGetBudget = 600

func computeLedgerHealth(runs []models.PipelineRun, lastBackfill *time.Time) models.LedgerHealth {
	total := len(runs)
	hasEvents := 0
	for i := range runs {
		if runs[i].LedgerState == "has_ledger" {
			hasEvents++
		}
	}
	return models.LedgerHealth{
		TotalRuns:      total,
		RunsWithEvents: hasEvents,
		RunsWithout:    total - hasEvents,
		LastBackfillAt: lastBackfill,
	}
}

func (uc *Usecase) setWatcherLedgerHealth(health models.LedgerHealth) {
	uc.watcherLedgerMu.Lock()
	uc.watcherLedgerHealth = health
	uc.watcherLedgerMu.Unlock()
}

func (uc *Usecase) watcherLedgerHealthSnapshot() models.LedgerHealth {
	uc.watcherLedgerMu.RLock()
	defer uc.watcherLedgerMu.RUnlock()
	return uc.watcherLedgerHealth
}

func watcherAnomalyReconcileLimit(limit int) int {
	if limit <= 0 {
		limit = 100
	}
	n := limit / 4
	if n < 5 {
		n = 5
	}
	if n > 25 {
		n = 25
	}
	return n
}

func needsWatcherAnomalyReconcile(run *models.PipelineRun, now time.Time) bool {
	if run == nil || isActiveDeploymentStatus(run.Status) {
		return false
	}
	if strings.TrimSpace(run.WorkflowName) == "" {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(run.LedgerState), "has_ledger") {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(run.LedgerState), "no_ledger") {
		return false
	}
	if isStaleWorkflowUnavailableMessage(run.Message) {
		return runObservedRecently(run, now, 7*24*time.Hour)
	}
	if !needsMisclassifiedReconcile(run) {
		return false
	}
	return runObservedRecently(run, now, 7*24*time.Hour)
}

// runObservedRecently reports whether the run's lifecycle timeline falls inside
// the given window. It intentionally uses only immutable-or-terminal timestamps
// (FinishedAt → StartedAt → CreatedAt) and skips UpdatedAt. UpdatedAt is bumped
// on every Save, so every observation of the run resets it — feeding it back
// into "recently observed" turns the reconcile treadmill into a perpetual loop
// (a 6-month-old TTL-cleaned run keeps re-entering the anomaly set forever).
// See CYB-3516.
func runObservedRecently(run *models.PipelineRun, now time.Time, window time.Duration) bool {
	if run == nil || window <= 0 {
		return false
	}
	ref := run.FinishedAt
	if ref == nil || ref.IsZero() {
		ref = run.StartedAt
	}
	if ref == nil || ref.IsZero() {
		ref = &run.CreatedAt
	}
	if ref == nil || ref.IsZero() {
		return false
	}
	return now.Sub(*ref) >= 0 && now.Sub(*ref) < window
}

// GetRunWatcherStatus returns the persisted pipeline watcher health snapshot.
func (uc *Usecase) GetRunWatcherStatus(ctx context.Context) (*models.PipelineRunWatcherState, error) {
	if uc.watcherRepo == nil {
		return watcherStateWithHealth(&models.PipelineRunWatcherState{
			ID:              "default",
			ActiveScanLimit: 100,
			LastError:       "pipeline run watcher state repository is not configured",
		}), nil
	}
	state, err := uc.watcherRepo.FindByID(ctx, "default")
	if err != nil {
		return nil, err
	}
	if state == nil {
		state = &models.PipelineRunWatcherState{ID: "default", ActiveScanLimit: 50}
	}
	if cached := uc.watcherLedgerHealthSnapshot(); cached.TotalRuns > 0 {
		state.LedgerHealth = cached
	}
	return watcherStateWithHealth(state), nil
}

// StartRunEventWatcher starts a polling watcher for active pipeline runs.
func (uc *Usecase) StartRunEventWatcher(ctx context.Context, interval time.Duration, limit int) {
	if interval <= 0 {
		interval = 3 * time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := uc.SyncActiveRunEvents(ctx, limit); err != nil {
					slog.Warn("pipeline run event watcher sync failed", "err", err)
				}
			}
		}
	}()
}

func (uc *Usecase) resolveExecutionTargetForCompatibility(targetID string) (*models.ExecutionTarget, error) {
	if targetID == "" || targetID == "default" {
		target := uc.defaultExecutionTarget()
		return &target, nil
	}
	return nil, fmt.Errorf("%w: target_id=%q", ErrExecutionTargetNotFound, targetID)
}

// ── Templates ─────────────────────────────────────────────────────

// SaveTemplate persists a pipeline template with auto-incremented version.
func (uc *Usecase) SaveTemplate(ctx context.Context, name string, pipeline map[string]interface{}, scope string, owner string) (*models.PipelineTemplate, error) {
	raw, err := json.Marshal(pipeline)
	if err != nil {
		return nil, fmt.Errorf("marshal pipeline: %w", err)
	}
	pipe, _, err := rawToPipeline(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: parse pipeline: %v", ErrInvalidArgument, err)
	}
	transpiler.NormalizePipeline(pipe)
	if err := transpiler.ValidatePipeline(pipe); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidArgument, err)
	}
	normalizedRaw, err := json.Marshal(pipe)
	if err != nil {
		return nil, fmt.Errorf("marshal normalized pipeline: %w", err)
	}
	normalizedPipeline := map[string]interface{}{}
	if err := json.Unmarshal(normalizedRaw, &normalizedPipeline); err != nil {
		return nil, fmt.Errorf("unmarshal normalized pipeline: %w", err)
	}

	version, err := uc.templateRepo.GetNextVersion(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get next version: %w", err)
	}
	// Propagate active version from the existing latest (if any).
	activeVersion := 0
	if prev, _ := uc.templateRepo.FindByNameAndVersion(ctx, name, version-1); prev != nil {
		activeVersion = prev.ActiveVersion
	}
	t := &models.PipelineTemplate{
		ID:            uuid.New().String(),
		Name:          name,
		Version:       version,
		ActiveVersion: activeVersion,
		Scope:         scope,
		Owner:         owner,
		Pipeline:      normalizedPipeline,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	t.NodeCount = len(pipe.Nodes)
	if err := uc.templateRepo.Save(ctx, t); err != nil {
		return nil, fmt.Errorf("save template: %w", err)
	}
	return t, nil
}

// ListVersions returns all versions of a pipeline template. The identifier is
// normally a template id; name fallback preserves compatibility with older
// callers that used the route param as a template name.
func (uc *Usecase) ListVersions(ctx context.Context, templateIDOrName string) ([]models.PipelineTemplate, error) {
	name := templateIDOrName
	if t, err := uc.templateRepo.FindByID(ctx, templateIDOrName); err != nil {
		return nil, fmt.Errorf("find template: %w", err)
	} else if t != nil {
		name = t.Name
	}
	return uc.templateRepo.FindVersionsByName(ctx, name)
}

// SetActiveVersion pins a pipeline's default run version. When version is 0,
// the pin is cleared (latest = active).
func (uc *Usecase) SetActiveVersion(ctx context.Context, templateIDOrName string, version int) error {
	t, err := uc.templateRepo.FindByID(ctx, templateIDOrName)
	if err != nil {
		return fmt.Errorf("find template: %w", err)
	}
	if t == nil {
		return ErrTemplateNotFound
	}
	return uc.templateRepo.SetActiveVersion(ctx, t.Name, version)
}

// Promote copies a dev template to the prod scope.
var ErrProdLocked = errors.New("prod templates are read-only")

func (uc *Usecase) Promote(ctx context.Context, templateID string) (*models.PipelineTemplate, error) {
	t, err := uc.templateRepo.FindByID(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("find template: %w", err)
	}
	if t == nil {
		return nil, ErrTemplateNotFound
	}
	if t.Scope == "prod" {
		return nil, fmt.Errorf("template %s is already in prod scope", t.Name)
	}
	return uc.SaveTemplate(ctx, t.Name, t.Pipeline, "prod", t.Owner)
}

// ListTemplates returns all pipeline templates.
func (uc *Usecase) ListTemplates(ctx context.Context) ([]models.PipelineTemplate, error) {
	return uc.templateRepo.FindAll(ctx)
}

// ListTemplatesPaged returns a paginated pipeline template list.
func (uc *Usecase) ListTemplatesPaged(ctx context.Context, filter models.PipelineTemplateListFilter) ([]models.PipelineTemplate, int, error) {
	return uc.templateRepo.FindLatestPaged(ctx, filter)
}

// GetTemplate returns a pipeline template by id.
func (uc *Usecase) GetTemplate(ctx context.Context, id string) (*models.PipelineTemplate, error) {
	return uc.templateRepo.FindByID(ctx, id)
}

func (uc *Usecase) GetTemplateVersion(ctx context.Context, id string, version int) (*models.PipelineTemplate, error) {
	t, err := uc.templateRepo.FindByID(ctx, id)
	if err != nil || t == nil || version <= 0 || t.Version == version {
		return t, err
	}
	versioned, err := uc.templateRepo.FindByNameAndVersion(ctx, t.Name, version)
	if err != nil {
		return nil, err
	}
	if versioned != nil {
		return versioned, nil
	}
	return t, nil
}

// DeleteTemplate removes a pipeline template.
var ErrTemplateNotOwned = errors.New("template is not owned by current user")

func (uc *Usecase) DeleteTemplate(ctx context.Context, id, owner string) error {
	t, err := uc.templateRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("find template: %w", err)
	}
	if t == nil {
		return nil
	}
	if t.Scope == "prod" {
		return ErrProdLocked
	}
	if owner != "" && t.Owner != "" && t.Owner != owner {
		return ErrTemplateNotOwned
	}
	if uc.runRepo != nil {
		if err := uc.runRepo.DeleteByTemplateID(ctx, id); err != nil {
			return fmt.Errorf("delete template runs: %w", err)
		}
	}
	if err := uc.deploymentRepo.DeleteByTemplateID(ctx, id); err != nil {
		return fmt.Errorf("delete template deployments: %w", err)
	}
	return uc.templateRepo.Delete(ctx, id)
}

// ── Deploy ────────────────────────────────────────────────────────

const (
	ssDeliveryLerobotNodeKey   = "ss_delivery_lerobot"
	tonyDeliveryLerobotStepKey = "tony_delivery_lerobot"
	cyberpipeNodeEnvName       = "CYBERPIPE_NODE"
)

func applyCyberpipeNodeCompatibilityAliases(pipe *transpiler.Pipeline) {
	if pipe == nil {
		return
	}
	var walk func(nodes []transpiler.Node)
	walk = func(nodes []transpiler.Node) {
		for i := range nodes {
			for j := range nodes[i].Component.Env {
				env := &nodes[i].Component.Env[j]
				if env.Name == cyberpipeNodeEnvName && env.Value == ssDeliveryLerobotNodeKey {
					env.Value = tonyDeliveryLerobotStepKey
				}
			}
			if len(nodes[i].SubNodes) > 0 {
				walk(nodes[i].SubNodes)
			}
		}
	}
	walk(pipe.Nodes)
}

// costTrackingLabelPrefix namespaces cost-attribution labels so they read
// clearly in GKE Cost Allocation / BigQuery billing export alongside Argo's
// and GKE's own labels (workflows.argoproj.io/*, topology.kubernetes.io/*).
const costTrackingLabelPrefix = "cyber-databrew/"

// buildCostTrackingLabels returns the pod labels used for GKE Cost Allocation
// attribution, skipping any identifier that is unknown (empty) rather than
// emitting an empty-valued label.
func buildCostTrackingLabels(owner, runID, assetID string) map[string]string {
	labels := map[string]string{}
	if v := sanitizeLabelValue(owner); v != "" {
		labels[costTrackingLabelPrefix+"owner"] = v
	}
	// run-id attributes real GCP cost (BigQuery billing export) to a single run;
	// asset-id to the one video it processed, joinable to video_durations by
	// asset_id (CYB-3118). Both are omitted when unknown/ambiguous.
	if v := sanitizeLabelValue(runID); v != "" {
		labels[costTrackingLabelPrefix+"run-id"] = v
	}
	if v := sanitizeLabelValue(assetID); v != "" {
		labels[costTrackingLabelPrefix+"asset-id"] = v
	}
	return labels
}

// sanitizeLabelValue coerces raw into a valid Kubernetes label value:
// [a-zA-Z0-9] at each end, only [-_.a-zA-Z0-9] in between, max 63 chars.
// Owner identifiers are emails, so "@" is escaped rather than dropped to
// keep the value legible (e.g. "a@b.com" -> "a-at-b.com").
func sanitizeLabelValue(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		case r == '@':
			b.WriteString("-at-")
		default:
			b.WriteRune('-')
		}
	}
	out := b.String()
	if len(out) > 63 {
		out = out[:63]
	}
	return strings.Trim(out, "-_.")
}

// Deploy transpiles a pipeline and submits it as an Argo Workflow.
// pipelineArg is the raw pipeline JSON map. name overrides the workflow name.
// assetIDs are passed as workflow-level parameters (F4.1).
func (uc *Usecase) Deploy(
	ctx context.Context,
	pipelineArg map[string]interface{},
	name string,
	assetIDs []string,
	opts ...DeployOptions,
) (*models.PipelineDeployment, error) {
	// Marshal pipeline to JSON for transpiler.
	raw, err := json.Marshal(pipelineArg)
	if err != nil {
		return nil, fmt.Errorf("marshal pipeline: %w", err)
	}

	pipe, nodeCount, err := rawToPipeline(raw)
	if err != nil {
		return nil, fmt.Errorf("parse pipeline: %w", err)
	}
	transpiler.NormalizePipeline(pipe)
	if err := transpiler.ValidatePipeline(pipe); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidArgument, err)
	}
	normalizedRaw, err := json.Marshal(pipe)
	if err != nil {
		return nil, fmt.Errorf("marshal normalized pipeline: %w", err)
	}
	if err := json.Unmarshal(normalizedRaw, &pipelineArg); err != nil {
		return nil, fmt.Errorf("unmarshal normalized pipeline: %w", err)
	}
	delete(pipelineArg, runConfigInputsPipelineJSONKey)
	applyCyberpipeNodeCompatibilityAliases(pipe)

	pipeName := pipe.Name
	if name != "" {
		pipeName = name
	}

	depID := uuid.New().String()
	templateID := ""
	templateVersion := 0
	dryRun := false
	costOwner := ""
	if len(opts) > 0 {
		templateID = opts[0].TemplateID
		templateVersion = opts[0].TemplateVersion
		dryRun = opts[0].DryRun
		costOwner = opts[0].Owner
		if strings.TrimSpace(opts[0].PreallocatedRunID) != "" {
			depID = strings.TrimSpace(opts[0].PreallocatedRunID)
		}
	}
	// Workflow name = run id (depID). The Argo pod name is <wfName>-<step>-<hash>,
	// so leading with the run id lets any pod map straight to /runs/<id>
	// (CYB-3076). depID is a UUID → a valid RFC1123 name. pipeName is retained
	// for the run's display name only.
	wfName := depID
	resolveTargetID := ""
	if len(opts) > 0 {
		resolveTargetID = opts[0].TargetID
	}
	target, err := uc.resolveExecutionTarget(ctx, resolveTargetID)
	if err != nil {
		return nil, err
	}
	targetNamespace := target.Namespace
	if targetNamespace == "" {
		targetNamespace = uc.namespace
	}
	if err := uc.validateResourceCeilings(pipe, target); err != nil {
		return nil, err
	}

	normalizedAssetIDs, err := uc.validateDeployAssetIDs(ctx, assetIDs, opts...)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAssetNotFound, err)
	}
	assetIDs = normalizedAssetIDs

	var runtimeConfig *resolvedRuntimeConfig
	if len(opts) > 0 {
		runtimeConfig, err = uc.resolveRuntimeConfig(ctx, opts[0].ConfigSelection)
		if err != nil {
			return nil, err
		}
	}
	nodeRuntimeConfigs, err := uc.resolveNodeRuntimeConfigs(ctx, pipe)
	if err != nil {
		return nil, err
	}

	// Assemble workflow-level params and global env vars from asset IDs.
	var wfParams []transpiler.Param
	globalEnv := []transpiler.EnvVar{
		{Name: "PIPELINE_DEPLOYMENT_ID", Value: depID},
		{Name: "REQUEST_ID", Value: depID},
	}
	// assetRegistered records which input assets actually exist in the assets
	// table (filled from the Get below, no extra query). Used to skip the
	// pipeline_processing asset_events append for unregistered/external assets:
	// asset_events.asset_id has an FK to assets, so those inserts fail with
	// 23503 and — at batch scale, with AllowUnknownAssets — spammed thousands
	// of WARN "pipeline side effect failed" logs (cyb-3491).
	assetRegistered := make(map[string]bool, len(assetIDs))
	if len(assetIDs) > 0 {
		wfParams = append(wfParams, transpiler.Param{
			Name:  "asset_ids",
			Value: strings.Join(assetIDs, ","),
		})
		globalEnv = append(globalEnv,
			transpiler.EnvVar{Name: "ASSET_IDS", Value: strings.Join(assetIDs, ",")},
			transpiler.EnvVar{Name: "ASSET_COUNT", Value: fmt.Sprintf("%d", len(assetIDs))},
		)
		if len(assetIDs) == 1 {
			globalEnv = append(globalEnv,
				transpiler.EnvVar{Name: "VIDEO_ID", Value: assetIDs[0]},
				transpiler.EnvVar{Name: "ASSET_ID", Value: assetIDs[0]},
			)
		}
		for i, aid := range assetIDs {
			prefix := fmt.Sprintf("ASSET_%d_", i)
			globalEnv = append(globalEnv, transpiler.EnvVar{Name: prefix + "ID", Value: aid})
			if uc.assetRepo != nil {
				a, err := uc.assetRepo.Get(ctx, aid)
				if err == nil && a != nil {
					assetRegistered[aid] = true
					if a.StorageURI != "" {
						globalEnv = append(globalEnv, transpiler.EnvVar{Name: prefix + "STORAGE_URI", Value: a.StorageURI})
					}
					if a.AssetType != "" {
						globalEnv = append(globalEnv, transpiler.EnvVar{Name: prefix + "TYPE", Value: a.AssetType})
					}
				}
			}
		}
	}

	var extraVolumes []transpiler.Volume
	var runtimeConfigProjection *RuntimeConfigProjection
	if runtimeConfig != nil || len(nodeRuntimeConfigs) > 0 {
		projection := buildRuntimeConfigProjection(runtimeConfig, nodeRuntimeConfigs)
		// CYB-3680: content-addressed name — identical config content across a
		// batch collapses to ONE shared ConfigMap per namespace instead of one
		// per run, and the name needs no server-side identity so the CM can be
		// ensured BEFORE the Workflow exists.
		projection.ContentHash = runtimeConfigProjectionHash(projection)
		volumeName := runtimeConfigVolumeNameForContent(projection.ContentHash)
		projection.VolumeName = volumeName
		runtimeConfigProjection = &projection
		if runtimeConfig != nil {
			runtimeConfig.VolumeName = volumeName
		}
		if len(nodeRuntimeConfigs) == 0 {
			applyRuntimeConfigToPipeline(pipe, runtimeConfig, &globalEnv, &extraVolumes)
		} else {
			extraVolumes = append(extraVolumes, transpiler.Volume{
				Name:          volumeName,
				ConfigMapName: volumeName,
			})
			if runtimeConfig != nil {
				applyRuntimeConfigToUnboundNodes(pipe, runtimeConfig, boundNodeIDs(nodeRuntimeConfigs))
			}
			applyNodeRuntimeConfigs(pipe, nodeRuntimeConfigs, volumeName)
		}
	}
	if configInputs := materializedRuntimeConfigRunInputs(runtimeConfig, nodeRuntimeConfigs); len(configInputs) > 0 {
		pipelineArg[runConfigInputsPipelineJSONKey] = configInputs
	}
	if err := uc.applyRuntimeMounts(pipe, target); err != nil {
		return nil, err
	}

	// A run around exactly one asset is one video; label its cost with that
	// asset id (CYB-3118). Ambiguous (multi-asset) / no-asset runs are left
	// unlabelled so per-video billing stays unambiguous.
	costAssetID := ""
	if len(assetIDs) == 1 && assetIDs[0] != "no-asset" {
		costAssetID = assetIDs[0]
	}

	// Transpile to Argo Workflow.
	wfOpts := &transpiler.Options{
		Name:                 wfName,
		Namespace:            targetNamespace,
		ServiceAccount:       target.ServiceAccount,
		TemplateNodeSelector: executionTargetTemplateNodeSelector(target),
		TemplateTolerations:  executionTargetTemplateTolerations(target),
		GpuStepNodeSelector:  executionTargetGpuStepNodeSelector(target),
		TTLSecondsAfter:      uc.argoWorkflowTTLSecondsAfter(),
		WorkflowParams:       wfParams,
		GlobalEnv:            globalEnv,
		ExtraVolumes:         extraVolumes,

		ExitHookURL:             uc.argoRunWebhookURL,
		ExitHookTokenSecretName: uc.argoRunWebhookTokenSecretName,
		ExitHookTokenSecretKey:  uc.argoRunWebhookTokenSecretKey,
		ExitHookImage:           uc.argoRunWebhookImage,

		PodLabels: buildCostTrackingLabels(costOwner, depID, costAssetID),
	}
	// CYB-3486 pool: inject the pool's scheduling config onto every workflow
	// pod. Fully data-driven from the target's resource_defaults.scheduling
	// (JSONB) — the backend injects verbatim and hardcodes nothing about koord
	// / ElasticQuota / any scheduler. A Koordinator pool works because its
	// config declares schedulerName="koord-scheduler" + the EQ podLabel; a
	// plain namespace pool declares none and stays scheduler-agnostic.
	// Empty values are no-ops (transpiler leaves the field unset).
	wfOpts.PodLabels = mergePoolPodLabels(wfOpts.PodLabels, target)
	wfOpts.PodAnnotations = executionTargetPodAnnotations(target)
	wfOpts.PodPriorityClassName = executionTargetPriorityClassName(target)
	wfOpts.SchedulerName = executionTargetSchedulerName(target)
	wf, err := transpiler.Transpile(pipe, wfOpts)
	if err != nil {
		return nil, fmt.Errorf("transpile: %w", err)
	}

	// sigsyaml (sigs.k8s.io/yaml) round-trips through encoding/json first, so it
	// correctly calls resource.Quantity's MarshalJSON (producing e.g. "500m")
	// instead of yaml.v3's default reflection, which only sees Quantity's
	// exported Format field and silently drops the actual numeric value — see
	// CYB-3065. Must stay paired with the sigsyaml.Unmarshal callers
	// (reconstruct_from_db.go, resource_usage.go) for round-trip fidelity.
	manifestBytes, err := sigsyaml.Marshal(wf)
	if err != nil {
		return nil, fmt.Errorf("marshal manifest: %w", err)
	}
	manifest := string(manifestBytes)
	if manifest == "" {
		manifest = "{}\n"
	}

	if dryRun {
		var templateVersionPtr *int
		if templateVersion > 0 {
			templateVersionPtr = &templateVersion
		}
		return &models.PipelineDeployment{
			ID:              depID,
			PipelineName:    pipeName,
			WorkflowName:    wfName,
			Status:          "Preview",
			TemplateVersion: templateVersionPtr,
			NodeCount:       nodeCount,
			Manifest:        &manifest,
			PipelineJSON:    pipelineArg,
			AssetIDs:        assetIDs,
			AssetCount:      len(assetIDs),
			ExecutionTarget: target,
			CreatedAt:       time.Now().UTC(),
			UpdatedAt:       time.Now().UTC(),
		}, nil
	}

	// Submit to the runtime backend. Prefer the Run Kernel adapter; keep the
	// legacy Argo client path as a fallback while migration is in progress.
	if !uc.runtimeSubmitConfigured() {
		return nil, ErrWorkflowUnavailable
	}
	if runtimeConfigProjection != nil && uc.runtimeConfigStore == nil && uc.runtimeConfigStoreFactory == nil {
		return nil, fmt.Errorf("%w: runtime config store is not configured", ErrInvalidArgument)
	}
	// CYB-3486 PR 4c.1: resolve the target's argo client ONCE and use it for
	// every subsequent workflow op in this Deploy call. Previously each op
	// hit uc.wfClient (the process-global singleton), so a delivery-clust
	// submit would silently read/delete against cyber-clust.
	client, err := uc.resolveArgoClient(ctx, target)
	if err != nil {
		return nil, err
	}
	// CYB-3680: ensure the runtime-config ConfigMap BEFORE the Workflow. The
	// content-addressed name needs no server identity, so the historical
	// ordering (owner UID required the Workflow first) is gone — and with it
	// the crash window that stranded pods on a ConfigMap that was never
	// created (the Init:0/1 FailedMount class). Owner-less by design: the CM
	// is shared across runs; its lifecycle belongs to the TTL janitor, and
	// the DB blob is the rebuildable source of truth.
	if runtimeConfigProjection != nil {
		configStore, err := uc.resolveRuntimeConfigStore(ctx, target)
		if err != nil {
			return nil, fmt.Errorf("resolve runtime config store: %w", err)
		}
		if _, err := configStore.Create(ctx, targetNamespace, depID, *runtimeConfigProjection, nil); err != nil {
			return nil, fmt.Errorf("ensure runtime config: %w", err)
		}
		uc.persistRuntimeConfigBlob(ctx, runtimeConfigProjection)
	}

	status := "Pending"
	runtimeJob, err := uc.submitRuntimeWorkflow(ctx, client, depID, pipeName, wf, targetNamespace)
	if err != nil {
		if strings.Contains(err.Error(), "argo server URL is empty") {
			return nil, fmt.Errorf("%w: create workflow", ErrWorkflowUnavailable)
		}
		if errors.Is(err, ErrWorkflowUnavailable) {
			return nil, fmt.Errorf("%w: create workflow", ErrWorkflowUnavailable)
		}
		return nil, fmt.Errorf("create workflow: %w", err)
	}
	wfUID := ""
	if runtimeJob != nil {
		wfUID = strings.TrimSpace(runtimeJob.Ref.UID)
	}
	wfDetail := workflowFromRuntimeJob(runtimeJob)
	// (CYB-3680) The post-submit ConfigMap block — owner-UID lookup, create,
	// and its delete-workflow compensations — is gone: the CM is ensured
	// before submission, owner-less and content-addressed.
	if client != nil && (wfDetail == nil || wfUID == "" || status == "Pending") {
		phase, err := client.GetWorkflowStatus(ctx, wfName, targetNamespace)
		if err == nil && phase != "" {
			status = string(phase)
		}
		if detail, err := client.GetWorkflow(ctx, wfName, targetNamespace); err == nil && detail != nil {
			wfDetail = detail
		}
	}
	if wfDetail != nil {
		wfUID = string(wfDetail.UID)
		if wfDetail.Status.Phase != "" {
			status = string(wfDetail.Status.Phase)
		}
	}

	// The submit call returned no error, yet no Argo UID ever materialized — the
	// best-effort GetWorkflow above was rate-limited (or the CR isn't readable
	// yet). CRITICAL: the CR was almost certainly really created (CRD
	// CreateWorkflow only returns nil on a k8s 201), so we must NOT delete it —
	// that would tear down a live workflow. But persisting the run now would
	// leave a uid-less Pending row the batch submitter can never advance
	// (invariant ②), silently stranding the item. Return a RETRYABLE error so
	// the caller leaves the item pending and re-submits next cycle; the
	// deterministic workflow name makes that submit AlreadyExists → uid backfill.
	if strings.TrimSpace(wfUID) == "" {
		return nil, fmt.Errorf("%w: workflow %q", ErrWorkflowSubmitIncomplete, wfName)
	}

	runScope := "dev"
	runOwner := ""
	if len(opts) > 0 {
		runOwner = opts[0].Owner
	}
	dep := &models.PipelineDeployment{
		ID:              depID,
		PipelineName:    pipeName,
		WorkflowName:    wfName,
		Status:          status,
		NodeCount:       nodeCount,
		Manifest:        &manifest,
		PipelineJSON:    pipelineArg,
		AssetIDs:        assetIDs,
		AssetCount:      len(assetIDs),
		ExecutionTarget: target,
		Scope:           runScope,
		Owner:           runOwner,
		CreatedAt:       time.Now().UTC(),
	}
	if templateID != "" {
		dep.TemplateID = &templateID
	}
	if templateVersion > 0 {
		dep.TemplateVersion = &templateVersion
	}
	// Embed input asset IDs into PipelineJSON for lineage queries (F4.7).
	if len(assetIDs) > 0 {
		pipelineArg["_input_asset_ids"] = assetIDs
	}

	if err := uc.deploymentRepo.Save(ctx, dep); err != nil {
		return nil, fmt.Errorf("save deployment: %w", err)
	}
	if err := uc.savePipelineRun(ctx, dep, templateVersion, wfUID, opts...); err != nil {
		return nil, fmt.Errorf("save pipeline run: %w", err)
	}

	// Record asset_events for input assets (F4.4).
	if uc.assetEventRepo != nil && len(assetIDs) > 0 {
		payload, _ := json.Marshal(map[string]interface{}{
			"deployment_id": depID,
			"pipeline_name": pipeName,
			"workflow_name": wfName,
		})
		for _, aid := range assetIDs {
			// Skip unregistered/external assets: asset_events.asset_id FKs to
			// assets, so appending lineage for an asset that isn't in the table
			// fails with 23503. There is no assets row to attach lineage to, so
			// the event is meaningless anyway. Only filter when we could check
			// (assetRepo present); otherwise preserve the prior best-effort.
			if uc.assetRepo != nil && !assetRegistered[aid] {
				continue
			}
			logPipelineSideEffect("append pipeline_processing event", uc.assetEventRepo.Append(ctx, repository.AssetEventAppendInput{
				EventType:     "pipeline_processing",
				AggregateType: "asset",
				AssetID:       aid,
				RunID:         depID,
				EventPayload:  payload,
			}))
		}
	}

	return dep, nil
}

func (uc *Usecase) runtimeSubmitConfigured() bool {
	return uc.runtimeAdapter != nil || uc.wfClient != nil
}

// submitRuntimeWorkflow uses the client resolved by the caller (see
// resolveArgoClient) for the actual CreateWorkflow call, then falls back to
// the runtime adapter when no direct client is available. Prior to PR 4c.1
// this function did its own factory lookup — that logic has moved up to
// Deploy so the SAME client can be reused for post-submit reads/deletes,
// preventing multi-cluster misroutes on the downstream ops.
//
// Preference order:
//  1. When factory is not wired but a runtime adapter IS wired, prefer the
//     adapter. Preserves the pre-3486 unit-test intent where SetRuntimeAdapter
//     was the primary wiring.
//  2. Otherwise, submit through the caller-provided client.
//  3. Nothing wired → ErrWorkflowUnavailable.
func (uc *Usecase) submitRuntimeWorkflow(ctx context.Context, client argo.WorkflowClient, runID, runName string, wf *wfv1.Workflow, namespace string) (*runtimeadapter.RuntimeJob, error) {
	if uc.argoFactory == nil && uc.runtimeAdapter != nil {
		return uc.runtimeAdapter.Submit(ctx,
			runtimeadapter.RunRef{ID: runID, Name: runName},
			runtimeadapter.RuntimeSpec{
				RuntimeType: "argo",
				Namespace:   namespace,
				Manifest:    wf,
			},
		)
	}
	if client != nil {
		if err := client.CreateWorkflow(ctx, wf, namespace); err != nil {
			return nil, err
		}
		return &runtimeadapter.RuntimeJob{
			Ref: runtimeadapter.RuntimeRef{
				RuntimeType: "argo",
				Name:        strings.TrimSpace(wf.Name),
				Namespace:   firstNonEmpty(namespace, wf.Namespace),
				UID:         string(wf.UID),
			},
			Raw: wf,
		}, nil
	}
	return nil, ErrWorkflowUnavailable
}

// targetClusterID returns a short label of the target's cluster for logging.
// Empty target or empty ClusterID → "cluster-default".
func targetClusterID(t *models.ExecutionTarget) string {
	if t == nil || strings.TrimSpace(t.ClusterID) == "" {
		return "cluster-default"
	}
	return t.ClusterID
}

func workflowFromRuntimeJob(job *runtimeadapter.RuntimeJob) *wfv1.Workflow {
	if job == nil {
		return nil
	}
	switch raw := job.Raw.(type) {
	case *wfv1.Workflow:
		return raw
	case wfv1.Workflow:
		return &raw
	default:
		return nil
	}
}

// DeployByTemplateID deploys a saved template.
func (uc *Usecase) DeployByTemplateID(ctx context.Context, templateID, name string, assetIDs []string, opts ...DeployOptions) (*models.PipelineDeployment, error) {
	t, err := uc.templateRepo.FindByID(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("find template: %w", err)
	}
	if t == nil {
		return nil, ErrTemplateNotFound
	}
	requestedVersion := 0
	if len(opts) > 0 {
		requestedVersion = opts[0].TemplateVersion
	}
	// Resolve which version to deploy: explicit request > current template version.
	// Do NOT auto-apply activeVersion here; user's templateID choice must be respected.
	resolvedVersion := t.Version
	if requestedVersion > 0 && requestedVersion != t.Version {
		resolvedVersion = requestedVersion
	}
	if resolvedVersion != t.Version {
		versioned, err := uc.templateRepo.FindByNameAndVersion(ctx, t.Name, resolvedVersion)
		if err != nil {
			return nil, fmt.Errorf("find template version: %w", err)
		}
		if versioned == nil {
			return nil, ErrTemplateNotFound
		}
		t = versioned
		templateID = t.ID
	}
	if name == "" {
		name = t.Name
	}
	deployOpts := DeployOptions{TemplateID: templateID, TemplateVersion: t.Version}
	if len(opts) > 0 {
		deployOpts.TargetID = opts[0].TargetID
		deployOpts.Owner = opts[0].Owner
		deployOpts.BatchJobID = opts[0].BatchJobID
		deployOpts.DryRun = opts[0].DryRun
		deployOpts.AllowUnknownAssets = opts[0].AllowUnknownAssets
		deployOpts.PreallocatedRunID = opts[0].PreallocatedRunID
		deployOpts.ConfigSelection = opts[0].ConfigSelection
	}
	return uc.Deploy(ctx, t.Pipeline, name, assetIDs, deployOpts)
}

func (uc *Usecase) validateDeployAssetIDs(ctx context.Context, assetIDs []string, opts ...DeployOptions) ([]string, error) {
	allowUnknown := len(opts) > 0 && opts[0].AllowUnknownAssets
	if allowUnknown {
		return assetvalidation.NormalizeAssetIDs("asset_ids", assetIDs)
	}
	normalized, err := assetvalidation.Validate(ctx, uc.assetRepo, "asset_ids", assetIDs)
	if err == nil {
		return normalized, nil
	}
	var validationErr *assetvalidation.ValidationError
	if !errors.As(err, &validationErr) ||
		len(validationErr.MissingIDs) == 0 ||
		len(validationErr.InvalidIDs) > 0 ||
		len(validationErr.DuplicateIDs) > 0 {
		return nil, err
	}

	var unsupportedMissing []string
	for _, missing := range validationErr.MissingIDs {
		if !isCompatibleExternalVideoID(missing) {
			unsupportedMissing = append(unsupportedMissing, missing)
		}
	}
	if len(unsupportedMissing) > 0 {
		compatErr := *validationErr
		compatErr.MissingIDs = unsupportedMissing
		return nil, &compatErr
	}
	return assetvalidation.NormalizeAssetIDs("asset_ids", assetIDs)
}

func isCompatibleExternalVideoID(value string) bool {
	return compatibleExternalVideoIDPattern.MatchString(strings.TrimSpace(value))
}

// ── Pipeline Runs ───────────────────────────────────────────────────────

// CreateRun creates a first-class pipeline run while keeping deployment
// compatibility storage in sync.
func (uc *Usecase) CreateRun(
	ctx context.Context,
	pipelineArg map[string]interface{},
	name string,
	assetIDs []string,
	opts ...DeployOptions,
) (*models.PipelineRun, error) {
	dep, err := uc.Deploy(ctx, pipelineArg, name, assetIDs, opts...)
	if err != nil {
		return nil, err
	}
	if uc.runRepo == nil {
		return uc.deploymentToRun(dep), nil
	}
	run, err := uc.runRepo.FindByID(ctx, dep.ID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return uc.deploymentToRun(dep), nil
	}
	uc.enrichRun(ctx, run, true)
	return run, nil
}

// CreateRunByTemplateID creates a first-class pipeline run from a saved template.
func (uc *Usecase) CreateRunByTemplateID(ctx context.Context, templateID, name string, assetIDs []string, opts ...DeployOptions) (*models.PipelineRun, error) {
	dep, err := uc.DeployByTemplateID(ctx, templateID, name, assetIDs, opts...)
	if err != nil {
		return nil, err
	}
	if uc.runRepo == nil {
		return uc.deploymentToRun(dep), nil
	}
	run, err := uc.runRepo.FindByID(ctx, dep.ID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return uc.deploymentToRun(dep), nil
	}
	uc.enrichRun(ctx, run, true)
	return run, nil
}

// CreateRunsByTemplateID deploys one pipeline run per asset when multiple asset
// IDs are provided; zero or one asset uses a single run as before.
// Multiple assets are created concurrently with a concurrency limit of 5
// to reduce total wall-clock time from O(N*T) to O(T).
func (uc *Usecase) CreateRunsByTemplateID(ctx context.Context, templateID, name string, assetIDs []string, opts ...DeployOptions) ([]models.PipelineRun, error) {
	if len(assetIDs) <= 1 {
		run, err := uc.CreateRunByTemplateID(ctx, templateID, name, assetIDs, opts...)
		if err != nil {
			return nil, err
		}
		return []models.PipelineRun{*run}, nil
	}

	type result struct {
		run *models.PipelineRun
		err error
		idx int
	}
	results := make(chan result, len(assetIDs))
	sem := make(chan struct{}, 5)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for idx, assetID := range assetIDs {
		go func(i int, aid string) {
			sem <- struct{}{}
			r, err := uc.CreateRunByTemplateID(ctx, templateID, name, []string{aid}, opts...)
			<-sem
			res := result{idx: i, err: err}
			if err == nil {
				res.run = r
			}
			results <- res
		}(idx, assetID)
	}

	runs := make([]models.PipelineRun, len(assetIDs))
	for range assetIDs {
		res := <-results
		if res.err != nil {
			cancel()
			return runs, res.err
		}
		runs[res.idx] = *res.run
	}
	return runs, nil
}

// ListRuns returns all first-class pipeline runs. When the run table is not
// wired, it projects legacy deployments for compatibility.
// refreshActive triggers live Argo status polls (capped); list endpoints should
// pass false and rely on the run watcher + GetRun for on-demand refresh.
func (uc *Usecase) ListRuns(ctx context.Context, refreshActive bool) ([]models.PipelineRun, error) {
	if uc.runRepo == nil {
		deps, err := uc.listDeployments(ctx, refreshActive)
		if err != nil {
			return nil, err
		}
		out := make([]models.PipelineRun, 0, len(deps))
		for i := range deps {
			out = append(out, *uc.deploymentToRun(&deps[i]))
		}
		return out, nil
	}
	list, err := uc.runRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	if refreshActive && uc.wfClient != nil {
		refreshed := 0
		for i := range list {
			if refreshed >= maxActiveDeploymentStatusRefresh {
				break
			}
			if isActiveDeploymentStatus(list[i].Status) {
				refreshed++
				uc.refreshPipelineRunStatus(ctx, &list[i])
			}
		}
	}
	for i := range list {
		uc.enrichRun(ctx, &list[i], false)
	}
	return list, nil
}

// defaultUnfilteredRunSummaryPageSize bounds a ListRunSummaries call made
// without an explicit filter. It matches the repo's own pageSize hard cap so
// an unfiltered list can never degrade into a full pipeline_runs scan
// (CYB-3491). Callers that need more must paginate.
const defaultUnfilteredRunSummaryPageSize = 500

// ListRunSummaries returns lightweight pipeline runs for list UIs. It skips
// manifest/pipeline_json hydration and per-run node/asset enrichment.
func (uc *Usecase) ListRunSummaries(ctx context.Context, filter ...models.PipelineRunListFilter) ([]models.PipelineRun, int, error) {
	if uc.runRepo == nil {
		deps, err := uc.deploymentRepo.FindAll(ctx)
		if err != nil {
			return nil, 0, err
		}
		out := make([]models.PipelineRun, 0, len(deps))
		for i := range deps {
			run := uc.deploymentToRun(&deps[i])
			stripRunHeavyFields(run)
			out = append(out, *run)
		}
		return out, len(out), nil
	}
	// CYB-3491: an unfiltered summary list must never full-scan pipeline_runs.
	// The old no-filter branch called FindAllSummaries (no LIMIT); at 30k+ rows
	// it once took ~39s and exhausted the connection pool. Default to a bounded
	// page and route every call through the single paginated path instead.
	// Callers that need more than the default must page explicitly.
	f := models.PipelineRunListFilter{PageSize: defaultUnfilteredRunSummaryPageSize}
	if len(filter) > 0 {
		f = filter[0]
	}
	t0 := time.Now()
	items, total, err := uc.runRepo.ListSummaries(ctx, f)
	if elapsed := time.Since(t0); elapsed > 300*time.Millisecond {
		slog.Warn("ListSummaries slow query", "elapsed", elapsed.String(), "total", total)
	}
	if err != nil {
		return nil, 0, err
	}
	// CYB-3490: list reads are pure — no Argo refresh on the request
	// path (RefreshActive is accepted but ignored). The background
	// watcher owns active-run refresh and misclassified-run healing.
	normalizeActiveRunRuntimeFields(items)
	if f.BatchJobID != "" {
		uc.attachBatchNodeProgress(ctx, items)
		uc.attachVideoDurations(ctx, items)
	}
	annotateRunDiagnostics(items)
	// The default list view keeps per-run nodes so callers can render the
	// estimated cost; the summary view drops them for a lighter payload.
	keepNodes := !f.SummaryOnly
	if keepNodes && uc.runNodeRepo != nil {
		uc.attachRunNodesForList(ctx, items)
	}
	for i := range items {
		stripRunListFields(&items[i], keepNodes)
	}
	return items, total, nil
}

func normalizeActiveRunRuntimeFields(items []models.PipelineRun) {
	for i := range items {
		normalizeActiveRunRuntimeField(&items[i])
	}
}

func normalizeActiveRunRuntimeField(run *models.PipelineRun) {
	if run == nil || !isActiveDeploymentStatus(run.Status) {
		return
	}
	run.FinishedAt = nil
	if isStaleWorkflowUnavailableMessage(run.Message) {
		run.Message = ""
	}
}

func annotateRunDiagnostics(items []models.PipelineRun) {
	for i := range items {
		runstate.AnnotateRunDiagnostics(&items[i])
	}
}

func (uc *Usecase) attachBatchNodeProgress(ctx context.Context, items []models.PipelineRun) {
	if uc.assetNodeRepo == nil || len(items) == 0 {
		return
	}
	runIDs := make([]string, 0, len(items))
	for i := range items {
		if items[i].ID != "" {
			runIDs = append(runIDs, items[i].ID)
		}
	}
	rows, err := uc.assetNodeRepo.ListByRunIDs(ctx, runIDs)
	if err != nil {
		return
	}
	progressByRun := batchprogress.ByRunID(rows, items)
	for i := range items {
		items[i].NodeProgress = progressByRun[items[i].ID]
		// CYB-3490: while a workflow runs, per-node rows are no longer
		// continuously projected — surface the workflow-level progress
		// ("done/total" from Argo status.progress) as the compact label so
		// batch lists stay near-realtime without per-node writes.
		if items[i].NodeProgress == nil && items[i].Progress != "" && isActiveDeploymentStatus(items[i].Status) {
			items[i].NodeProgress = &models.PipelineRunNodeProgress{
				FocusStatus: "Running",
				Label:       items[i].Progress,
			}
		}
	}
}

// ListBatchAssetRuns returns all pipeline runs for one asset within a batch job.
func (uc *Usecase) ListBatchAssetRuns(ctx context.Context, batchJobID, assetID string) ([]models.PipelineRun, error) {
	if uc.runRepo == nil {
		return nil, nil
	}
	runs, err := uc.runRepo.FindAllByBatchJobAndAssetID(ctx, batchJobID, assetID)
	if err != nil {
		return nil, err
	}
	// CYB-3490: pure read — per-run live refresh happens on single-run
	// drill-in (GetRun) and in the background watcher, not on list reads.
	return runs, nil
}

// ListAssetNodesByRunIDs returns asset-node rows for multiple runs.
func (uc *Usecase) ListAssetNodesByRunIDs(ctx context.Context, runIDs []string) ([]models.PipelineRunAssetNode, error) {
	if uc.assetNodeRepo == nil || len(runIDs) == 0 {
		return nil, nil
	}
	return uc.assetNodeRepo.ListByRunIDs(ctx, runIDs)
}

func stripRunHeavyFields(run *models.PipelineRun) {
	stripRunListFields(run, false)
}

// stripRunListFields removes fields that are only needed on the detail page.
// When keepNodes is true the per-run nodes are preserved so list callers can
// compute the estimated cost.
func stripRunListFields(run *models.PipelineRun, keepNodes bool) {
	if run == nil {
		return
	}
	run.PipelineJSON = nil
	run.Manifest = nil
	run.TargetSnapshot = nil
	run.ExecutionTarget = nil
	if !keepNodes {
		run.Nodes = nil
	}
}

// attachRunNodesForList hydrates each run's nodes directly from the node repo
// without the full GetRun status refresh, keeping the list path cheap.
func (uc *Usecase) attachRunNodesForList(ctx context.Context, items []models.PipelineRun) {
	for i := range items {
		if len(items[i].Nodes) > 0 {
			continue
		}
		nodes, err := uc.runNodeRepo.FindByRunID(ctx, items[i].ID)
		if err != nil {
			slog.Warn("list runs: load nodes failed", "runID", items[i].ID, "err", err)
			continue
		}
		items[i].Nodes = nodes
	}
}

// GetRunByWorkflowName returns a pipeline run by its Argo workflow name.
func (uc *Usecase) GetRunByWorkflowName(ctx context.Context, workflowName string) (*models.PipelineRun, error) {
	workflowName = strings.TrimSpace(workflowName)
	if workflowName == "" {
		return nil, nil
	}
	if uc.runRepo == nil {
		return nil, nil
	}
	run, err := uc.runRepo.FindByWorkflowName(ctx, workflowName)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, nil
	}
	// CYB-3490: single-run drill-in is the sanctioned live path — it also
	// projects fresh node rows while the workflow runs (bounded: one wf).
	uc.refreshPipelineRunStatusLive(ctx, run)
	uc.reconcileTerminalRunFromLedger(ctx, run)
	uc.reconcileMisclassifiedRunFromArgo(ctx, run, nodeProjectLive)
	uc.enrichRun(ctx, run, true)
	return run, nil
}

// GetRun returns a single first-class pipeline run.
func (uc *Usecase) GetRun(ctx context.Context, id string) (*models.PipelineRun, error) {
	if uc.runRepo == nil {
		dep, err := uc.GetDeployment(ctx, id)
		if err != nil || dep == nil {
			return nil, err
		}
		return uc.deploymentToRun(dep), nil
	}
	run, err := uc.runRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		run, err = uc.runRepo.FindSummaryByID(ctx, id)
		if err != nil {
			return nil, err
		}
	}
	if run == nil {
		return nil, nil
	}
	initialStatus := run.Status
	initialMessage := run.Message
	// CYB-3490: single-run drill-in is the sanctioned live path — it also
	// projects fresh node rows while the workflow runs (bounded: one wf).
	uc.refreshPipelineRunStatusLive(ctx, run)
	uc.reconcileTerminalRunFromLedger(ctx, run)
	uc.reconcileMisclassifiedRunFromArgo(ctx, run, nodeProjectLive)
	// CYB-3707: un-stick a run left mislabeled Succeeded by the prior revival
	// race when its durable ledger shows a failed leaf, so the retry control
	// returns and the failed step can be re-run.
	uc.recoverStuckSucceededRun(ctx, run)
	uc.enrichRun(ctx, run, true)
	if run.Status != initialStatus || run.Message != initialMessage {
		slog.Warn("GetRun status changed",
			"runID", run.ID,
			"workflowName", run.WorkflowName,
			"oldStatus", initialStatus,
			"newStatus", run.Status,
			"oldMessage", initialMessage,
			"newMessage", run.Message,
		)
	}
	return run, nil
}

// DeleteRun removes a first-class run and the legacy deployment record.
func (uc *Usecase) DeleteRun(ctx context.Context, id string) error {
	run, err := uc.GetRun(ctx, id)
	if err != nil {
		return err
	}
	if run == nil {
		return ErrDeploymentNotFound
	}
	uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
		EventType:      runEventDeleteRequested,
		SubjectType:    "run",
		SubjectID:      run.ID,
		Status:         run.Status,
		Message:        "pipeline run delete requested",
		IdempotencyKey: fmt.Sprintf("run_delete_requested:%s:%d", run.ID, time.Now().UTC().UnixNano()),
	})
	// CYB-3486d5: delete via the run's cluster client. Nil client → skip
	// (test-only path with neither factory nor singleton wired).
	if client, resolveErr := uc.resolveArgoClientForRun(ctx, run); resolveErr == nil && client != nil {
		namespace := run.ArgoNamespace
		if namespace == "" {
			namespace = uc.namespace
		}
		if err := client.DeleteWorkflow(ctx, run.WorkflowName, namespace); err != nil {
			uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
				EventType:      runEventDeleteFailed,
				SubjectType:    "run",
				SubjectID:      run.ID,
				Status:         run.Status,
				Message:        "pipeline run delete failed",
				Reason:         err.Error(),
				IdempotencyKey: fmt.Sprintf("run_delete_failed:%s:%d", run.ID, time.Now().UTC().UnixNano()),
				Payload: map[string]interface{}{
					"workflowName": run.WorkflowName,
					"namespace":    namespace,
				},
			})
			logPipelineSideEffect("delete workflow", err)
		} else {
			uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
				EventType:      runEventDeleted,
				SubjectType:    "run",
				SubjectID:      run.ID,
				Status:         "Deleted",
				Message:        "pipeline run deleted",
				IdempotencyKey: fmt.Sprintf("run_deleted:%s", run.ID),
				Payload: map[string]interface{}{
					"workflowName": run.WorkflowName,
					"namespace":    namespace,
				},
			})
		}
	}
	if uc.runNodeRepo != nil {
		logPipelineSideEffect("delete pipeline run nodes", uc.runNodeRepo.DeleteByRunID(ctx, id))
	}
	if uc.runRepo != nil {
		if err := uc.runRepo.Delete(ctx, id); err != nil {
			return err
		}
	}
	return uc.deploymentRepo.Delete(ctx, id)
}

// RetryRun re-runs a first-class pipeline run.
func (uc *Usecase) RetryRun(ctx context.Context, id string) (*models.PipelineRun, error) {
	run, err := uc.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrDeploymentNotFound
	}
	uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
		EventType:      runEventRetryRequested,
		SubjectType:    "run",
		SubjectID:      run.ID,
		Status:         run.Status,
		Message:        "pipeline run retry requested",
		IdempotencyKey: fmt.Sprintf("run_retry_requested:%s:%d", run.ID, time.Now().UTC().UnixNano()),
	})
	next, err := uc.CreateRun(ctx, run.PipelineJSON, run.PipelineName+"-retry", run.AssetIDs, deployOptionsFromSourceRun(run))
	if err != nil {
		uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
			EventType:      runEventRetryFailed,
			SubjectType:    "run",
			SubjectID:      run.ID,
			Status:         run.Status,
			Message:        "pipeline run retry failed",
			Reason:         err.Error(),
			IdempotencyKey: fmt.Sprintf("run_retry_failed:%s:%d", run.ID, time.Now().UTC().UnixNano()),
		})
	}
	if err == nil && next != nil {
		uc.appendRunEvent(ctx, next, models.PipelineRunEvent{
			EventType:      runEventResubmitted,
			SubjectType:    "run",
			SubjectID:      next.ID,
			Status:         next.Status,
			Message:        "pipeline run created from retry",
			IdempotencyKey: fmt.Sprintf("run_resubmitted:%s:%s", next.ID, run.ID),
			Payload: map[string]interface{}{
				"sourceRunId": run.ID,
				"childRunId":  next.ID,
				"relation":    "retry_of",
			},
		})
		uc.appendChildRunRelationEvent(ctx, run, next, "retry_of", runEventResubmitted, "pipeline run retry child created")
	}
	return next, err
}

// RerunRun creates a new Run from the original Run spec. Unlike runtime retry,
// this is a product-level full rerun and therefore returns a new Run identity.
func (uc *Usecase) RerunRun(ctx context.Context, id string) (*models.PipelineRun, error) {
	run, err := uc.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrDeploymentNotFound
	}
	uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
		EventType:      runEventRerunRequested,
		SubjectType:    "run",
		SubjectID:      run.ID,
		Status:         run.Status,
		Message:        "run rerun requested",
		IdempotencyKey: fmt.Sprintf("run_rerun_requested:%s:%d", run.ID, time.Now().UTC().UnixNano()),
	})
	next, err := uc.CreateRun(ctx, run.PipelineJSON, run.PipelineName+"-rerun", run.AssetIDs, deployOptionsFromSourceRun(run))
	if err != nil {
		uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
			EventType:      runEventRerunFailed,
			SubjectType:    "run",
			SubjectID:      run.ID,
			Status:         run.Status,
			Message:        "run rerun failed",
			Reason:         err.Error(),
			IdempotencyKey: fmt.Sprintf("run_rerun_failed:%s:%d", run.ID, time.Now().UTC().UnixNano()),
		})
	}
	if err == nil && next != nil {
		uc.appendRunEvent(ctx, next, models.PipelineRunEvent{
			EventType:      runEventRerunCreated,
			SubjectType:    "run",
			SubjectID:      next.ID,
			Status:         next.Status,
			Message:        "run created from rerun",
			IdempotencyKey: fmt.Sprintf("run_rerun_created:%s:%s", next.ID, run.ID),
			Payload: map[string]interface{}{
				"sourceRunId": run.ID,
				"childRunId":  next.ID,
				"relation":    "rerun_of",
			},
		})
		uc.appendChildRunRelationEvent(ctx, run, next, "rerun_of", runEventRerunCreated, "run rerun child created")
	}
	return next, err
}

// StopRun stops a run's workflow.
func (uc *Usecase) StopRun(ctx context.Context, id string) error {
	return uc.runtimeWorkflowOperation(ctx, id, runEventStopRequested, runEventStopSucceeded, runEventStopFailed, "run stop", uc.stopRuntimeRun)
}

// ListRunEvents returns a chronological page of stored events for a run.
func (uc *Usecase) ListRunEvents(ctx context.Context, id string, opts models.PipelineRunEventListOptions) (*models.PipelineRunEventListResult, error) {
	if uc.runRepo == nil {
		return nil, ErrDeploymentNotFound
	}
	run, err := uc.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrDeploymentNotFound
	}
	uc.refreshPipelineRunStatus(ctx, run)
	if uc.runEventRepo == nil {
		return &models.PipelineRunEventListResult{Items: []models.PipelineRunEvent{}}, nil
	}
	result, err := uc.runEventRepo.ListByRunID(ctx, id, opts)
	if err != nil {
		return nil, err
	}
	if result.Items == nil {
		result.Items = []models.PipelineRunEvent{}
	}
	return result, nil
}

// ListRunAssetNodes returns derived asset × node snapshots for a run.
func (uc *Usecase) ListRunAssetNodes(ctx context.Context, id string, opts models.PipelineRunAssetNodeListOptions) (*models.PipelineRunAssetNodeListResult, error) {
	run, err := uc.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrDeploymentNotFound
	}
	if uc.assetNodeRepo == nil {
		rows := deriveAssetNodeRows(run, run.Nodes)
		summary := models.PipelineRunAssetNodeSummary{
			Statuses:   map[string]int{},
			CostSource: "not_available",
		}
		assets := map[string]bool{}
		nodes := map[string]bool{}
		var totalCost float64
		hasCost := false
		for _, row := range rows {
			summary.Statuses[row.Status]++
			assets[row.AssetID] = true
			nodes[row.PipelineNodeID] = true
			if row.EstimatedCostUSD != nil {
				totalCost += *row.EstimatedCostUSD
				hasCost = true
			}
		}
		summary.AssetCount = len(assets)
		summary.NodeCount = len(nodes)
		if hasCost {
			summary.TotalEstimatedCostUSD = &totalCost
			summary.CostSource = "estimated_resource_duration"
		}
		return &models.PipelineRunAssetNodeListResult{Items: rows, Total: len(rows), Summary: summary}, nil
	}
	result, err := uc.assetNodeRepo.ListByRunID(ctx, id, opts)
	if err != nil {
		return nil, err
	}
	projectTerminalActiveAssetNodes(result, run)
	return result, nil
}

func projectTerminalActiveAssetNodes(result *models.PipelineRunAssetNodeListResult, run *models.PipelineRun) {
	if result == nil || run == nil || !isTerminalFailureRunStatus(run.Status) {
		return
	}
	changed := false
	for i := range result.Items {
		status := strings.TrimSpace(result.Items[i].Status)
		isActiveRunning := strings.EqualFold(status, "Running")
		isPendingDiagnostic := strings.EqualFold(status, "Pending") &&
			(isImageStartupFailureMessage(result.Items[i].Message) ||
				isUnschedulableSchedulerMessage(result.Items[i].Message))
		if !isActiveRunning && !isPendingDiagnostic {
			continue
		}
		result.Items[i].Status = "Error"
		if strings.TrimSpace(result.Items[i].Message) == "" {
			result.Items[i].Message = strings.TrimSpace(run.Message)
		}
		if result.Items[i].FinishedAt == nil && run.FinishedAt != nil && !run.FinishedAt.IsZero() {
			finishedAt := *run.FinishedAt
			result.Items[i].FinishedAt = &finishedAt
		}
		changed = true
	}
	if !changed {
		return
	}
	result.Summary = summarizeAssetNodeRows(result.Items)
}

func isTerminalFailureRunStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed", "error", "expired":
		return true
	default:
		return false
	}
}

func summarizeAssetNodeRows(rows []models.PipelineRunAssetNode) models.PipelineRunAssetNodeSummary {
	summary := models.PipelineRunAssetNodeSummary{
		Statuses:   map[string]int{},
		CostSource: "not_available",
	}
	assets := map[string]bool{}
	nodes := map[string]bool{}
	var totalCost float64
	hasCost := false
	for _, row := range rows {
		summary.Statuses[row.Status]++
		assets[row.AssetID] = true
		nodes[row.PipelineNodeID] = true
		if row.EstimatedCostUSD != nil {
			totalCost += *row.EstimatedCostUSD
			hasCost = true
		}
	}
	summary.AssetCount = len(assets)
	summary.NodeCount = len(nodes)
	if hasCost {
		summary.TotalEstimatedCostUSD = &totalCost
		summary.CostSource = "estimated_resource_duration"
	}
	return summary
}

func durationSeconds(startedAt, finishedAt *time.Time) *int64 {
	if startedAt == nil || finishedAt == nil || finishedAt.Before(*startedAt) {
		return nil
	}
	seconds := int64(finishedAt.Sub(*startedAt).Seconds())
	return &seconds
}

// GetRunCostSummary returns estimated cost and duration summaries for a run.
func (uc *Usecase) GetRunCostSummary(ctx context.Context, id string) (*models.PipelineRunCostSummary, error) {
	run, err := uc.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrDeploymentNotFound
	}
	if uc.runCostSnapshotMissing(run) {
		if isActiveDeploymentStatus(run.Status) {
			uc.refreshRunStatus(ctx, run, nodeProjectTerminalArchive)
		} else {
			uc.backfillRunStatus(ctx, run)
		}
		uc.enrichRun(ctx, run, true)
	}
	summary := &models.PipelineRunCostSummary{
		RunID:              id,
		CostSource:         "not_available",
		NodeSummaries:      []models.PipelineRunNodeCostSummary{},
		AssetNodeSummaries: []models.PipelineRunAssetNodeCostSummary{},
		GeneratedAt:        time.Now().UTC(),
	}
	var total float64
	hasCost := false
	for _, node := range run.Nodes {
		if isWorkflowControlRunNode(node) {
			continue
		}
		if node.EstimatedCostUSD != nil {
			total += *node.EstimatedCostUSD
			hasCost = true
		}
		podCount := 0
		if node.PodName != "" {
			podCount = 1
		}
		summary.NodeSummaries = append(summary.NodeSummaries, models.PipelineRunNodeCostSummary{
			NodeID:           node.PipelineNodeID,
			DisplayName:      node.DisplayName,
			Status:           node.Phase,
			PodCount:         podCount,
			EstimatedCostUSD: node.EstimatedCostUSD,
			CostSource:       costSourceFromPtr(node.EstimatedCostUSD),
			DurationSeconds:  durationSeconds(node.StartedAt, node.FinishedAt),
		})
	}
	assetNodes, err := uc.ListRunAssetNodes(ctx, id, models.PipelineRunAssetNodeListOptions{Limit: 500})
	if err == nil && assetNodes != nil {
		for _, row := range assetNodes.Items {
			summary.AssetNodeSummaries = append(summary.AssetNodeSummaries, models.PipelineRunAssetNodeCostSummary{
				AssetID:          row.AssetID,
				NodeID:           row.PipelineNodeID,
				DisplayName:      row.DisplayName,
				Status:           row.Status,
				EstimatedCostUSD: row.EstimatedCostUSD,
				CostSource:       row.CostSource,
			})
		}
	}
	if hasCost {
		summary.TotalEstimatedCostUSD = &total
		summary.CostSource = "estimated_resource_duration"
	}
	return summary, nil
}

func costSourceFromPtr(v *float64) string {
	if v == nil {
		return "not_available"
	}
	return "estimated_resource_duration"
}

// ListRunNodes returns the DataBrew node ledger for a run.
func (uc *Usecase) ListRunNodes(ctx context.Context, id string) ([]models.PipelineRunNode, error) {
	run, err := uc.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrDeploymentNotFound
	}
	if run.Nodes == nil {
		return []models.PipelineRunNode{}, nil
	}
	return run.Nodes, nil
}

// ListRunInputs returns a Phase 1 projection of RunInput records.
func (uc *Usecase) ListRunInputs(ctx context.Context, id string) (*models.RunInputList, error) {
	run, err := uc.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrDeploymentNotFound
	}
	if uc.runInputRepo != nil {
		items, err := uc.runInputRepo.ListByRunID(ctx, run.ID)
		if err != nil {
			return nil, err
		}
		if len(items) > 0 {
			return &models.RunInputList{RunID: run.ID, Items: items, Total: len(items)}, nil
		}
	}
	items := materializeRunInputs(run)
	return &models.RunInputList{RunID: run.ID, Items: items, Total: len(items)}, nil
}

// ListRunOutputs returns a Phase 1 projection of RunOutput records.
func (uc *Usecase) ListRunOutputs(ctx context.Context, id string) (*models.RunOutputList, error) {
	run, err := uc.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrDeploymentNotFound
	}
	items := make([]models.RunOutput, 0, len(run.Nodes)*2)
	for _, node := range run.Nodes {
		nodeID := firstNonEmpty(node.PipelineNodeID, node.ArgoNodeID, node.ID)
		if len(node.Outputs) > 0 {
			items = append(items, models.RunOutput{
				ID:       fmt.Sprintf("%s:%s:outputs", run.ID, nodeID),
				RunID:    run.ID,
				NodeID:   nodeID,
				Type:     "node_outputs",
				Snapshot: copyStringAnyMap(node.Outputs),
			})
		}
		if strings.TrimSpace(node.LogRef) != "" {
			items = append(items, models.RunOutput{
				ID:     fmt.Sprintf("%s:%s:logs", run.ID, nodeID),
				RunID:  run.ID,
				NodeID: nodeID,
				Type:   "logs",
				URI:    node.LogRef,
			})
		}
		metrics := map[string]interface{}{}
		if len(node.ResourcesDuration) > 0 {
			metrics["resourcesDuration"] = node.ResourcesDuration
		}
		if len(node.ResourceSummary) > 0 {
			metrics["resourceSummary"] = node.ResourceSummary
		}
		if len(metrics) > 0 {
			items = append(items, models.RunOutput{
				ID:       fmt.Sprintf("%s:%s:metrics", run.ID, nodeID),
				RunID:    run.ID,
				NodeID:   nodeID,
				Type:     "metrics",
				Snapshot: metrics,
			})
		}
	}
	return &models.RunOutputList{RunID: run.ID, Items: items, Total: len(items)}, nil
}

// ListRunChildren returns child runs for Phase 1 Run Tree views. A batch parent
// can be represented by a run whose id is used as child batchJobId.
func (uc *Usecase) ListRunChildren(ctx context.Context, id string, filters ...models.PipelineRunListFilter) (*models.RunChildList, error) {
	filter := normalizeRunChildrenFilter(filters...)
	run, err := uc.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		if uc.runRepo != nil {
			if fallback, fallbackErr := uc.listBatchRunChildren(ctx, id, "", filter); fallbackErr != nil {
				return nil, fallbackErr
			} else if fallback.Total > 0 {
				return fallback, nil
			}
		}
		return nil, ErrDeploymentNotFound
	}
	if uc.runRepo == nil {
		return &models.RunChildList{
			RunID:     run.ID,
			Items:     []models.PipelineRun{},
			Relations: []models.RunRelation{},
			Summary:   runstate.AggregateChildRuns(nil),
		}, nil
	}
	if uc.runRelationRepo != nil {
		if result, err := uc.listDurableRunChildren(ctx, run.ID, filter); err != nil {
			return nil, err
		} else if result.Total > 0 || len(result.Relations) > 0 {
			return result, nil
		}
	}
	result, err := uc.listBatchRunChildren(ctx, id, run.ID, filter)
	if err != nil {
		return nil, err
	}
	childIDs := make(map[string]struct{}, len(result.Items))
	relationKeys := make(map[string]struct{}, len(result.Relations))
	for _, child := range result.Items {
		childIDs[child.ID] = struct{}{}
	}
	for _, relation := range result.Relations {
		relationKeys[runRelationKey(relation.ChildRunID, relation.RelationType)] = struct{}{}
	}
	eventChildren, eventRelations, err := uc.listEventDerivedRunChildren(ctx, run.ID, childIDs, relationKeys)
	if err != nil {
		return nil, err
	}
	previousTotal := result.Total
	result.Items = append(result.Items, eventChildren...)
	result.Relations = append(result.Relations, eventRelations...)
	annotateRunDiagnostics(result.Items)
	if previousTotal > 0 {
		result.Total = previousTotal + len(eventChildren)
	} else {
		result.Total = len(result.Items)
	}
	result.Summary = runstate.AggregateChildRuns(result.Items)
	for i := range result.Items {
		result.Items[i].Manifest = nil
		result.Items[i].PipelineJSON = nil
	}
	return result, nil
}

func (uc *Usecase) listDurableRunChildren(ctx context.Context, parentRunID string, filter models.PipelineRunListFilter) (*models.RunChildList, error) {
	if uc.runRelationRepo == nil || uc.runRepo == nil {
		return &models.RunChildList{
			RunID:     parentRunID,
			Items:     []models.PipelineRun{},
			Relations: []models.RunRelation{},
			Summary:   runstate.AggregateChildRuns(nil),
		}, nil
	}
	relations, err := uc.runRelationRepo.ListByParentRunID(ctx, parentRunID)
	if err != nil {
		return nil, err
	}
	total := len(relations)
	start := (filter.Page - 1) * filter.PageSize
	if start > total {
		start = total
	}
	end := start + filter.PageSize
	if end > total {
		end = total
	}
	pageRelations := relations[start:end]
	children := make([]models.PipelineRun, 0, len(pageRelations))
	outRelations := make([]models.RunRelation, 0, len(pageRelations))
	seenChildren := map[string]struct{}{}
	for _, relation := range pageRelations {
		childID := strings.TrimSpace(relation.ChildRunID)
		if childID == "" || childID == parentRunID {
			continue
		}
		child, err := uc.runRepo.FindByID(ctx, childID)
		if err != nil {
			return nil, err
		}
		if child == nil {
			continue
		}
		// CYB-3490: pure list read — no per-child GetRun/Argo refresh here;
		// the webhook + background watcher keep children fresh.
		if relation.Source == "" {
			relation.Source = "run_relations"
		}
		outRelations = append(outRelations, relation)
		if _, ok := seenChildren[child.ID]; !ok {
			children = append(children, *child)
			seenChildren[child.ID] = struct{}{}
		}
	}
	annotateRunDiagnostics(children)
	return &models.RunChildList{
		RunID:     parentRunID,
		Items:     children,
		Relations: outRelations,
		Summary:   runstate.AggregateChildRuns(children),
		Total:     total,
		Page:      filter.Page,
		PageSize:  filter.PageSize,
	}, nil
}

func normalizeRunChildrenFilter(filters ...models.PipelineRunListFilter) models.PipelineRunListFilter {
	var filter models.PipelineRunListFilter
	if len(filters) > 0 {
		filter = filters[0]
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	return filter
}

func (uc *Usecase) listBatchRunChildren(ctx context.Context, batchJobID, parentRunID string, filter models.PipelineRunListFilter) (*models.RunChildList, error) {
	// CYB-3822: forward filter.Status so batch-export "仅成功 / 仅失败" filters
	// at the repo (WHERE status=...) instead of the frontend having to fetch
	// every child then discard 95%. Repo already knows how to compare against
	// batchItemRunStatusExpr (Argo TitleCase) — see pipeline_repo.go:1025.
	items, total, err := uc.runRepo.ListSummaries(ctx, models.PipelineRunListFilter{
		BatchJobID: batchJobID,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		Status:     filter.Status,
	})
	if err != nil {
		return nil, err
	}
	// CYB-3490: pure list read — no per-child GetRun/Argo refresh on the
	// request path; the webhook + background watcher keep children fresh.
	if parentRunID == "" {
		parentRunID = batchJobID
	}
	children := make([]models.PipelineRun, 0, len(items))
	for _, child := range items {
		if child.ID == parentRunID {
			continue
		}
		children = append(children, child)
	}
	annotateRunDiagnostics(children)
	for i := range children {
		children[i].Manifest = nil
		children[i].PipelineJSON = nil
	}
	relations := runstate.BuildBatchChildRelations(parentRunID, children)
	return &models.RunChildList{
		RunID:     parentRunID,
		Items:     children,
		Relations: relations,
		Summary:   runstate.AggregateChildRuns(children),
		Total:     total,
		Page:      filter.Page,
		PageSize:  filter.PageSize,
	}, nil
}

func (uc *Usecase) listEventDerivedRunChildren(ctx context.Context, parentRunID string, childIDs map[string]struct{}, relationKeys map[string]struct{}) ([]models.PipelineRun, []models.RunRelation, error) {
	if uc.runEventRepo == nil || uc.runRepo == nil {
		return nil, nil, nil
	}
	events, err := uc.runEventRepo.ListByRunID(ctx, parentRunID, models.PipelineRunEventListOptions{Limit: 500})
	if err != nil {
		return nil, nil, err
	}
	if events == nil || len(events.Items) == 0 {
		return nil, nil, nil
	}
	children := []models.PipelineRun{}
	relations := []models.RunRelation{}
	for _, event := range events.Items {
		childID, relationType, ok := runRelationFromEvent(parentRunID, event)
		if !ok {
			continue
		}
		key := runRelationKey(childID, relationType)
		if _, exists := relationKeys[key]; exists {
			continue
		}
		child, err := uc.runRepo.FindByID(ctx, childID)
		if err != nil {
			return nil, nil, err
		}
		if child == nil || child.ID == parentRunID {
			continue
		}
		if _, exists := childIDs[child.ID]; !exists {
			children = append(children, *child)
			childIDs[child.ID] = struct{}{}
		}
		relations = append(relations, models.RunRelation{
			ID:           fmt.Sprintf("%s:%s:%s", parentRunID, child.ID, relationType),
			ParentRunID:  parentRunID,
			ChildRunID:   child.ID,
			RelationType: relationType,
			Source:       "pipeline_run_events",
		})
		relationKeys[key] = struct{}{}
	}
	return children, relations, nil
}

func runRelationFromEvent(parentRunID string, event models.PipelineRunEvent) (string, string, bool) {
	sourceRunID := strings.TrimSpace(stringFromAny(event.Payload["sourceRunId"]))
	if sourceRunID != "" && sourceRunID != parentRunID {
		return "", "", false
	}
	relationType := strings.TrimSpace(stringFromAny(event.Payload["relation"]))
	if !supportedRunRelationType(relationType) {
		return "", "", false
	}
	childID := strings.TrimSpace(stringFromAny(event.Payload["childRunId"]))
	if childID == "" && event.SubjectType == "run" && event.SubjectID != parentRunID {
		childID = strings.TrimSpace(event.SubjectID)
	}
	if childID == "" || childID == parentRunID {
		return "", "", false
	}
	return childID, relationType, true
}

func supportedRunRelationType(relationType string) bool {
	switch relationType {
	case "retry_of", "resubmit_of", "rerun_of":
		return true
	default:
		return false
	}
}

func runRelationKey(childRunID, relationType string) string {
	return childRunID + "\x00" + relationType
}

// GetRunRuntime returns runtime debug references without requiring the runtime
// object to still exist.
func (uc *Usecase) GetRunRuntime(ctx context.Context, id string) (*models.RunRuntime, error) {
	run, err := uc.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrDeploymentNotFound
	}
	namespace := run.ArgoNamespace
	if namespace == "" {
		namespace = uc.namespace
	}

	runtimeType := "argo"
	workflowName := strings.TrimSpace(run.WorkflowName)
	if isBatchParentWorkflowName(workflowName) {
		runtimeType = "none"
		workflowName = ""
	}
	debugURL := ""
	if workflowName != "" {
		debugURL = fmt.Sprintf("/api/v1/workflows/%s", workflowName)
	}
	return &models.RunRuntime{
		RunID: run.ID,
		Runtime: models.RunRuntimeRef{
			RuntimeType:       runtimeType,
			WorkflowName:      workflowName,
			Namespace:         namespace,
			UID:               run.ArgoWorkflowUID,
			Status:            run.Status,
			Message:           run.Message,
			ExecutionTargetID: run.ExecutionTargetID,
			TargetSnapshot:    copyStringAnyMap(run.TargetSnapshot),
			DebugURL:          debugURL,
		},
	}, nil
}

// RuntimeRetryRun retries failed runtime nodes in-place when the runtime
// adapter supports it. It does not create a new Run.
func (uc *Usecase) RuntimeRetryRun(ctx context.Context, id string) (*models.PipelineRun, error) {
	run, err := uc.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrDeploymentNotFound
	}
	if strings.TrimSpace(run.WorkflowName) == "" {
		return nil, fmt.Errorf("%w: run has no workflowName", ErrInvalidArgument)
	}
	if !isRetryableRunStatusForRuntimeRetry(run.Status) {
		return nil, fmt.Errorf("%w: runtime retry only supports failed or errored runs", ErrInvalidArgument)
	}
	if !uc.runtimeControlConfigured() {
		return nil, ErrWorkflowUnavailable
	}
	uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
		EventType:      runEventRuntimeRetryRequested,
		SubjectType:    "run",
		SubjectID:      run.ID,
		Status:         run.Status,
		Message:        "run runtime retry requested",
		IdempotencyKey: fmt.Sprintf("run_runtime_retry_requested:%s:%d", run.ID, time.Now().UTC().UnixNano()),
	})
	if err := uc.retryRuntimeRun(ctx, run); err != nil {
		if mapped := classifyRuntimeRetryError(err); mapped != nil {
			err = mapped
		}
		uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
			EventType:      runEventRuntimeRetryFailed,
			SubjectType:    "run",
			SubjectID:      run.ID,
			Status:         run.Status,
			Message:        "run runtime retry failed",
			Reason:         err.Error(),
			IdempotencyKey: fmt.Sprintf("run_runtime_retry_failed:%s:%d", run.ID, time.Now().UTC().UnixNano()),
		})
		return nil, err
	}
	uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
		EventType:      runEventRuntimeRetrySucceeded,
		SubjectType:    "run",
		SubjectID:      run.ID,
		Status:         run.Status,
		Message:        "run runtime retry submitted",
		IdempotencyKey: fmt.Sprintf("run_runtime_retry_succeeded:%s:%d", run.ID, time.Now().UTC().UnixNano()),
	})
	return uc.GetRun(ctx, id)
}

func isRetryableRunStatusForRuntimeRetry(status string) bool {
	switch runstate.NormalizeRunStatus(status) {
	case runstate.StatusFailed, runstate.StatusError:
		return true
	default:
		return false
	}
}

func classifyRuntimeRetryError(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	if msg == "" {
		return nil
	}
	switch {
	case strings.Contains(msg, "retry only supports failed"):
		return fmt.Errorf("%w: %s", ErrInvalidArgument, msg)
	case strings.Contains(msg, "to retry a succeeded workflow"):
		return fmt.Errorf("%w: %s", ErrInvalidArgument, msg)
	default:
		return nil
	}
}

// ResubmitRun creates a new Run from an existing run spec.
func (uc *Usecase) ResubmitRun(ctx context.Context, id string) (*models.PipelineRun, error) {
	run, err := uc.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrDeploymentNotFound
	}
	uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
		EventType:      runEventResubmitRequested,
		SubjectType:    "run",
		SubjectID:      run.ID,
		Status:         run.Status,
		Message:        "run resubmit requested",
		IdempotencyKey: fmt.Sprintf("run_resubmit_requested:%s:%d", run.ID, time.Now().UTC().UnixNano()),
	})
	next, err := uc.CreateRun(ctx, run.PipelineJSON, run.PipelineName+"-resubmit", run.AssetIDs, deployOptionsFromSourceRun(run))
	if err != nil {
		uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
			EventType:      runEventResubmitFailed,
			SubjectType:    "run",
			SubjectID:      run.ID,
			Status:         run.Status,
			Message:        "run resubmit failed",
			Reason:         err.Error(),
			IdempotencyKey: fmt.Sprintf("run_resubmit_failed:%s:%d", run.ID, time.Now().UTC().UnixNano()),
		})
	}
	if err == nil && next != nil {
		uc.appendRunEvent(ctx, next, models.PipelineRunEvent{
			EventType:      runEventResubmitted,
			SubjectType:    "run",
			SubjectID:      next.ID,
			Status:         next.Status,
			Message:        "run created from resubmit",
			IdempotencyKey: fmt.Sprintf("run_resubmitted:%s:%s", next.ID, run.ID),
			Payload: map[string]interface{}{
				"sourceRunId": run.ID,
				"childRunId":  next.ID,
				"relation":    "resubmit_of",
			},
		})
		uc.appendChildRunRelationEvent(ctx, run, next, "resubmit_of", runEventResubmitted, "run resubmit child created")
	}
	return next, err
}

func (uc *Usecase) appendChildRunRelationEvent(ctx context.Context, source, child *models.PipelineRun, relation, eventType, message string) {
	if source == nil || child == nil || strings.TrimSpace(relation) == "" {
		return
	}
	logPipelineSideEffect("persist child run relation", uc.persistRunRelation(ctx, &models.RunRelation{
		ParentRunID:  source.ID,
		ChildRunID:   child.ID,
		RelationType: strings.TrimSpace(relation),
		Source:       "run_kernel",
		Snapshot: map[string]interface{}{
			"eventType":      eventType,
			"sourceRunId":    source.ID,
			"childRunId":     child.ID,
			"relation":       strings.TrimSpace(relation),
			"sourceStatus":   source.Status,
			"childStatus":    child.Status,
			"sourceWorkflow": source.WorkflowName,
			"childWorkflow":  child.WorkflowName,
		},
	}))
	uc.appendRunEvent(ctx, source, models.PipelineRunEvent{
		EventType:      eventType,
		SubjectType:    "run",
		SubjectID:      child.ID,
		Status:         child.Status,
		Message:        message,
		IdempotencyKey: fmt.Sprintf("%s:%s:%s:%s", eventType, source.ID, child.ID, relation),
		Payload: map[string]interface{}{
			"sourceRunId": source.ID,
			"childRunId":  child.ID,
			"relation":    relation,
		},
	})
}

func deployOptionsFromSourceRun(run *models.PipelineRun) DeployOptions {
	opts := DeployOptions{}
	if run == nil {
		return opts
	}
	if run.ExecutionTargetID != "" {
		opts.TargetID = run.ExecutionTargetID
	} else if run.ExecutionTarget != nil {
		opts.TargetID = run.ExecutionTarget.ID
	}
	opts.ConfigSelection = deployLevelConfigSelectionFromRunInputs(run.PipelineJSON)
	return opts
}

func deployLevelConfigSelectionFromRunInputs(pipeline map[string]interface{}) *RuntimeConfigSelection {
	rawInputs, ok := interfaceSlice(pipeline[runConfigInputsPipelineJSONKey])
	if !ok {
		return nil
	}
	for _, rawInput := range rawInputs {
		cfg, ok := stringAnyMap(rawInput)
		if !ok || stringFromAny(cfg["scope"]) != "global" {
			continue
		}
		mode := stringFromAny(cfg["mode"])
		if mode == "" {
			mode = stringFromAny(cfg["source"])
		}
		if mode != "saved" {
			return nil
		}
		configID := stringFromAny(cfg["configId"])
		version, _ := strconv.Atoi(stringFromAny(cfg["version"]))
		if configID == "" || version <= 0 {
			return nil
		}
		return &RuntimeConfigSelection{
			Mode:           "saved",
			ConfigID:       configID,
			Version:        version,
			FileName:       stringFromAny(cfg["fileName"]),
			MountPath:      stringFromAny(cfg["mountPath"]),
			TargetFilename: stringFromAny(cfg["targetFilename"]),
		}
	}
	return nil
}

func (uc *Usecase) SuspendRun(ctx context.Context, id string) error {
	return uc.runtimeWorkflowOperation(ctx, id, runEventSuspendRequested, runEventSuspendSucceeded, runEventSuspendFailed, "run suspend", uc.suspendRuntimeRun)
}

func (uc *Usecase) ResumeRun(ctx context.Context, id string) error {
	return uc.runtimeWorkflowOperation(ctx, id, runEventResumeRequested, runEventResumeSucceeded, runEventResumeFailed, "run resume", uc.resumeRuntimeRun)
}

func (uc *Usecase) TerminateRun(ctx context.Context, id string) error {
	return uc.runtimeWorkflowOperation(ctx, id, runEventTerminateRequested, runEventTerminateSucceeded, runEventTerminateFailed, "run terminate", uc.terminateRuntimeRun)
}

func (uc *Usecase) runtimeWorkflowOperation(ctx context.Context, id, requestedEvent, succeededEvent, failedEvent, message string, op func(context.Context, *models.PipelineRun) error) error {
	run, err := uc.GetRun(ctx, id)
	if err != nil {
		return err
	}
	if run == nil {
		return ErrDeploymentNotFound
	}
	if strings.TrimSpace(run.WorkflowName) == "" {
		return fmt.Errorf("%w: run has no workflowName", ErrInvalidArgument)
	}
	if !uc.runtimeControlConfigured() {
		return ErrWorkflowUnavailable
	}
	uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
		EventType:      requestedEvent,
		SubjectType:    "run",
		SubjectID:      run.ID,
		Status:         run.Status,
		Message:        message + " requested",
		IdempotencyKey: fmt.Sprintf("%s:%s:%d", requestedEvent, run.ID, time.Now().UTC().UnixNano()),
	})
	if err := op(ctx, run); err != nil {
		uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
			EventType:      failedEvent,
			SubjectType:    "run",
			SubjectID:      run.ID,
			Status:         run.Status,
			Message:        message + " failed",
			Reason:         err.Error(),
			IdempotencyKey: fmt.Sprintf("%s:%s:%d", failedEvent, run.ID, time.Now().UTC().UnixNano()),
		})
		return err
	}
	uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
		EventType:      succeededEvent,
		SubjectType:    "run",
		SubjectID:      run.ID,
		Status:         run.Status,
		Message:        message + " submitted",
		IdempotencyKey: fmt.Sprintf("%s:%s:%d", succeededEvent, run.ID, time.Now().UTC().UnixNano()),
	})
	return nil
}

func (uc *Usecase) runtimeControlConfigured() bool {
	return uc.runtimeAdapter != nil || uc.wfClient != nil
}

func (uc *Usecase) runtimeRefForRun(run *models.PipelineRun) runtimeadapter.RuntimeRef {
	if run == nil {
		return runtimeadapter.RuntimeRef{RuntimeType: "argo", Namespace: uc.namespace}
	}
	return runtimeadapter.RuntimeRef{
		RuntimeType: "argo",
		Name:        strings.TrimSpace(run.WorkflowName),
		Namespace:   firstNonEmpty(run.ArgoNamespace, uc.namespace),
		UID:         strings.TrimSpace(run.ArgoWorkflowUID),
	}
}

// retryRuntimeRun / stopRuntimeRun / suspendRuntimeRun / resumeRuntimeRun /
// terminateRuntimeRun all resolve the argo client for the run's cluster
// (CYB-3486 PR 4d.5). runtimeAdapter still wins when set — used by tests.

func (uc *Usecase) retryRuntimeRun(ctx context.Context, run *models.PipelineRun) error {
	if uc.runtimeAdapter != nil {
		_, err := uc.runtimeAdapter.Retry(ctx, uc.runtimeRefForRun(run), runtimeadapter.RetryOptions{})
		return err
	}
	client, err := uc.resolveArgoClientForRun(ctx, run)
	if err != nil {
		return err
	}
	if client == nil {
		return ErrWorkflowUnavailable
	}
	return client.RetryWorkflow(ctx, run.WorkflowName, firstNonEmpty(run.ArgoNamespace, uc.namespace))
}

func (uc *Usecase) stopRuntimeRun(ctx context.Context, run *models.PipelineRun) error {
	if uc.runtimeAdapter != nil {
		return uc.runtimeAdapter.Stop(ctx, uc.runtimeRefForRun(run))
	}
	client, err := uc.resolveArgoClientForRun(ctx, run)
	if err != nil {
		return err
	}
	if client == nil {
		return ErrWorkflowUnavailable
	}
	return client.StopWorkflow(ctx, run.WorkflowName, firstNonEmpty(run.ArgoNamespace, uc.namespace))
}

func (uc *Usecase) suspendRuntimeRun(ctx context.Context, run *models.PipelineRun) error {
	if uc.runtimeAdapter != nil {
		return uc.runtimeAdapter.Suspend(ctx, uc.runtimeRefForRun(run))
	}
	client, err := uc.resolveArgoClientForRun(ctx, run)
	if err != nil {
		return err
	}
	if client == nil {
		return ErrWorkflowUnavailable
	}
	return client.SuspendWorkflow(ctx, run.WorkflowName, firstNonEmpty(run.ArgoNamespace, uc.namespace))
}

func (uc *Usecase) resumeRuntimeRun(ctx context.Context, run *models.PipelineRun) error {
	if uc.runtimeAdapter != nil {
		return uc.runtimeAdapter.Resume(ctx, uc.runtimeRefForRun(run))
	}
	client, err := uc.resolveArgoClientForRun(ctx, run)
	if err != nil {
		return err
	}
	if client == nil {
		return ErrWorkflowUnavailable
	}
	return client.ResumeWorkflow(ctx, run.WorkflowName, firstNonEmpty(run.ArgoNamespace, uc.namespace))
}

func (uc *Usecase) terminateRuntimeRun(ctx context.Context, run *models.PipelineRun) error {
	if uc.runtimeAdapter != nil {
		return uc.runtimeAdapter.Terminate(ctx, uc.runtimeRefForRun(run))
	}
	client, err := uc.resolveArgoClientForRun(ctx, run)
	if err != nil {
		return err
	}
	if client == nil {
		return ErrWorkflowUnavailable
	}
	return client.TerminateWorkflow(ctx, run.WorkflowName, firstNonEmpty(run.ArgoNamespace, uc.namespace))
}

func collectRunInputsFromPipelineJSON(runID string, pipeline map[string]interface{}, items *[]models.RunInput) {
	if len(pipeline) == 0 || items == nil {
		return
	}
	hasMaterializedConfigInputs := collectMaterializedRunConfigInputs(runID, pipeline, items)
	nodes, _ := interfaceSlice(pipeline["nodes"])
	for _, nodeAny := range nodes {
		node, ok := stringAnyMap(nodeAny)
		if !ok {
			continue
		}
		collectRunInputsFromPipelineNode(runID, node, items, !hasMaterializedConfigInputs)
	}
}

func collectMaterializedRunConfigInputs(runID string, pipeline map[string]interface{}, items *[]models.RunInput) bool {
	rawInputs, ok := interfaceSlice(pipeline[runConfigInputsPipelineJSONKey])
	if !ok || len(rawInputs) == 0 {
		return false
	}
	start := len(*items)
	for _, rawInput := range rawInputs {
		cfg, ok := stringAnyMap(rawInput)
		if !ok {
			continue
		}
		snapshot := copyStringAnyMap(cfg)
		delete(snapshot, "content")
		source := stringFromAny(cfg["mode"])
		if source == "" {
			source = stringFromAny(cfg["source"])
		}
		*items = append(*items, models.RunInput{
			ID:             fmt.Sprintf("%s:config:%d", runID, len(*items)),
			RunID:          runID,
			NodeID:         stringFromAny(cfg["nodeId"]),
			Type:           "config",
			RefID:          stringFromAny(cfg["configId"]),
			RefVersion:     stringFromAny(cfg["version"]),
			FileName:       stringFromAny(cfg["fileName"]),
			MountPath:      stringFromAny(cfg["mountPath"]),
			TargetFilename: stringFromAny(cfg["targetFilename"]),
			ContentHash:    stringFromAny(cfg["contentHash"]),
			ProjectionKey:  stringFromAny(cfg["projectionKey"]),
			Source:         source,
			Snapshot:       snapshot,
		})
	}
	return len(*items) > start
}

func collectRunInputsFromPipelineNode(runID string, node map[string]interface{}, items *[]models.RunInput, includeConfig bool) {
	nodeID := stringFromAny(node["id"])
	if includeConfig {
		if cfg, ok := stringAnyMap(node["runtimeConfig"]); ok {
			contentHash := ""
			if rawContent, ok := cfg["content"]; ok {
				if content, ok := rawContent.(string); ok {
					contentHash = runtimeConfigContentHash(content)
				} else {
					contentHash = runtimeConfigContentHash(fmt.Sprintf("%v", rawContent))
				}
			}
			snapshot := copyStringAnyMap(cfg)
			delete(snapshot, "content")
			version := stringFromAny(cfg["version"])
			*items = append(*items, models.RunInput{
				ID:             fmt.Sprintf("%s:%s:config:%d", runID, nodeID, len(*items)),
				RunID:          runID,
				NodeID:         nodeID,
				Type:           "config",
				RefID:          stringFromAny(cfg["configId"]),
				RefVersion:     version,
				FileName:       stringFromAny(cfg["fileName"]),
				MountPath:      stringFromAny(cfg["mountPath"]),
				TargetFilename: stringFromAny(cfg["targetFilename"]),
				ContentHash:    contentHash,
				ProjectionKey:  stringFromAny(cfg["projectionKey"]),
				Source:         stringFromAny(cfg["mode"]),
				Snapshot:       snapshot,
			})
		}
	}
	if component, ok := stringAnyMap(node["component"]); ok {
		if args, ok := interfaceSlice(component["args"]); ok {
			for _, argAny := range args {
				arg, ok := stringAnyMap(argAny)
				if !ok {
					continue
				}
				name := stringFromAny(arg["name"])
				if name == "" {
					continue
				}
				*items = append(*items, models.RunInput{
					ID:       fmt.Sprintf("%s:%s:param:%s", runID, nodeID, name),
					RunID:    runID,
					NodeID:   nodeID,
					Type:     "parameter",
					RefID:    name,
					Source:   "component_args",
					Snapshot: copyStringAnyMap(arg),
				})
			}
		}
	}
	for _, key := range []string{"nodes", "subNodes"} {
		children, _ := interfaceSlice(node[key])
		for _, childAny := range children {
			child, ok := stringAnyMap(childAny)
			if ok {
				collectRunInputsFromPipelineNode(runID, child, items, includeConfig)
			}
		}
	}
}

func copyStringAnyMap(in map[string]interface{}) map[string]interface{} {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func stringAnyMap(in interface{}) (map[string]interface{}, bool) {
	switch v := in.(type) {
	case map[string]interface{}:
		return v, true
	default:
		return nil, false
	}
}

func interfaceSlice(in interface{}) ([]interface{}, bool) {
	switch v := in.(type) {
	case []interface{}:
		return v, true
	default:
		return nil, false
	}
}

func stringFromAny(in interface{}) string {
	switch v := in.(type) {
	case string:
		return strings.TrimSpace(v)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return fmt.Sprintf("%v", v)
	default:
		return ""
	}
}

// SaveFromDeployment creates a new template from a deployment's pipeline JSON (F7.8).
func (uc *Usecase) SaveFromDeployment(ctx context.Context, deploymentID, templateName string) (*models.PipelineTemplate, error) {
	d, err := uc.deploymentRepo.FindByID(ctx, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("find deployment: %w", err)
	}
	if d == nil {
		return nil, ErrDeploymentNotFound
	}
	if d.PipelineJSON == nil {
		return nil, fmt.Errorf("deployment %s has no pipeline JSON", deploymentID)
	}
	name := templateName
	if name == "" {
		name = d.PipelineName + "-from-deployment"
	}
	return uc.SaveTemplate(ctx, name, d.PipelineJSON, "dev", "")
}

// ── Deployments ───────────────────────────────────────────────────

// maxActiveDeploymentStatusRefresh caps Argo status polls per ListDeployments call.
const maxActiveDeploymentStatusRefresh = 50

const deploymentStatusExpired = "Expired"

const (
	staleWorkflowTTLCleanupMessage = "Argo 工作流已被 TTL 清理"
	messageWorkflowAwaitingDeploy  = "等待绑定 Argo workflow"
	messageWorkflowUnavailable     = "Argo workflow 在集群中不可访问（可能已 TTL 清理）"
)

// staleActiveRunMaxAge caps how long a legitimate active run can be — beyond
// this, staleness is a display-side blocking hint (runstate.AnnotateRunDiagnostics
// stamps BlockingReason=stale_running), NOT a status flip.
//
// CYB-3491 rationale (from #417): tasks may wait. The old writer path used
// this threshold to hard-fail runs whose Argo workflow was still queued —
// violating the "waiting is not failure" invariant. The workflow is untouched
// in Argo; if a webhook or poll later brings an update, the diagnostic hint
// naturally clears.
const staleActiveRunMaxAge = 48 * time.Hour

// workflowCreateVisibilityGracePeriod avoids marking brand-new runs as expired
// while Argo is still creating the workflow CR.
const workflowCreateVisibilityGracePeriod = 5 * time.Minute

func isActiveDeploymentStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case "", "Running", "Pending", "Unknown", "Suspended":
		return true
	default:
		return false
	}
}

// IsActiveDeploymentStatus is the exported form of isActiveDeploymentStatus,
// for callers outside this package (e.g. the legacy workflow handler) that
// need to apply the exact same active-vs-terminal check as the Run Kernel path.
func IsActiveDeploymentStatus(status string) bool {
	return isActiveDeploymentStatus(status)
}

// isSucceededRunStatus reports whether the run status is a successful terminal
// state. Success is final in Argo (a workflow never un-succeeds), so it is the
// only status protected by the monotonicity guard in persistRunObservation.
func isSucceededRunStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case string(wfv1.WorkflowSucceeded), "completed", "success":
		return true
	default:
		return false
	}
}

// isDefinitiveTerminalFailure reports whether a run represents a permanent
// pre-submission or terminal failure that must never be revived to an active
// status — as opposed to a transient/misclassified terminal state that may
// legitimately be recovered. The distinguishing signal is the message: a real
// failure (e.g. the resource guard rejecting an over-spec node before the
// workflow is ever created) carries a substantive error message, while a
// misclassification (TTL-cleanup false positive, awaiting-deploy placeholder)
// carries one of the known stale/transient "workflow unavailable" messages.
func isDefinitiveTerminalFailure(run *models.PipelineRun) bool {
	if run == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(run.Status)) {
	case "failed", "error":
	default:
		return false
	}
	msg := strings.TrimSpace(run.Message)
	if msg == "" || isStaleWorkflowUnavailableMessage(msg) {
		return false
	}
	// Legacy verdicts from the (removed) unschedulable guard are NOT
	// definitive: the workflow was never stopped in Argo — it kept queueing.
	// Leaving these revivable lets the misclassified-reconcile path restore
	// them to their true active phase (CYB-3491: waiting is not failure).
	if isUnschedulableGuardVerdict(msg) {
		return false
	}
	return true
}

// isUnschedulableGuardVerdict matches the message minted by the pre-CYB-3491
// unschedulable guard ("Kubernetes 调度失败:…"), which mass-failed runs whose
// pods were merely waiting for capacity.
func isUnschedulableGuardVerdict(message string) bool {
	return strings.HasPrefix(strings.TrimSpace(message), "Kubernetes 调度失败")
}

func shouldWaitForWorkflowCreation(run *models.PipelineRun, now time.Time) bool {
	if run == nil {
		return false
	}
	if strings.TrimSpace(run.WorkflowName) == "" {
		return true
	}
	if isPendingBatchWorkflowCreation(run) {
		return true
	}
	if !isActiveDeploymentStatus(run.Status) {
		return false
	}
	if run.CreatedAt.IsZero() {
		return false
	}
	return now.Sub(run.CreatedAt) < workflowCreateVisibilityGracePeriod
}

func isPendingBatchWorkflowCreation(run *models.PipelineRun) bool {
	if run == nil || run.BatchJobID == nil || strings.TrimSpace(*run.BatchJobID) == "" {
		return false
	}
	if strings.TrimSpace(run.ArgoWorkflowUID) != "" {
		return false
	}
	if isBatchSubtaskPlaceholderWorkflowName(run.WorkflowName) {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(run.Status), "Pending")
}

func isMisclassifiedTerminalRunStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed", "error", "expired":
		return true
	default:
		return false
	}
}

func (uc *Usecase) refreshDeploymentStatus(ctx context.Context, d *models.PipelineDeployment) {
	if d == nil || !isActiveDeploymentStatus(d.Status) {
		return
	}
	// CYB-3486d1: probe status on the deployment's own cluster. Deploy dual-writes
	// a pipeline_runs row (same id / workflow_name) that carries the execution
	// target, so recover the run and resolve its cluster client. Without this a
	// non-default-cluster deployment is probed on the cyber-clust singleton,
	// 404s, and gets wrongly marked Expired (#436/#437 class). A nil client (no
	// factory + no singleton) means we skip the refresh, as before.
	var run *models.PipelineRun
	if uc.runRepo != nil {
		run, _ = uc.runRepo.FindByWorkflowName(ctx, d.WorkflowName)
	}
	client, err := uc.resolveArgoClientForRun(ctx, run)
	if err != nil || client == nil {
		return
	}
	namespace := uc.namespace
	if run != nil && strings.TrimSpace(run.ArgoNamespace) != "" {
		namespace = strings.TrimSpace(run.ArgoNamespace)
	}
	phase, err := client.GetWorkflowStatus(ctx, d.WorkflowName, namespace)
	if err != nil {
		if errors.Is(err, argo.ErrNotFound) {
			d.Status = deploymentStatusExpired
			logPipelineSideEffect("mark expired deployment", uc.deploymentRepo.UpdateStatus(ctx, d.ID, deploymentStatusExpired))
		}
		return
	}
	d.Status = string(phase)
	if phase == "Succeeded" || phase == "Failed" || phase == "Error" {
		now := time.Now().UTC()
		d.FinishedAt = &now
	}
	logPipelineSideEffect("update deployment status", uc.deploymentRepo.UpdateStatus(ctx, d.ID, string(phase)))
}

// ListDeployments returns all deployments without live Argo refresh (fast list).
func (uc *Usecase) ListDeployments(ctx context.Context) ([]models.PipelineDeployment, error) {
	return uc.listDeployments(ctx, false)
}

func (uc *Usecase) listDeployments(ctx context.Context, refreshActive bool) ([]models.PipelineDeployment, error) {
	if uc.runRepo != nil {
		// CYB-3491(紧急):ListRunSummaries(ctx) 无 filter 走 FindAllSummaries,
		// 全表扫。dev 30k+ 行时 39s + 撑爆连接。这个 endpoint 是
		// /api/v1/pipelines 的支线("最近的部署")——最近 500 条已够,
		// legacy deployments 走独立 FindAll(不受影响)。
		runs, _, err := uc.ListRunSummaries(ctx, models.PipelineRunListFilter{PageSize: 500})
		if err != nil {
			return nil, err
		}
		legacy, err := uc.deploymentRepo.FindAll(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]models.PipelineDeployment, 0, len(runs)+len(legacy))
		seen := make(map[string]struct{}, len(runs))
		for i := range runs {
			dep := runToDeployment(&runs[i])
			out = append(out, *dep)
			seen[dep.ID] = struct{}{}
		}
		for i := range legacy {
			if _, ok := seen[legacy[i].ID]; ok {
				continue
			}
			uc.enrichDeployment(&legacy[i])
			out = append(out, legacy[i])
		}
		sort.SliceStable(out, func(i, j int) bool {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		})
		return out, nil
	}
	list, err := uc.deploymentRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	// Refresh status for active workflows (capped to avoid N+1 storms).
	if refreshActive && uc.wfClient != nil {
		refreshed := 0
		for i := range list {
			if refreshed >= maxActiveDeploymentStatusRefresh {
				break
			}
			if isActiveDeploymentStatus(list[i].Status) {
				refreshed++
				uc.refreshDeploymentStatus(ctx, &list[i])
			}
		}
	}
	for i := range list {
		uc.enrichDeployment(&list[i])
	}
	return list, nil
}

// GetDeployment returns a single deployment with refreshed status.
func (uc *Usecase) GetDeployment(ctx context.Context, id string) (*models.PipelineDeployment, error) {
	d, err := uc.deploymentRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		if uc.runRepo != nil {
			run, err := uc.GetRun(ctx, id)
			if err != nil || run == nil {
				return nil, err
			}
			return runToDeployment(run), nil
		}
		return nil, nil
	}
	uc.refreshDeploymentStatus(ctx, d)
	uc.enrichDeployment(d)
	return d, nil
}

func (uc *Usecase) enrichDeployment(d *models.PipelineDeployment) {
	if d == nil {
		return
	}
	if len(d.AssetIDs) == 0 {
		d.AssetIDs = assetIDsFromPipelineJSON(d.PipelineJSON)
	}
	d.AssetCount = len(d.AssetIDs)
	target := uc.defaultExecutionTarget()
	d.ExecutionTarget = &target
}

// DeleteDeployment removes a deployment record and optionally deletes the K8s
// workflow.
//
// CYB-3486d1: cluster routing lives in the runRepo path. In production runRepo
// is always wired (cmd/server/core.go), so this delegates to DeleteRun, which
// resolves the run's cluster via resolveArgoClientForRun and deletes on the
// right cluster. The uc.wfClient branch below only runs in the legacy
// runRepo==nil configuration, where the default-cluster singleton is the only
// cluster — cluster-correct by construction (no non-default clusters exist
// without the run store).
func (uc *Usecase) DeleteDeployment(ctx context.Context, id string) error {
	if uc.runRepo != nil {
		return uc.DeleteRun(ctx, id)
	}
	if uc.wfClient != nil {
		d, err := uc.deploymentRepo.FindByID(ctx, id)
		if err == nil && d != nil {
			logPipelineSideEffect("delete workflow", uc.wfClient.DeleteWorkflow(ctx, d.WorkflowName, uc.namespace))
		}
	}
	return uc.deploymentRepo.Delete(ctx, id)
}

// RetryDeployment re-deploys from a saved deployment's pipeline JSON.
func (uc *Usecase) RetryDeployment(ctx context.Context, id string) (*models.PipelineDeployment, error) {
	if uc.runRepo != nil {
		run, err := uc.RetryRun(ctx, id)
		if err != nil {
			return nil, err
		}
		return runToDeployment(run), nil
	}
	d, err := uc.deploymentRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find deployment: %w", err)
	}
	if d == nil {
		return nil, ErrDeploymentNotFound
	}
	assetIDs := assetIDsFromPipelineJSON(d.PipelineJSON)
	return uc.Deploy(ctx, d.PipelineJSON, d.PipelineName+"-retry", assetIDs)
}

// StopDeployment stops a running workflow by setting its Shutdown strategy.
//
// CYB-3486d1: same cluster-routing split as DeleteDeployment — production
// (runRepo wired) delegates to the cluster-aware StopRun; the uc.wfClient
// branch is the legacy runRepo==nil single-cluster path only.
func (uc *Usecase) StopDeployment(ctx context.Context, id string) error {
	if uc.runRepo != nil {
		return uc.StopRun(ctx, id)
	}
	d, err := uc.deploymentRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("find deployment: %w", err)
	}
	if d == nil {
		return ErrDeploymentNotFound
	}
	if uc.wfClient == nil {
		return fmt.Errorf("workflow client not available")
	}
	return uc.wfClient.StopWorkflow(ctx, d.WorkflowName, uc.namespace)
}

// ── Pipeline Output Registration (F4.3) ──────────────────────────

// RegisterPipelineOutputInput describes an asset produced by a pipeline step.
type RegisterPipelineOutputInput struct {
	DeploymentID string                 `json:"deployment_id"`
	NodeID       string                 `json:"node_id"`
	AssetID      string                 `json:"asset_id"`
	StorageURI   string                 `json:"storage_uri"`
	AssetType    string                 `json:"asset_type"`
	Files        map[string]string      `json:"files,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// RegisterOutput creates a new asset from a pipeline step output and records
// an asset_event for lineage tracing. The container calls back to this endpoint
// (via the PIPELINE_DEPLOYMENT_ID injected env var) to register results.
func (uc *Usecase) RegisterOutput(ctx context.Context, in RegisterPipelineOutputInput) (*models.Asset, error) {
	// Validate deployment exists.
	dep, err := uc.deploymentRepo.FindByID(ctx, in.DeploymentID)
	if err != nil {
		return nil, fmt.Errorf("find deployment: %w", err)
	}
	if dep == nil {
		return nil, ErrDeploymentNotFound
	}

	assetID := in.AssetID
	if assetID == "" {
		assetID = uuid.New().String()
	}

	asset := &models.Asset{
		AssetID:    assetID,
		StorageURI: in.StorageURI,
		AssetType:  in.AssetType,
		Files:      in.Files,
		Metadata:   in.Metadata,
		CreatedAt:  time.Now().UTC(),
	}
	if err := seedPipelineOutputVersion(ctx, uc.logicalRepo, asset); err != nil {
		return nil, fmt.Errorf("seed pipeline output version: %w", err)
	}
	if err := uc.assetRepo.InsertNew(ctx, asset); err != nil {
		return nil, fmt.Errorf("insert asset: %w", err)
	}

	// Record asset_event for lineage.
	if uc.assetEventRepo != nil {
		eventPayload, _ := json.Marshal(map[string]interface{}{
			"deployment_id": in.DeploymentID,
			"node_id":       in.NodeID,
			"pipeline_name": dep.PipelineName,
		})
		logPipelineSideEffect("append pipeline_output event", uc.assetEventRepo.Append(ctx, repository.AssetEventAppendInput{
			EventType:     "pipeline_output",
			AggregateType: "asset",
			AssetID:       assetID,
			RunID:         in.DeploymentID,
			EventPayload:  eventPayload,
		}))
	}

	// Create asset_relations from input assets to output asset (F4.7).
	if uc.relationWriter != nil {
		switch inputIDs := dep.PipelineJSON["_input_asset_ids"].(type) {
		case []interface{}:
			for _, id := range inputIDs {
				if s, ok := id.(string); ok {
					logPipelineSideEffect("insert pipeline_output relation", uc.relationWriter.InsertRelation(ctx, s, assetID, "pipeline_output", in.DeploymentID))
				}
			}
		case []string:
			for _, s := range inputIDs {
				logPipelineSideEffect("insert pipeline_output relation", uc.relationWriter.InsertRelation(ctx, s, assetID, "pipeline_output", in.DeploymentID))
			}
		}
	}

	return asset, nil
}

// ── Lineage Query (F4.5) ──────────────────────────────────────────

// AssetLineage describes how an asset was produced by a pipeline.
type AssetLineage struct {
	AssetID      string   `json:"asset_id"`
	DeploymentID string   `json:"deployment_id,omitempty"`
	PipelineName string   `json:"pipeline_name,omitempty"`
	WorkflowName string   `json:"workflow_name,omitempty"`
	NodeID       string   `json:"node_id,omitempty"`
	InputAssets  []string `json:"input_assets,omitempty"`
	ProducedAt   string   `json:"produced_at,omitempty"`
}

// GetLineage returns pipeline provenance for an asset (which run + step produced it).
func (uc *Usecase) GetLineage(ctx context.Context, assetID string) (*AssetLineage, error) {
	if uc.assetEventRepo == nil {
		return nil, fmt.Errorf("asset event repo not available")
	}

	events, err := uc.assetEventRepo.ListByAsset(ctx, assetID, repository.AssetEventListOptions{
		EventTypes: []string{"pipeline_output"},
		Limit:      1,
	})
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	if len(events) == 0 {
		return &AssetLineage{AssetID: assetID}, nil
	}

	ev := events[0]
	lineage := &AssetLineage{
		AssetID:    assetID,
		ProducedAt: ev.OccurredAt.Format(time.RFC3339),
	}

	// Parse event payload for deployment_id, node_id, pipeline_name.
	var payload map[string]interface{}
	if len(ev.EventPayload) > 0 {
		_ = json.Unmarshal(ev.EventPayload, &payload)
		if did, ok := payload["deployment_id"].(string); ok {
			lineage.DeploymentID = did
		}
		if nid, ok := payload["node_id"].(string); ok {
			lineage.NodeID = nid
		}
		if pn, ok := payload["pipeline_name"].(string); ok {
			lineage.PipelineName = pn
		}
	}

	// Look up deployment for pipeline details + input assets.
	dep, err := uc.deploymentRepo.FindByID(ctx, lineage.DeploymentID)
	if err == nil && dep != nil {
		if lineage.PipelineName == "" {
			lineage.PipelineName = dep.PipelineName
		}
		lineage.WorkflowName = dep.WorkflowName
		// Extract input asset IDs from PipelineJSON.
		if dep.PipelineJSON != nil {
			switch raw := dep.PipelineJSON["_input_asset_ids"].(type) {
			case []interface{}:
				for _, id := range raw {
					if s, ok := id.(string); ok {
						lineage.InputAssets = append(lineage.InputAssets, s)
					}
				}
			case []string:
				lineage.InputAssets = raw
			}
		}
	}

	return lineage, nil
}

// ── Resource Usage (F5.8) ────────────────────────────────────────

// ResourceUsageReport describes per-pod resource usage for a deployment.
type ResourceUsageReport struct {
	DeploymentID         string              `json:"deployment_id,omitempty"`
	WorkflowName         string              `json:"workflow_name"`
	Status               string              `json:"status"`
	ObservedAt           string              `json:"observed_at"`
	Source               ResourceUsageSource `json:"source"`
	LiveMetricsAvailable bool                `json:"live_metrics_available"`
	Pods                 []PodResourceUsage  `json:"pods"`
}

// ResourceUsageSource describes where each resource signal came from.
type ResourceUsageSource struct {
	Workflow string `json:"workflow"`
	Metrics  string `json:"metrics"`
	Spec     string `json:"spec"`
}

// ResourceValues carries CPU and memory resource quantities.
type ResourceValues struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// PodResourceUsage summarizes per-pod runtime duration and template resource requests.
type PodResourceUsage struct {
	PodName                string         `json:"pod_name"`
	NodeID                 string         `json:"node_id,omitempty"`
	NodeName               string         `json:"node_name,omitempty"`
	TemplateName           string         `json:"template_name,omitempty"`
	ObservedAt             string         `json:"observed_at,omitempty"`
	LiveMetricsAvailable   bool           `json:"live_metrics_available"`
	Requests               ResourceValues `json:"requests"`
	Limits                 ResourceValues `json:"limits"`
	ResourceDuration       ResourceValues `json:"resource_duration"`
	CPUUsage               string         `json:"cpu_usage"`
	MemoryUsage            string         `json:"memory_usage"`
	CPUResourceDuration    string         `json:"cpu_resource_duration,omitempty"`
	MemoryResourceDuration string         `json:"memory_resource_duration,omitempty"`
	CPURequest             string         `json:"cpu_request"`
	MemoryRequest          string         `json:"memory_request"`
	CPULimit               string         `json:"cpu_limit"`
	MemoryLimit            string         `json:"memory_limit"`
}

// GetResourceUsage returns resource usage for pods belonging to a deployment.
func (uc *Usecase) GetResourceUsage(ctx context.Context, deploymentID string) (*ResourceUsageReport, error) {
	d, err := uc.deploymentRepo.FindByID(ctx, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("find deployment: %w", err)
	}
	if d == nil {
		return nil, ErrDeploymentNotFound
	}

	report := &ResourceUsageReport{
		DeploymentID: deploymentID,
		WorkflowName: d.WorkflowName,
		Status:       d.Status,
		ObservedAt:   time.Now().UTC().Format(time.RFC3339),
		Source: ResourceUsageSource{
			Workflow: "unavailable",
			Metrics:  "unavailable",
			Spec:     specSource(d.Manifest),
		},
		LiveMetricsAvailable: false,
		Pods:                 []PodResourceUsage{},
	}

	if d.WorkflowName != "" {
		// CYB-3486d1: the pipeline_deployments row carries no cluster, but Deploy
		// dual-writes a pipeline_runs row (same id / workflow_name) that does.
		// Resolve it so a non-default-cluster run is read through its own argo
		// client instead of the cyber-clust singleton (#436/#437 misroute class).
		var run *models.PipelineRun
		if uc.runRepo != nil {
			run, _ = uc.runRepo.FindByWorkflowName(ctx, d.WorkflowName)
		}
		client, err := uc.resolveArgoClientForRun(ctx, run)
		if err != nil {
			return nil, err
		}
		if client != nil {
			namespace := uc.namespace
			if run != nil && strings.TrimSpace(run.ArgoNamespace) != "" {
				namespace = strings.TrimSpace(run.ArgoNamespace)
			} else if d.ExecutionTarget != nil && strings.TrimSpace(d.ExecutionTarget.Namespace) != "" {
				namespace = strings.TrimSpace(d.ExecutionTarget.Namespace)
			}
			wf, err := client.GetWorkflow(ctx, d.WorkflowName, namespace)
			if err != nil {
				return nil, fmt.Errorf("get workflow: %w", err)
			}
			if wf != nil {
				if wf.Status.Phase != "" {
					report.Status = string(wf.Status.Phase)
				}
				report.Source.Workflow = "argo-live"
				report.Pods = buildPodResourceUsageReport(wf, d.Manifest, report.ObservedAt, "")
			}
		}
	}

	return report, nil
}

// GetWorkflowResourceUsage returns resource usage metadata for a workflow.
func (uc *Usecase) GetWorkflowResourceUsage(ctx context.Context, workflowName string) (*ResourceUsageReport, error) {
	return uc.getWorkflowResourceUsage(ctx, workflowName, "")
}

// GetWorkflowNodeResourceUsage returns resource usage metadata for one workflow node.
func (uc *Usecase) GetWorkflowNodeResourceUsage(ctx context.Context, workflowName, nodeID string) (*ResourceUsageReport, error) {
	return uc.getWorkflowResourceUsage(ctx, workflowName, nodeID)
}

func (uc *Usecase) getWorkflowResourceUsage(ctx context.Context, workflowName, nodeID string) (*ResourceUsageReport, error) {
	if strings.TrimSpace(workflowName) == "" {
		return nil, ErrInvalidArgument
	}

	var manifest *string
	var deploymentID string
	namespace := uc.namespace
	// CYB-3486d1: keep the resolved run so the workflow read below goes through
	// the run's own cluster client. This path is live (WorkflowNodeDetailPanel →
	// GET /workflows/:name/nodes/:id/resources); a run on a non-default cluster
	// must not be read through the cyber-clust singleton or GetWorkflow 404s
	// (the #436/#437 cross-cluster misroute class).
	var run *models.PipelineRun
	// Prefer the first-class pipeline_runs table (workflow_name UNIQUE) so we
	// avoid a full table scan over pipeline_deployments. Fall back to the
	// legacy compatibility read only when the run repo has no matching row.
	if uc.runRepo != nil {
		if r, err := uc.runRepo.FindByWorkflowName(ctx, workflowName); err == nil && r != nil {
			run = r
			manifest = r.Manifest
			deploymentID = r.ID
			if strings.TrimSpace(r.ArgoNamespace) != "" {
				namespace = strings.TrimSpace(r.ArgoNamespace)
			}
		}
	}
	if manifest == nil && uc.deploymentRepo != nil {
		if deployments, err := uc.deploymentRepo.FindAll(ctx); err == nil {
			for i := range deployments {
				if deployments[i].WorkflowName == workflowName {
					manifest = deployments[i].Manifest
					deploymentID = deployments[i].ID
					if deployments[i].ExecutionTarget != nil && strings.TrimSpace(deployments[i].ExecutionTarget.Namespace) != "" {
						namespace = strings.TrimSpace(deployments[i].ExecutionTarget.Namespace)
					}
					break
				}
			}
		}
	}

	// run is nil for legacy deployment-only rows; resolveArgoClientForRun then
	// routes to cluster-default (the singleton), correct for pre-multi-cluster
	// rows. A nil client (no factory + no singleton) means workflow unavailable.
	client, err := uc.resolveArgoClientForRun(ctx, run)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, ErrWorkflowUnavailable
	}

	wf, err := client.GetWorkflow(ctx, workflowName, namespace)
	if err != nil {
		return nil, fmt.Errorf("get workflow: %w", err)
	}
	observedAt := time.Now().UTC().Format(time.RFC3339)
	report := &ResourceUsageReport{
		DeploymentID: deploymentID,
		WorkflowName: workflowName,
		Status:       string(wf.Status.Phase),
		ObservedAt:   observedAt,
		Source: ResourceUsageSource{
			Workflow: "argo-live",
			Metrics:  "unavailable",
			Spec:     specSource(manifest),
		},
		LiveMetricsAvailable: false,
		Pods:                 buildPodResourceUsageReport(wf, manifest, observedAt, nodeID),
	}
	if nodeID != "" && len(report.Pods) == 0 {
		return nil, ErrDeploymentNotFound
	}
	return report, nil
}

// ── Pipeline Diff (F2.12) ────────────────────────────────────────

// PipelineDiff describes the structural diff between two template versions.
type PipelineDiff struct {
	AddedNodes    []DiffNode `json:"added_nodes"`
	RemovedNodes  []DiffNode `json:"removed_nodes"`
	ModifiedNodes []DiffNode `json:"modified_nodes"`
	AddedEdges    []DiffEdge `json:"added_edges"`
	RemovedEdges  []DiffEdge `json:"removed_edges"`
}

// DiffNode is a node in a diff result.
type DiffNode struct {
	ID        string                 `json:"id"`
	Component map[string]interface{} `json:"component,omitempty"`
	Inputs    []interface{}          `json:"inputs,omitempty"`
	Outputs   []interface{}          `json:"outputs,omitempty"`
}

// DiffEdge is an edge in a diff result.
type DiffEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// DiffTemplates compares two pipeline templates and returns a structural diff.
func (uc *Usecase) DiffTemplates(ctx context.Context, id1, id2 string) (*PipelineDiff, error) {
	t1, err := uc.templateRepo.FindByID(ctx, id1)
	if err != nil {
		return nil, fmt.Errorf("find template %s: %w", id1, err)
	}
	if t1 == nil {
		return nil, fmt.Errorf("template %s: %w", id1, ErrTemplateNotFound)
	}
	t2, err := uc.templateRepo.FindByID(ctx, id2)
	if err != nil {
		return nil, fmt.Errorf("find template %s: %w", id2, err)
	}
	if t2 == nil {
		return nil, fmt.Errorf("template %s: %w", id2, ErrTemplateNotFound)
	}

	p1Raw, err := json.Marshal(t1.Pipeline)
	if err != nil {
		return nil, fmt.Errorf("marshal t1 pipeline: %w", err)
	}
	p2Raw, err := json.Marshal(t2.Pipeline)
	if err != nil {
		return nil, fmt.Errorf("marshal t2 pipeline: %w", err)
	}

	p1, _, err := rawToPipeline(p1Raw)
	if err != nil {
		return nil, fmt.Errorf("parse t1 pipeline: %w", err)
	}
	p2, _, err := rawToPipeline(p2Raw)
	if err != nil {
		return nil, fmt.Errorf("parse t2 pipeline: %w", err)
	}

	// Build lookup maps.
	nodes1 := make(map[string]map[string]interface{})
	for _, n := range p1.Nodes {
		nodes1[n.ID] = nodeToMap(n)
	}
	nodes2 := make(map[string]map[string]interface{})
	for _, n := range p2.Nodes {
		nodes2[n.ID] = nodeToMap(n)
	}

	edges1 := make(map[string]bool)
	for _, e := range p1.Edges {
		edges1[e.Source+"|"+e.Target] = true
	}
	edges2 := make(map[string]bool)
	for _, e := range p2.Edges {
		edges2[e.Source+"|"+e.Target] = true
	}

	diff := &PipelineDiff{}

	// Find added/modified nodes.
	for _, n2 := range p2.Nodes {
		n1Raw, exists := nodes1[n2.ID]
		if !exists {
			diff.AddedNodes = append(diff.AddedNodes, nodeToDiffNode(n2))
		} else {
			n2Raw := nodeToMap(n2)
			if !mapsEqual(n1Raw, n2Raw) {
				diff.ModifiedNodes = append(diff.ModifiedNodes, nodeToDiffNode(n2))
			}
		}
	}

	// Find removed nodes.
	for _, n1 := range p1.Nodes {
		if _, exists := nodes2[n1.ID]; !exists {
			diff.RemovedNodes = append(diff.RemovedNodes, nodeToDiffNode(n1))
		}
	}

	// Find added edges.
	for _, e2 := range p2.Edges {
		key := e2.Source + "|" + e2.Target
		if !edges1[key] {
			diff.AddedEdges = append(diff.AddedEdges, DiffEdge{Source: e2.Source, Target: e2.Target})
		}
	}

	// Find removed edges.
	for _, e1 := range p1.Edges {
		key := e1.Source + "|" + e1.Target
		if !edges2[key] {
			diff.RemovedEdges = append(diff.RemovedEdges, DiffEdge{Source: e1.Source, Target: e1.Target})
		}
	}

	return diff, nil
}

func nodeToMap(n transpiler.Node) map[string]interface{} {
	m := map[string]interface{}{
		"id": n.ID,
	}
	if n.Component.Name != "" || n.Component.Image != "" {
		comp := map[string]interface{}{
			"name":  n.Component.Name,
			"image": n.Component.Image,
		}
		if len(n.Component.Command) > 0 {
			comp["command"] = n.Component.Command
		}
		if n.Component.Resources != nil {
			comp["resources"] = n.Component.Resources
		}
		m["component"] = comp
	}
	if len(n.Inputs) > 0 {
		m["inputs"] = n.Inputs
	}
	if len(n.Outputs) > 0 {
		m["outputs"] = n.Outputs
	}
	return m
}

func nodeToDiffNode(n transpiler.Node) DiffNode {
	dn := DiffNode{ID: n.ID}
	if n.Component.Name != "" || n.Component.Image != "" {
		comp := map[string]interface{}{
			"name":  n.Component.Name,
			"image": n.Component.Image,
		}
		if len(n.Component.Command) > 0 {
			comp["command"] = n.Component.Command
		}
		if n.Component.Resources != nil {
			comp["resources"] = n.Component.Resources
		}
		dn.Component = comp
	}
	if len(n.Inputs) > 0 {
		dn.Inputs = make([]interface{}, len(n.Inputs))
		for i, p := range n.Inputs {
			dn.Inputs[i] = map[string]string{"name": p.Name, "type": p.Type}
		}
	}
	if len(n.Outputs) > 0 {
		dn.Outputs = make([]interface{}, len(n.Outputs))
		for i, p := range n.Outputs {
			dn.Outputs[i] = map[string]string{"name": p.Name, "type": p.Type}
		}
	}
	return dn
}

func mapsEqual(a, b map[string]interface{}) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}

// ── Helpers ───────────────────────────────────────────────────────

func rawToPipeline(raw json.RawMessage) (*transpiler.Pipeline, int, error) {
	var pipe transpiler.Pipeline
	if err := json.Unmarshal(raw, &pipe); err != nil {
		return nil, 0, err
	}
	return &pipe, len(pipe.Nodes), nil
}

func assetIDsFromPipelineJSON(pipeline map[string]interface{}) []string {
	if pipeline == nil {
		return nil
	}
	switch raw := pipeline["_input_asset_ids"].(type) {
	case []string:
		return raw
	case []interface{}:
		out := make([]string, 0, len(raw))
		for _, id := range raw {
			if s, ok := id.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
