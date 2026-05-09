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

## Dev stack with in-cluster dependencies (Postgres + Elasticsearch + CDC)

For test environments where backend dependencies should run inside the same namespace:

```bash
K8S_NAMESPACE=cyber-databrew-dev ./deploy/k8s/apply-dev-stack.sh
```

`apply-dev-stack.sh` waits for `job/cdc-connect-init` to complete and fails fast by default when connector bootstrap does not finish.
Set `ALLOW_CDC_INIT_FAILURE=1` only when you explicitly want to continue with backend rollout while debugging CDC.

This applies:

- `deploy/k8s/dev-deps/`:
  - PostgreSQL (`StatefulSet` + PVC) with logical WAL enabled
  - Elasticsearch (`StatefulSet` + PVC)
  - Redpanda (`Deployment`)
  - Debezium Connect (`Deployment`)
  - `cdc-connect-init` job (creates `postgres-unified-cdc` connector)
- `deploy/k8s/dev/`:
  - backend base manifests plus dev config patch (`CDC_ENABLED=true`, in-cluster service URLs)

### Secret values for in-cluster dev deps

Use these values in `deploy/k8s/base/secret.local.yaml` for the dev stack:

- `DB_HOST=postgres`
- `DB_PORT=5432`
- `DB_USER=postgres`
- `DB_PASSWORD=postgres`
- `DB_NAME=data4cyber`
- `GRACE_TOKEN=<your-dev-token>`

> Note: Postgres schema migrations are not auto-applied by `apply-dev-stack.sh`.
> After Postgres is up, run your migration workflow against `svc/postgres` (for example via `kubectl port-forward` + `backend/scripts/ensure_migrations.sh` / your preferred migration command).

## Frontend deployment (Kubernetes)

### 1) Build and push frontend image

```bash
IMAGE_TAG=us-central1-docker.pkg.dev/green-valley-442103/rick-cyber-databrew-images/cyber-databrew-frontend:dev-latest \
  ./scripts/build-frontend-image.sh

docker push us-central1-docker.pkg.dev/green-valley-442103/rick-cyber-databrew-images/cyber-databrew-frontend:dev-latest
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
