#!/usr/bin/env bash
# Import existing GCP Certificate Manager + Cloudflare DNS resources into
# the local Terraform state for environments/dev. Run this BEFORE the first
# `terraform apply` if those resources were created manually (Console / CLI).
#
# Requires:
#   - gcloud ADC (`gcloud auth application-default login`) for terraform's
#     google provider, but NOT for the import IDs themselves (they are static).
#   - TF_VAR_cloudflare_api_token exported (used both by terraform's cloudflare
#     provider and by this script's curl calls to look up record IDs).
#
# Idempotent-ish: each `terraform import` will fail with "already managed" if
# the resource is already in state — that is fine, the script continues.

set -u
cd "$(dirname "$0")"

# ---- Values (mirror terraform.tfvars; keep in sync if you change names) -----
PROJECT_ID="green-valley-442103"
NAME_PREFIX="developer-gateway-cyber-databrew-dev"
CERT_MAP="developer-gateway-cert-map"
ZONE_NAME="cyberorigin.ai"
FRONTEND_FQDN="cyber-databrew-dev.cyberorigin.ai"
API_FQDN="api-cyber-databrew-dev.cyberorigin.ai"
FRONTEND_ACME_FQDN="_acme-challenge.${FRONTEND_FQDN}"
API_ACME_FQDN="_acme-challenge.${API_FQDN}"

if [[ -z "${TF_VAR_cloudflare_api_token:-}" ]]; then
  echo "ERROR: export TF_VAR_cloudflare_api_token=... before running" >&2
  exit 1
fi
CF_TOKEN="${TF_VAR_cloudflare_api_token}"

run_import() {
  local addr="$1" id="$2"
  echo "=== terraform import $addr"
  echo "    id: $id"
  terraform import "$addr" "$id" || true
}

# ---- 1. GCP Certificate Manager (6 resources) -------------------------------
GCP_BASE="projects/${PROJECT_ID}/locations/global"

run_import \
  "module.gateway_domain.google_certificate_manager_dns_authorization.frontend" \
  "${GCP_BASE}/dnsAuthorizations/${NAME_PREFIX}-dns-auth"

run_import \
  "module.gateway_domain.google_certificate_manager_dns_authorization.api" \
  "${GCP_BASE}/dnsAuthorizations/${NAME_PREFIX}-api-dns-auth"

run_import \
  "module.gateway_domain.google_certificate_manager_certificate.frontend" \
  "${GCP_BASE}/certificates/${NAME_PREFIX}-cert"

run_import \
  "module.gateway_domain.google_certificate_manager_certificate.api" \
  "${GCP_BASE}/certificates/${NAME_PREFIX}-api-cert"

run_import \
  "module.gateway_domain.google_certificate_manager_certificate_map_entry.frontend" \
  "${GCP_BASE}/certificateMaps/${CERT_MAP}/certificateMapEntries/${NAME_PREFIX}-entry"

run_import \
  "module.gateway_domain.google_certificate_manager_certificate_map_entry.api" \
  "${GCP_BASE}/certificateMaps/${CERT_MAP}/certificateMapEntries/${NAME_PREFIX}-api-entry"

# ---- 2. Cloudflare DNS (4 records) ------------------------------------------
echo
echo "=== Looking up Cloudflare zone + record IDs ..."

cf_api() {
  curl -fsS -H "Authorization: Bearer ${CF_TOKEN}" -H 'Content-Type: application/json' "$@"
}

ZONE_ID="$(cf_api "https://api.cloudflare.com/client/v4/zones?name=${ZONE_NAME}" \
  | python3 -c 'import json,sys; print(json.load(sys.stdin)["result"][0]["id"])')"
echo "    zone_id: $ZONE_ID"

lookup_record_id() {
  # $1 = full record name, $2 = type
  cf_api "https://api.cloudflare.com/client/v4/zones/${ZONE_ID}/dns_records?name=$1&type=$2" \
    | python3 -c 'import json,sys
d=json.load(sys.stdin); 
print(d["result"][0]["id"] if d["result"] else "")'
}

FRONTEND_A_ID="$(lookup_record_id "${FRONTEND_FQDN}" A)"
API_A_ID="$(lookup_record_id "${API_FQDN}" A)"
FRONTEND_ACME_ID="$(lookup_record_id "${FRONTEND_ACME_FQDN}" CNAME)"
API_ACME_ID="$(lookup_record_id "${API_ACME_FQDN}" CNAME)"

import_cf() {
  local addr="$1" rid="$2"
  if [[ -z "$rid" ]]; then
    echo "WARN: no Cloudflare record id for $addr — skipping" >&2
    return
  fi
  run_import "$addr" "${ZONE_ID}/${rid}"
}

import_cf "module.gateway_domain.cloudflare_dns_record.frontend_a"           "$FRONTEND_A_ID"
import_cf "module.gateway_domain.cloudflare_dns_record.api_a"                "$API_A_ID"
import_cf "module.gateway_domain.cloudflare_dns_record.frontend_acme_cname"  "$FRONTEND_ACME_ID"
import_cf "module.gateway_domain.cloudflare_dns_record.api_acme_cname"       "$API_ACME_ID"

echo
echo "Done. Now run:  terraform plan"
