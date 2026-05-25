#!/usr/bin/env bash
# Fallback load sampler using Apache Bench (macOS: /usr/sbin/ab).
# Safe defaults; override BASE / TOKEN / N / C for stronger runs.
set -euo pipefail

BASE="${BASE:-http://localhost:8080}"
TOKEN="${TOKEN:-dev-token}"
N="${N:-200}"
C="${C:-20}"

hdr=( -H "X-Databrew-Token: ${TOKEN}" -H "X-Request-ID: ab-mix-$$" )

run_ab() {
	local name="$1"
	local path="$2"
	local auth="$3"
	echo "=== ${name} (${N} reqs, concurrency ${C}) ==="
	if [[ "${auth}" == "yes" ]]; then
		/usr/sbin/ab -n "${N}" -c "${C}" "${hdr[@]}" "${BASE}${path}" | grep -E '^(Complete|Failed|Requests per second|Time taken for|Document Length|Concurrency Level|Non-2xx)' || true
	else
		/usr/sbin/ab -n "${N}" -c "${C}" "${BASE}${path}" | grep -E '^(Complete|Failed|Requests per second|Time taken for|Document Length|Concurrency Level|Non-2xx)' || true
	fi
	echo
}

run_ab "healthz" "/healthz" "no"
run_ab "assets" "/api/v1/assets?page=1&page_size=10" "yes"
run_ab "search" "/api/v1/search/assets?page=1&page_size=5" "yes"
run_ab "lakehouse_status" "/api/v1/lakehouse/status" "yes"
