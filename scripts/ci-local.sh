#!/usr/bin/env bash
# Run a subset of PR CI locally before push (faster feedback than waiting on GitHub).
#
# Usage:
#   scripts/ci-local.sh              # quick (~1–3 min): pre-commit + commitlint
#   scripts/ci-local.sh --full       # + backend go test + frontend lint/build/test
#   scripts/ci-local.sh --pre-commit-only
#
# Requires: pre-commit, node+npx (scripts/commitlint-run.sh), go, npm (for --full).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

MODE="quick"
for arg in "$@"; do
  case "${arg}" in
    --full) MODE="full" ;;
    --pre-commit-only) MODE="pre-commit-only" ;;
    -h|--help)
      sed -n '2,8p' "$0"
      exit 0
      ;;
    *)
      echo "Unknown option: ${arg}" >&2
      exit 2
      ;;
  esac
done

echo "==> ci-local (${MODE})"

if ! command -v pre-commit >/dev/null 2>&1; then
  echo "Install pre-commit: pip install pre-commit==3.7.1" >&2
  exit 1
fi

echo "==> pre-commit (all files, same as CI)"
pre-commit run --all-files --show-diff-on-failure

if [[ "${MODE}" == "pre-commit-only" ]]; then
  echo "OK: pre-commit only"
  exit 0
fi

if [[ -n "${CL_LOCAL_RANGE:-}" ]]; then
  FROM_SHA="${CL_LOCAL_RANGE%..*}"
  TO_SHA="${CL_LOCAL_RANGE#*..}"
  echo "==> commitlint (${CL_LOCAL_RANGE}, pre-push range)"
else
  BASE_REF="${CI_LOCAL_BASE:-origin/main}"
  if ! git rev-parse --verify "${BASE_REF}" >/dev/null 2>&1; then
    BASE_REF="main"
  fi
  if ! git rev-parse --verify "${BASE_REF}" >/dev/null 2>&1; then
    echo "ERROR: no ${BASE_REF} for commitlint. Fetch your base branch, e.g.:" >&2
    echo "  git fetch origin main && git branch -u origin/main main  # or set CI_LOCAL_BASE=origin/your-base" >&2
    exit 1
  fi
  FROM_SHA="$(git merge-base "${BASE_REF}" HEAD)"
  echo "==> commitlint (${FROM_SHA}..HEAD, base ${BASE_REF})"
fi
bash "${ROOT}/scripts/commitlint-run.sh" --from "${FROM_SHA}" --to "${TO_SHA:-HEAD}" --verbose

if [[ "${MODE}" != "full" ]]; then
  echo "OK: quick ci-local passed. For full parity: scripts/ci-local.sh --full"
  exit 0
fi

echo "==> backend (go vet + test)"
(
  cd backend
  go vet ./...
  go test -timeout 300s ./...
)

echo "==> frontend (lint + build + test)"
(
  cd Frontend
  npm run lint
  npm run build
  npm run test
)

echo "OK: full ci-local passed"
