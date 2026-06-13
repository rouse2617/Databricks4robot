# Preview deployments

## Fast preview (GKE/Tekton build + GKE Pod deploy)

Fast preview avoids Cloud Build queue/worker startup. It runs a single Tekton
PipelineRun in GKE, downloads the pushed commit archive from GitHub, builds or
reuses backend and frontend images through the warm `preview-buildkitd`, then
deploys preview backend/frontend Pods into the dev namespace.

```bash
bash deploy/preview/fast-preview.sh HEAD --frontend remote
bash deploy/preview/fast-preview.sh HEAD --frontend local
```

`remote` deploys backend and frontend preview Pods. `local` deploys only the
backend and prints the local Vite command.

## Deployment modes

| Mode | Backend | Frontend | Typical use |
|------|---------|----------|-------------|
| **Local Vite + preview backend** (`--frontend local`) | GKE preview Pod (~1–2 min) | Local Vite with HMR | Daily dev: hot-reload UI against an isolated backend at a specific commit |
| **Full preview** (`--frontend remote`) | GKE preview Pod | GKE preview Pod | Share a hosted preview URL or test the production frontend build |
| **Cloud Run dev** (`npm run dev:remote`) | Shared Cloud Run dev | Local Vite | Legacy path; backend updates are slower (~4–5 min via Cloud Build) |

### Local frontend + preview backend (recommended daily workflow)

Terminal 1 — deploy backend preview:

```bash
bash deploy/preview/fast-preview.sh HEAD --frontend local --wait
```

Terminal 2 — local frontend with hot reload:

```bash
cd Frontend
VITE_PREVIEW_ID=<preview-id> npm run dev
```

Open `http://127.0.0.1:5176/`. Vite proxies `/api/*` to the preview backend through
the dev gateway. Use **`VITE_PREVIEW_ID`**, not `VITE_API_BASE_URL=.../api/v1`:
the preview router strips `/preview/<id>/api` before forwarding to the backend,
so the proxy must rewrite paths correctly (handled automatically when
`VITE_PREVIEW_ID` is set).

**Verify the pairing:** Dashboard → 概览 → bottom-left **前端版本** should read
`v<package.json-version> (preview/<preview-id>)`, e.g. `v0.1.1 (preview/b723ff083383)`.
If it still shows `(local)`, restart Vite with `VITE_PREVIEW_ID` set.

**Verify pipeline deploy works:** preview Pods override Argo env to talk to the
in-cluster API (`argo-server.cyber-databrew-dev.svc.cluster.local:2746`), not the
Cloud Run `pipeline-ui` proxy from `cyber-databrew-config`. Without this override,
deploy returns `argo resource not found: 404 page not found`.

### Monitor build/deploy progress

```bash
# list recent runs
bash deploy/preview/preview-status.sh

# watch a specific PipelineRun
bash deploy/preview/preview-status.sh --watch preview-fast-<preview-id>-<HHMMSS>
```

`fast-preview.sh --wait` also prints step progress inline. Use `--no-wait` plus
`preview-status.sh --watch` in another terminal if you prefer.

Before the deploy finishes, the script prints deterministic preview URLs:

```text
https://cyber-databrew-dev.cyberorigin.ai/preview/<preview-id>/
https://cyber-databrew-dev.cyberorigin.ai/preview/<preview-id>/api
```

### Fast preview resources

The build and runtime resources are intentionally colocated in one namespace:

```text
namespace: cyber-databrew-dev
```

Resource split:

```text
PipelineRun / TaskRun Pods:
  cyber-databrew-dev

BuildKit daemon:
  deployment/preview-buildkitd
  service/preview-buildkitd
  pvc/preview-buildkitd-cache
  endpoint tcp://preview-buildkitd.cyber-databrew-dev.svc.cluster.local:1234

Preview runtime:
  deployment cdb-pvc-b-<backend-content-hash>-api
  deployment cdb-pvc-f-<frontend-content-hash>-web
  service cdb-pv-<preview-id>-api
  service cdb-pv-<preview-id>-web
```

