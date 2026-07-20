package workflow

import (
	"context"
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/argo"
	"github.com/CyberOrigin2077/cyber-databrew/internal/k8s"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// clusterDefault is the cluster id the legacy single-cluster row uses; also the
// fallback whenever a run's cluster can't be resolved. Matches the seed id and
// argo.ClientFactory's own empty-target default.
const clusterDefault = "cluster-default"

// cluster_routing.go makes the /workflows/:name handler cluster-aware (CYB-3486).
//
// The handler used to hold a single Argo WorkflowClient wired at startup, which
// always targets the default cluster (cyber-clust argo-server). Any request for
// a workflow that lives on another cluster (delivery-clust) failed with
// "workflows.argoproj.io <name> not found". These helpers resolve, per request,
// the workflow's owning cluster (by name → run → execution target → cluster_id)
// and hand back the matching Argo client from the factory. Everything degrades
// to the injected singleton (h.wfClient) when the factory isn't wired or the
// cluster can't be resolved, so no-PG test infra and default-cluster workflows
// behave exactly as before. Mirrors usecase/pipeline resolveRunClusterID /
// resolveArgoClientForRun (the already-shipped /runs/* path, PR #437).

// resolveRunClusterID determines which cluster a run's workflow lives on.
//
// PipelineRun has no direct cluster_id column; the truth is
// execution_targets.cluster_id, reachable two ways:
//  1. run.ExecutionTarget populated → read it directly.
//  2. only run.ExecutionTargetID present (the FindByWorkflowName path, which does
//     not preload the nested target) → look the target up by id (memoized).
//
// A not-found target is deliberately NOT cached, so a transient DB error can't
// pin the run to cluster-default for the process lifetime.
func (h *Handler) resolveRunClusterID(ctx context.Context, run *models.PipelineRun) string {
	if run == nil {
		return clusterDefault
	}
	if run.ExecutionTarget != nil {
		if id := strings.TrimSpace(run.ExecutionTarget.ClusterID); id != "" {
			return id
		}
	}
	tid := strings.TrimSpace(run.ExecutionTargetID)
	if tid == "" || h.targetRepo == nil {
		return clusterDefault
	}
	if cached, ok := h.targetClusterCache.Load(tid); ok {
		if cid, _ := cached.(string); cid != "" {
			return cid
		}
	}
	t, err := h.targetRepo.FindByID(ctx, tid)
	if err != nil || t == nil {
		return clusterDefault
	}
	cid := strings.TrimSpace(t.ClusterID)
	if cid == "" {
		cid = clusterDefault
	}
	h.targetClusterCache.Store(tid, cid)
	return cid
}

// argoClientForCluster returns the Argo client for clusterID. Factory path when
// wired; singleton fallback when the factory is nil, the lookup errors, or it
// yields no client (so a misconfigured/unknown cluster degrades to prior
// behavior instead of erroring the whole request at resolution time).
func (h *Handler) argoClientForCluster(ctx context.Context, clusterID string) argo.WorkflowClient {
	if h.argoFactory == nil {
		return h.wfClient
	}
	if strings.TrimSpace(clusterID) == "" {
		clusterID = clusterDefault
	}
	client, err := h.argoFactory.ForCluster(ctx, clusterID)
	if err != nil || client == nil {
		slog.Warn("workflow handler: argo client resolution failed; using singleton fallback",
			"clusterID", clusterID, "err", err)
		return h.wfClient
	}
	return client
}

// argoClientForRun resolves the Argo client for the run's owning cluster.
func (h *Handler) argoClientForRun(ctx context.Context, run *models.PipelineRun) argo.WorkflowClient {
	if h.argoFactory == nil {
		return h.wfClient
	}
	return h.argoClientForCluster(ctx, h.resolveRunClusterID(ctx, run))
}

// namespaceFromRun derives the Argo namespace for a run: the run's own
// ArgoNamespace, else its execution target namespace (nested object or
// snapshot), else the handler's default namespace.
func (h *Handler) namespaceFromRun(run *models.PipelineRun) string {
	if run != nil {
		if namespace := strings.TrimSpace(run.ArgoNamespace); namespace != "" {
			return namespace
		}
		if run.ExecutionTarget != nil {
			if namespace := strings.TrimSpace(run.ExecutionTarget.Namespace); namespace != "" {
				return namespace
			}
		}
		if namespace := strings.TrimSpace(targetString(run.TargetSnapshot, "namespace")); namespace != "" {
			return namespace
		}
	}
	return h.namespace
}

// namespaceForRequest honors an explicit namespace override (context or query)
// then falls back to the run-derived namespace. Same precedence as the previous
// namespaceForWorkflow, but takes an already-loaded run to avoid a second lookup.
func (h *Handler) namespaceForRequest(c *gin.Context, run *models.PipelineRun) string {
	if namespace := strings.TrimSpace(c.GetString("namespace")); namespace != "" {
		return namespace
	}
	if namespace := strings.TrimSpace(c.Query("namespace")); namespace != "" {
		return namespace
	}
	return h.namespaceFromRun(run)
}

// resolveWorkflowRouting loads the run for workflowName once and returns both the
// per-cluster Argo client and the namespace for a /workflows/:name request. This
// is the single entry point handlers use so a request does exactly one run
// lookup for routing.
func (h *Handler) resolveWorkflowRouting(ctx context.Context, c *gin.Context, workflowName string) (argo.WorkflowClient, string) {
	run, _ := h.findPipelineRunByWorkflow(ctx, workflowName)
	return h.argoClientForRun(ctx, run), h.namespaceForRequest(c, run)
}

// podClientForCluster resolves the K8s Pod diagnostics client for clusterID via
// the per-cluster factory, falling back to the env-based singleton podClient
// (which may be nil — the caller handles that). CYB-3486.
func (h *Handler) podClientForCluster(ctx context.Context, clusterID string) (k8s.PodClient, error) {
	if h.k8sFactory == nil {
		return h.podClient, nil
	}
	if strings.TrimSpace(clusterID) == "" {
		clusterID = clusterDefault
	}
	clientset, err := h.k8sFactory.ForCluster(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return k8s.NewPodClientWithClientset(clientset, clusterID), nil
}
