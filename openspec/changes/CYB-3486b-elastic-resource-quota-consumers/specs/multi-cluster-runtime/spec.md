# Spec Delta — quota endpoints route via ClientFactory

## MODIFIED

### `GET /api/v1/elastic-quotas`

- **Adds** optional query param `clusterId` (default `cluster-default`).
- **Adds** cluster-scoped listing: when the K8s ClientFactory is wired, the
  handler pulls a dynamic client from `factory.DynamicForCluster(clusterID)`
  and lists ElasticQuota CRDs on that cluster.
- Behavior when `clusterId` is omitted matches the pre-4b response (default
  cluster, whose factory config is env-derived).

### `GET /api/v1/resource-quotas`

- **Adds** optional query param `clusterId` (default `cluster-default`).
- **Adds** cluster-scoped listing via `factory.ForCluster(clusterID)`.
- Same default-cluster behavior when `clusterId` is omitted.

## Compat guarantees

- Legacy callers (no `?clusterId=`) hit `cluster-default`; both endpoints
  return the same rows they did before 4b landed.
- Handlers built without a factory (unit tests, no-PG boot) fall back to
  the pre-3486 env singleton — no regression path.
