# Cloud Run Migration Scripts

This folder contains incremental migration scripts for moving selected services
from GKE to Cloud Run.

## Current scope

- `mcap-preview-dev.sh`: deploys `mcap-preview` to Cloud Run in `us-central1`.
- `frontend-dev.sh`: builds a Cloud Run-specific frontend image and deploys
  `cyber-databrew-frontend-dev`.
- `backend-dev.sh`: deploys `cyber-databrew-backend-dev` and can source env
  values from `cyber-databrew-dev` K8s ConfigMap/Secret.

## Notes

- This is a non-breaking migration path: deploy Cloud Run first, verify with
  the generated `run.app` URL, then switch ingress traffic later.
- `mcap-preview` on Cloud Run points upstream to
  `https://api-cyber-databrew-dev.cyberorigin.ai`.
- Frontend Cloud Run image (`frontend-cloudrun.Dockerfile`) reuses the registry
  `cyber-databrew-frontend` base image and replaces nginx with
  `deploy/cloudrun/frontend-nginx.conf`: **`/api/v1/preview/` → mcap-preview-dev**,
  **`/api/` → cyber-databrew-backend-dev** (same layout as `Frontend/nginx.conf`).
- `frontend-dev.sh` defaults to local `docker build && docker push` for speed.
  Set `USE_CLOUD_BUILD=true` to use remote Cloud Build instead.
  Optional: `BASE_IMAGE=<registry/.../cyber-databrew-frontend:tag>` passes
  `--build-arg BASE_IMAGE=...` so the SPA bundle is not silently taken from
  the Dockerfile default (`dev-latest`).
- `backend-dev.sh` defaults to local `docker build && docker push`; set
  `USE_CLOUD_BUILD=true` for remote build, or `USE_EXISTING_IMAGE=true` to
  skip build and deploy the provided image directly.
- `backend-dev.sh` supports `DB_PASSWORD_SECRET=<secret-name>` to load
  `DB_PASSWORD` from GCP Secret Manager at deploy time (recommended over
  plaintext `DB_PASSWORD_OVERRIDE`).
- When sourcing env from Kubernetes, **`LAKEHOUSE_BACKEND` is not overwritten** if
  the ConfigMap/secret merge already defines it; only missing keys get
  `LAKEHOUSE_BACKEND_DEFAULT` (`false`). Use `LAKEHOUSE_BACKEND_OVERRIDE=true` if you
  need to force-enable without changing the merged file.
- If your Iceberg namespace in BigLake is **`poc`** but the cluster ConfigMap still has
  `LAKEHOUSE_BQ_DATASET=robot`, pass **`LAKEHOUSE_BQ_DATASET_OVERRIDE=poc`** on deploy so
  lakehouse SQL resolves (namespace skew only; unrelated to Nessie vs REST).
- Optional: `ELASTICSEARCH_PASSWORD_SECRET` loads `ELASTICSEARCH_PASSWORD` when
  your Elasticsearch cluster uses HTTP Basic auth (not used by the default
  passwordless dev ES manifest).
- `backend-dev.sh` supports `ENV_FILE=/path/to/backend.env` to merge additional
  key/value config before applying Cloud Run safety defaults and explicit
  `*_OVERRIDE` variables.
- For Pub/Sub outbox POC, `backend-dev.sh` supports explicit
  `PUBSUB_PROJECT_OVERRIDE`, `TOPIC_ASSET_EVENTS_OVERRIDE`, and
  `OUTBOX_ES_SUBSCRIPTION_OVERRIDE` so Cloud Run can switch transport without
  editing the source ConfigMap/Secret first.
- Lakehouse APIs on Cloud Run require explicit BigQuery wiring:
  `LAKEHOUSE_BACKEND_OVERRIDE=true`, `LAKEHOUSE_BQ_PROJECT_OVERRIDE=http://<LAKEHOUSE_ILB_IP>:8080`
  (and optionally `LAKEHOUSE_BQ_CATALOG_OVERRIDE`, `LAKEHOUSE_BQ_DATASET_OVERRIDE`).
