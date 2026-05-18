#!/usr/bin/env bash
set -euo pipefail

# IaC-like bootstrap for cyber-databrew gateway exposure.
# Mirrors the cyber-iac flow:
# 1) Certificate Manager DNS authorization + certificate + cert-map entry
# 2) Print Cloudflare records (A + ACME CNAME) that must exist
# 3) Apply Gateway HTTPRoute / ReferenceGrant / GCPBackendPolicy
#
# Usage:
#   ./deploy/k8s/gateway/bootstrap-like-iac.sh
#
# Optional env:
#   PROJECT_ID=green-valley-442103
#   APP_NAMESPACE=cyber-databrew-dev
#   GATEWAY_NAMESPACE=developer-gateway
#   GATEWAY_NAME=developer-gateway
#   CERT_MAP_NAME=developer-gateway-cert-map
#   FRONTEND_DOMAIN=cyber-databrew-dev.cyberorigin.ai
#   API_DOMAIN=api-cyber-databrew-dev.cyberorigin.ai
#   IAP_CLIENT_ID=<oauth-client-id>
#   IAP_CLIENT_SECRET=<oauth-client-secret>
#   SOURCE_IAP_SECRET_NAMESPACE=video-proc-dev
#   SOURCE_IAP_SECRET_NAME=iap-oauth-dagster-dev

PROJECT_ID="${PROJECT_ID:-green-valley-442103}"
APP_NAMESPACE="${APP_NAMESPACE:-cyber-databrew-dev}"
GATEWAY_NAMESPACE="${GATEWAY_NAMESPACE:-developer-gateway}"
GATEWAY_NAME="${GATEWAY_NAME:-developer-gateway}"
CERT_MAP_NAME="${CERT_MAP_NAME:-developer-gateway-cert-map}"
FRONTEND_DOMAIN="${FRONTEND_DOMAIN:-cyber-databrew-dev.cyberorigin.ai}"
API_DOMAIN="${API_DOMAIN:-api-cyber-databrew-dev.cyberorigin.ai}"

SOURCE_IAP_SECRET_NAMESPACE="${SOURCE_IAP_SECRET_NAMESPACE:-video-proc-dev}"
SOURCE_IAP_SECRET_NAME="${SOURCE_IAP_SECRET_NAME:-iap-oauth-dagster-dev}"

FRONTEND_AUTH_NAME="developer-gateway-cyber-databrew-dev-dns-auth"
API_AUTH_NAME="developer-gateway-cyber-databrew-dev-api-dns-auth"
FRONTEND_CERT_NAME="developer-gateway-cyber-databrew-dev-cert"
API_CERT_NAME="developer-gateway-cyber-databrew-dev-api-cert"
FRONTEND_ENTRY_NAME="developer-gateway-cyber-databrew-dev-entry"
API_ENTRY_NAME="developer-gateway-cyber-databrew-dev-api-entry"

ensure_dns_auth() {
  local name="$1"
  local domain="$2"
  if gcloud certificate-manager dns-authorizations describe "$name" --location=global --project="$PROJECT_ID" >/dev/null 2>&1; then
    echo "dns-authorization exists: $name"
  else
    echo "creating dns-authorization: $name ($domain)"
    gcloud certificate-manager dns-authorizations create "$name" \
      --domain="$domain" \
      --location=global \
      --project="$PROJECT_ID" >/dev/null
  fi
}

ensure_cert() {
  local name="$1"
  local domain="$2"
  local auth_name="$3"
  if gcloud certificate-manager certificates describe "$name" --location=global --project="$PROJECT_ID" >/dev/null 2>&1; then
    echo "certificate exists: $name"
  else
    echo "creating certificate: $name ($domain)"
    gcloud certificate-manager certificates create "$name" \
      --domains="$domain" \
      --dns-authorizations="$auth_name" \
      --location=global \
      --project="$PROJECT_ID" >/dev/null
  fi
}

ensure_cert_map_entry() {
  local entry="$1"
  local hostname="$2"
  local cert="$3"
  if gcloud certificate-manager maps entries describe "$entry" --map="$CERT_MAP_NAME" --location=global --project="$PROJECT_ID" >/dev/null 2>&1; then
    echo "cert-map entry exists: $entry"
  else
    echo "creating cert-map entry: $entry ($hostname)"
    gcloud certificate-manager maps entries create "$entry" \
      --map="$CERT_MAP_NAME" \
      --hostname="$hostname" \
      --certificates="$cert" \
      --location=global \
      --project="$PROJECT_ID" >/dev/null
  fi
}

