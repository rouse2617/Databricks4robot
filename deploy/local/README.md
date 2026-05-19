# Local Docker Compose

There is **one** Compose file: **`docker-compose.yml`**. Optional stacks are selected with **[profiles](https://docs.docker.com/compose/how-tos/profiles/)** (Compose v2+).

## Local UI + API without rebuilding Docker images (recommended day-to-day)

For frontend or MCAP preview work you usually **do not** need `docker compose build frontend` / `--profile full`.

1. **Backend on the host** (same machine as the browser): from repo root `cd backend && go run ./cmd/server` (or `make run`), listening on **`:8080`** with env pointing at your DB/emulators as usual.
2. **Frontend dev server**: `cd Frontend && npm install && npm run dev` → Vite serves on **`:5173`** and proxies **`/api`** to `http://localhost:8080` (see `Frontend/vite.config.ts`; override with `VITE_API_BASE_URL` if needed).
3. Run unit checks anytime without containers: `cd Frontend && npm test && npm run build`, and `cd backend && go test ./...`.

Use Compose **`--profile full`** when you need the **nginx-packaged** frontend container, Elasticsearch, lakehouse, Grafana/Prometheus, etc.—not for every code/test iteration.

## Commands (from repo root)

| Goal | Command |
|------|---------|
| Postgres + PgBouncer + legacy emulators | `make dev-up` → `docker compose up -d` |
| Tear down **everything** in this file (base + profiles) | `make dev-down` |
| Full product stack (backend, frontend, ES, lakehouse, Grafana, …) | `make all-up` → `docker compose --profile full up -d --build` |
| Stop full stack only (keep Postgres running) | `make all-down` |
| Lakehouse only (MinIO, Iceberg REST, Spark, BigQuery) | `make iceberg-up` → `docker compose --profile lakehouse up -d` |

### Faster Docker image rebuilds (optional)

- **BuildKit** powers cache mounts in `Frontend/Dockerfile` and `backend/Dockerfile`. `make all-up` sets `DOCKER_BUILDKIT=1`; enable it manually if your Docker engine disables BuildKit.
- **npm registry (slow / China):** `export NPM_REGISTRY=https://registry.npmmirror.com` before `docker compose build` / `make all-up`.
- **Go modules:** `export GOPROXY=https://goproxy.cn,direct` (or keep default `proxy.golang.org`).
- **Day-to-day:** rebuild only what changed, e.g. `cd deploy/local && docker compose build frontend` — avoid `--no-cache` unless you need a cold install.

## Ports (typical)

| Service | Host port |
|---------|-----------|
| Postgres | 5432 |
| PgBouncer | 6432 |
| Backend (profile `full`) | 8080 |
| Frontend (profile `full`) | 5173 |
| BigQuery (profiles `full` / `lakehouse`) | 8082 |
| Elasticsearch (profile `full`) | 9200 |
| Iceberg REST (profiles `full` / `lakehouse`) | 8183 |
| Spark UI | 8083 |
| Spark Jupyter | 8889 |

## macOS → Linux tarball (BigQuery / bind mounts)

On macOS, `tar` can add AppleDouble sidecar files such as `._iceberg.properties` under `bigquery/catalog/`. BigQuery treats **every file** in that directory as a catalog; those `._*` files then break startup (`does not contain connector.name`). When copying `deploy/local` to a Linux VM:

- Prefer `COPYFILE_DISABLE=1 tar ...` when creating the archive, **or**
- On the target host before `docker compose up`: `find deploy/local -name '._*' -delete`

## Remote VM: Git clone + update (recommended)

Use a normal Git remote on the VM so **`git pull` matches whatever you pushed from your laptop**, instead of copying opaque tarballs.

### First-time setup (Linux VM, e.g. GCE)

1. Install Docker Engine + Compose (see repo `tmp/gcp-vm-docker-bootstrap.sh` as an example userdata script).

2. Authenticate Git (pick one):

   - **SSH**: add the VM’s `~/.ssh/id_ed25519.pub` (or a deploy key read-only) to your Git host.
   - **HTTPS**: `git clone https://...` and use a credential helper / PAT when prompted.

3. Clone into the same path the Compose file expects relative to `backend/` and `Frontend/`:

   ```bash
   cd ~
   git clone <YOUR_REPO_URL> cyber-databrew
   cd cyber-databrew
   ```

4. **One-time on Linux** (Prometheus in Compose scrapes the backend container; the checked-in config uses `host.docker.internal` for host-run backends — see `backend/README.md`). On a VM running the **full Compose stack**, point Prometheus at the service name:

   ```bash
   sed -i.bak 's/host.docker.internal:8080/backend:8080/g' deploy/local/monitoring/prometheus/prometheus.yml
   ```

5. Strip any stray AppleDouble files (harmless if empty), then start the stack:

   ```bash
   find deploy/local -name '._*' -delete 2>/dev/null || true
   cd deploy/local
   sudo docker compose --profile full up -d --build
   ```

### Update after you push new code

```bash
cd ~/cyber-databrew
git fetch origin
git pull --ff-only   # or: git pull --rebase origin main
find deploy/local -name '._*' -delete 2>/dev/null || true
cd deploy/local
sudo docker compose --profile full up -d --build
```

Rebuild is required when `backend/` or `Frontend/` (or image bases) change; config-only edits may sometimes work with `docker compose up -d` without `--build` — when in doubt, use `--build`.

### Replacing an old tarball tree on the VM

Use this when `~/cyber-databrew` exists **without** a `.git` directory (e.g. from `scp`/`tar`). Goal: keep **Docker named volumes** (Postgres data, ES indices, …) by default — only replace the **checkout on disk**.

**Repository** (this project): `https://github.com/CyberOrigin2077/cyber-databrew.git` — swap if you use a fork or SSH remote.

**Optional automation**: [`deploy/local/scripts/replace-tarball-with-git.sh`](scripts/replace-tarball-with-git.sh) performs the same flow (stop stack, backup directory, `git clone`, sed, `find`, `compose up --build`). Copy the script to the VM if the repo is not there yet, `chmod +x`, set `REPO_URL` if needed, run from `~`.

Or follow the manual steps:

1. **SSH into the VM**, then stop the Compose stack **without** `-v` (so volumes stay):

   ```bash
   cd ~/cyber-databrew/deploy/local
   sudo docker compose --profile full down
   ```

2. **Rename** the old tree (recoverable), **clone** fresh into the same path:

   ```bash
   cd ~
   mv cyber-databrew "cyber-databrew.bak.$(date +%Y%m%d%H%M%S)"
   git clone https://github.com/CyberOrigin2077/cyber-databrew.git cyber-databrew
   cd cyber-databrew
   ```

3. **One-time Linux Prometheus scrape fix** (same as §First-time setup step 4):

   ```bash
   sed -i.bak 's/host.docker.internal:8080/backend:8080/g' deploy/local/monitoring/prometheus/prometheus.yml
   ```

4. **Strip AppleDouble junk** (safe if none exist), **rebuild and start**:

   ```bash
   find deploy/local -name '._*' -delete 2>/dev/null || true
   cd deploy/local
   sudo docker compose --profile full up -d --build
   ```

5. **Verify**: `curl -s http://127.0.0.1:8080/healthz` on the VM; from your laptop against the VM public IP if firewall allows.

**Private repo**: use SSH clone (`git@github.com:…`) after installing a deploy key on the VM, or HTTPS with a [PAT](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token).

**Reset DB/data intentionally**: only then use `sudo docker compose --profile full down -v` **before** clone — **destructive**, removes named volumes.

## Related docs

- [`iceberg/README.md`](iceberg/README.md) — notebooks / smoke scripts
- [`loadtest/README.md`](loadtest/README.md) — k6 / `ab` API smoke, soak (default 10m plateau), Docker example
