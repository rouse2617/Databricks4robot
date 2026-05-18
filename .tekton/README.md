# Tekton Pipelines-as-Code

Mirrors the pattern from
[CyberOrigin2077/tekton-playground](https://github.com/CyberOrigin2077/tekton-playground).
PAC (`pipelines-as-code` namespace) reads this directory and fires a
PipelineRun on matching git events.

## Pipeline matrix (target end state)

| File | Trigger | Purpose |
|---|---|---|
| `plan-gcp-data.yaml` ✅ | PR `deploy/iac/terraform/gcp-data/**` | `terraform plan` → PR comment |
| `plan-gateway-domain.yaml` (TODO) | PR `deploy/iac/terraform/gateway-domain/**` | same |
| `plan-service-identity.yaml` (TODO) | PR `deploy/iac/terraform/service-identity/**` | same |
| `apply-*-iac.yaml` (TODO) | merge → main | `terraform apply` |
| `build-backend.yaml` (TODO) | PR `backend/**` | BuildKit → `:<sha>` + `:pr-N` |
| `deploy-cloudrun-dev.yaml` ✅ | PR comment `/deploy-cloudrun-dev` (PR → `main`) | BuildKit → deploy `cyber-databrew-backend-dev` on Cloud Run |
| `push-backend-cloudrun-dev.yaml` ✅ | push **non-`main`** (CEL) + `backend/**` etc. | BuildKit → deploy `cyber-databrew-backend-dev` (flags match live dev: VPC, 512Mi, …) |
| `push-backend-cloudrun-prod.yaml` ✅ | push **`main`** + paths | BuildKit → deploy `cyber-databrew-backend-prod` **if** service exists (same runtime shape as dev deploy; tune after prod exists) |
| `push-frontend-cloudrun-dev.yaml` ✅ | push **non-`main`** (CEL) + `Frontend/**` etc. | BuildKit → deploy `cyber-databrew-frontend-dev` (Cloud Run) |
| `push-frontend-cloudrun-prod.yaml` ✅ | push **`main`** + paths | BuildKit → deploy `cyber-databrew-frontend-prod` **if** service exists; else image-only |
| `promote-backend.yaml` (TODO) | merge → main | retag `:dev-latest` + `:latest`, kubectl patch |
| `build-frontend.yaml` (TODO) | PR `Frontend/**` | same as backend |
| `promote-frontend.yaml` (TODO) | merge → main | same as backend |

## Prerequisites

- PAC controller (`pipelines-as-code` ns) ✅ already running on the cluster
- Reusable tasks in `tekton-pipelines` ns ✅ (`buildkit`, `retag-image`, `git-clone`)
- IaC ServiceAccount **TODO**: ask `cyber-iac` to provision
  `cyber-databrew-iac-dev` (and `-prod`) with:
  - `roles/storage.objectAdmin` on `terraform_staging_state_store`
  - project-level roles needed by each stack (storage.admin for buckets,
    compute.admin for gateway, iam.serviceAccountAdmin for service-identity)
- GitHub App installation on the repo so PAC can receive webhooks

## Feishu (Lark) — PAC completion notify

Several Cloud Run PAC pipelines end with a `finally` task `notify-feishu-open`.

If the Kubernetes Secret is missing or incomplete, the task prints `SKIP` and exits 0
so the PipelineRun still succeeds.

### Option A — Custom bot webhook (simplest)

In the target Feishu group, add a **custom bot** and copy the **Webhook URL** (one long
HTTPS URL, often containing `/open-apis/bot/v2/hook/...`). That URL is the credential:
anyone with it can post to the chat, so store it only in the cluster Secret.

Create (or update) in namespace `tekton-pipelines`:

| Key | Meaning |
|-----|---------|
| `webhook_url` | Full webhook URL from the group custom bot settings |

Example (replace the URL; do not commit real values):

```bash
kubectl create secret generic feishu-open-notify -n tekton-pipelines \
  --from-literal=webhook_url='https://open.feishu.cn/open-apis/bot/v2/hook/xxxxxxxx' \
  --dry-run=client -o yaml | kubectl apply -f -
```

You do **not** need a separate “bot id” in the Secret when using this path: the hook
URL already identifies the bot and chat.

### Option B — Open Platform app (tenant token + IM)

Uses `tenant_access_token` then `POST /open-apis/im/v1/messages` with
`receive_id_type=chat_id`.

| Key | Meaning |
|-----|---------|
| `app_id` | Feishu app ID from the developer console |
| `app_secret` | App secret (treat as credential; rotate if leaked) |
| `chat_id` | Group chat ID for `receive_id_type=chat_id` (often `oc_...`) |

Example (replace placeholders; do not commit real values):

```bash
kubectl create secret generic feishu-open-notify -n tekton-pipelines \
  --from-literal=app_id='YOUR_APP_ID' \
  --from-literal=app_secret='YOUR_APP_SECRET' \
  --from-literal=chat_id='oc_xxxxxxxx' \
  --dry-run=client -o yaml | kubectl apply -f -
```

If **both** `webhook_url` and the three app keys are present, **`webhook_url` wins**.

### App setup (Option B only, high level)

1. In the Feishu Open Platform, create a **custom bot / enterprise app** and enable
   **bot** capabilities as required by your tenant policy.
2. Grant API scopes needed to **send messages to chats** the bot is in (e.g. message
   send / IM scopes; exact names depend on the console version).
3. **Publish / install** the app to your tenant and **add the bot to the target group**.
4. Obtain the group `chat_id` (`oc_...`) for the Secret above.

Pipelines that include this notifier: `push-backend-cloudrun-dev.yaml`,
`push-backend-cloudrun-prod.yaml`, `push-frontend-cloudrun-dev.yaml`,
`push-frontend-cloudrun-prod.yaml`, `deploy-cloudrun-dev.yaml`. Each passes a fixed
`pipeline-label` in the message body so you can tell which definition fired.

Each definition also sets `notify-deploy-line`: a short Chinese line that spells out
what **aggregate `Succeeded`** means for **Cloud Run** in that pipeline (service
name, region, and prod “service may not exist yet” behavior). The Feishu text is
**not** a second webhook: it is one message that includes both pipeline status and
deploy semantics.

## Tag taxonomy (when build pipelines land)

| Tag | Set by | Retention |
|---|---|---|
| `:<sha>` | every build | recent 5 / 30 days |
| `:pr-{n}` | PR build | recent 5 / 30 days |
| `:dev-latest` | merge to main | permanent |
| `:latest` | merge to main | permanent |
| `:prod-latest` / `:prod-prev-N` | prod promote | permanent |

Reference: tekton-playground `docs/implementation-guide.md §3`.