- When sourcing env from Kubernetes, **`backend-dev.sh` auto-patches** in-cluster
  `LAKEHOUSE_BQ_PROJECT` (for example `http://bigquery:8080`) to a VPC-reachable ILB: it prefers
  `kubectl … bigquery-ilb` in `K8S_NAMESPACE`, then `CLOUDRUN_LAKEHOUSE_BQ_PROJECT` if you set it,
  else a last-resort dev IP from `deploy/cloudrun/backend.env.example`. Override
  with `LAKEHOUSE_BQ_PROJECT_OVERRIDE` / `LAKEHOUSE_BACKEND_OVERRIDE` when your topology differs.
- Backend runtime now supports `BACKEND_ENV_FILE=<path>` and will load that env
  file before reading process env vars. Use `deploy/cloudrun/backend.env.example`
  as the starter template for PG/ES/BigQuery and core runtime flags.
- If K8s secret uses `DB_HOST=postgres`, Cloud Run revision will fail because
  that DNS name is cluster-internal. Set `DB_HOST_OVERRIDE` (and optionally
  `VPC_CONNECTOR`) to a reachable PostgreSQL endpoint before rollout.
- `backend-dev.sh` defaults to the current dev Cloud Run wiring:
  `DB_HOST_OVERRIDE=172.27.160.7`, `DB_NAME_OVERRIDE=cyber_databrew_dev`,
  `DB_PASSWORD_SECRET=cyber-databrew-dev-postgres-password`,
  `VPC_CONNECTOR=cr-central-conn`, `ARGO_SERVER_URL_OVERRIDE=http://10.2.1.211:2746`,
  `K8S_API_ENDPOINT_OVERRIDE=https://34.59.48.233`,
  `K8S_BEARER_TOKEN_SECRET=cyber-databrew-dev-k8s-bearer-token`, and
  `K8S_CA_DATA_SECRET=cyber-databrew-dev-k8s-ca-data`. Override these variables
  only when deploying to a different dev topology.
- `backend-dev.sh` still auto-falls back to the PostgreSQL pod IP if you clear
  `DB_HOST_OVERRIDE` and the merged K8s env says `DB_HOST=postgres`. This is a
  temporary migration convenience, not a stable long-term endpoint.
- Cloud Run target database name is now `cyber_databrew_dev`. Override with
  `DB_NAME` via the script when K8s-sourced env points to a
  different database than the one Cloud Run should target.

## Elasticsearch (GKE `cyber-databrew-dev`) from Cloud Run

The dev Elasticsearch StatefulSet is in GKE. In-cluster callers use
`http://elasticsearch:9200`. Cloud Run cannot resolve that DNS name or reach a
plain `ClusterIP` Service from the VPC, so we expose ES with an **internal**
`LoadBalancer` Service: `deploy/k8s/dev-deps/elasticsearch-internal-lb.yaml`
(`Service/elasticsearch-ilb`).

1. **Read the ILB VIP** (shown under `EXTERNAL-IP`; it is still private to the
   VPC):

   ```bash
   kubectl -n cyber-databrew-dev get svc elasticsearch-ilb \
     -o jsonpath='{.status.loadBalancer.ingress[0].ip}{"\n"}'
   ```

2. **Attach Serverless VPC Access** (project `green-valley-442103` already has
   connector `cr-central-conn` on `cyber-vpc` in `us-central1`).

3. **Deploy the backend** with private egress and an explicit ES URL (use the
   IP from step 1). No Elasticsearch HTTP password is required for this dev
   template (`xpack.security.enabled=false`).

   ```bash
   VPC_CONNECTOR=cr-central-conn \
   VPC_EGRESS=private-ranges-only \
   ELASTICSEARCH_URL_OVERRIDE=http://<ILB_IP>:9200 \
   ./deploy/cloudrun/backend-dev.sh
   ```