Apply or refresh it with:

```bash
kubectl apply -f deploy/preview/k8s/preview-buildkitd.yaml
kubectl rollout status deploy/preview-buildkitd -n cyber-databrew-dev
kubectl apply -f deploy/preview/k8s/preview-env-reader.yaml
kubectl apply -f deploy/preview/k8s/preview-buildkit-prewarm.yaml
```

`preview-env-reader.yaml` creates `serviceaccount/tekton-builder` in
`cyber-databrew-dev` and gives it the minimum Kubernetes permissions needed to:

```text
get dev config/secrets
create/update preview Deployments
create/update preview Services
watch rollout status
```

The BuildKit daemon stores local cache on `pvc/preview-buildkitd-cache`
(`standard-rwo`, 30Gi) and runs BuildKit GC with about 24GB keep-storage. The
pipeline also keeps registry cache as a fallback:

```text
cyber-databrew-backend:preview-buildcache
cyber-databrew-frontend:preview-buildcache
```

### BuildKit cache prewarm

`deploy/preview/k8s/preview-buildkit-prewarm.yaml` installs
`cronjob/preview-buildkit-prewarm` in `cyber-databrew-dev`. It runs every
6 hours with `concurrencyPolicy: Forbid`, fetches the configured Git ref
(`dev` by default), and builds the backend through the same warm
`preview-buildkitd` with registry cache export enabled. It does not push a
preview image and does not create preview runtime Deployments or Services; the
goal is to keep base image layers, Go module cache, and Go build cache warm
after BuildKit restarts or cache eviction.

Kubernetes does not shrink existing PVCs in place. If an existing cluster still
has a larger `preview-buildkitd-cache` PVC, applying the manifest alone will not
reduce the disk. Recreate only the BuildKit Deployment/PVC when you explicitly
want to trade cache history for lower disk cost; dev runtime Deployments are not
affected.

Run one prewarm manually:

```bash
bash deploy/preview/prewarm-buildkit.sh
```

Prewarm a feature branch while validating preview changes:

```bash
bash deploy/preview/prewarm-buildkit.sh --ref infra/preview-cloudrun
```

Frontend cache prewarm is opt-in because the backend is the dominant frequent
edit path:

```bash
bash deploy/preview/prewarm-buildkit.sh --frontend true
```

Preview image tags are based on content hash, not commit id:

```text
cyber-databrew-backend:preview-b-<backend-content-hash>
cyber-databrew-frontend:preview-f-<frontend-content-hash>
```

The commit id still owns the public URL and the alias Service names. If a new
commit does not change backend or frontend runtime content, the PipelineRun
reuses the existing image tag and content Deployment, then only creates or
updates the lightweight `cdb-pv-<preview-id>-*` Services.

Frontend preview builds also mount persistent BuildKit caches for npm package
downloads, TypeScript build info, and Vite cache data. This helps repeated
small commits on the same branch, while still producing a complete static
bundle for each changed frontend content hash.

### GKE path preview router

For pod-based previews, keep a single public entry point and route by commit id:

```text
https://cyber-databrew-dev.cyberorigin.ai/preview/<sha12>/
```

Cloudflare Worker `cyber-databrew-dev` handles that user-facing URL and proxies
it to the GKE Gateway origin:

```text
https://api-cyber-databrew-dev.cyberorigin.ai/preview/<sha12>/
```

The Worker uses Cloudflare `resolveOverride` to resolve that API host through
the DNS-only internal origin
`preview-origin-cyber-databrew-dev.cyberorigin.ai` (`A 34.111.117.201`).
This keeps the existing `api-cyber-databrew-dev.cyberorigin.ai` Cloud Run
mapping untouched for normal dev API traffic.

The GKE `preview-router` nginx then dispatches by service name in namespace
`cyber-databrew-dev`:

