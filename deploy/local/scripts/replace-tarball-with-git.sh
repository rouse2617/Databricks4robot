#!/usr/bin/env bash
# Run ON THE VM (e.g. GCE) to replace a tarball tree with git clone. See deploy/local/README.md.
set -euxo pipefail
REPO_URL="${REPO_URL:-https://github.com/CyberOrigin2077/cyber-databrew.git}"
TARGET="${TARGET:-$HOME/cyber-databrew}"

if [ -d "$TARGET/.git" ]; then
  echo "Already a git clone at $TARGET — run: cd $TARGET && git pull && cd deploy/local && sudo docker compose --profile full up -d --build"
  exit 0
fi

export DEBIAN_FRONTEND=noninteractive
sudo apt-get update -y
sudo apt-get install -y git

if [ -d "$TARGET" ]; then
  if [ -f "$TARGET/deploy/local/docker-compose.yml" ]; then
    (cd "$TARGET/deploy/local" && sudo docker compose --profile full down) || true
  fi
  BK="${TARGET}.bak.$(date +%Y%m%d%H%M%S)"
  mv "$TARGET" "$BK"
  echo "Renamed old tree to $BK"
fi

git clone "$REPO_URL" "$TARGET"
cd "$TARGET"

sed -i.bak 's/host.docker.internal:8080/backend:8080/g' deploy/local/monitoring/prometheus/prometheus.yml || true
find deploy/local -name '._*' -delete 2>/dev/null || true

cd deploy/local
sudo docker compose --profile full up -d --build

echo "OK — verify: curl -s http://127.0.0.1:8080/healthz"
