# Tasks — CYB-3486 PR 4d: argo client K8s CRD mode + pool 概念收口

Original scope was 4d.1-4d.5 (Argo CRD client series). The user then bundled
adjacent items (auth abstraction, pool.1/pool.2 columns + UI) into the same
branch — those are recorded below in the same file for a single source of
truth.

## PR 4d.1 · CRD skeleton + create/get/delete/list/status(~4h) — landed in PR #423

- [x] `backend/internal/argo/crd_client.go`
  - [x] type `crdWorkflowClient` with fields: dyn (dynamic.Interface), pods (kubernetes.Interface), defaultNS
  - [x] constructor `newCRDWorkflowClient(dyn, pods, defaultNS)`
  - [x] compile-time `var _ WorkflowClient = (*crdWorkflowClient)(nil)`
- [x] 5 read/create/delete methods
  - [x] `CreateWorkflow` via `dyn.Resource(workflowGVR).Namespace(ns).Create(unstructured, CreateOptions)` — TypeMeta normalized
  - [x] `GetWorkflow` via dynamic Get + unstructured → wfv1.Workflow via `runtime.DefaultUnstructuredConverter`
  - [x] `GetWorkflowStatus` via GetWorkflow + extract phase; NotFound → ErrNotFound + WorkflowUnknown
  - [x] `DeleteWorkflow` via dynamic Delete
  - [x] `ListWorkflows` via dynamic List with LabelSelector
- [x] `translateK8sErr`: IsNotFound → ErrNotFound, IsAlreadyExists → ErrAlreadyExists
- [x] `backend/internal/argo/factory.go` mode selection
  - [x] Local `K8sFactory` interface (avoid `argo → k8s → pipeline → argo` import cycle)
  - [x] `dbClientFactory.k8sFactory K8sFactory` (nullable)
  - [x] `WithK8sFactory(K8sFactory)` option
  - [x] `buildClient` branches on resolved `cfg.ServerURL != ""` → HTTP; else CRD (needs k8sFactory)
  - [x] Nil k8sFactory + empty URL → `ErrClusterMisconfigured`
  - [x] `argo_namespace` from cluster row; blank → `defaultArgoNamespace = "argo"`
- [x] `backend/cmd/server/infra.go` wire `argo.WithK8sFactory(k8sFactory)`
- [x] Tests: `crd_client_test.go` (12 subtests) + `factory_test.go` extended (5 mode-selection tests)
- [x] `go test ./...` clean

## PR 4d.2 · lifecycle ops(~4h) — landed on `feat/cyb-3486d-remaining`

- [x] `StopWorkflow` — merge-patch `spec.shutdown = "Stop"`
- [x] `TerminateWorkflow` — merge-patch `spec.shutdown = "Terminate"`
- [x] `SuspendWorkflow` — merge-patch `spec.suspend = true`
- [x] `ResumeWorkflow` — merge-patch `spec.suspend = false`
- [x] Extracted `patchSpec` helper (JSON merge patch)
- [x] Tests: 8 subtests (4 op patch content + merge preservation + NotFound translation)

## PR 4d.3 · Retry / Resubmit(~4h) — landed

- [x] `RetryWorkflow` — hand-rolled (workflow/util drags in HDFS/Kerberos/OTel/cron;
  D3 explicitly allowed a simplified subset)
  - [x] guard: must be in Failed / Error / Succeeded
  - [x] reset workflow status.phase → Running, clear finishedAt / message
  - [x] reset failed / errored node phase → Pending, collect their pods
  - [x] delete stale pods (NotFound tolerated)
  - [x] Update workflow via dynamic Update
- [x] `ResubmitWorkflow` / `ResubmitWorkflowWithResult`
  - [x] deep-copy spec, clear metadata (uid, resourceVersion, generation, ownerRefs)
  - [x] fresh generateName derived from source (strip argo hash suffix)
  - [x] strip argo-managed labels/annotations (`workflows.argoproj.io/*`)
  - [x] Create new workflow via dynamic Create
- [x] Compile-time `var _ WorkflowResubmitResultClient = (*crdWorkflowClient)(nil)`
- [x] Tests: 6 subtests (retry pods deletion, retry rejects Running, retry NotFound,
      resubmit clones + strips argo labels, resubmit NotFound, generateName fallback 3 shapes)