echo "==> Ensuring Certificate Manager resources"
ensure_dns_auth "$FRONTEND_AUTH_NAME" "$FRONTEND_DOMAIN"
ensure_dns_auth "$API_AUTH_NAME" "$API_DOMAIN"
ensure_cert "$FRONTEND_CERT_NAME" "$FRONTEND_DOMAIN" "$FRONTEND_AUTH_NAME"
ensure_cert "$API_CERT_NAME" "$API_DOMAIN" "$API_AUTH_NAME"
ensure_cert_map_entry "$FRONTEND_ENTRY_NAME" "$FRONTEND_DOMAIN" "$FRONTEND_CERT_NAME"
ensure_cert_map_entry "$API_ENTRY_NAME" "$API_DOMAIN" "$API_CERT_NAME"

echo
echo "==> Resolving DNS challenge records + gateway IP"
FRONTEND_ACME_VALUE="$(gcloud certificate-manager dns-authorizations describe "$FRONTEND_AUTH_NAME" --location=global --project="$PROJECT_ID" --format='value(dnsResourceRecord.data)')"
API_ACME_VALUE="$(gcloud certificate-manager dns-authorizations describe "$API_AUTH_NAME" --location=global --project="$PROJECT_ID" --format='value(dnsResourceRecord.data)')"
GATEWAY_IP="$(kubectl -n "$GATEWAY_NAMESPACE" get gateway "$GATEWAY_NAME" -o jsonpath='{.status.addresses[0].value}')"

cat <<EOF
Cloudflare records required (apply in zone cyberorigin.ai):

1) A      cyber-databrew-dev                     -> ${GATEWAY_IP}   (proxied ON suggested)
2) CNAME  _acme-challenge.cyber-databrew-dev     -> ${FRONTEND_ACME_VALUE} (proxied OFF)
3) A      api-cyber-databrew-dev                 -> ${GATEWAY_IP}   (proxied ON suggested)
4) CNAME  _acme-challenge.api-cyber-databrew-dev -> ${API_ACME_VALUE} (proxied OFF)
EOF

echo
echo "==> Applying Gateway routes + ReferenceGrant"
kubectl apply -f "$(dirname "$0")/frontend-route.yaml" >/dev/null
kubectl apply -f "$(dirname "$0")/backend-route.yaml" >/dev/null
kubectl apply -f "$(dirname "$0")/reference-grant.yaml" >/dev/null

echo
echo "==> Preparing IAP client ID / secret"
if [[ -z "${IAP_CLIENT_ID:-}" ]]; then
  IAP_CLIENT_ID="$(kubectl get gcpbackendpolicy -A -o jsonpath='{.items[0].spec.default.iap.clientID}')"
fi
if [[ -z "${IAP_CLIENT_SECRET:-}" ]]; then
  IAP_CLIENT_SECRET_B64="$(kubectl -n "$SOURCE_IAP_SECRET_NAMESPACE" get secret "$SOURCE_IAP_SECRET_NAME" -o jsonpath='{.data.key}')"
  IAP_CLIENT_SECRET="$(printf '%s' "$IAP_CLIENT_SECRET_B64" | base64 --decode)"
fi

kubectl -n "$APP_NAMESPACE" create secret generic iap-oauth-cyber-databrew-dev-frontend \
  --from-literal=key="$IAP_CLIENT_SECRET" \
  --dry-run=client -o yaml | kubectl apply -f - >/dev/null
kubectl -n "$APP_NAMESPACE" create secret generic iap-oauth-cyber-databrew-dev-api \
  --from-literal=key="$IAP_CLIENT_SECRET" \
  --dry-run=client -o yaml | kubectl apply -f - >/dev/null

sed "s/REPLACE_WITH_IAP_OAUTH_CLIENT_ID/${IAP_CLIENT_ID}/g" "$(dirname "$0")/frontend-backend-policy.yaml" | kubectl apply -f - >/dev/null
sed "s/REPLACE_WITH_IAP_OAUTH_CLIENT_ID/${IAP_CLIENT_ID}/g" "$(dirname "$0")/backend-backend-policy.yaml" | kubectl apply -f - >/dev/null

echo
echo "==> Status snapshot"
kubectl -n developer-gateway get httproute cyber-databrew-dev-frontend-route cyber-databrew-dev-api-route
kubectl -n "$APP_NAMESPACE" get gcpbackendpolicy
gcloud certificate-manager maps entries list --map="$CERT_MAP_NAME" --project="$PROJECT_ID" | rg 'cyber-databrew-dev|api-cyber-databrew-dev|NAME'

echo
echo "Done."
echo "If cert state is still PENDING/PROVISIONING, wait for Cloudflare DNS propagation and rerun this script."