```text
/preview/<sha12>/api/* -> service/cdb-pv-<sha12>-api
/preview/<sha12>/*     -> service/cdb-pv-<sha12>-web
```

Apply the fixed router once:

```bash
kubectl apply -f deploy/preview/k8s/preview-router.yaml
kubectl rollout status deploy/preview-router -n cyber-databrew-dev
```

Frontend pod preview images use relative Vite assets and runtime preview path
detection. React Router and API calls derive `/preview/<sha12>` from
`window.location.pathname`, so a new backend/infra-only commit can reuse the
frontend build layer:

```text
VITE_BASE_PATH=./
VITE_API_BASE_URL=
```

`deploy/preview/frontend-runtime.Dockerfile` exposes these as build args. The
fast preview PipelineRun sets them automatically.

### GitHub token

The PipelineRun expects this Kubernetes Secret in `cyber-databrew-dev`:

```text
cyber-databrew-preview-github-token
```

with key:

```text
token
```

If the Secret is missing, `fast-preview.sh` creates it from `GITHUB_TOKEN` or
`gh auth token`. Use a fine-grained token restricted to:

```text
Repository: CyberOrigin2077/cyber-databrew
Permission: Contents read-only
```

Refresh an existing Secret explicitly:

```bash
REFRESH_GITHUB_TOKEN_SECRET=true bash deploy/preview/fast-preview.sh HEAD
```

### Timing

Fast preview logs explicit timing lines:

```text
TIMING source_archive_seconds=...
TIMING backend_image_reused=...
TIMING backend_buildctl_seconds=...
TIMING frontend_image_reused=...
TIMING frontend_buildctl_seconds=...
TIMING backend_k8s_deploy_seconds=...
TIMING frontend_k8s_deploy_seconds=...
```

In warm-cache tests, GitHub archive fetch was about 1 second and BuildKit
image build/push was about 2-3 seconds per image. Backend and frontend
Kubernetes deploys run in parallel. When the content-hash image already exists,
the corresponding build step is skipped.

```text
TIMING source_archive_seconds=1
TIMING backend_image_reused=1
TIMING backend_buildctl_seconds=0
TIMING frontend_image_reused=1
TIMING frontend_buildctl_seconds=0
TIMING backend_k8s_deploy_seconds=...
TIMING frontend_k8s_deploy_seconds=...
```

This preview path builds from a pushed Git commit. Local machines do not upload
source to GCP and do not need `gcloud` credentials for the normal flow.

## Trigger

Add one of these flags to the head commit message before pushing:

```text
[preview]
[preview:remote]
[preview:local]
```

`remote` deploys both backend and frontend preview Pods. `local` deploys
only the backend preview and prints the local Vite command.

The GitHub workflow authenticates to GCP, gets GKE credentials, and starts the
same `fast-preview.sh` path. The Tekton PipelineRun itself then runs inside
GKE, uses its Kubernetes ServiceAccount directly for preview Deployment/Service
updates, and downloads the target commit archive using the Kubernetes Secret:

```text
cyber-databrew-preview-github-token
```

This avoids uploading source from local machines or GitHub runners to GCP, and
it avoids Cloud Build queue/worker startup for the preview build.

Running `fast-preview.sh` locally is the lowest-latency control path because it
talks to Tekton directly. GitHub Actions uses the same Tekton pipeline after
push, but adds runner startup, OIDC authentication, and GKE credential setup
before the PipelineRun starts.

Legacy manual test from a checked-out repo through the older Cloud Build path:

```bash
sha="$(git rev-parse --short=12 HEAD)"
gcloud builds submit --no-source \
  --project green-valley-442103 \
  --region us-central1 \
  --config deploy/preview/cloudbuild.yaml \
  --substitutions "_PREVIEW_ID=${sha},_FRONTEND_MODE=remote,_COMMIT_SHA=$(git rev-parse HEAD)"
```

## URL convention

Preview URLs are derived from the first 12 characters of the commit SHA:

