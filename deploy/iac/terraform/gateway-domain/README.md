# Gateway Domain IaC (GCP + Cloudflare)

Terraform stack for **GCP Certificate Manager** (DNS authorization → managed TLS certs → entries on an **existing** certificate map) plus matching **Cloudflare** records (`A` to the Gateway IP, grey-cloud `CNAME` for ACME). Kubernetes **Gateway / HTTPRoute / IAP** stays in `deploy/k8s/gateway/` (or your central `cyber-iac`); this stack only covers **DNS + certs**.

## Structure

- `modules/gateway_domain/`: reusable module
  - GCP Certificate Manager (`dns_authorization`, `certificate`, `certificate_map_entry`)
  - Cloudflare DNS (`cloudflare_dns_record`: `A`, `_acme-challenge` `CNAME`)
- `environments/dev/`: dev configuration (`terraform.tfvars`)

## Usage (dev)

```bash
cd deploy/iac/terraform/gateway-domain/environments/dev
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars: project_id, certificate_map_name, gateway_ip, domains, record names, name_prefix.
gcloud config set project YOUR_PROJECT_ID   # Certificate Manager uses provider default project

terraform init
terraform plan
terraform apply
```

## Prerequisites

- **GCP**: `certificatemanager.googleapis.com` enabled; IAM to manage Certificate Manager and cert map entries on the shared map (e.g. `developer-gateway-cert-map`).
- **Cloudflare**: API token stored in GCP Secret Manager (default secret name `cloudflare-api-token`) with DNS edit on zone `cyberorigin.ai` (or your zone).
- **Cert map** must already exist (created by cluster / `cyber-iac` infra); this module only **adds entries**.

## Notes

- Keep Cloudflare token in Secret Manager (not in git / tfvars); `environments/dev` reads it via `ephemeral.google_secret_manager_secret_version`.
- If **GCP or Cloudflare resources already exist** from manual setup, **import** before apply so Terraform does not try to create duplicates. See `environments/dev/README.md`.
- If **`gateway_ip`** changes, run `terraform apply` again to refresh Cloudflare `A` records.
- **State**: default local `terraform.tfstate`; for teams, configure a GCS `backend` in a follow-up change.

## `terraform init` appears stuck on “Finding … versions”

Usually **slow or blocked access** to **registry.terraform.io** (DNS, firewall, VPN, or regional network). Not a Terraform bug.

- Prefer **`terraform init`** without **`-upgrade`** when `.terraform.lock.hcl` is already committed—resolver does less work and may skip re-downloads.
- Raise timeout: `export TF_REGISTRY_CLIENT_TIMEOUT=300` (seconds).
- See where it blocks: `TF_LOG=INFO terraform init` or `TF_LOG=TRACE`.
- Corporate proxy: `export HTTPS_PROXY=…` / `HTTP_PROXY=…`.
- **China / unstable international egress**: use an official [Terraform registry mirror](https://developer.hashicorp.com/terraform/cli/config/config-file#provider-installation) or a reachable mirror endpoint so provider ZIPs do not pull directly from HashiCorp/Cloudflare CDNs.

**Provider major**: this repo’s module uses **`cloudflare_dns_record`** (Cloudflare provider **v5**). Do not set `cloudflare` to `~> 4.0` in `required_providers`—that forces a different major, conflicts with the lock file, and does not match the module schema.
