package pipeline

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	"github.com/CyberOrigin2077/cyber-databrew/internal/metrics"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// CYB-3681 — DB-driven bulk-pull writeback.
//
// The watcher's active-run refresh used to issue one GetWorkflow per active
// run per tick (bounded to a rotating window of ≤50). With thousands of
// active runs that is both slow (starvation across windows) and expensive
// (N GETs against the K8s API). The bulk path replaces it:
//
//  1. per (cluster, namespace): ONE full LIST of active workflows, selected
//     by the positive label `workflows.argoproj.io/completed=false` (the Argo
//     controller stamps it on first reconcile). No pagination: the active set
//     is bounded by cluster capacity, and a consistent snapshot beats a
//     Continue-chain racing the controller.
//  2. every DB-active run present in the snapshot is applied from the listed
//     object — change-gated on resourceVersion so an unchanged workflow costs
//     zero DB writes. Every recalibrateEvery-th scan ignores the gate and
//     re-applies everything (drift sweep).
//  3. DB-active runs ABSENT from the snapshot get a bounded targeted GET
//     (refreshPipelineRunStatus): a just-finished workflow no longer matches
//     the selector but still exists (terminal apply), and a 404 flows into
//     the existing orphan grading (preserve window → ledger reconcile →
//     expired). Freshly submitted workflows without the label yet land here
//     too and simply read back their live phase.
//
// `WATCHER_MODE=legacy` restores the per-run rotating-window path (rollback
// hatch, same convention as BATCH_DISPATCH_MODE).

const (
	watcherModeEnv    = "WATCHER_MODE"
	watcherModeLegacy = "legacy"

	// activeWorkflowSelector matches workflows the Argo controller considers
	// live. Positive match (not `!=true`) so an apiserver that drops the
	// selector can only over-return, never silently hide active workflows.
	activeWorkflowSelector = "workflows.argoproj.io/completed=false"

	// recalibrateEvery: every N-th scan bypasses the resourceVersion gate and
	// re-applies every listed workflow, so a bug in the gate (or a missed
	// write) self-heals within N ticks instead of persisting forever.
	recalibrateEvery = 10
)

func watcherBulkModeEnabled() bool {
	return !strings.EqualFold(strings.TrimSpace(os.Getenv(watcherModeEnv)), watcherModeLegacy)
}

// markWorkflowApplied gates repeat applies of an unchanged workflow within
// this process. Returns true when rv is NEW for the run (caller must apply).
// Memory only: a restart re-applies everything once, which is exactly the
// recalibration semantic.
func (uc *Usecase) markWorkflowApplied(runID, rv string) bool {
	uc.watcherRVMu.Lock()
	defer uc.watcherRVMu.Unlock()
	if uc.watcherAppliedRV == nil {
		uc.watcherAppliedRV = map[string]string{}
	}
	if uc.watcherAppliedRV[runID] == rv && rv != "" {
		return false
	}
	uc.watcherAppliedRV[runID] = rv
	return true
}

// forgetWorkflowApplied drops a run from the RV gate (terminal runs must not
// pin map memory forever).
func (uc *Usecase) forgetWorkflowApplied(runID string) {
	uc.watcherRVMu.Lock()
	defer uc.watcherRVMu.Unlock()
	delete(uc.watcherAppliedRV, runID)
}

// recordActiveWorkflowCount stores (and publishes as a gauge) the active
// (completed=false) workflow count the bulk watcher observed for a namespace
// this scan. ActiveWorkflowCount reads it back for backfill admission
// backpressure (CYB-3681).
func (uc *Usecase) recordActiveWorkflowCount(namespace string, n int) {
	uc.activeWFMu.Lock()
	if uc.activeWFCount == nil {
		uc.activeWFCount = map[string]int{}
	}
	uc.activeWFCount[namespace] = n
	uc.activeWFMu.Unlock()
	metrics.DispatcherActiveWorkflows.WithLabelValues(namespace).Set(float64(n))
}

// ActiveWorkflowCount returns the last active (pending+running) workflow count
// observed for a namespace and whether any observation exists yet. The
// backfill submitter gates dispatch on it; a missing observation (false) fails
// open so dispatch is never wedged by a cold start or a stalled watcher.
func (uc *Usecase) ActiveWorkflowCount(namespace string) (int, bool) {
	uc.activeWFMu.Lock()
	defer uc.activeWFMu.Unlock()
	n, ok := uc.activeWFCount[namespace]
	return n, ok
}

// bulkSyncActiveRuns refreshes all DB-active runs from per-cluster LIST
// snapshots. Returns the number of runs whose state was applied or probed.
// residualLimit bounds the targeted GETs for snapshot-absent runs per cluster
// per tick (the residual set is naturally small: just-finished or just-created
// workflows).
func (uc *Usecase) bulkSyncActiveRuns(ctx context.Context, runs []models.PipelineRun, activeIdx []int, residualLimit int, recalibrate bool) int {
	if len(activeIdx) == 0 {
		return 0
	}
	byCluster := groupRunIndicesByCluster(runs, activeIdx, func(r *models.PipelineRun) string {
		return uc.resolveRunClusterID(ctx, r)
	})
	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		synced int
	)
	for cluster, group := range byCluster {
		wg.Add(1)
		go func(cluster string, group []int) {
			defer wg.Done()
			n := uc.bulkSyncClusterRuns(ctx, runs, group, residualLimit, recalibrate)
			mu.Lock()
			synced += n
			mu.Unlock()
			_ = cluster
		}(cluster, group)
	}
	wg.Wait()
	return synced
}