```text
https://cyber-databrew-dev.cyberorigin.ai/preview/<sha12>/
https://cyber-databrew-dev.cyberorigin.ai/preview/<sha12>/api
```

## Legacy Cloud Build timing

The older Cloud Build preview path prints explicit timing lines:

```text
TIMING clone_source_seconds=...
TIMING backend_buildx_push_seconds=...
TIMING backend_cloud_run_deploy_seconds=...
TIMING frontend_npm_ci_seconds=...
TIMING frontend_vite_build_seconds=...
TIMING frontend_buildx_push_seconds=...
TIMING frontend_cloud_run_deploy_seconds=...
```

`frontend=local` skips the frontend timings because it only deploys the backend.
The build uses `docker buildx --push` with registry-backed layer caches:

```text
cyber-databrew-backend:preview-buildcache
cyber-databrew-frontend:preview-buildcache
```

Frontend preview builds use stable Vite build metadata so unchanged frontend
source can reuse the full build layer.

## Expiration

Commit alias Services add labels:

```text
preview=true
preview-id=<sha12>
expires-epoch=<unix timestamp>
ttl-hours=<hours>
preview-content-hash=<hash>
```

Default TTL is 24 hours. Override it with `--ttl-hours ...` for fast preview or
`_TTL_HOURS=...` in the legacy Cloud Build substitutions.

Content Deployments are shared across preview ids and use:

```text
preview-content=true
preview-content-hash=<hash>
```

They do not carry `preview=true`. Cleanup first deletes expired alias Services,
then deletes content Deployments only when no remaining alias Service points at
the same content hash.

Delete expired Kubernetes preview resources with:

```bash
DRY_RUN=true bash deploy/preview/cleanup-expired.sh
bash deploy/preview/cleanup-expired.sh
```

Delete old Cloud Run preview services with the legacy target:

```bash
PREVIEW_CLEANUP_TARGET=cloudrun DRY_RUN=true bash deploy/preview/cleanup-expired.sh
PREVIEW_CLEANUP_TARGET=cloudrun bash deploy/preview/cleanup-expired.sh
```

## Required one-time setup

- Store a GitHub token that can read `CyberOrigin2077/cyber-databrew`:
  `cyber-databrew-preview-github-token`.
  A classic PAT can be used temporarily; replace it with a fine-grained
  read-only token when organization approval is available.
- Configure GitHub secrets:
  - `GCP_WORKLOAD_IDENTITY_PROVIDER`
  - `GCP_PREVIEW_SERVICE_ACCOUNT`
- Grant the GitHub Actions service account permission to get GKE credentials.
  The current expected GSA is:

  ```text
  cyber-databrew-gha@green-valley-442103.iam.gserviceaccount.com
  ```

- Apply the one-time K8s bootstrap manifests:

  ```bash
  kubectl apply -f deploy/preview/k8s/preview-buildkitd.yaml
  kubectl apply -f deploy/preview/k8s/preview-buildkit-prewarm.yaml
  kubectl apply -f deploy/preview/k8s/preview-env-reader.yaml
  kubectl apply -f deploy/preview/k8s/preview-trigger-rbac.yaml
  ```

- Allow the `cyber-databrew-dev/tekton-builder` Kubernetes ServiceAccount to
  use the GCP service account:

  ```bash
  gcloud iam service-accounts add-iam-policy-binding \
    tekton-builder@green-valley-442103.iam.gserviceaccount.com \
    --project green-valley-442103 \
    --role roles/iam.workloadIdentityUser \
    --member "serviceAccount:green-valley-442103.svc.id.goog[cyber-databrew-dev/tekton-builder]"
  ```

  GitHub Actions runs `fast-preview.sh` with `PREVIEW_BOOTSTRAP=false`; it only
  creates the Tekton PipelineRun and reads logs/results. The PipelineRun itself
  runs as `tekton-builder` in `cyber-databrew-dev`, which needs Artifact
  Registry writer, Workload Identity for the GSA, and the dev Kubernetes RBAC
  above.
