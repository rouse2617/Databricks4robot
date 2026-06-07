#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INCLUDE_DOCS="${INCLUDE_DOCS:-1}"
INSTALL_DEPS="${INSTALL_DEPS:-0}"
DEPLOY_WORKER="${DEPLOY_WORKER:-1}"

cd "$ROOT"

install_deps() {
  local dir="$1"
  if [[ -f "$dir/package-lock.json" || -f "$dir/npm-shrinkwrap.json" ]]; then
    (cd "$dir" && npm ci)
  else
    (cd "$dir" && npm install --no-package-lock)
  fi
}

if [[ "$INSTALL_DEPS" == "1" || ! -d Frontend/node_modules ]]; then
  install_deps Frontend
fi

(cd Frontend && npm run build:dev)

rm -rf site
mkdir -p site
cp -R Frontend/dist/. site/

if [[ "$INCLUDE_DOCS" == "1" ]]; then
  if [[ "$INSTALL_DEPS" == "1" || ! -d docs-site/node_modules ]]; then
    install_deps docs-site
  fi
  (cd docs-site && npm run build)
  mkdir -p site/doc
  cp -R docs-site/build/. site/doc/
fi

if [[ "$DEPLOY_WORKER" == "1" ]]; then
  npx wrangler deploy --env dev
else
  echo "DEPLOY_WORKER=0: built dev Worker assets in $ROOT/site"
fi