4. **Create the `assets` index** once per fresh ES data volume:

   ```bash
   ./deploy/local/elasticsearch/init-index.sh "http://<ILB_OR_LOCALHOST>:9200"
   ```

   If you later enable HTTP Basic on ES, set `ELASTICSEARCH_PASSWORD` (and
   optionally `ELASTICSEARCH_USERNAME`) for `init-index.sh`, and use
   `ELASTICSEARCH_PASSWORD_SECRET` with `backend-dev.sh`.

## BigQuery (GKE `cyber-databrew-dev`) from Cloud Run

Cloud Run cannot reach `http://bigquery:8080` (cluster DNS), so expose BigQuery with
an internal LoadBalancer first:

```bash
kubectl -n cyber-databrew-dev apply -f deploy/k8s/lakehouse-gcs/bigquery-internal-lb.yaml
```

Read the private ILB IP:

```bash
kubectl -n cyber-databrew-dev get svc bigquery-ilb \
  -o jsonpath='{.status.loadBalancer.ingress[0].ip}{"\n"}'
```

Deploy backend with both ES and BigQuery private endpoints:

```bash
VPC_CONNECTOR=cr-central-conn \
VPC_EGRESS=private-ranges-only \
ELASTICSEARCH_URL_OVERRIDE=http://<ES_ILB_IP>:9200 \
LAKEHOUSE_BACKEND_OVERRIDE=true \
LAKEHOUSE_BQ_PROJECT_OVERRIDE=http://data-platform@<LAKEHOUSE_ILB_IP>:8080 \
LAKEHOUSE_BQ_CATALOG_OVERRIDE=iceberg \
LAKEHOUSE_BQ_DATASET_OVERRIDE=<LAKEHOUSE_BQ_DATASET> \
./deploy/cloudrun/backend-dev.sh
```

`LAKEHOUSE_BQ_DATASET_OVERRIDE` must match the **Iceberg namespace** where tables are registered in BigLake (PyIceberg / Bronze job default **`robot`**). Use `poc` only if your BigLake namespace is actually `poc`.

The in-cluster BigQuery catalog for `deploy/k8s/lakehouse-gcs/` is **BigLake Iceberg REST** (same endpoint as Bronze); if you still see `Schema 'robot' does not exist`, the running BigQuery revision is likely on an older Nessie-only catalog — re-apply `kubectl apply -n cyber-databrew-dev -k deploy/k8s/lakehouse-gcs/` and restart BigQuery.

## Outbox Pub/Sub POC (Cloud Run backend)

To test relay/subscriber via Pub/Sub instead of in-process internal transport:

1. Create topic + subscription (once per project):

   ```bash
   gcloud pubsub topics create cyber-databrew-asset-events
   gcloud pubsub subscriptions create cyber-databrew-asset-events-search-sub \
     --topic=cyber-databrew-asset-events
   ```

2. Deploy backend with outbox transport set to Pub/Sub:

   ```bash
   USE_EXISTING_IMAGE=true \
   VPC_CONNECTOR=cr-central-conn \
   VPC_EGRESS=private-ranges-only \
   CLOUDRUN_OUTBOX_TRANSPORT=pubsub \
   PUBSUB_PROJECT_OVERRIDE=green-valley-442103 \
   TOPIC_ASSET_EVENTS_OVERRIDE=cyber-databrew-asset-events \
   OUTBOX_ES_SUBSCRIPTION_OVERRIDE=cyber-databrew-asset-events-search-sub \
   DB_PASSWORD_SECRET=cyber-databrew-dev-postgres-password \
   ./deploy/cloudrun/backend-dev.sh
   ```

3. Roll back to in-process transport quickly:

   ```bash
   USE_EXISTING_IMAGE=true \
   CLOUDRUN_OUTBOX_TRANSPORT=internal \
   ./deploy/cloudrun/backend-dev.sh
   ```
