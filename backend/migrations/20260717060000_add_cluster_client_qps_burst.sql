-- CYB-3486: per-cluster K8s client rate limits, editable online.
--
-- The backend's K8s clients (typed + dynamic, built per cluster by
-- k8s.ClientFactory) never set QPS/Burst on their rest.Config, so they ran at
-- client-go's defaults (QPS=5 / Burst=10). That throttled every K8s call to a
-- cluster's API at ~5 req/s — the real ceiling on batch submission throughput
-- (each submit does create-workflow + get-uid + create-configmap). Raising
-- parallelism / submit workers did nothing because the client throttled first.
--
-- Storing the limits on the cluster row (instead of an env var) makes them
-- editable from the cluster admin UI: the factory already Invalidate()s a
-- cluster's cached clients on update, so a change takes effect within seconds
-- with no redeploy/restart. Per-cluster so a busy delivery cluster can run
-- hotter than the default one.
--
-- Defaults 50/100 are a safe lift over the client-go defaults (well under a
-- typical apiserver's capacity) and apply to every existing row.

ALTER TABLE "clusters"
  ADD COLUMN "client_qps"   real    NOT NULL DEFAULT 50,
  ADD COLUMN "client_burst" integer NOT NULL DEFAULT 100;