// bulkSyncClusterRuns handles one cluster's group: LIST once per namespace,
// apply matches, probe absentees.
func (uc *Usecase) bulkSyncClusterRuns(ctx context.Context, runs []models.PipelineRun, group []int, residualLimit int, recalibrate bool) int {
	if len(group) == 0 {
		return 0
	}
	client, err := uc.resolveArgoClientForRun(ctx, &runs[group[0]])
	if err != nil || client == nil {
		if err != nil {
			slog.Warn("bulk watcher: resolve argo client failed, falling back to per-run refresh",
				"clusterID", uc.resolveRunClusterID(ctx, &runs[group[0]]), "err", err)
		}
		return uc.legacyRefreshGroup(ctx, runs, group, residualLimit)
	}

	// One LIST per namespace present in the group (almost always exactly one).
	namespaces := map[string]bool{}
	for _, i := range group {
		ns := runs[i].ArgoNamespace
		if ns == "" {
			ns = uc.namespace
		}
		namespaces[ns] = true
	}
	listed := map[string]*wfv1.Workflow{} // "ns/name" → workflow
	nsActive := map[string]int{}          // namespace → active workflow count
	listOK := true
	for ns := range namespaces {
		items, err := client.ListWorkflows(ctx, ns, activeWorkflowSelector)
		if err != nil {
			// A failed LIST must NOT make every run in the namespace look
			// absent (mass orphan-probing a healthy cluster). Degrade to the
			// bounded per-run path for this tick.
			slog.Warn("bulk watcher: list active workflows failed, falling back to per-run refresh",
				"namespace", ns, "err", err)
			listOK = false
			break
		}
		nsActive[ns] = len(items)
		for i := range items {
			listed[ns+"/"+items[i].Name] = &items[i]
		}
	}
	if !listOK {
		return uc.legacyRefreshGroup(ctx, runs, group, residualLimit)
	}
	// Publish per-namespace active (completed=false) workflow counts so the
	// backfill submitter can backpressure dispatch against control-plane
	// saturation (CYB-3681). Per-namespace, not per-cluster: the Argo
	// controller (the thing that OOMs) is scoped to one namespace, and a
	// cluster can map to several namespaces.
	for ns, n := range nsActive {
		uc.recordActiveWorkflowCount(ns, n)
	}

	// Snapshot-hit apply is serialized (no network — just DB writes with
	// pooled connections; keeping it inline avoids goroutine setup for the
	// common case). Residual GETs are the slow path (K8s round-trip + DB
	// writes per run) — those we fan out via a bounded worker pool so a
	// tick's residual budget is drained in parallel rather than dragged
	// through serially. CYB-3746: the per-cluster serialization comment
	// upstairs predates PR #470's per-cluster QPS lift; QPS=50 has ample
	// headroom for a 20-way fanout without stampeding the client-side rate
	// limiter, and the effect on a 600-item budget is ~20× faster (~20 min
	// → ~1 min per scan).
	var synced atomic.Int64 // written from both the serial snapshot-hit path
	// and the residual worker goroutines, so it must be atomic (data race
	// otherwise when a group interleaves snapshot hits with residual GETs).
	residuals := 0
	var (
		wg  sync.WaitGroup
		sem = make(chan struct{}, watcherResidualConcurrency())
	)
	for _, i := range group {
		run := &runs[i]
		ns := run.ArgoNamespace
		if ns == "" {
			ns = uc.namespace
		}
		if wf, ok := listed[ns+"/"+strings.TrimSpace(run.WorkflowName)]; ok && run.WorkflowName != "" {
			if recalibrate || uc.markWorkflowApplied(run.ID, wf.ResourceVersion) {
				uc.applyWorkflowToRun(ctx, run, wf, nodeProjectTerminalArchive)
				if !isActiveDeploymentStatus(run.Status) {
					uc.forgetWorkflowApplied(run.ID)
				}
				synced.Add(1)
			}
			continue
		}
		// Absent from the active snapshot: terminal, orphaned, or not yet
		// labeled. Targeted GET resolves which; bounded per tick.
		if residuals >= residualLimit {
			continue
		}
		residuals++
		wg.Add(1)
		sem <- struct{}{}
		go func(run *models.PipelineRun) {
			defer wg.Done()
			defer func() { <-sem }()
			uc.refreshPipelineRunStatus(ctx, run)
			uc.forgetWorkflowApplied(run.ID)
			synced.Add(1)
		}(run)
	}
	wg.Wait()
	return int(synced.Load())
}

// watcherResidualConcurrency reads the bounded fanout size for residual GETs
// (env WATCHER_RESIDUAL_CONCURRENCY, default 20). Kept small enough to stay
// under the per-cluster client-go QPS/burst (50/100 on this deployment) even
// with brief bursts, and large enough to drain a 600-item budget in ~1 min.
func watcherResidualConcurrency() int {
	if raw := strings.TrimSpace(os.Getenv("WATCHER_RESIDUAL_CONCURRENCY")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			return n
		}
	}
	return 20
}

// legacyRefreshGroup is the bounded per-run fallback used when a cluster's
// LIST is unavailable this tick.
func (uc *Usecase) legacyRefreshGroup(ctx context.Context, runs []models.PipelineRun, group []int, limit int) int {
	n := 0
	for _, i := range group {
		if n >= limit {
			break
		}
		uc.refreshPipelineRunStatus(ctx, &runs[i])
		n++
	}
	return n
}
