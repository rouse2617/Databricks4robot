#!/usr/bin/env bash
set -euo pipefail

PREVIEW_PUBLIC_BASE_URL="${PREVIEW_PUBLIC_BASE_URL:-https://cyber-databrew-dev.cyberorigin.ai}"
PREVIEW_PUBLIC_BASE_URL="${PREVIEW_PUBLIC_BASE_URL%/}"

ref="${1:-HEAD}"
if git rev-parse --is-inside-work-tree >/dev/null 2>&1 && git rev-parse --verify "${ref}^{commit}" >/dev/null 2>&1; then
  sha="$(git rev-parse --short=12 "${ref}^{commit}")"
else
  sha="$(printf '%s' "${ref}" | tr '[:upper:]' '[:lower:]' | tr -c 'a-f0-9' '-' | sed -E 's/^-+//; s/-+$//' | cut -c1-12)"
fi

if [[ ! "${sha}" =~ ^[a-f0-9]{7,12}$ ]]; then
  echo "usage: $0 [git-ref-or-sha]" >&2
  echo "error: could not derive a 7-12 char git sha from: ${ref}" >&2
  exit 2
fi

api_url="${PREVIEW_PUBLIC_BASE_URL}/preview/${sha}/api"
web_url="${PREVIEW_PUBLIC_BASE_URL}/preview/${sha}/"

cat <<EOF
preview_id: ${sha}
frontend:   ${web_url}
backend:    ${api_url}

local frontend:
  cd Frontend && VITE_API_BASE_URL=${api_url}/v1 npm run dev
EOF
