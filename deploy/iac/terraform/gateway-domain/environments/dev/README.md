# Dev Environment

## Run

```bash
cd deploy/iac/terraform/gateway-domain/environments/dev
cp terraform.tfvars.example terraform.tfvars
# Fill in terraform.tfvars (project_id, gateway_ip, domains, name_prefix, certificate_map_name).
terraform init
terraform plan
terraform apply
```

Cloudflare token is read from GCP Secret Manager (`cloudflare-api-token` by default via `cloudflare_api_token_secret_name`).
The Terraform runner identity needs `roles/secretmanager.secretAccessor`.

If your identity cannot access Secret Manager, use direct token fallback:

```bash
export TF_VAR_cloudflare_api_token="YOUR_CF_TOKEN"
terraform plan
```

Replace `PROJECT_ID` and resource **names** below if yours differ from `terraform.tfvars.example` (`name_prefix` drives Certificate Manager resource names).

## Existing resources already created manually

If GCP Certificate Manager objects and Cloudflare records were created **outside** Terraform but match this module’s naming, **import** them before the first `apply` so state matches reality.

### One-shot script (recommended)

`import.sh` runs all 6 GCP imports + 4 Cloudflare imports and looks up Cloudflare record IDs automatically.

```bash
gcloud auth application-default login
export TF_VAR_cloudflare_api_token='YOUR_CF_DNS_EDIT_TOKEN'

cd deploy/iac/terraform/gateway-domain/environments/dev
terraform init
./import.sh

terraform plan          # expect: No changes (or only minor TTL drift)
terraform apply
```

Re-running the script is safe: already-imported resources just print `Resource already managed by Terraform` and the script continues.

### Manual one-by-one (if you prefer)

### GCP Certificate Manager

```bash
terraform import 'module.gateway_domain.google_certificate_manager_dns_authorization.frontend' \
  'projects/PROJECT_ID/locations/global/dnsAuthorizations/developer-gateway-cyber-databrew-dev-dns-auth'

terraform import 'module.gateway_domain.google_certificate_manager_dns_authorization.api' \
  'projects/PROJECT_ID/locations/global/dnsAuthorizations/developer-gateway-cyber-databrew-dev-api-dns-auth'

terraform import 'module.gateway_domain.google_certificate_manager_certificate.frontend' \
  'projects/PROJECT_ID/locations/global/certificates/developer-gateway-cyber-databrew-dev-cert'

terraform import 'module.gateway_domain.google_certificate_manager_certificate.api' \
  'projects/PROJECT_ID/locations/global/certificates/developer-gateway-cyber-databrew-dev-api-cert'

terraform import 'module.gateway_domain.google_certificate_manager_certificate_map_entry.frontend' \
  'projects/PROJECT_ID/locations/global/certificateMaps/developer-gateway-cert-map/certificateMapEntries/developer-gateway-cyber-databrew-dev-entry'

terraform import 'module.gateway_domain.google_certificate_manager_certificate_map_entry.api' \
  'projects/PROJECT_ID/locations/global/certificateMaps/developer-gateway-cert-map/certificateMapEntries/developer-gateway-cyber-databrew-dev-api-entry'
```

Use exact IDs from **Google Cloud Console → Certificate Manager** or `gcloud certificate-manager …` if your names differ.

### Cloudflare DNS (`cloudflare_dns_record`, provider v5)

Import ID format: **`ZONE_ID/RECORD_ID`**. Find IDs in the Cloudflare dashboard (DNS record details) or API:

```bash
ZONE_ID='…'   # cyberorigin.ai zone
CF_TOKEN="$(gcloud secrets versions access latest --secret cloudflare-api-token --project PROJECT_ID)"
curl -sS -H "Authorization: Bearer $CF_TOKEN" \
  "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/dns_records?name=cyber-databrew-dev.cyberorigin.ai"
```

Then:

```bash
terraform import 'module.gateway_domain.cloudflare_dns_record.frontend_a'              'ZONE_ID/RECORD_ID'
terraform import 'module.gateway_domain.cloudflare_dns_record.frontend_acme_cname'   'ZONE_ID/RECORD_ID'
terraform import 'module.gateway_domain.cloudflare_dns_record.api_a'                 'ZONE_ID/RECORD_ID'
terraform import 'module.gateway_domain.cloudflare_dns_record.api_acme_cname'        'ZONE_ID/RECORD_ID'
```

After imports, `terraform plan` should show **no** create/destroy for those six GCP resources and four Cloudflare records (only tiny drift fixes like TTL are OK).

### If names do not match

Either align **`name_prefix`** / **`frontend_record_name`** / **`api_record_name`** in `terraform.tfvars` with what you created, or use **`terraform state mv`** / adjust module addresses—avoid duplicate Certificate Manager names in the same project.
