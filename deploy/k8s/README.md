# Kubernetes (cyber-databrew backend)

## Prerequisites

- Namespace: `kubectl apply -f deploy/k8s/namespaces.yaml`
- Secret **`cyber-databrew-secrets`** in the target namespace (DB + `GRACE_TOKEN`). Copy `base/secret.example.yaml` to `base/secret.local.yaml`, replace placeholders, apply. `secret.local.yaml` is gitignored.

## Apply ConfigMap + Deployment + Service

From repo root (checks that the Secret exists unless `SKIP_SECRET_CHECK=1`):

```bash
K8S_NAMESPACE=cyber-databrew-dev ./deploy/k8s/apply-base.sh
```

Equivalent:

```bash
kubectl apply -n cyber-databrew-dev -k deploy/k8s/base/
```

Env injection follows `backend/.env.example` (`cyber-databrew-config` ConfigMap + `cyber-databrew-secrets`).

## BigQuery + Iceberg on GCS (GKE PoC)

In-cluster **BigQuery** with **BigLake Iceberg REST** catalog and **GCS** warehouse (`gs://cyber-databrew-iceberg-warehouse-prod` by default — same as PyIceberg Bronze). BigQuery uses the namespace **`default` ServiceAccount**; annotate it to a dev **GCP SA**, grant **bucket IAM**, then **Workload Identity** (`...svc.id.goog[cyber-databrew-dev/default]`). See **`deploy/k8s/lakehouse-gcs/README.md`**.

```bash
kubectl apply -n cyber-databrew-dev -k deploy/k8s/lakehouse-gcs/
```

Details, IAM commands, and SQL smoke: **`deploy/k8s/lakehouse-gcs/README.md`**.

## Dev stack with in-cluster dependencies (outbox-only)

For test environments where dependencies run in-cluster (`Postgres + Elasticsearch`):

```bash
K8S_NAMESPACE=cyber-databrew-dev ./deploy/k8s/apply-dev-stack.sh
```

This applies:

- `deploy/k8s/dev-deps/`:
  - PostgreSQL (`StatefulSet` + PVC)
  - Elasticsearch (`StatefulSet` + PVC; **no HTTP auth** in this dev template)
  - Internal ILB for VPC callers: `elasticsearch-internal-lb.yaml` → `Service/elasticsearch-ilb`
- `deploy/k8s/overlays/dev/`:
  - backend base manifests plus dev config patch (`ELASTICSEARCH_URL=http://elasticsearch:9200`)

### Secret values for in-cluster dev deps

Use these values in `deploy/k8s/base/secret.local.yaml` for the dev stack:

- `DB_HOST=postgres`
- `DB_PORT=5432`
- `DB_USER=postgres`
- `DB_PASSWORD=postgres`
- `DB_NAME=cyber_databrew_dev`
- `GRACE_TOKEN=<your-dev-token>`

> Note: Postgres schema migrations are not auto-applied by `apply-dev-stack.sh`.
> After Postgres is up, run your migration workflow against `svc/postgres` (for example via `kubectl port-forward` + `backend/scripts/ensure_migrations.sh` / your preferred migration command).

## Frontend deployment (Kubernetes)

### 1) Build and push frontend image

```bash
IMAGE_TAG=us-central1-docker.pkg.dev/green-valley-442103/video-proc-images/cyber-databrew-frontend:dev-latest \
  ./scripts/build-frontend-image.sh

docker push us-central1-docker.pkg.dev/green-valley-442103/video-proc-images/cyber-databrew-frontend:dev-latest
```

### 2) Apply frontend manifests

```bash
kubectl apply -n cyber-databrew-dev -k deploy/k8s/frontend/
```

This creates:

- `deployment/cyber-databrew-frontend`
- `service/cyber-databrew-frontend`
- `ingress/cyber-databrew-frontend`

The frontend pod uses a Kubernetes-specific Nginx config (`deploy/k8s/frontend/nginx-config.yaml`) and proxies `/api/*` to `service/cyber-databrew-backend`.

### 3) Quick verification

If DNS for ingress host is not set yet, verify with port-forward:

```bash
kubectl -n cyber-databrew-dev port-forward svc/cyber-databrew-frontend 5173:80
```

Then open `http://localhost:5173`.

## Expose via developer-gateway (`*.cyberorigin.ai`)

This repository now includes Gateway API manifests under `deploy/k8s/gateway/` for:

- `cyber-databrew-dev.cyberorigin.ai` -> `service/cyber-databrew-frontend`
- `api-cyber-databrew-dev.cyberorigin.ai` -> `service/cyber-databrew-backend`

The manifests assume an existing shared Gateway:

- Gateway name: `developer-gateway`
- Gateway namespace: `developer-gateway`
- App namespace: `cyber-databrew-dev`

### IaC-style bootstrap (recommended)

Run the repository script that follows the same sequence as `cyber-iac`:

1. ensure Certificate Manager DNS authorization/certificate/cert-map entries
2. print Cloudflare records to apply
3. apply HTTPRoute + ReferenceGrant + GCPBackendPolicy (+ frontend IAP secret if enabled)

```bash
./deploy/k8s/gateway/bootstrap-like-iac.sh
```

The script is idempotent. If cert is still `PROVISIONING`, add/verify the Cloudflare records it prints and rerun.

### Manual flow (equivalent)

### 1) (Optional) Create frontend IAP OAuth secret in namespace

Only required when enabling IAP on the frontend route:

```bash
IAP_SECRET="$(gcloud secrets versions access latest --secret iap-oauth-client-secret --project green-valley-442103)"

kubectl -n cyber-databrew-dev create secret generic iap-oauth-cyber-databrew-dev-frontend \
  --from-literal=key="${IAP_SECRET}" \
  --dry-run=client -o yaml | kubectl apply -f -

```

### 2) Fill `clientID` in frontend BackendPolicy manifest (if frontend IAP enabled)

Get OAuth client ID:

```bash
gcloud secrets versions access latest --secret iap-oauth-client-id --project green-valley-442103
```

Replace `REPLACE_WITH_IAP_OAUTH_CLIENT_ID` in:

- `deploy/k8s/gateway/frontend-backend-policy.yaml`

API note: `deploy/k8s/gateway/backend-backend-policy.yaml` now sets `iap.enabled: false` for easier API integration testing.

### 3) Apply Gateway resources

```bash
kubectl apply -k deploy/k8s/gateway/
```

### 4) Validate

```bash
kubectl -n developer-gateway get httproute cyber-databrew-dev-frontend-route cyber-databrew-dev-api-route
kubectl -n cyber-databrew-dev get gcpbackendpolicy
kubectl -n cyber-databrew-dev get referencegrant
```

After DNS/cert propagation, open:

- `https://cyber-databrew-dev.cyberorigin.ai`
- `https://api-cyber-databrew-dev.cyberorigin.ai/healthz`

## Troubleshooting (pods not Ready / CrashLoop)

From repo root:

```bash
K8S_NAMESPACE=cyber-databrew-dev ./deploy/k8s/diagnose.sh
```

Paste the full output when asking for help. Typical patterns:

| Symptom | Likely cause |
|--------|----------------|
| `ImagePullBackOff` | Registry auth, wrong tag, or image only on laptop — configure `docker-credential-gcloud` / Artifact Registry IAM |
| `CrashLoopBackOff`, logs show postgres error | `DB_HOST` / firewall / Cloud SQL authorized networks / Private Service Connect |
| `CreateContainerConfigError` | Secret `cyber-databrew-secrets` missing or wrong name |
