# `deploy/iac/terraform/` — Layer A (Infrastructure)

Terraform-managed infrastructure for `cyber-databrew`. Scope is intentionally
narrow: only resources whose lifecycle is **independent of application
releases** live here. Application Kubernetes manifests stay under
`deploy/k8s/` (Layer B) and are reconciled by `kubectl` / Kustomize
(eventually GitOps).

## Layout

Each "stack" is self-contained (own modules, own state, own provider
configuration). Stacks do **not** import each other; cross-stack references
go through Terraform `output` → consumed via `terraform_remote_state` data
source or copied into K8s manifests.

```
deploy/iac/terraform/
├── service-identity/      # GSA + IAM + Workload Identity (skeleton)
└── <future>/              # gke-cluster, gateway-ingress, ...
```

Within each stack:

```
<stack>/
├── modules/<name>/        # reusable module (no provider config, no backend)
│   ├── main.tf
│   ├── variables.tf
│   └── outputs.tf
└── environments/<env>/    # concrete instantiation per env
    ├── main.tf            # provider + module call
    ├── variables.tf
    ├── outputs.tf
    ├── terraform.tfvars   # gitignored; real values
    └── terraform.tfvars.example
```

## Conventions

- **Provider config lives in `environments/<env>/main.tf`**, never in
  `modules/`. Modules only declare `required_providers`.
- **State is per-stack-per-env, stored in GCS.** Convention follows
  `cyber-iac` org standard:
  - dev → `gs://terraform_staging_state_store/terraform/staging/cyber-databrew/<stack>/`
  - prod → `gs://terraform_production_state_store/terraform/prod/cyber-databrew/<stack>/`
- **Secrets are not in tfvars.** Use Secret Manager + a `data` source
  with optional `TF_VAR_*` override.
- **Existing live resources** are taken over via per-env `import.sh`.
- **Names are stable.** Changing `name_prefix` or any `*_name` variable
  forces resource recreation; treat them as immutable once apply'd.

## Quality gates (aligned with `cyber-iac`)

Repo root carries the same lint/secret-scan toolchain as the org IaC
repo:

- `.tflint.hcl` — TFLint with `tflint-ruleset-google` plugin
- `.gitleaks.toml` / `.gitleaksignore` — gitleaks scanning
- `.secrets.baseline` — detect-secrets baseline (regenerate after
  large changes: `detect-secrets scan --baseline .secrets.baseline`)
- `.pre-commit-config.yaml` — wires the above plus `terraform_fmt` /
  `terraform_validate` / `terraform_tflint`

Bootstrap once per clone:

```bash
brew install pre-commit gitleaks tflint detect-secrets   # or pipx install
pre-commit install
```

After this, every commit auto-runs the hooks against staged files.

## Order of apply (dev bootstrap)

1. `service-identity/` — GSA + IAM + WI bindings consumed by K8s SAs.
2. (Future) `gke-cluster/`, `gateway-ingress/`.

## What is **not** here

- `Deployment` / `Service` / `HTTPRoute` / `GCPBackendPolicy` /
  `HealthCheckPolicy` — see `deploy/k8s/`.
- Image tags, replica counts, env values — Layer B / Kustomize overlays.
- Secret values — written by humans / CI via `gcloud secrets versions add`.
