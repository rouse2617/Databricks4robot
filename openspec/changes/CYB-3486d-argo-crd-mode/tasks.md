# Tasks — CYB-3486 PR 4d: argo client K8s CRD mode

## PR 4d.1 · CRD skeleton + create/get/delete/list/status(~4h)

- [x] `backend/internal/argo/crd_client.go` — new file
  - [x] type `crdWorkflowClient` with fields: dyn (dynamic.Interface), pods (kubernetes.Interface), defaultNS
  - [x] constructor `newCRDWorkflowClient(dyn, pods, defaultNS)`
  - [x] compile-time `var _ WorkflowClient = (*crdWorkflowClient)(nil)`
- [x] Implement 5 read/create/delete methods:
  - [x] `CreateWorkflow` via `dyn.Resource(workflowGVR).Namespace(ns).Create(unstructured, CreateOptions)` — TypeMeta normalized to argoproj.io/v1alpha1 Workflow
  - [x] `GetWorkflow` via `dyn.Resource(...).Namespace(ns).Get(name, GetOptions)` + unstructured → wfv1.Workflow via `runtime.DefaultUnstructuredConverter`
  - [x] `GetWorkflowStatus` via GetWorkflow + extract phase; NotFound → ErrNotFound + WorkflowUnknown
  - [x] `DeleteWorkflow` via `dyn.Resource(...).Namespace(ns).Delete(name, DeleteOptions)`
  - [x] `ListWorkflows` via `dyn.Resource(...).Namespace(ns).List({LabelSelector})`
- [x] Stub 4d.2/4d.3/4d.4 methods with `errCRDMethodNotImplemented` sentinel
- [x] `translateK8sErr` maps IsNotFound → ErrNotFound, IsAlreadyExists → ErrAlreadyExists
- [x] `backend/internal/argo/factory.go` — mode selection
  - [x] Local `K8sFactory` interface to avoid `argo → k8s → pipeline → argo` import cycle
  - [x] `dbClientFactory` gains `k8sFactory K8sFactory` field (nullable)
  - [x] `WithK8sFactory(K8sFactory)` option
  - [x] `buildClient` branches on resolved `cfg.ServerURL != ""` → HTTP; else CRD (requires k8sFactory)
  - [x] Nil k8sFactory + empty resolved URL → `ErrClusterMisconfigured`
  - [x] `argo_namespace` from cluster row; blank → fallback to "argo" (`defaultArgoNamespace`)
- [x] `backend/cmd/server/infra.go` — wire `argo.WithK8sFactory(k8sFactory)`
- [x] Tests:
  - [x] `crd_client_test.go` uses fake dynamic client — 12 subtests: 5 methods happy path + NotFound + AlreadyExists + empty-ns fallback + stub methods return sentinel
  - [x] `factory_test.go` extended: HTTP mode (cluster URL), HTTP via env fallback, CRD mode (empty URL), default-ns fallback, misconfigured guard
- [x] Full `go test ./...` clean
- [ ] Dev smoke:
  - [ ] Update delivery-clust cluster row: `argo_server_url = ''`
  - [ ] Via SDK / API: submit a minimal workflow to delivery-clust-dev target
  - [ ] Verify: workflow appears in delivery-clust `kubectl get workflow -n cyber-delivery-dev`
  - [ ] Verify: run_watcher / status polling reads back Succeeded phase (via GetWorkflow, still cyber-clust singleton for now)

## PR 4d.2 · lifecycle ops(~4h)

- [ ] Extend `crdWorkflowClient`:
  - [ ] `StopWorkflow` — patch `spec.shutdown = "Stop"`
  - [ ] `TerminateWorkflow` — patch `spec.shutdown = "Terminate"`
  - [ ] `SuspendWorkflow` — patch `spec.suspend = true`
  - [ ] `ResumeWorkflow` — patch `spec.suspend = false`
- [ ] Use JSON merge patch (simpler than strategic merge)
- [ ] Tests for each op (fake dynamic client)
- [ ] Dev smoke on delivery-clust:
  - [ ] Submit long-running workflow, hit stop via UI → phase transitions to Terminating → Terminated
  - [ ] Suspend/Resume roundtrip visible in argo CRD spec

## PR 4d.3 · Retry / Resubmit(~4h + hard bits)

- [ ] Study argo v3.5 source: `pkg/apiclient/workflow/workflow-server.go` :: RetryWorkflow / ResubmitWorkflow
- [ ] Extend `crdWorkflowClient`:
  - [ ] `RetryWorkflow` — Get → clear workflow.status.phase + reset failed nodes → Update
  - [ ] `ResubmitWorkflow` — Get → deep-copy spec → clear metadata → Create with new generateName
  - [ ] `ResubmitWorkflowWithResult` — same as above but return the new workflow
- [ ] Simplification: only support "restart all failed" (no partial node selectors); document in code
- [ ] Tests + delivery-clust smoke:
  - [ ] Failed workflow → Retry via UI → new pod spawns → Succeeded
  - [ ] Any workflow → Resubmit → new workflow with new UID appears + runs

## PR 4d.4 · Logs(~4h)

- [ ] Extend `crdWorkflowClient`:
  - [ ] `GetWorkflowLogs` — enumerate workflow.status.nodes for pod nodes, read main container logs, concat by startedAt
  - [ ] `GetWorkflowLogStream` — same but stream io.Reader
- [ ] Use `typedPods.CoreV1().Pods(ns).GetLogs(pod, corev1.PodLogOptions{Container: "main", TailLines: ..., LimitBytes: ...})`
- [ ] Multi-pod: sort by node.StartedAt, interleave sequentially
- [ ] Tests (fake typed client + fake dynamic)
- [ ] Dev smoke:
  - [ ] Submit multi-step workflow on delivery-clust
  - [ ] Frontend workflow log tab renders logs (each step's `main` container output)
  - [ ] Streaming works (SSE endpoint)

## PR 4d.5(optional)· run_watcher per-cluster(~4h)

- [ ] Currently `usecase/pipeline.Usecase.SyncActiveRunEvents` uses `uc.wfClient` singleton
- [ ] Refactor: enumerate distinct clusterIDs across active runs, per-cluster goroutine
- [ ] Reconcile loop: when new cluster appears in DB → spawn goroutine; when idle > N min → drain
- [ ] Tests + dev smoke (kill delivery-clust workflow, verify status persists correctly)

## Cross-PR / post-4d

- [ ] Update delivery-clust cluster row: `UPDATE clusters SET argo_server_url = ''`
- [ ] Verify prod SDK smoke on delivery-clust (customer-owned batch job end-to-end)
- [ ] Memory: record F path is now live; when adding a new cloud in future, follow the pattern

## Definition of Done

- [x] OpenSpec approved by user (this checkpoint)
- [ ] 4d.1 - 4d.4 merged
- [ ] delivery-clust workflow submission via SDK/UI works end-to-end
- [ ] argo UI on cyber-clust still shows every workflow correctly
- [ ] Backend `go test ./...` all green through the series
