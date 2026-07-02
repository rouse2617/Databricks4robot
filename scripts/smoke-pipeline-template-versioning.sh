#!/usr/bin/env bash
# Smoke test for CYB-1542 pipeline template versioning.
#
# Usage:
#   BASE=http://localhost:8080 TOKEN=dev-token bash scripts/smoke-pipeline-template-versioning.sh
#   RUN_PIPELINE_VERSION_SMOKE_DEPLOY=1 BASE=http://localhost:8080 TOKEN=dev-token bash scripts/smoke-pipeline-template-versioning.sh
#
# The default path avoids submitting an Argo workflow. Set
# RUN_PIPELINE_VERSION_SMOKE_DEPLOY=1 to verify selected-version no-asset run.
set -euo pipefail

BASE="${BASE:-http://localhost:8080}"
TOKEN="${TOKEN:-dev-token}"
NAME="cyb1542-smoke-$(date +%s)"

api() {
	local method="$1"
	local path="$2"
	local body="${3:-}"
	if [[ -n "$body" ]]; then
		curl -sS -X "$method" "$BASE$path" \
			-H "X-Databrew-Token: $TOKEN" \
			-H "Content-Type: application/json" \
			-d "$body"
	else
		curl -sS -X "$method" "$BASE$path" \
			-H "X-Databrew-Token: $TOKEN"
	fi
}

pipeline_json() {
	local version="$1"
	cat <<EOF
{
  "name": "$NAME",
  "nodes": [
    {
      "id": "step-1",
      "component": {
        "name": "v$version step",
        "image": "alpine:3.18",
        "type": "container",
        "source": "custom",
        "command": ["sh", "-c"],
        "args": [{"name": "script", "value": "echo $NAME v$version"}]
      },
      "inputs": [],
      "outputs": []
    }
  ],
  "edges": []
}
EOF
}

save_body() {
	local version="$1"
	printf '{"name":%s,"pipeline":%s}' "$(printf '%s' "$NAME" | jq -R .)" "$(pipeline_json "$version")"
}

v1="$(api POST /api/v1/pipelines "$(save_body 1)")"
v2="$(api POST /api/v1/pipelines "$(save_body 2)")"

v1_id="$(printf '%s' "$v1" | jq -r '.id')"
v2_id="$(printf '%s' "$v2" | jq -r '.id')"
v1_version="$(printf '%s' "$v1" | jq -r '.version')"
v2_version="$(printf '%s' "$v2" | jq -r '.version')"

[[ "$v1_version" == "1" ]] || { echo "expected v1 version=1, got $v1_version" >&2; exit 1; }
[[ "$v2_version" == "2" ]] || { echo "expected v2 version=2, got $v2_version" >&2; exit 1; }

latest="$(api GET /api/v1/pipelines)"
latest_id="$(printf '%s' "$latest" | jq -r --arg name "$NAME" '.items[] | select(.name == $name) | .id' | head -1)"
latest_version="$(printf '%s' "$latest" | jq -r --arg name "$NAME" '.items[] | select(.name == $name) | .version' | head -1)"
[[ "$latest_id" == "$v2_id" ]] || { echo "expected latest id $v2_id, got $latest_id" >&2; exit 1; }
[[ "$latest_version" == "2" ]] || { echo "expected latest version=2, got $latest_version" >&2; exit 1; }

versions="$(api GET "/api/v1/pipelines/$v2_id/versions")"
version_count="$(printf '%s' "$versions" | jq '[.items[].version] | length')"
has_v1="$(printf '%s' "$versions" | jq '[.items[].version] | index(1) != null')"
has_v2="$(printf '%s' "$versions" | jq '[.items[].version] | index(2) != null')"
[[ "$version_count" -ge 2 && "$has_v1" == "true" && "$has_v2" == "true" ]] || {
	echo "expected versions response to include v1 and v2" >&2
	exit 1
}

if [[ "${RUN_PIPELINE_VERSION_SMOKE_DEPLOY:-0}" == "1" ]]; then
	run="$(api POST "/api/v1/pipeline-runs/template/$v2_id" '{"asset_ids":[],"target_id":"default","version":1}')"
	run_template_id="$(printf '%s' "$run" | jq -r '.templateId')"
	run_template_version="$(printf '%s' "$run" | jq -r '.templateVersion')"
	[[ "$run_template_id" == "$v1_id" ]] || { echo "expected run templateId $v1_id, got $run_template_id" >&2; exit 1; }
	[[ "$run_template_version" == "1" ]] || { echo "expected run templateVersion=1, got $run_template_version" >&2; exit 1; }
fi

echo "OK pipeline template versioning smoke: $NAME v1=$v1_id v2=$v2_id"
