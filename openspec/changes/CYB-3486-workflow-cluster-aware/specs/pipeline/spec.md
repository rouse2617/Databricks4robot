# Pipeline Spec Delta

## Modified Requirements

### Requirement: Workflow monitoring endpoints resolve the workflow's cluster

The `/api/v1/workflows/*` endpoints SHALL operate against the Argo control plane
of the cluster that actually owns the workflow, rather than always the default
cluster. The owning cluster is resolved from the workflow name via its
`pipeline_run` and that run's execution target `cluster_id`.

#### Scenario: Read a workflow that lives on a non-default cluster

- **Given** a pipeline run whose execution target resolves to a non-default
  cluster (e.g. delivery-clust)
- **When** a client calls `GET /api/v1/workflows/:name` or
  `GET /api/v1/workflows/:name/logs` or the log stream for that workflow
- **Then** the handler resolves the run's `cluster_id` and uses that cluster's
  Argo client (argo-server HTTP or in-cluster CRD)
- **And** the workflow detail / node logs are returned from the owning cluster
- **And** the request does not fail with "workflows.argoproj.io not found" from
  the default cluster's argo-server

#### Scenario: Lifecycle operations target the owning cluster

- **Given** a workflow on a non-default cluster
- **When** a client calls retry / resubmit / suspend / stop / resume /
  terminate / delete for that workflow name
- **Then** the operation is issued against the owning cluster's Argo client

#### Scenario: Node pod diagnostics read the owning cluster

- **Given** a workflow node on a non-default cluster
- **When** a client calls `GET /api/v1/workflows/:name/nodes/:nodeId/pod`
- **Then** the pod name is resolved from the owning cluster's workflow object
- **And** pod diagnostics are read from the owning cluster's Kubernetes API

#### Scenario: Default cluster and no-factory paths are unchanged

- **Given** a workflow on the default cluster, or a deployment with no
  per-cluster factory wired (e.g. no Postgres)
- **When** any `/workflows/*` endpoint is called
- **Then** the handler uses the startup-injected singleton Argo client and
  default namespace exactly as before this change

#### Scenario: Cluster resolution failure falls back to the singleton

- **Given** the workflow's run or cluster cannot be resolved (unknown run,
  missing target, or the factory errors for the resolved cluster)
- **When** a `/workflows/*` endpoint is called
- **Then** the handler falls back to the injected singleton Argo client so the
  request degrades to prior behavior instead of erroring on resolution