## PR 4d.4 · Logs(~4h) — landed

- [x] `GetWorkflowLogs`
  - [x] enumerate `workflow.status.nodes` for Pod-type nodes when podName is empty
  - [x] sort by `node.StartedAt` (insertion sort — no `sort` import, workflows <100 pods)
  - [x] `typedPods.CoreV1().Pods(ns).GetLogs(pod, PodLogOptions).DoRaw` per pod
  - [x] concat in run order; NotFound pods (GC'd) skipped, not error
  - [x] `LimitBytes` truncates post-fetch and sets Truncated=true
- [x] `GetWorkflowLogStream` — single pod only (requires explicit podName)
- [x] `toPodLogOptions` maps WorkflowLogOptions → corev1.PodLogOptions; container defaults to `"main"`
- [x] Tests: 8 subtests (single pod, LimitBytes truncation, empty-pod enumeration,
      no-nodes empty result, stream requires podName, sortPodLogEntries orders correctly,
      toPodLogOptions defaults main, toPodLogOptions passes fields through)

## PR 4d.5 · run watcher routes by cluster — landed (correctness half)

- [x] `resolveArgoClientForRun(ctx, run)` — reads `run.ExecutionTarget.ClusterID`,
      falls back to `cluster-default` for legacy rows; factory-first with singleton fallback
- [x] `runClusterID` helper (used both by the resolver and by 4d.5.b grouping)
- [x] Refactored call sites (each was `uc.wfClient.<X>`):
  - [x] `refreshRunStatus` (watcher core)
  - [x] `backfillRunStatus` (7-day historical ledger sweep)
  - [x] `reconcileMisclassifiedRunFromArgo` (list-view drift correction)
  - [x] `RefreshRunFromWorkflowByName` (Argo run-status webhook)
  - [x] `retryRuntimeRun` / `stopRuntimeRun` / `suspendRuntimeRun` /
        `resumeRuntimeRun` / `terminateRuntimeRun` (UI lifecycle actions)
  - [x] `DeleteRun` argo workflow cleanup
- [x] Tests: 4 resolver tests + 5-case `runClusterID` table

## PR 4d.5.b · goroutine-per-cluster parallelism — landed

- [x] `SyncActiveRunEvents` picks the cursor window sequentially (fairness preserved),
      then groups by cluster and fans out one goroutine per cluster
- [x] Same-cluster runs stay sequential (avoid stampeding one API's client-go rate limiter)
- [x] `groupRunIndicesByCluster` helper extracted for unit testing
- [x] Concurrency safety notes on the commit; watcherActiveCursor written only after wg.Wait
- [x] Tests: 2 subtests (partitions by cluster, respects passed window)

## PR auth.1 · cluster auth abstraction — landed

- [x] Migration `20260716120000_add_cluster_auth_fields.sql`
  - [x] `clusters.auth_type text NOT NULL DEFAULT 'gke_wif'`
  - [x] `clusters.auth_secret_ref text NOT NULL DEFAULT ''`
- [x] `models.Cluster.AuthType` + `AuthSecretRef`
- [x] `postgres.ClusterRepo` selects / inserts / updates the columns (AuthType defaults to `gke_wif` when blank)
- [x] `k8s.buildConfigFromCluster` dispatches on AuthType
  - [x] `""` / `"gke_wif"` → extracted `buildConfigGKEWIF` (byte-identical to prior code)
  - [x] anything else → `ErrClusterMisconfigured`
- [x] Tests: 2 subtests (explicit gke_wif matches empty behavior, unsupported auth_type returns ErrClusterMisconfigured)
- [ ] `bearer` / `ack_wif` / `eks_wif` real implementations — deferred until first non-GCP customer

## PR pool.1 · ExecutionTarget.elastic_quota_name — landed

- [x] Migration `20260716120500_add_execution_target_elastic_quota.sql`
- [x] `models.ExecutionTarget.ElasticQuotaName`
- [x] `postgres.ExecutionTargetRepo` selects / inserts / updates
- [x] usecase/pipeline injects `quota.scheduling.koordinator.sh/name=<value>` into `wfOpts.PodLabels`
      via `applyElasticQuotaPodLabel` helper (empty name = no-op, preserves existing labels)
- [x] Tests: 6 subtests on `applyElasticQuotaPodLabel` including label-key pin

## PR pool.2 · ExecutionTarget.priority_class_name — landed

- [x] Migration `20260716121000_add_execution_target_priority_class.sql`
- [x] `models.ExecutionTarget.PriorityClassName`
- [x] `postgres.ExecutionTargetRepo` selects / inserts / updates
- [x] `transpiler.Options.PodPriorityClassName`; `Transpile` sets `wf.Spec.PodPriorityClassName` when non-empty
- [x] usecase/pipeline copies `target.PriorityClassName` into wfOpts
- [x] Tests: 2 transpiler subtests (propagates when set, stays empty when unset)

## PR ux.1 · pool-first UI polish — landed

- [x] PoolManager 「集群」 column title + cell rendered `type=secondary` at font-size 11
      (matches the 默认 tag hierarchy — info stays but recedes)
- [x] ElasticQuotaPanel: "名称" + "命名空间" merged into a single "池" column
      (EQ name as primary line, namespace as mono-font secondary label)
- [x] Quotas sorted by namespace then name so pools sharing a namespace visually cluster

## PR ux.2 · Target modal exposes EQ / PriorityClass — landed

- [x] `ExecutionTarget` TS type gains `elasticQuotaName` + `priorityClassName` (matches BE tags)
- [x] `FormValues` mirrors them
- [x] `openEdit` hydrates fields from the target row
- [x] `handleSave` includes trimmed values in the payload (blank preserves pre-pool.1/2 defaults)
- [x] Two new `Form.Item`'s in the modal with plain Input controls + Chinese help text
- [x] No dropdown: value must exist in the target cluster (EQ CRD name / PriorityClass name);
      free-text + placeholders is enough for admins who know their cluster

## Dev smoke — pending until PR review + merge

- [ ] Regression: cluster-default (cyber-clust) workflow via argo-server HTTP path unchanged
- [ ] `UPDATE clusters SET argo_server_url = '' WHERE name = 'delivery-clust'`
- [ ] SDK submit workflow to delivery-clust-dev → visible via `kubectl get workflow -n cyber-delivery-dev`
- [ ] Watcher polls delivery-clust workflow status back to run row
- [ ] UI Stop / Retry / Suspend / Resume / Terminate against delivery-clust workflow
- [ ] Workflow log tab renders delivery-clust pod logs
- [ ] Set target.elastic_quota_name → pod carries the koord label
- [ ] Set target.priority_class_name → `wf.Spec.PodPriorityClassName` populated

## Definition of Done

- [x] OpenSpec approved
- [x] 4d.1 merged (PR #423)
- [ ] 4d.2 / 4d.3 / 4d.4 / 4d.5 / 4d.5.b + auth.1 + pool.1 + pool.2 + ux.1 + ux.2 merged
      (all sit on `feat/cyb-3486d-remaining`, PR #426 draft awaiting user review)
- [ ] Dev smoke green
- [x] Backend `go test ./...` all green through the series

## PR 4d.6 · fix: CRD client pod naming (POD_NAMES=v2) — code-review follow-up

Bug found in code review of the merged 4d series. See `decisions.md` D8.

- [x] Extract v2 pod-naming into `argo.PodNameForNode` (single source of truth)
- [x] `RetryWorkflow` deletes stale pods by resolved pod name, not `node.ID`
- [x] `podsForLogs` (aggregate log fetch) enumerates by resolved pod name
- [x] `handlers/workflow.resolveWorkflowPodName` delegates to `argo.PodNameForNode`
      (removes the duplicated implementation; TTL-cache wrapper unchanged)
- [x] Unit tests: `argo.PodNameForNode` (v2 vs node.ID, sanitize, displayName
      fallback, root-node fallback, non-pod, missing inputs)
- [x] Regression: CRD retry test uses realistic node IDs + a node-ID-named
      decoy pod that must survive; verified tests fail against the old logic
- [x] `go build ./...` + `go test ./...` green
